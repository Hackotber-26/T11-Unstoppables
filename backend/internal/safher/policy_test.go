package safher

import "testing"

func TestEvaluateSafetyPolicy(t *testing.T) {
	decision, err := Evaluate(Context{DriverVerified: true, VehicleVerified: true, MaxRouteRisk: 0.5, RouteRisk: 0.2, MaxWalkingMeters: 500, WalkingMeters: 100})
	if err != nil || decision.Outcome != Allow {
		t.Fatalf("unexpected decision: %+v, %v", decision, err)
	}
	decision, err = Evaluate(Context{DriverVerified: true, VehicleVerified: true, MaxRouteRisk: 0.5, RouteRisk: 0.2, MaxWalkingMeters: 500, EmergencyReported: true})
	if err != nil || decision.Outcome != Review {
		t.Fatalf("expected manual review: %+v, %v", decision, err)
	}
}