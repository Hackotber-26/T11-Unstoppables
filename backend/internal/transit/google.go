package transit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotConfigured   = errors.New("transit routing is not configured")
	ErrInvalidRequest  = errors.New("transit route request is invalid")
	ErrProviderFailure = errors.New("transit routing provider failed")
	ErrNoTransitRoute  = errors.New("no transit itinerary is available")
)

const (
	defaultEndpoint = "https://routes.googleapis.com/directions/v2:computeRoutes"
	fieldMask       = "routes.duration,routes.legs.startTime,routes.legs.endTime,routes.legs.steps.travelMode,routes.legs.steps.transitDetails"
)

type Route struct {
	Origin          string       `json:"origin"`
	Destination     string       `json:"destination"`
	DepartureTime   time.Time    `json:"departureTime"`
	ArrivalTime     time.Time    `json:"arrivalTime"`
	DurationSeconds float64      `json:"durationSeconds"`
	RetrievedAt     time.Time    `json:"retrievedAt"`
	DataSource      string       `json:"dataSource"`
	TransitLegs     []TransitLeg `json:"transitLegs"`
}

type TransitLeg struct {
	DepartureStop string    `json:"departureStop"`
	ArrivalStop   string    `json:"arrivalStop"`
	DepartureTime time.Time `json:"departureTime"`
	ArrivalTime   time.Time `json:"arrivalTime"`
	LineName      string    `json:"lineName,omitempty"`
	LineShortName string    `json:"lineShortName,omitempty"`
	Headsign      string    `json:"headsign,omitempty"`
	VehicleType   string    `json:"vehicleType,omitempty"`
}

type GoogleRoutes struct {
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

func NewGoogleRoutes(apiKey string) *GoogleRoutes {
	return &GoogleRoutes{
		apiKey:     strings.TrimSpace(apiKey),
		endpoint:   defaultEndpoint,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GoogleRoutes) Plan(ctx context.Context, origin, destination string, departure time.Time) (Route, error) {
	if g == nil || g.apiKey == "" {
		return Route{}, ErrNotConfigured
	}
	origin = strings.TrimSpace(origin)
	destination = strings.TrimSpace(destination)
	now := time.Now()
	if origin == "" || destination == "" || origin == destination || departure.IsZero() ||
		departure.Before(now.Add(-7*24*time.Hour)) || departure.After(now.Add(100*24*time.Hour)) {
		return Route{}, ErrInvalidRequest
	}

	requestBody, err := json.Marshal(map[string]interface{}{
		"origin":        map[string]string{"address": origin},
		"destination":   map[string]string{"address": destination},
		"travelMode":    "TRANSIT",
		"departureTime": departure.UTC().Format(time.RFC3339),
		"transitPreferences": map[string]interface{}{
			"allowedTravelModes": []string{"SUBWAY", "TRAIN", "LIGHT_RAIL", "RAIL"},
		},
	})
	if err != nil {
		return Route{}, fmt.Errorf("encode transit route request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return Route{}, fmt.Errorf("create transit route request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Goog-Api-Key", g.apiKey)
	request.Header.Set("X-Goog-FieldMask", fieldMask)
	response, err := g.httpClient.Do(request)
	if err != nil {
		return Route{}, fmt.Errorf("%w: %v", ErrProviderFailure, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Route{}, ErrProviderFailure
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return Route{}, fmt.Errorf("%w: could not read response", ErrProviderFailure)
	}

	var result googleRouteResponse
	if err := json.Unmarshal(payload, &result); err != nil {
		return Route{}, fmt.Errorf("%w: could not decode response", ErrProviderFailure)
	}
	if len(result.Routes) == 0 {
		return Route{}, ErrNoTransitRoute
	}

	providerRoute := result.Routes[0]
	duration, err := parseDuration(providerRoute.Duration)
	if err != nil {
		return Route{}, fmt.Errorf("%w: invalid route duration", ErrProviderFailure)
	}
	if len(providerRoute.Legs) == 0 {
		return Route{}, ErrNoTransitRoute
	}
	startTime, err := parseProviderTime(providerRoute.Legs[0].StartTime)
	if err != nil {
		return Route{}, fmt.Errorf("%w: invalid departure time", ErrProviderFailure)
	}
	lastLeg := providerRoute.Legs[len(providerRoute.Legs)-1]
	endTime, err := parseProviderTime(lastLeg.EndTime)
	if err != nil {
		return Route{}, fmt.Errorf("%w: invalid arrival time", ErrProviderFailure)
	}

	route := Route{
		Origin: origin, Destination: destination, DepartureTime: startTime, ArrivalTime: endTime,
		DurationSeconds: duration, RetrievedAt: time.Now().UTC(), DataSource: "Google Routes API",
		TransitLegs: make([]TransitLeg, 0),
	}
	for _, leg := range providerRoute.Legs {
		for _, step := range leg.Steps {
			if step.TravelMode != "TRANSIT" || step.TransitDetails == nil {
				continue
			}
			details := step.TransitDetails
			vehicleType := details.TransitLine.Vehicle.Type
			if vehicleType != "SUBWAY" && vehicleType != "TRAIN" && vehicleType != "LIGHT_RAIL" && vehicleType != "RAIL" {
				continue
			}
			departureTime, err := parseProviderTime(details.StopDetails.DepartureTime)
			if err != nil {
				return Route{}, fmt.Errorf("%w: invalid transit departure time", ErrProviderFailure)
			}
			arrivalTime, err := parseProviderTime(details.StopDetails.ArrivalTime)
			if err != nil {
				return Route{}, fmt.Errorf("%w: invalid transit arrival time", ErrProviderFailure)
			}
			route.TransitLegs = append(route.TransitLegs, TransitLeg{
				DepartureStop: details.StopDetails.DepartureStop.Name,
				ArrivalStop:   details.StopDetails.ArrivalStop.Name,
				DepartureTime: departureTime,
				ArrivalTime:   arrivalTime,
				LineName:      details.TransitLine.Name,
				LineShortName: details.TransitLine.NameShort,
				Headsign:      details.Headsign,
				VehicleType:   vehicleType,
			})
		}
	}
	if len(route.TransitLegs) == 0 {
		return Route{}, ErrNoTransitRoute
	}
	return route, nil
}

type googleRouteResponse struct {
	Routes []struct {
		Duration string `json:"duration"`
		Legs     []struct {
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
			Steps     []struct {
				TravelMode     string `json:"travelMode"`
				TransitDetails *struct {
					Headsign    string `json:"headsign"`
					StopDetails struct {
						DepartureStop struct {
							Name string `json:"name"`
						} `json:"departureStop"`
						ArrivalStop struct {
							Name string `json:"name"`
						} `json:"arrivalStop"`
						DepartureTime string `json:"departureTime"`
						ArrivalTime   string `json:"arrivalTime"`
					} `json:"stopDetails"`
					TransitLine struct {
						Name      string `json:"name"`
						NameShort string `json:"nameShort"`
						Vehicle   struct {
							Type string `json:"type"`
						} `json:"vehicle"`
					} `json:"transitLine"`
				} `json:"transitDetails"`
			} `json:"steps"`
		} `json:"legs"`
	} `json:"routes"`
}

func parseDuration(value string) (float64, error) {
	if !strings.HasSuffix(value, "s") {
		return 0, errors.New("duration must be expressed in seconds")
	}
	seconds, err := strconv.ParseFloat(strings.TrimSuffix(value, "s"), 64)
	if err != nil || seconds < 0 {
		return 0, errors.New("duration is invalid")
	}
	return seconds, nil
}

func parseProviderTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
