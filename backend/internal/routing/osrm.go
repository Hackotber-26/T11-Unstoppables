package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrRouterUnavailable = errors.New("road routing service is unavailable")
var ErrInvalidCoordinates = errors.New("route coordinates are invalid")

type NavigationStep struct {
	Name     string  `json:"name"`
	Distance float64 `json:"distanceMeters"`
	Duration float64 `json:"durationSeconds"`
	Maneuver string  `json:"maneuver"`
}

type NavigationRoute struct {
	DistanceMeters  float64          `json:"distanceMeters"`
	DurationSeconds float64          `json:"durationSeconds"`
	Geometry        json.RawMessage  `json:"geometry"`
	Steps           []NavigationStep `json:"steps"`
	Legs            []NavigationLeg  `json:"legs,omitempty"`
}

type Coordinate struct {
	Latitude  float64
	Longitude float64
}

type NavigationLeg struct {
	DistanceMeters  float64 `json:"distanceMeters"`
	DurationSeconds float64 `json:"durationSeconds"`
}

func OSRMRoute(ctx context.Context, baseURL string, startLat, startLng, endLat, endLng float64) (NavigationRoute, error) {
	return OSRMRouteThrough(ctx, baseURL, []Coordinate{{Latitude: startLat, Longitude: startLng}, {Latitude: endLat, Longitude: endLng}})
}

func OSRMRouteThrough(ctx context.Context, baseURL string, points []Coordinate) (NavigationRoute, error) {
	if len(points) < 2 || len(points) > 20 {
		return NavigationRoute{}, ErrInvalidCoordinates
	}
	coordinates := make([]string, len(points))
	for i, point := range points {
		if !validCoordinate(point.Latitude, -90, 90) || !validCoordinate(point.Longitude, -180, 180) {
			return NavigationRoute{}, ErrInvalidCoordinates
		}
		coordinates[i] = fmt.Sprintf("%.6f,%.6f", point.Longitude, point.Latitude)
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || base.Scheme != "http" && base.Scheme != "https" || base.Host == "" {
		return NavigationRoute{}, ErrRouterUnavailable
	}
	endpoint := *base
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/route/v1/driving/" + strings.Join(coordinates, ";")
	query := endpoint.Query()
	query.Set("overview", "full")
	query.Set("geometries", "geojson")
	query.Set("steps", "true")
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return NavigationRoute{}, fmt.Errorf("create road routing request: %w", err)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return NavigationRoute{}, fmt.Errorf("%w: %v", ErrRouterUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return NavigationRoute{}, ErrRouterUnavailable
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return NavigationRoute{}, fmt.Errorf("read road routing response: %w", err)
	}
	var result struct {
		Code   string `json:"code"`
		Routes []struct {
			Distance float64         `json:"distance"`
			Duration float64         `json:"duration"`
			Geometry json.RawMessage `json:"geometry"`
			Legs     []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Steps    []struct {
					Name     string  `json:"name"`
					Distance float64 `json:"distance"`
					Duration float64 `json:"duration"`
					Maneuver struct {
						Type     string `json:"type"`
						Modifier string `json:"modifier"`
					} `json:"maneuver"`
				} `json:"steps"`
			} `json:"legs"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return NavigationRoute{}, fmt.Errorf("decode road routing response: %w", err)
	}
	if result.Code != "Ok" || len(result.Routes) == 0 {
		return NavigationRoute{}, ErrRouterUnavailable
	}
	route := result.Routes[0]
	if route.Distance < 0 || route.Duration < 0 || len(route.Geometry) == 0 {
		return NavigationRoute{}, errors.New("road routing service returned an invalid route")
	}
	steps := make([]NavigationStep, 0)
	legs := make([]NavigationLeg, 0, len(route.Legs))
	for _, leg := range route.Legs {
		legs = append(legs, NavigationLeg{DistanceMeters: leg.Distance, DurationSeconds: leg.Duration})
		for _, step := range leg.Steps {
			direction := strings.TrimSpace(step.Maneuver.Type + " " + step.Maneuver.Modifier)
			steps = append(steps, NavigationStep{Name: step.Name, Distance: step.Distance, Duration: step.Duration, Maneuver: direction})
		}
	}
	return NavigationRoute{
		DistanceMeters:  route.Distance,
		DurationSeconds: route.Duration,
		Geometry:        route.Geometry,
		Steps:           steps,
		Legs:            legs,
	}, nil
}

func validCoordinate(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}
