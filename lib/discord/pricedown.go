package discord

import (
	"context"
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"

	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/v6/table"
)

func CommandPricedown() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		ApplicationID: secrets.DiscordApplicationID,
		Type:          discordgo.ChatApplicationCommand,
		Name:          COMMAND_PRICEDOWN,
		Description:   "Prices crafted items against their ingredient costs on a world. (version 1)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "world_name",
				Description: "The world name corresponding to the world in which to price down the crafted item.",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "item_id",
				Description: "The FFXIV internal item ID for the item in question.",
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "item_name",
				Description: "The name of the item in question (case insensitive).",
			},
		},
	}
}

func (dc *Discord) handlePricedown(ctx context.Context, ic *discordgo.InteractionCreate) {
	commandData := ic.ApplicationCommandData()
	var itemID int
	var itemName, worldName string
	for _, option := range commandData.Options {
		optName := option.Name
		switch optName {
		case "world_name":
			worldName = option.StringValue()
		case "item_id":
			itemID = int(option.IntValue())
		case "item_name":
			itemName = option.StringValue()
		default:
			dc.logger.Warnw(logWithEvent(interactionCreateEventName, "unexpected option name"),
				"command_name", commandData.Name,
				"option_name", optName)
		}
	}

	if itemID == 0 && itemName == "" {
		dc.respondInstant(ctx, ic, "At least one of `item_id` and `item_name` must be provided.")
		return
	}

	if itemName != "" {
		convItemID, err := dc.pg.ConvertItemNameToItemID(ctx, itemName)
		itemID = int(convItemID)
		if err != nil {
			dc.logger.Errorw("failed to get item ID for item",
				"item_name", itemName,
				"error", err)
			dc.respondInstant(ctx, ic, fmt.Sprintf("Failed to find an item for %s.", itemName))
			return
		}
	}

	worldID, err := dc.pg.WorldIDFromWorldName(ctx, worldName)
	if err != nil {
		dc.logger.Errorw("failed to get item ID for item",
			"item_name", itemName,
			"error", err)
		dc.respondInstant(ctx, ic, fmt.Sprintf("Failed to find the world: %s", worldName))
	}

	// Verified parameters, so ack the message while we compute.
	dc.respondAck(ctx, ic)

	if itemID == 0 {
		dc.logger.Errorw("critical error in pricedown command: failed to have item ID for recipe lookup",
			"item_id", itemID,
			"item_name", itemName)
		dc.respondFollowup(ctx, ic, "Critical error occurred, tell Req to check the logs.")
		return
	}

	recipe, err := dc.pg.RecipesDetailsForItemID(ctx, int32(itemID))
	if err != nil {
		dc.logger.Errorw("failed to get recipe details for item",
			"item_id", itemID,
			"error", err)
		dc.respondFollowup(ctx, ic, fmt.Sprintf("Failed to find a recipe for item ID: %v.", itemID))
		return
	}

	type priceForItem struct {
		name       string
		worldName  string
		minPriceNQ int
		minPriceHQ int
	}
	// item_name -> price
	priceMap := make(map[string]*priceForItem)

	waitCount := len(recipe.Ingredients)

	type lookupResult struct {
		foundPrices []*postgres.AllWorldsPriceRowExpensive
		err         error
	}
	resChan := make(chan lookupResult)
	for _, ing := range recipe.Ingredients {
		go func(itemID int32, worldID int) {
			prices, err := dc.pg.GetPriceForItemIDWorldSpecificExpensive(ctx, int(itemID), worldID)
			resChan <- lookupResult{
				foundPrices: prices,
				err:         err,
			}
		}(ing.ItemID, worldID)
	}

	go func(itemID int32, worldID int) {
		prices, err := dc.pg.GetPriceForItemIDWorldSpecificExpensive(ctx, int(itemID), worldID)
		resChan <- lookupResult{
			foundPrices: prices,
			err:         err,
		}
	}(int32(itemID), worldID)

	for i := 0; i < waitCount+1; i++ {
		res := <-resChan
		if res.err != nil {
			dc.logger.Warnf("subquery for pricedown price lookup failed",
				"error", err,
				"crafted_item", recipe.CraftedItemName)
			continue
		}

		for _, fp := range res.foundPrices {
			if _, ok := priceMap[fp.Name]; ok {
				if fp.HighQuality {
					priceMap[fp.Name].minPriceHQ = fp.MinPrice
				} else {
					priceMap[fp.Name].minPriceNQ = fp.MinPrice
				}
			} else {
				pr := &priceForItem{
					name:      fp.Name,
					worldName: fp.WorldName,
				}
				if fp.HighQuality {
					pr.minPriceHQ = fp.MinPrice
				} else {
					pr.minPriceNQ = fp.MinPrice
				}
				priceMap[fp.Name] = pr
			}
		}
	}
	type pricingRow struct {
		itemName     string
		isIngredient bool
		// either ingredient count in the recipe or produced items
		quantity    int
		itemWorld   string
		minPriceNQ  int
		minPriceHQ  int
		missingInfo bool
	}
	var pricingRows []pricingRow

	targetItemPricingRow := pricingRow{
		itemName:     recipe.CraftedItemName,
		isIngredient: false,
		quantity:     int(recipe.CraftedItemCount),
	}
	targetItemPrice, ok := priceMap[recipe.CraftedItemName]
	if ok {
		targetItemPricingRow.minPriceNQ = targetItemPrice.minPriceNQ
		targetItemPricingRow.minPriceHQ = targetItemPrice.minPriceHQ
		targetItemPricingRow.itemWorld = targetItemPrice.worldName
	} else {
		targetItemPricingRow.missingInfo = true
	}
	pricingRows = append(pricingRows, targetItemPricingRow)

	for _, ing := range recipe.Ingredients {
		ingPriceRow := pricingRow{
			itemName:     ing.Name,
			isIngredient: true,
			quantity:     int(ing.Count),
		}
		ingItemPrice, ok := priceMap[ing.Name]
		if ok {
			ingPriceRow.minPriceNQ = ingItemPrice.minPriceNQ
			ingPriceRow.minPriceHQ = ingItemPrice.minPriceHQ
			ingPriceRow.itemWorld = ingItemPrice.worldName
		} else {
			ingPriceRow.missingInfo = true
		}
		pricingRows = append(pricingRows, ingPriceRow)
	}

	saleTotalHQ := 0
	costTotalHQ := 0
	saleTotalNQ := 0
	costTotalNQ := 0
	for _, pr := range pricingRows {
		if pr.isIngredient {
			costTotalHQ += pr.minPriceHQ
			costTotalNQ += pr.minPriceNQ
		} else {
			saleTotalHQ += pr.minPriceHQ
			saleTotalNQ += pr.minPriceNQ
		}
	}
	t := table.NewWriter()

	t.AppendHeader(table.Row{"Item", "World", "Price per unit (HQ)", "Price per unit (NQ)", "Quantity", "Total Price (HQ)", "Total Price (NQ)"})
	for _, pr := range pricingRows {
		if !pr.isIngredient {
			if !pr.missingInfo {
				t.AppendRow(table.Row{
					pr.itemName,
					pr.itemWorld,
					pr.minPriceHQ,
					pr.minPriceNQ,
					pr.quantity,
					pr.quantity * pr.minPriceHQ,
					pr.quantity * pr.minPriceNQ,
				})
			} else {
				t.AppendRow(table.Row{
					pr.itemName,
				})
			}
		}
	}
	t.AppendSeparator()
	for _, pr := range pricingRows {
		if pr.isIngredient {
			if !pr.missingInfo {
				t.AppendRow(table.Row{
					pr.itemName,
					pr.itemWorld,
					pr.minPriceHQ,
					pr.minPriceNQ,
					pr.quantity,
					pr.quantity * pr.minPriceHQ,
					pr.quantity * pr.minPriceNQ,
				})
			} else {
				t.AppendRow(table.Row{
					pr.itemName,
				})
			}
		}
	}

	t.AppendSeparator()

	t.AppendRow(table.Row{
		"Expected profit", "", "", "", "", saleTotalHQ - costTotalHQ, saleTotalNQ - costTotalNQ,
	})
	dc.respondFollowupWithFile(ctx, ic, fmt.Sprintf("Price data for %s:", itemName), t.Render())
}
