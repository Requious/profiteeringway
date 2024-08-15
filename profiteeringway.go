package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"profiteeringway/lib/discord"
	"profiteeringway/lib/hotlist"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	HotlistDawntrailMateria                 = "Dawntrail Materia"
	HotlistDawntrailConsumables             = "Dawntrail Consumables"
	HotlistDawntrailTierOneCraftedEquipment = "Dawntrail Tier One Crafted Equipment"
	HotlistDawntrailMaterialsSetOne         = "Dawntrail Materials (Set One)"
	HotlistDawntrailMaterialsSetTwo         = "Dawntrail Materials (Set Two)"
	HotlistCrystals                         = "Crystals"
)

func loggerInit(production bool) (*zap.Logger, zap.AtomicLevel, error) {
	if production {
		config := zap.NewProductionConfig()
		config.OutputPaths = []string{"stdout"}
		zapLogger, err := config.Build()
		return zapLogger, zap.NewAtomicLevel(), err
	}
	// Copied from the BasicConfiguration example.
	rawJSON := []byte(`{
	  "level": "debug",
	  "encoding": "json",
	  "outputPaths": ["stdout", "/tmp/logs"],
	  "errorOutputPaths": ["stderr"],
	  "initialFields": {"application": "profiteeringway"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		return nil, zap.NewAtomicLevelAt(zap.FatalLevel), fmt.Errorf("failed to unmarshal Zap config from json %v", err)
	}
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   "/var/log/profiteeringway/process.log",
		MaxSize:    10, //megabytes
		MaxBackups: 3,
		MaxAge:     14, //days
	})
	atom := zap.NewAtomicLevel()
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		w,
		atom,
	)
	return zap.New(core), atom, nil
}

func makeHotlist(worldIDs []int, itemIDs []int, name string) *hotlist.Hotlist {
	fifteenMinutes, err := time.ParseDuration("15m")
	if err != nil {
		panic(fmt.Sprintf("failed to parse duration: %s", err))
	}

	return &hotlist.Hotlist{
		Name:          name,
		ItemIDs:       itemIDs,
		PollFrequency: fifteenMinutes,
		WorldIDs:      worldIDs,
	}
}

func copyIntSlice(s []int) []int {
	c := make([]int, len(s))

	copy(c, s)

	return c
}

func dawntrailTierOneHotlists(p *postgres.Postgres) ([]*hotlist.Hotlist, error) {
	var ret []*hotlist.Hotlist

	worldIDs, err := p.NorthAmericanWorlds()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	materiaIDs, err := p.DawntrailMateriaIDs()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	consumableIDs, err := p.DawntrailConsumables()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	craftedIDs, err := p.DawntrailTierOneCraftedEquipment()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	materialsOneID, err := p.DawntrailMaterialsSetOne()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	materialsTwoID, err := p.DawntrailMaterialsSetTwo()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	crystalIDs, err := p.AllCrystals()
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materiaIDs, HotlistDawntrailMateria))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), consumableIDs, HotlistDawntrailConsumables))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), craftedIDs, HotlistDawntrailTierOneCraftedEquipment))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materialsOneID, HotlistDawntrailMaterialsSetOne))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materialsTwoID, HotlistDawntrailMaterialsSetTwo))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), crystalIDs, HotlistCrystals))

	return ret, nil
}

func main() {
	bot := flag.Bool("bot", false, "set this to enable bot behavior")
	polling := flag.Bool("polling", false, "set this enable polling behavior")
	production := flag.Bool("production", false, "set this to go to production mode")
	flag.Parse()

	logger, _, err := loggerInit(*production)
	if err != nil {
		panic(fmt.Sprintf("failed to init logger %s", err))
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	sugar.Infow("process init:",
		"bot", *bot,
		"polling", *polling,
		"production", *production)

	pg, err := postgres.NewPostgres(secrets.PostgresConnectionString, sugar)
	defer pg.CleanUp()
	if err != nil {
		fmt.Printf("failed to initialize postgres: %v\n", err)
		return
	}
	pg.InitializePriceTables()

	hub := hotlist.NewHotlistHub(pg, sugar)

	// Universalis polling
	if *polling {
		var hotlists []*hotlist.Hotlist
		hotlists, err = dawntrailTierOneHotlists(pg)
		if err != nil {
			panic(fmt.Sprintf("%s", err))
		}

		hub.ConfiguredHotlists = make(map[string]*hotlist.Hotlist)
		for _, hotlist := range hotlists {
			hub.ConfiguredHotlists[hotlist.Name] = hotlist
		}
		if err := hub.BeginPollingAll(); err != nil {
			panic(fmt.Sprintf("%s", err))
		}
		sugar.Infow("began polling for hotlists",
			"hotlists", hotlists)
	}

	// Discord setup
	if *bot {
		sess, err := discordgo.New(fmt.Sprintf("Bot %s", secrets.DiscordBotToken))
		if err != nil {
			panic(fmt.Sprintf("failed to connect to Discord: %s", err))
		}
		discord := discord.NewDiscord(sess, sugar, pg)
		if err := discord.Initialize(); err != nil {
			panic(fmt.Sprintf("failed to initialize Discord bot user connection: %s", err))
		}
		defer discord.CleanUp()
	}

	sigStopChan := make(chan os.Signal, 1)
	signal.Notify(sigStopChan, syscall.SIGTSTP)
	signal.Notify(sigStopChan, syscall.SIGINT)
	for {
		<-sigStopChan
		hub.CleanUp()
		break
	}
}
