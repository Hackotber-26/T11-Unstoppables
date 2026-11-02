package routing

import (
	"errors"
	"math"
)

type Edge struct {
	From          string
	To            string
	TravelSeconds float64
	DistanceMeters float64
}

type Route struct {
	Nodes         []string `json:"nodes"`
	TravelSeconds float64  `json:"travelSeconds"`
	DistanceMeters float64 `json:"distanceMeters"`
}

func Shortest(edges []Edge, start, destination string) (Route, error) {
	if start == "" || destination == "" {
		return Route{}, errors.New("start and destination are required")
	}
	adjacency := make(map[string][]Edge)
	for _, edge := range edges {
		if edge.From == "" || edge.To == "" || edge.TravelSeconds < 0 || edge.DistanceMeters < 0 || math.IsNaN(edge.TravelSeconds) || math.IsNaN(edge.DistanceMeters) || math.IsInf(edge.TravelSeconds, 0) || math.IsInf(edge.DistanceMeters, 0) {
			return Route{}, errors.New("route edge contains invalid values")
		}
		adjacency[edge.From] = append(adjacency[edge.From], edge)
	}
	distance := map[string]float64{start: 0}
	meters := map[string]float64{start: 0}
	previous := make(map[string]string)
	visited := make(map[string]bool)
	for {
		current := ""
		best := math.Inf(1)
		for node, value := range distance {
			if !visited[node] && value < best {
				current, best = node, value
			}
		}
		if current == "" {
			break
		}
		if current == destination {
			break
		}
		visited[current] = true
		for _, edge := range adjacency[current] {
			candidate := best + edge.TravelSeconds
			oldDistance, exists := distance[edge.To]
			if !exists || candidate < oldDistance {
				distance[edge.To] = candidate
				meters[edge.To] = meters[current] + edge.DistanceMeters
				previous[edge.To] = current
			}
		}
	}
	travelSeconds, found := distance[destination]
	if !found {
		return Route{}, errors.New("no route found")
	}
	path := []string{destination}
	for path[len(path)-1] != start {
		prior, exists := previous[path[len(path)-1]]
		if !exists {
			return Route{}, errors.New("route path is incomplete")
		}
		path = append(path, prior)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return Route{Nodes: path, TravelSeconds: travelSeconds, DistanceMeters: meters[destination]}, nil
}

type ETA struct {
	ExpectedSeconds int `json:"expectedSeconds"`
	LowerSeconds    int `json:"lowerSeconds"`
	UpperSeconds    int `json:"upperSeconds"`
}

func EstimateETA(route Route, dwellSeconds int, varianceRatio float64) (ETA, error) {
	if route.TravelSeconds < 0 || math.IsNaN(route.TravelSeconds) || math.IsInf(route.TravelSeconds, 0) || dwellSeconds < 0 || varianceRatio < 0 || math.IsNaN(varianceRatio) || math.IsInf(varianceRatio, 0) {
		return ETA{}, errors.New("ETA inputs must be non-negative finite values")
	}
	expected := int(math.Ceil(route.TravelSeconds)) + dwellSeconds
	return ETA{
		ExpectedSeconds: expected,
		LowerSeconds:    int(math.Floor(float64(expected) * (1 - varianceRatio))),
		UpperSeconds:    int(math.Ceil(float64(expected) * (1 + varianceRatio))),
	}, nil
}