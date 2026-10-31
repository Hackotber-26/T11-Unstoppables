package analytics

import "testing"

func TestAggregate(t *testing.T) {
	summary, err := Aggregate([]TripObservation{
		{WaitSeconds: 100, FareCents: 500, SoloFareCents: 700, Occupancy: 2, Completed: true},
		{WaitSeconds: 200, FareCents: 600, SoloFareCents: 600, Occupancy: 1},
	})
	if err != nil || summary.PoolFormationRate != 0.5 || summary.AverageWaitSeconds != 150 || summary.AverageSavingsCents != 100 {
		t.Fatalf("unexpected summary: %+v, %v", summary, err)
	}
}