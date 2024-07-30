package hotlist

import (
	"context"
	"fmt"
	"profiteeringway/lib/postgres"
	"profiteeringway/lib/universalis"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type Hotlist struct {
	Name          string
	ItemIDs       []int
	PollFrequency time.Duration
	WorldIDs      []int
}

type HotlistHub struct {
	ConfiguredHotlists map[string]*Hotlist
	enabledHotlists    map[string]struct{}
	resultChan         chan *timerResult
	stopChans          map[string]chan struct{}
	cleanupChan        chan struct{}
	universalisLimiter *rate.Limiter
	postgresLimiter    *rate.Limiter
	pg                 *postgres.Postgres
}

type timerResult struct {
	success bool
	message string
	err     error
}

func NewHotlistHub(db *postgres.Postgres) *HotlistHub {
	cleanupChan := make(chan struct{})
	resultChan := make(chan *timerResult)

	// Set up HTTP rate limiting to 5 qps.
	var five_qps rate.Limit = 5.0
	// Postgres write limit to 20 qps.
	var twenty_qps rate.Limit = 20.0
	l := rate.NewLimiter(five_qps, 2)
	pgl := rate.NewLimiter(twenty_qps, 10)

	go func(resultChan chan *timerResult, cleanupChan chan struct{}) {
		done := false
		for {
			if done {
				break
			}
			var result *timerResult
			select {
			case result = <-resultChan:
				if result.err != nil {
					fmt.Printf("failed to poll with error %s: %s", result.message, result.err)
				} else if !result.success {
					fmt.Printf("failed to poll with details %s", result.message)
				} else {
					fmt.Printf("successfully polled %s", result.message)
				}
			case <-cleanupChan:
				for {
					// Ensure resultChan is closed before cleaning up this goroutine.
					if _, ok := <-resultChan; ok {
						done = true
					}
				}
			}
		}
	}(resultChan, cleanupChan)

	return &HotlistHub{
		ConfiguredHotlists: map[string]*Hotlist{},
		enabledHotlists:    map[string]struct{}{},
		resultChan:         resultChan,
		stopChans:          map[string]chan struct{}{},
		cleanupChan:        cleanupChan,
		universalisLimiter: l,
		postgresLimiter:    pgl,
		pg:                 db,
	}
}

func (h *HotlistHub) BeginPollingAll() error {
	for name, hl := range h.ConfiguredHotlists {
		if _, ok := h.enabledHotlists[name]; ok {
			continue
		}

		stopChan := make(chan struct{})

		go func(hotlist *Hotlist, result chan *timerResult, stopSignal chan struct{}) {
			ctx := context.Background()
			duration, err := time.ParseDuration("5s")
			if err != nil {
				panic("failed to parse duration when initializing timer")
			}
			fetchSignal := time.After(duration)
			done := false
			for {
				if done {
					break
				}
				select {
				case <-fetchSignal:
					successCount := 0
					var sb strings.Builder
					for _, worldID := range hotlist.WorldIDs {
						if err := h.universalisLimiter.Wait(ctx); err != nil {
							sb.WriteString(fmt.Sprintf("error universalis rate limiting %s", err))
							continue
						}
						marketData, err := universalis.GetItemData(worldID, hotlist.ItemIDs)
						successCount += 1
						if err != nil {
							sb.WriteString(fmt.Sprintf("error getting data %s", err))
							continue
						}
						if err := h.postgresLimiter.Wait(ctx); err != nil {
							sb.WriteString(fmt.Sprintf("error postgres rate limiting %s", err))
							continue
						}
						if err := h.pg.WriteUniversalisPriceData(ctx, marketData); err != nil {
							sb.WriteString(fmt.Sprintf("error writing to postgres %s", err))
							continue
						}
						successCount += 1
						sb.WriteString(fmt.Sprintf("successfully wrote for hotlist %s", hotlist.Name))
					}

					// Reset the timer.
					fetchSignal = time.After(hotlist.PollFrequency)
				case <-stopSignal:
					done = true
				}
			}
		}(hl, h.resultChan, stopChan)

		h.stopChans[name] = stopChan
		h.enabledHotlists[name] = struct{}{}
	}
	return nil
}

func (h *HotlistHub) CleanUp() error {
	h.cleanupChan <- struct{}{}
	for _, stopChan := range h.stopChans {
		stopChan <- struct{}{}
	}
	close(h.resultChan)
	return nil
}
