package matching

import "testing"

func TestRankFiltersAndOrdersCandidates(t *testing.T) {
	request := Request{PartySize: 1, MaxWaitSeconds: 600, MaxDetourMeters: 2000, MaxPickupDeltaSecs: 900}
	results, err := Rank(request, []Candidate{
		{ID: "lower", AvailableSeats: 2, WaitSeconds: 200, DetourMeters: 800, PickupDeltaSeconds: 200, CorridorSimilarity: 0.8, SafetyCompatible: true},
		{ID: "unsafe", AvailableSeats: 2, WaitSeconds: 10, DetourMeters: 10, PickupDeltaSeconds: 10, CorridorSimilarity: 1, SafetyCompatible: false},
		{ID: "best", AvailableSeats: 2, WaitSeconds: 10, DetourMeters: 10, PickupDeltaSeconds: 10, CorridorSimilarity: 1, SafetyCompatible: true},
	})
	if err != nil || len(results) != 2 || results[0].CandidateID != "best" {
		t.Fatalf("unexpected ranking: %+v, %v", results, err)
	}
}