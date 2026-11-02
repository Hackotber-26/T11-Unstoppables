package safher

import (
	"errors"
	"math"
)

type Outcome string

const (
	Allow       Outcome = "allow"
	Review      Outcome = "manual_review"
	Restrict    Outcome = "restrict"
)

type Context struct {
	DriverVerified       bool
	VehicleVerified      bool
	RequiresAccessible   bool
	AccessibilityMatched bool
	EmergencyReported    bool
	RouteRisk             float64
	MaxRouteRisk          float64
	WalkingMeters         float64
	MaxWalkingMeters      float64
}

type Decision struct {
	Outcome Outcome `json:"outcome"`
	Reasons []string `json:"reasons,omitempty"`
}

func Evaluate(input Context) (Decision, error) {
	if input.RouteRisk < 0 || input.MaxRouteRisk <= 0 || input.WalkingMeters < 0 || input.MaxWalkingMeters < 0 || math.IsNaN(input.RouteRisk) || math.IsInf(input.RouteRisk, 0) || math.IsNaN(input.MaxRouteRisk) || math.IsInf(input.MaxRouteRisk, 0) || math.IsNaN(input.WalkingMeters) || math.IsInf(input.WalkingMeters, 0) || math.IsNaN(input.MaxWalkingMeters) || math.IsInf(input.MaxWalkingMeters, 0) {
		return Decision{}, errors.New("safety policy thresholds must be valid non-negative values")
	}
	decision := Decision{Outcome: Allow}
	if !input.DriverVerified || !input.VehicleVerified {
		decision.Outcome = Restrict
		decision.Reasons = append(decision.Reasons, "driver-or-vehicle-not-verified")
	}
	if input.RequiresAccessible && !input.AccessibilityMatched {
		decision.Outcome = Restrict
		decision.Reasons = append(decision.Reasons, "accessibility-requirement-not-met")
	}
	if input.RouteRisk > input.MaxRouteRisk {
		decision.Outcome = Restrict
		decision.Reasons = append(decision.Reasons, "route-risk-exceeds-policy")
	}
	if input.WalkingMeters > input.MaxWalkingMeters {
		decision.Outcome = Restrict
		decision.Reasons = append(decision.Reasons, "walking-distance-exceeds-preference")
	}
	if input.EmergencyReported {
		decision.Outcome = Review
		decision.Reasons = append(decision.Reasons, "human-safety-review-required")
	}
	return decision, nil
}