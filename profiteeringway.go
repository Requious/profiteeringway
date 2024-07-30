package main

import (
	"fmt"
	"os"
	"os/signal"
	"profiteeringway/lib/hotlist"
	"profiteeringway/lib/postgres"
	"profiteeringway/secrets"
	"syscall"
	"time"
)

func MakeHotlist(worldIDs []int, itemIDs []int, name string) *hotlist.Hotlist {
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

func main() {
	var hotlists []*hotlist.Hotlist

	pg, err := postgres.NewPostgres(secrets.PostgresConnectionString)
	defer pg.CleanUp()

	if err != nil {
		fmt.Printf("failed to initialize postgres: %v\n", err)
		return
	}

	hub := hotlist.NewHotlistHub(pg)

	sigStopChan := make(chan os.Signal, 1)
	signal.Notify(sigStopChan, syscall.SIGTSTP)
	for {
		<-sigStopChan
		hub.CleanUp()
		break
	}
}
