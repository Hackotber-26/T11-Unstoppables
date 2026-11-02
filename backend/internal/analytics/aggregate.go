package analytics

import "errors"

type TripObservation struct {
	WaitSeconds int
	FareCents   int64
	SoloFareCents int64
	Occupancy   int
	Completed   bool
}

type Summary struct {
	TripCount          int     `json:"tripCount"`
	CompletedCount     int     `json:"completedCount"`
	AverageWaitSeconds float64 `json:"averageWaitSeconds"`
	AverageFareCents   float64 `json:"averageFareCents"`
	PoolFormationRate  float64 `json:"poolFormationRate"`
	AverageSavingsCents float64 `json:"averageSavingsCents"`
}

func Aggregate(observations []TripObservation) (Summary, error) {
	var result Summary
	result.TripCount = len(observations)
	if len(observations) == 0 {
		return result, nil
	}
	var waitSum, fareSum, savingsSum int64
	var pooled int
	for _, observation := range observations {
		if observation.WaitSeconds < 0 || observation.FareCents < 0 || observation.SoloFareCents < 0 || observation.Occupancy < 1 {
			return Summary{}, errors.New("trip observation contains invalid values")
		}
		waitSum += int64(observation.WaitSeconds)
		fareSum += observation.FareCents
		if observation.Occupancy > 1 {
			pooled++
			savings := observation.SoloFareCents - observation.FareCents
			if savings > 0 {
				savingsSum += savings
			}
		}
		if observation.Completed {
			result.CompletedCount++
		}
	}
	count := float64(len(observations))
	result.AverageWaitSeconds = float64(waitSum) / count
	result.AverageFareCents = float64(fareSum) / count
	result.AverageSavingsCents = float64(savingsSum) / count
	result.PoolFormationRate = float64(pooled) / count
	return result, nil
}