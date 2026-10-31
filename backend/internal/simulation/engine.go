package simulation

import (
	"errors"
	"sort"
	"time"

	"t11-unstoppables/internal/pooling"
)

type Result struct {
	RiderCount         int     `json:"riderCount"`
	ServedRiders       int     `json:"servedRiders"`
	UnservedRiders     int     `json:"unservedRiders"`
	PoolCount          int     `json:"poolCount"`
	PoolFormationRate  float64 `json:"poolFormationRate"`
	AverageWaitSec     float64 `json:"averageWaitSeconds"`
	AverageOccupancy   float64 `json:"averageOccupancy"`
	AverageSavingsUSD  float64 `json:"averageSavingsUSD"`
}

type Scenario struct {
	Name                   string `json:"name"`
	RiderCount             int    `json:"riderCount"`
	CorridorCount          int    `json:"corridorCount"`
	ArrivalWindowSeconds   int    `json:"arrivalWindowSeconds"`
	MaxWaitSeconds         int    `json:"maxWaitSeconds"`
	MaxPoolSize            int    `json:"maxPoolSize"`
	DriverCount            int    `json:"driverCount"`
	VehicleCapacity        int    `json:"vehicleCapacity"`
	Seed                   int64  `json:"seed"`
	SoloFareCents          int64  `json:"soloFareCents"`
	PoolDiscountBPS        int64  `json:"poolDiscountBps"`
}

func DefaultScenario() Scenario {
	return Scenario{
		Name: "weekday-commute", RiderCount: 200, CorridorCount: 4,
		ArrivalWindowSeconds: 900, MaxWaitSeconds: 300, MaxPoolSize: 4,
		DriverCount: 50, VehicleCapacity: 4, Seed: 42,
		SoloFareCents: 950, PoolDiscountBPS: 1500,
	}
}

type group struct {
	requests []pooling.Request
	start    time.Time
	last     time.Time
}

func RunScenario(s Scenario) (Result, error) {
	if s.Name == "" || s.RiderCount < 0 || s.CorridorCount < 1 || s.ArrivalWindowSeconds < 1 || s.MaxWaitSeconds < 0 || s.MaxPoolSize < 2 || s.DriverCount < 0 || s.VehicleCapacity < 1 || s.SoloFareCents < 0 || s.PoolDiscountBPS < 0 || s.PoolDiscountBPS > 10000 {
		return Result{}, errors.New("simulation scenario contains invalid values")
	}
	if s.MaxPoolSize > s.VehicleCapacity {
		s.MaxPoolSize = s.VehicleCapacity
	}
	if s.MaxPoolSize < 2 {
		return Result{}, errors.New("vehicle capacity must be at least two for pooling")
	}
	random := newRandom(uint64(s.Seed))
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	requests := make([]pooling.Request, 0, s.RiderCount)
	for index := 0; index < s.RiderCount; index++ {
		corridor := int(random.next() * float64(s.CorridorCount))
		arrival := int(random.next() * float64(s.ArrivalWindowSeconds))
		requestedAt := base.Add(time.Duration(arrival) * time.Second)
		requests = append(requests, pooling.Request{
			ID: "rider-" + itoa(index+1), OriginID: "station-" + itoa(corridor+1),
			Destination: "corridor-" + itoa(corridor+1), RequestedAt: requestedAt,
			PartySize: 1, MaxWaitSeconds: s.MaxWaitSeconds,
		})
	}
	pools, err := pooling.Form(requests, s.MaxPoolSize, false)
	if err != nil {
		return Result{}, err
	}
	grouped := make(map[string]bool, len(requests))
	groups := make([]group, 0, len(requests))
	for _, pool := range pools {
		members := append([]pooling.Request(nil), pool.Requests...)
		g := group{requests: members, start: members[0].RequestedAt, last: members[0].RequestedAt}
		for _, member := range members {
			grouped[member.ID] = true
			if member.RequestedAt.Before(g.start) {
				g.start = member.RequestedAt
			}
			if member.RequestedAt.After(g.last) {
				g.last = member.RequestedAt
			}
		}
		groups = append(groups, g)
	}
	for _, request := range requests {
		if !grouped[request.ID] {
			groups = append(groups, group{requests: []pooling.Request{request}, start: request.RequestedAt, last: request.RequestedAt})
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].start.Before(groups[j].start) })
	result := Result{RiderCount: s.RiderCount, PoolCount: len(pools)}
	var waitSum int64
	var pooledRiders int
	var savedCents int64
	servedGroups := s.DriverCount
	if servedGroups > len(groups) {
		servedGroups = len(groups)
	}
	for index, current := range groups {
		if index >= servedGroups {
			result.UnservedRiders += len(current.requests)
			continue
		}
		result.ServedRiders += len(current.requests)
		for _, request := range current.requests {
			waitSum += int64(current.last.Sub(request.RequestedAt).Seconds())
		}
		if len(current.requests) > 1 {
			pooledRiders += len(current.requests)
			savedCents += int64(len(current.requests)) * s.SoloFareCents * s.PoolDiscountBPS / 10000
		}
	}
	if result.ServedRiders > 0 {
		result.AverageWaitSec = float64(waitSum) / float64(result.ServedRiders)
		result.PoolFormationRate = float64(pooledRiders) / float64(result.ServedRiders)
		result.AverageSavingsUSD = float64(savedCents) / float64(result.ServedRiders) / 100
	}
	if servedGroups > 0 {
		result.AverageOccupancy = float64(result.ServedRiders) / float64(servedGroups)
	}
	return result, nil
}


type randomSource struct{ state uint64 }

func newRandom(seed uint64) *randomSource {
	if seed == 0 {
		seed = 1
	}
	return &randomSource{state: seed}
}

func (r *randomSource) next() float64 {
	r.state = 6364136223846793005*r.state + 1442695040888963407
	return float64(r.state>>11) / float64(uint64(1)<<53)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}

