package metro

import (
	"testing"
	"time"
)

func TestNextDeparturesFiltersAndOrders(t *testing.T) {
	now := time.Date(2026, 10, 31, 8, 0, 0, 0, time.UTC)
	departures, err := NextDepartures([]Departure{
		{StationID: "s1", LineID: "red", DepartsAt: now.Add(5 * time.Minute)},
		{StationID: "s1", LineID: "red", DepartsAt: now.Add(time.Minute)},
		{StationID: "s2", LineID: "red", DepartsAt: now},
	}, "s1", "red", now, 2)
	if err != nil || len(departures) != 2 || !departures[0].DepartsAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("unexpected departures: %+v, %v", departures, err)
	}
}