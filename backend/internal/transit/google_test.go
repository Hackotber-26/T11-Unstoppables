package transit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGoogleRoutesReturnsTransitTimesAndLegs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/computeRoutes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Goog-Api-Key") != "server-key" {
			t.Fatal("expected API key header")
		}
		if r.Header.Get("X-Goog-FieldMask") != fieldMask {
			t.Fatalf("field mask = %q", r.Header.Get("X-Goog-FieldMask"))
		}
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request["travelMode"] != "TRANSIT" {
			t.Fatalf("travel mode = %v", request["travelMode"])
		}
		preferences, ok := request["transitPreferences"].(map[string]interface{})
		if !ok || len(preferences["allowedTravelModes"].([]interface{})) != 4 {
			t.Fatalf("unexpected transit preferences: %v", request["transitPreferences"])
		}
		_, _ = w.Write([]byte(`{
			"routes": [{
				"duration": "1800s",
				"legs": [{
					"startTime": "2026-04-04T10:00:00Z",
					"endTime": "2026-04-04T10:30:00Z",
					"steps": [
						{"travelMode": "WALK"},
						{"travelMode": "TRANSIT", "transitDetails": {
							"headsign": "Whitefield",
							"stopDetails": {
								"departureStop": {"name": "Majestic"},
								"arrivalStop": {"name": "Indiranagar"},
								"departureTime": "2026-04-04T10:05:00Z",
								"arrivalTime": "2026-04-04T10:22:00Z"
							},
							"transitLine": {
								"name": "Purple Line",
								"nameShort": "Purple",
								"vehicle": {"type": "SUBWAY"}
							}
						}}
					]
				}]
			}]
		}`))
	}))
	defer server.Close()

	planner := NewGoogleRoutes("server-key")
	planner.endpoint = server.URL + "/computeRoutes"
	planner.httpClient = server.Client()
	route, err := planner.Plan(context.Background(), "Majestic, Bengaluru", "Indiranagar, Bengaluru", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if route.DurationSeconds != 1800 || len(route.TransitLegs) != 1 {
		t.Fatalf("unexpected route: %+v", route)
	}
	leg := route.TransitLegs[0]
	if leg.DepartureStop != "Majestic" || leg.ArrivalStop != "Indiranagar" || leg.LineShortName != "Purple" || leg.Headsign != "Whitefield" {
		t.Fatalf("unexpected transit leg: %+v", leg)
	}
	if route.DataSource != "Google Routes API" || route.RetrievedAt.IsZero() {
		t.Fatalf("missing provider freshness metadata: %+v", route)
	}
}

func TestGoogleRoutesRejectsUnconfiguredOrInvalidRequest(t *testing.T) {
	if _, err := NewGoogleRoutes("").Plan(context.Background(), "Majestic", "Indiranagar", time.Now()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
	planner := NewGoogleRoutes("server-key")
	if _, err := planner.Plan(context.Background(), "Majestic", "Majestic", time.Now()); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func TestGoogleRoutesDoesNotInventRouteWhenProviderHasNoTransitLeg(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"routes":[{"duration":"300s","legs":[{"startTime":"2026-04-04T10:00:00Z","endTime":"2026-04-04T10:05:00Z","steps":[{"travelMode":"WALK"}]}]}]}`))
	}))
	defer server.Close()
	planner := NewGoogleRoutes("server-key")
	planner.endpoint = server.URL
	planner.httpClient = server.Client()

	_, err := planner.Plan(context.Background(), "Majestic", "Indiranagar", time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNoTransitRoute) {
		t.Fatalf("error = %v, want ErrNoTransitRoute", err)
	}
}

func TestGoogleRoutesRejectsBusOnlyItineraries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"routes": [{
				"duration": "1800s",
				"legs": [{
					"startTime": "2026-04-04T10:00:00Z",
					"endTime": "2026-04-04T10:30:00Z",
					"steps": [{
						"travelMode": "TRANSIT",
						"transitDetails": {"transitLine": {"vehicle": {"type": "BUS"}}}
					}]
				}]
			}]
		}`))
	}))
	defer server.Close()
	planner := NewGoogleRoutes("server-key")
	planner.endpoint = server.URL
	planner.httpClient = server.Client()

	_, err := planner.Plan(context.Background(), "Majestic", "Indiranagar", time.Now().Add(time.Hour))
	if !errors.Is(err, ErrNoTransitRoute) {
		t.Fatalf("error = %v, want ErrNoTransitRoute", err)
	}
}

func TestGoogleRoutesReportsProviderFailureWithoutResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "private provider detail", http.StatusForbidden)
	}))
	defer server.Close()
	planner := NewGoogleRoutes("server-key")
	planner.endpoint = server.URL
	planner.httpClient = server.Client()

	_, err := planner.Plan(context.Background(), "Majestic", "Indiranagar", time.Now().Add(time.Hour))
	if !errors.Is(err, ErrProviderFailure) {
		t.Fatalf("error = %v, want ErrProviderFailure", err)
	}
	if err.Error() == "transit routing provider failed: private provider detail" {
		t.Fatal("provider response body must not be surfaced")
	}
}
