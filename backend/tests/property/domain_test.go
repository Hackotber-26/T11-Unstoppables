package property

import (
	"fmt"
	"testing"
	"testing/quick"
	"time"

	"t11-unstoppables/internal/fares"
	"t11-unstoppables/internal/pooling"
)

func TestFareQuoteInvariantsForRandomDistances(t *testing.T) {
	property := func(distance uint32) bool {
		meters := float64(distance % 500001)
		corridor, corridorErr := fares.QuoteFare(meters, "corridor")
		solo, soloErr := fares.QuoteFare(meters, "solo")
		if corridorErr != nil || soloErr != nil {
			return false
		}
		return corridor.TotalCents >= 0 && corridor.TotalCents <= solo.TotalCents && corridor.DiscountCents > 0 && solo.DiscountCents == 0
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 500}); err != nil {
		t.Fatal(err)
	}
}

func TestPoolGroupsNeverExceedSeatCapacity(t *testing.T) {
	property := func(count uint8) bool {
		requests := make([]pooling.Request, 0, int(count%50))
		for index := 0; index < int(count%50); index++ {
			requests = append(requests, pooling.Request{
				ID: fmt.Sprintf("r-%d", index), OriginID: "station-1", Destination: "corridor-a",
				RequestedAt: time.Unix(int64(index), 0), PartySize: index%4 + 1, MaxWaitSeconds: 100,
			})
		}
		groups, err := pooling.Form(requests, 4, false)
		if err != nil {
			return false
		}
		for _, group := range groups {
			seats := 0
			for _, request := range group.Requests {
				seats += request.PartySize
			}
			if seats > 4 {
				return false
			}
		}
		return true
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 250}); err != nil {
		t.Fatal(err)
	}
}
