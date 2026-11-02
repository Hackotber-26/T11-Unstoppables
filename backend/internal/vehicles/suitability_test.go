package vehicles

import "testing"

func TestSelectRespectsNeedAndDeadhead(t *testing.T) {
	vehicle, err := Select(Need{PartySize: 2, RequiresAccessible: true}, []Vehicle{
		{ID: "small", Capacity: 1, Available: true, Accessible: true},
		{ID: "near", Capacity: 4, Available: true, Accessible: true, DeadheadMeters: 300},
		{ID: "far", Capacity: 4, Available: true, Accessible: true, DeadheadMeters: 500},
	})
	if err != nil || vehicle.ID != "near" {
		t.Fatalf("unexpected vehicle: %+v, %v", vehicle, err)
	}
}