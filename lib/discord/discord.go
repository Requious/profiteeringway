package discord

import (
	"context"
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jedib0t/go-pretty/table"
	"go.uber.org/zap"
)

const (
	interactionCreateEventName string = "INTERACTION_CREATE"
	COMMAND_LOOKUP             string = "lookup"
)

type Discord struct {
	client         *discordgo.Session
	logger         *zap.SugaredLogger
	updateCommands bool
	pg             *postgres.Postgres
}

func NewDiscord(session *discordgo.Session, logger *zap.SugaredLogger, pg *postgres.Postgres) *Discord {
	return &Discord{
		client:         session,
		logger:         logger,
		updateCommands: false,
		pg:             pg,
	}
}

// Initializes Gateway websocket connection to Discord and supplies
// callbacks with closures to the sugared Zap logger.
func (dc *Discord) Initialize() error {
	// For GUILD_CREATE, detecting when the bot is acked by a Discord server.
	dc.client.Identify.Intents = discordgo.IntentsGuilds

	// Register callbacks for Gateway events.
	dc.client.AddHandlerOnce(dc.ready())
	dc.client.AddHandler(dc.guildCreate())
	dc.client.AddHandler(dc.interactionCreate())

	err := dc.client.Open()
	if err != nil {
		dc.logger.Fatalw("failed to open websocket connection to Discord gateway",
			"suberror", err,
		)
		return err
	}

	return nil
}

func logWithEvent(eventName string, msg string) string {
	return fmt.Sprintf("%s: %s", eventName, msg)
}

// ready is fired once upon session initialization. For now we just log the servers
// the bot is registered in.
func (dc *Discord) ready() func(*discordgo.Session, *discordgo.Ready) {
	readyEventName := "READY"
	return func(s *discordgo.Session, r *discordgo.Ready) {
		dc.logger.Infow(logWithEvent(readyEventName, "initialized Gateway connection"),
			"gateway_version", r.Version,
			"session_id", r.SessionID,
		)
		for _, guild := range r.Guilds {
			dc.logger.Infow(logWithEvent(readyEventName, "registered to guild"),
				"guild_id", guild.ID,
				"guild_name", guild.Name,
			)
		}
	}
}

// guildCreate fires upon a guild coming online - we check for all bot commands here
// and update/create any out of date.
func (dc *Discord) guildCreate() func(*discordgo.Session, *discordgo.GuildCreate) {
	guildCreateEventName := "GUILD_CREATE"
	return func(s *discordgo.Session, g *discordgo.GuildCreate) {
		dc.logger.Infow(logWithEvent(guildCreateEventName, "guild online"),
			"guild_id", g.ID,
			"guild_name", g.Name,
			"guild_unavailable", g.Unavailable,
		)
		if !g.Unavailable && g.ID != "" {
			dc.registerCommands(g.ID)
		}
	}
}

func (dc *Discord) interactionCreate() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, ic *discordgo.InteractionCreate) {
		ctx := context.Background()
		if ic.AppID != secrets.DiscordApplicationID {
			dc.logger.Warnw(logWithEvent(interactionCreateEventName, "invalid associated application_id"),
				"application_id", ic.AppID,
			)
			return
		}
		switch ic.Type {
		case discordgo.InteractionApplicationCommand:
			dc.handleApplicationCommand(ctx, ic)
		default:
			dc.logger.Warnw(logWithEvent(interactionCreateEventName, "received unexpected Interaction"),
				"application_id", ic.AppID,
				"interaction_type", ic.Type.String(),
			)
		}
	}
}

// Registers all application commands with the associated
// guild, skipping any whose descriptions are exact matches
// with the current version of the command.
func (dc *Discord) registerCommands(guildID string) error {
	preexistingCmds, err := dc.client.ApplicationCommands(secrets.DiscordApplicationID, guildID)
	if err != nil {
		dc.logger.Errorw("failed to retrieve preexisting registered application commands",
			"guild_id", guildID,
			"suberror", err,
		)
	}

	var found bool
	var count int
	for _, command := range AllCommands() {
		found = false
		count = 0
		for _, preexistingCmd := range preexistingCmds {
			// Remove duplicates.
			if preexistingCmd.Name == command.Name {
				found = true
				count += 1
				if count > 1 {
					if err := dc.client.ApplicationCommandDelete(secrets.DiscordApplicationID, guildID, preexistingCmd.ID); err != nil {
						dc.logger.Errorw("failed to delete duplicate application command",
							"command", command.Name,
							"command_id", preexistingCmd.ID,
							"guild_id", guildID,
							"suberror", err,
						)
					}
				} else if dc.updateCommands {
					if _, err := dc.client.ApplicationCommandEdit(secrets.DiscordApplicationID, guildID, preexistingCmd.ID, command); err != nil {
						dc.logger.Errorw("failed to update application command",
							"command", command.Name,
							"command_id", preexistingCmd.ID,
							"guild_id", guildID,
							"suberror", err,
						)
					}
					dc.logger.Infow("updated application command",
						"guild_id", guildID,
						"command", command.Name,
						"command_id", preexistingCmd.ID,
					)
				}
				continue
			}
			dc.logger.Infof("preexisting command found",
				"guild_id", guildID,
				"command_id", preexistingCmd.ID,
				"command", preexistingCmd.Name,
				"version", preexistingCmd.Version,
				"description", preexistingCmd.Description,
			)
		}

		if found {
			continue
		}

		if _, err := dc.client.ApplicationCommandCreate(secrets.DiscordApplicationID, guildID, command); err != nil {
			dc.logger.Errorw("failed to create command",
				"guild_id", guildID,
				"command", command.Name,
				"suberror", err,
			)
		}
		dc.logger.Infow("successfully registered command",
			"guild_id", guildID,
			"command", command.Name,
			"version", command.Version,
		)
	}
	return nil
}

