package routing

import "testing"

func TestShortestRouteAndETA(t *testing.T) {
	route, err := Shortest([]Edge{
		{From: "A", To: "B", TravelSeconds: 20, DistanceMeters: 100},
		{From: "A", To: "C", TravelSeconds: 5, DistanceMeters: 200},
		{From: "C", To: "B", TravelSeconds: 5, DistanceMeters: 200},
	}, "A", "B")
	if err != nil || route.TravelSeconds != 10 || route.DistanceMeters != 400 || len(route.Nodes) != 3 || route.Nodes[1] != "C" {
		t.Fatalf("unexpected route: %+v, %v", route, err)
	}
	eta, err := EstimateETA(route, 5, 0.2)
	if err != nil || eta.ExpectedSeconds != 15 || eta.LowerSeconds != 12 || eta.UpperSeconds != 18 {
		t.Fatalf("unexpected ETA: %+v, %v", eta, err)
	}
}