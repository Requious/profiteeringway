package discord

import (
	"context"
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"

	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/table"
)

func CommandLookup() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		ApplicationID: secrets.DiscordApplicationID,
		Type:          discordgo.ChatApplicationCommand,
		Name:          COMMAND_LOOKUP,
		Description:   "Looks up prices for the specified item. (version 1)",
		Options: []*discordgo.ApplicationCommandOption{
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

func tabularPrintHQ(rows []*postgres.HQPriceRow) (string, string) {
	t := table.NewWriter()
	var itemName string

	t.AppendHeader(table.Row{"World", "Minimum Price (HQ)", "Minimum Price (NQ)"})
	for _, row := range rows {
		if itemName == "" {
			itemName = row.Name
		}
		t.AppendRow(table.Row{
			row.WorldName,
			row.MinPriceHQ,
			row.MinPriceNQ,
		})
	}
	return itemName, t.Render()
}

func tabularPrintNQ(rows []*postgres.NQPriceRow) (string, string) {
	t := table.NewWriter()
	var itemName string

	t.AppendHeader(table.Row{"World", "Minimum Price (NQ)"})
	for _, row := range rows {
		if itemName == "" {
			itemName = row.Name
		}
		t.AppendRow(table.Row{
			row.WorldName,
			row.MinPriceNQ,
		})
	}
	return itemName, t.Render()
}

type expensivePriceTableRow struct {
	worldName  string
	datacenter string
	minPriceHQ int
	minPriceNQ int
}

func tabularPrintExpensive(priceRows []*postgres.AllWorldsPriceRowExpensive) (string, string) {
	itemName := ""

	// We'll do some finicky stuff to preserve sort order from the query.
	// Run through the slice twice, once HQ only and once NQ only; on the NQ run
	// check if we've already seen the worldName, if so, append the NQ price.
	var printRows []*expensivePriceTableRow
	for _, row := range priceRows {
		if itemName == "" {
			itemName = row.Name
		}
		if !row.HighQuality {
			continue
		}
		printRows = append(printRows, &expensivePriceTableRow{
			worldName:  row.WorldName,
			datacenter: row.Datacenter,
			minPriceHQ: row.MinPrice,
		})
	}

	for _, row := range priceRows {
		if row.HighQuality {
			continue
		}
		found := false
		for _, printRow := range printRows {
			if printRow.worldName == row.WorldName {
				found = true

				printRow.minPriceNQ = row.MinPrice
			}
		}
		if found {
			continue
		}
		printRows = append(printRows, &expensivePriceTableRow{
			worldName:  row.WorldName,
			datacenter: row.Datacenter,
			minPriceNQ: row.MinPrice,
		})
	}

	t := table.NewWriter()

	t.AppendHeader(table.Row{"Datacenter", "World", "Price per unit (HQ)", "Price per unit (NQ)"})
	for _, pr := range printRows {
		t.AppendRow(table.Row{
			pr.datacenter,
			pr.worldName,
			pr.minPriceHQ,
			pr.minPriceNQ,
		})
	}
	return itemName, t.Render()
}

func (dc *Discord) handleLookup(ctx context.Context, ic *discordgo.InteractionCreate) {
	commandData := ic.ApplicationCommandData()
	var itemID int
	var itemName string
	for _, option := range commandData.Options {
		optName := option.Name
		switch optName {
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

	// Verified parameters, so ack the message while we compute.
	dc.respondAck(ctx, ic)

	var err error
	var priceData []*postgres.AllWorldsPriceRowExpensive
	if itemID > 0 {
		priceData, err = dc.pg.GetPriceForItemIDExpensive(ctx, itemID)
	} else {
		priceData, err = dc.pg.GetPriceForItemNameExpensive(ctx, itemName)
	}
	if err != nil {
		dc.logger.Errorw(logWithEvent(interactionCreateEventName, "failed to get item prices"),
			"command_name", commandData.Name,
			"database_error", err)
		dc.respondFollowup(ctx, ic, "A database lookup error has occurred. Tell Req to check the logs.")
		return
	}

	if len(priceData) == 0 {
		// Now we're pretty sure the item can't be found.
		dc.respondFollowup(ctx, ic, "No items were found with that lookup.")
		return
	}

	var table string
	itemName, table = tabularPrintExpensive(priceData)
	dc.respondFollowupWithFile(ctx, ic, fmt.Sprintf("Price data for %s:", itemName), table)
}
