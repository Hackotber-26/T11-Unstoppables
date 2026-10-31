package vehicles

import (
	"errors"
	"math"
	"sort"
)

type Vehicle struct {
	ID            string
	Capacity      int
	Available     bool
	Accessible    bool
	ServiceType   string
	DeadheadMeters float64
}

type Need struct {
	PartySize          int
	RequiresAccessible bool
	ServiceType        string
}

func Select(need Need, candidates []Vehicle) (Vehicle, error) {
	if need.PartySize < 1 {
		return Vehicle{}, errors.New("party size must be positive")
	}
	filtered := make([]Vehicle, 0, len(candidates))
	for _, vehicle := range candidates {
		if vehicle.ID == "" || !vehicle.Available || vehicle.Capacity < need.PartySize || vehicle.DeadheadMeters < 0 || math.IsNaN(vehicle.DeadheadMeters) || math.IsInf(vehicle.DeadheadMeters, 0) {
			continue
		}
		if need.RequiresAccessible && !vehicle.Accessible {
			continue
		}
		if need.ServiceType != "" && vehicle.ServiceType != need.ServiceType {
			continue
		}
		filtered = append(filtered, vehicle)
	}
	if len(filtered) == 0 {
		return Vehicle{}, errors.New("no suitable vehicle available")
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].DeadheadMeters == filtered[j].DeadheadMeters {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].DeadheadMeters < filtered[j].DeadheadMeters
	})
	return filtered[0], nil
}