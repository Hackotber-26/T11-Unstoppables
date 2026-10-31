package bays

import (
	"errors"
	"math"
	"sort"
)

type Bay struct {
	ID           string
	StationID    string
	Capacity     int
	QueueLength  int
	WalkingMeters float64
	Accessible   bool
	Open         bool
}

type Request struct {
	StationID          string
	PartySize          int
	MaxWalkMeters      float64
	RequiresAccessible bool
}

func Assign(request Request, available []Bay) (Bay, error) {
	if request.StationID == "" || request.PartySize < 1 || request.MaxWalkMeters < 0 || math.IsNaN(request.MaxWalkMeters) || math.IsInf(request.MaxWalkMeters, 0) {
		return Bay{}, errors.New("invalid bay assignment request")
	}
	candidates := make([]Bay, 0, len(available))
	for _, bay := range available {
		if bay.ID == "" || bay.StationID != request.StationID || !bay.Open || bay.Capacity < request.PartySize || bay.QueueLength < 0 || bay.WalkingMeters < 0 || math.IsNaN(bay.WalkingMeters) || math.IsInf(bay.WalkingMeters, 0) || bay.WalkingMeters > request.MaxWalkMeters {
			continue
		}
		if request.RequiresAccessible && !bay.Accessible {
			continue
		}
		candidates = append(candidates, bay)
	}
	if len(candidates) == 0 {
		return Bay{}, errors.New("no suitable open bay")
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i].WalkingMeters + float64(candidates[i].QueueLength)*25
		right := candidates[j].WalkingMeters + float64(candidates[j].QueueLength)*25
		if math.Abs(left-right) < 0.001 {
			return candidates[i].ID < candidates[j].ID
		}
		return left < right
	})
	return candidates[0], nil
}