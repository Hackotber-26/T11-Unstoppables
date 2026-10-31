package simulation

import "testing"

func TestScenarioIsReproducibleAndCapacityBounded(t *testing.T) {
	scenario := DefaultScenario()
	first, err := RunScenario(scenario)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunScenario(scenario)
	if err != nil || first != second {
		t.Fatalf("scenario is not reproducible: %+v, %+v, %v", first, second, err)
	}
	if first.ServedRiders+first.UnservedRiders != scenario.RiderCount || first.AverageOccupancy > float64(scenario.VehicleCapacity) {
		t.Fatalf("scenario violated capacity accounting: %+v", first)
	}
}

func TestScenarioRejectsInvalidInputs(t *testing.T) {
	scenario := DefaultScenario()
	scenario.PoolDiscountBPS = 10001
	if _, err := RunScenario(scenario); err == nil {
		t.Fatal("expected invalid discount rejection")
	}
}