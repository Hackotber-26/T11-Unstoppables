package bays

import "testing"

func TestAssignRespectsAccessibilityAndRanksQueue(t *testing.T) {
	bay, err := Assign(Request{StationID: "s1", PartySize: 2, MaxWalkMeters: 500, RequiresAccessible: true}, []Bay{
		{ID: "near-inaccessible", StationID: "s1", Capacity: 4, WalkingMeters: 20, Open: true},
		{ID: "good", StationID: "s1", Capacity: 4, WalkingMeters: 100, Open: true, Accessible: true},
	})
	if err != nil || bay.ID != "good" {
		t.Fatalf("unexpected bay: %+v, %v", bay, err)
	}
}