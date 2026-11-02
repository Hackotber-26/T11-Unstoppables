package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPoolableTripGroupsAllowNearbyDistinctDropoffsWithoutDuplicateRiders(t *testing.T) {
	firstRider, secondRider, thirdRider := uuid.New(), uuid.New(), uuid.New()
	requests := []pendingTrip{
		{ID: uuid.New(), RiderID: firstRider, OriginID: "majestic", Destination: "drop one", CreatedAt: time.Unix(1, 0), PickupLat: 12.9716, PickupLng: 77.5946, DropoffLat: 12.98, DropoffLng: 77.6, PartySize: 1},
		{ID: uuid.New(), RiderID: secondRider, OriginID: "majestic", Destination: "drop two", CreatedAt: time.Unix(2, 0), PickupLat: 12.9717, PickupLng: 77.5947, DropoffLat: 12.99, DropoffLng: 77.61, PartySize: 1},
		{ID: uuid.New(), RiderID: firstRider, OriginID: "majestic", Destination: "drop three", CreatedAt: time.Unix(3, 0), PickupLat: 12.9716, PickupLng: 77.5946, DropoffLat: 12.981, DropoffLng: 77.601, PartySize: 1},
		{ID: uuid.New(), RiderID: thirdRider, OriginID: "indiranagar", CreatedAt: time.Unix(4, 0), PickupLat: 12.9716, PickupLng: 77.5946, DropoffLat: 12.98, DropoffLng: 77.6, PartySize: 1},
	}
	groups := poolableTripGroups(requests)
	var compatibleGroup []pendingTrip
	for _, group := range groups {
		if len(group) == 2 && group[0].OriginID == "majestic" {
			compatibleGroup = group
			break
		}
	}
	if len(groups) != 3 || len(compatibleGroup) != 2 {
		t.Fatalf("unexpected pooled groups: %+v", groups)
	}
	if compatibleGroup[0].RiderID == compatibleGroup[1].RiderID {
		t.Fatal("one rider was added to the same pool more than once")
	}
	if compatibleGroup[0].Destination == compatibleGroup[1].Destination {
		t.Fatal("test requires distinct drop-offs")
	}
}
