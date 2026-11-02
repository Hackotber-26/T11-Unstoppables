package pooling

import (
	"testing"
	"time"
)

func TestFormGroupsByOriginAndWindow(t *testing.T) {
	start := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	groups, err := Form([]Request{
		{ID: "r1", OriginID: "s1", Destination: "north", RequestedAt: start, PartySize: 1, MaxWaitSeconds: 300},
		{ID: "r2", OriginID: "s1", Destination: "south", RequestedAt: start.Add(time.Minute), PartySize: 1, MaxWaitSeconds: 300},
		{ID: "r3", OriginID: "s2", Destination: "north", RequestedAt: start, PartySize: 1, MaxWaitSeconds: 300},
	}, 4, false)
	if err != nil || len(groups) != 1 || len(groups[0].Requests) != 2 {
		t.Fatalf("unexpected pools: %+v, %v", groups, err)
	}
}