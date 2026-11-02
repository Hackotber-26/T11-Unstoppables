package fares

import (
	"errors"
	"math"
)

const (
	baseFareCents     int64 = 3000
	ratePerKmCents    int64 = 1200
	pooledDiscountBPS int64 = 1500
)

type Fare struct {
	BaseCents     int64
	DistanceCents int64
	DiscountCents int64
	TotalCents    int64
}

func QuoteFare(distanceMeters float64, poolMode string) (Fare, error) {
	if math.IsNaN(distanceMeters) || math.IsInf(distanceMeters, 0) || distanceMeters < 0 || distanceMeters > 500000 {
		return Fare{}, errors.New("distanceMeters must be between 0 and 500000")
	}
	if poolMode != "corridor" && poolMode != "solo" {
		return Fare{}, errors.New("poolMode must be corridor or solo")
	}
	distanceCents := int64(math.Ceil(distanceMeters * float64(ratePerKmCents) / 1000))
	undiscounted := baseFareCents + distanceCents
	discount := int64(0)
	if poolMode == "corridor" {
		discount = undiscounted * pooledDiscountBPS / 10000
	}
	return Fare{
		BaseCents:     baseFareCents,
		DistanceCents: distanceCents,
		DiscountCents: discount,
		TotalCents:    undiscounted - discount,
	}, nil
}