package matching

import (
	"errors"
	"math"
	"sort"
)

type Request struct {
	PartySize          int
	MaxWaitSeconds     int
	MaxDetourMeters    float64
	MaxPickupDeltaSecs int
}

type Candidate struct {
	ID                 string
	AvailableSeats     int
	WaitSeconds        int
	DetourMeters       float64
	PickupDeltaSeconds int
	CorridorSimilarity float64
	SafetyCompatible   bool
}

type Result struct {
	CandidateID string  `json:"candidateId"`
	Eligible    bool    `json:"eligible"`
	Score       float64 `json:"score"`
	Reasons     []string `json:"reasons,omitempty"`
}

func Score(request Request, candidate Candidate) (Result, error) {
	if request.PartySize < 1 || request.MaxWaitSeconds < 1 || request.MaxDetourMeters <= 0 || math.IsNaN(request.MaxDetourMeters) || math.IsInf(request.MaxDetourMeters, 0) || request.MaxPickupDeltaSecs < 1 {
		return Result{}, errors.New("matching constraints must be positive")
	}
	if candidate.ID == "" || candidate.AvailableSeats < 0 || candidate.WaitSeconds < 0 || candidate.DetourMeters < 0 || candidate.PickupDeltaSeconds < 0 || math.IsNaN(candidate.DetourMeters) || math.IsInf(candidate.DetourMeters, 0) || math.IsNaN(candidate.CorridorSimilarity) || math.IsInf(candidate.CorridorSimilarity, 0) || candidate.CorridorSimilarity < 0 || candidate.CorridorSimilarity > 1 {
		return Result{}, errors.New("candidate contains invalid values")
	}
	result := Result{CandidateID: candidate.ID, Eligible: true}
	if !candidate.SafetyCompatible {
		result.Reasons = append(result.Reasons, "safety-incompatible")
	}
	if candidate.AvailableSeats < request.PartySize {
		result.Reasons = append(result.Reasons, "insufficient-capacity")
	}
	if candidate.WaitSeconds > request.MaxWaitSeconds {
		result.Reasons = append(result.Reasons, "wait-limit-exceeded")
	}
	if candidate.DetourMeters > request.MaxDetourMeters {
		result.Reasons = append(result.Reasons, "detour-limit-exceeded")
	}
	if candidate.PickupDeltaSeconds > request.MaxPickupDeltaSecs {
		result.Reasons = append(result.Reasons, "pickup-window-mismatch")
	}
	result.Eligible = len(result.Reasons) == 0
	waitScore := math.Max(0, 1-float64(candidate.WaitSeconds)/float64(request.MaxWaitSeconds))
	detourScore := math.Max(0, 1-candidate.DetourMeters/request.MaxDetourMeters)
	timeScore := math.Max(0, 1-float64(candidate.PickupDeltaSeconds)/float64(request.MaxPickupDeltaSecs))
	result.Score = math.Round((candidate.CorridorSimilarity*0.35+timeScore*0.25+detourScore*0.2+waitScore*0.2)*10000) / 10000
	return result, nil
}

func Rank(request Request, candidates []Candidate) ([]Result, error) {
	results := make([]Result, 0, len(candidates))
	for _, candidate := range candidates {
		result, err := Score(request, candidate)
		if err != nil {
			return nil, err
		}
		if result.Eligible {
			results = append(results, result)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].CandidateID < results[j].CandidateID
		}
		return results[i].Score > results[j].Score
	})
	return results, nil
}