func (dc *Discord) CleanUp() {
	dc.client.Close()
}

func interactionFromInteractionCreate(ic *discordgo.InteractionCreate) *discordgo.Interaction {
	return &discordgo.Interaction{
		ID:          ic.ID,
		AppID:       ic.AppID,
		Type:        ic.Type,
		Data:        ic.Data,
		GuildID:     ic.GuildID,
		ChannelID:   ic.ChannelID,
		Message:     ic.Message,
		Member:      ic.Member,
		User:        ic.User,
		Locale:      ic.Locale,
		GuildLocale: ic.GuildLocale,
		Token:       ic.Token,
		Version:     ic.Version,
	}
}

func (dc *Discord) respondInstant(ctx context.Context, ic *discordgo.InteractionCreate, message string) error {
	icInteraction := interactionFromInteractionCreate(ic)
	if err := dc.client.InteractionRespond(icInteraction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
		},
	}); err != nil {
		dc.logger.Errorw(logWithEvent(interactionCreateEventName, "failed to send interaction response"),
			"suberror", err)
		return err
	}
	return nil
}

func (dc *Discord) respondTextFile(ctx context.Context, ic *discordgo.InteractionCreate, message string, text string) error {
	icInteraction := interactionFromInteractionCreate(ic)
	if err := dc.client.InteractionRespond(icInteraction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Files: []*discordgo.File{
				{
					Name:        "response.txt",
					ContentType: "text/plain",
					Reader:      strings.NewReader(text),
				},
			},
		},
	}); err != nil {
		dc.logger.Errorw(logWithEvent(interactionCreateEventName, "failed to send interaction response"),
			"suberror", err)
		return err
	}
	return nil
}

func (dc *Discord) handleApplicationCommand(ctx context.Context, ic *discordgo.InteractionCreate) {
	commandData := ic.ApplicationCommandData()
	switch name := commandData.Name; name {
	case COMMAND_LOOKUP:
		dc.handleLookup(ctx, ic)
	default:
		dc.logger.Warnw(logWithEvent(interactionCreateEventName, "unexpected command received"),
			"command_name", name)
	}
}

func AllCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		CommandLookup(),
	}
}

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

	var hqPriceData []*postgres.HQPriceRow
	var nqPriceData []*postgres.NQPriceRow
	var err error
	if itemID > 0 {
		hqPriceData, err = dc.pg.GetItemPricesFromItemID(ctx, itemID)
	} else {
		hqPriceData, err = dc.pg.GetItemPricesFromItemName(ctx, itemName)
	}
	if err != nil {
		dc.logger.Errorw(logWithEvent(interactionCreateEventName, "failed to get item prices"),
			"command_name", commandData.Name,
			"database_error", err)
		dc.respondInstant(ctx, ic, "A database lookup error has occurred. Tell Req to check the logs.")
		return
	}

	if len(hqPriceData) == 0 {
		// It's possible this item has no HQ variant, so check the NQ only lookup.
		if itemID > 0 {
			nqPriceData, err = dc.pg.GetNQItemPricesFromItemID(ctx, itemID)
		} else {
			nqPriceData, err = dc.pg.GetNQItemPricesFromItemName(ctx, itemName)
		}
		if err != nil {
			dc.logger.Errorw(logWithEvent(interactionCreateEventName, "failed to get item prices"),
				"command_name", commandData.Name,
				"database_error", err)
			dc.respondInstant(ctx, ic, "A database lookup error has occurred. Tell Req to check the logs.")
			return
		}
		if len(nqPriceData) == 0 {
			// Now we're pretty sure the item can't be found.
			dc.respondInstant(ctx, ic, "No items were found with that lookup.")
			return
		}
	}

	var table string
	if len(hqPriceData) > 0 {
		itemName, table = tabularPrintHQ(hqPriceData)
	} else {
		itemName, table = tabularPrintNQ(nqPriceData)
	}
	dc.respondTextFile(ctx, ic, fmt.Sprintf("Price data for %s:", itemName), table)
}
