package stations

import (
	"errors"
	"math"
)

type Station struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Active    bool    `json:"active"`
}

func (s Station) Validate() error {
	if s.ID == "" || s.Name == "" || math.IsNaN(s.Latitude) || math.IsNaN(s.Longitude) || s.Latitude < -90 || s.Latitude > 90 || s.Longitude < -180 || s.Longitude > 180 {
		return errors.New("station identity or coordinates are invalid")
	}
	return nil
}

func DistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000
	toRadians := func(value float64) float64 { return value * math.Pi / 180 }
	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func Nearest(stations []Station, latitude, longitude float64) (Station, float64, error) {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return Station{}, 0, errors.New("coordinates are invalid")
	}
	var nearest Station
	distance := math.Inf(1)
	for _, station := range stations {
		if !station.Active || station.Validate() != nil {
			continue
		}
		candidate := DistanceMeters(latitude, longitude, station.Latitude, station.Longitude)
		if candidate < distance {
			nearest, distance = station, candidate
		}
	}
	if math.IsInf(distance, 1) {
		return Station{}, 0, errors.New("no active station available")
	}
	return nearest, distance, nil
}