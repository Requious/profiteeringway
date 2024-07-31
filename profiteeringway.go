package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"profiteeringway/lib/hotlist"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"
	"syscall"
	"time"

	"go.uber.org/zap"
)

const (
	HotlistDawntrailMateria                 = "Dawntrail Materia"
	HotlistDawntrailConsumables             = "Dawntrail Consumables"
	HotlistDawntrailTierOneCraftedEquipment = "Dawntrail Tier One Crafted Equipment"
	HotlistDawntrailMaterialsSetOne         = "Dawntrail Materials (Set One)"
	HotlistDawntrailMaterialsSetTwo         = "Dawntrail Materials (Set Two)"
)

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

	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materiaIDs, HotlistDawntrailMateria))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), consumableIDs, HotlistDawntrailConsumables))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), craftedIDs, HotlistDawntrailTierOneCraftedEquipment))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materialsOneID, HotlistDawntrailMaterialsSetOne))
	ret = append(ret, makeHotlist(copyIntSlice(worldIDs), materialsTwoID, HotlistDawntrailMaterialsSetTwo))

	return ret, nil
}

func main() {
	botOnly := flag.Bool("bot_only", false, "set this to disable all polling behavior")

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	var hotlists []*hotlist.Hotlist

	pg, err := postgres.NewPostgres(secrets.PostgresConnectionString, sugar)
	defer pg.CleanUp()

	if err != nil {
		fmt.Printf("failed to initialize postgres: %v\n", err)
		return
	}

	hub := hotlist.NewHotlistHub(pg, sugar)

	hotlists, err = dawntrailTierOneHotlists(pg)
	if err != nil {
		panic(fmt.Sprintf("%s", err))
	}

	hub.ConfiguredHotlists = make(map[string]*hotlist.Hotlist)
	for _, hotlist := range hotlists {
		hub.ConfiguredHotlists[hotlist.Name] = hotlist
	}

	if !*botOnly {
		if err := hub.BeginPollingAll(); err != nil {
			panic(fmt.Sprintf("%s", err))
		}
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
