package hotlist

import "time"

type Hotlist struct {
	Name          string
	ItemIDs       []int32
	PollFrequency time.Time
	WorldIDs      []int32
}
