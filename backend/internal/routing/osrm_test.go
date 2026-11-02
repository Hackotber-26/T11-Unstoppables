package routing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOSRMRouteParsesRoadRouteAndSteps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/route/v1/driving/77.594600,12.971600;77.600000,12.980000" {
			t.Errorf("unexpected route path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("steps") != "true" || request.URL.Query().Get("geometries") != "geojson" {
			t.Errorf("route response was not requested with geometry and steps: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{
			"code": "Ok",
			"routes": []interface{}{map[string]interface{}{
				"distance": 2400.0, "duration": 420.0,
				"geometry": map[string]interface{}{"type": "LineString", "coordinates": [][]float64{{77.5946, 12.9716}, {77.6, 12.98}}},
				"legs": []interface{}{map[string]interface{}{"steps": []interface{}{map[string]interface{}{
					"name": "Demo Road", "distance": 2400.0, "duration": 420.0,
					"maneuver": map[string]string{"type": "turn", "modifier": "right"},
				}}}},
			}},
		})
	}))
	defer server.Close()

	route, err := OSRMRoute(context.Background(), server.URL, 12.9716, 77.5946, 12.98, 77.6)
	if err != nil {
		t.Fatal(err)
	}
	if route.DistanceMeters != 2400 || route.DurationSeconds != 420 || len(route.Steps) != 1 || route.Steps[0].Name != "Demo Road" {
		t.Fatalf("unexpected route result: %+v", route)
	}
}

func TestOSRMRouteRejectsInvalidCoordinatesAndMissingProvider(t *testing.T) {
	if _, err := OSRMRoute(context.Background(), "http://localhost:5000", 91, 77, 12, 77); !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatal("expected invalid coordinate error")
	}
	if _, err := OSRMRoute(context.Background(), "", 12, 77, 13, 78); err != ErrRouterUnavailable {
		t.Fatalf("expected router unavailable, got %v", err)
	}
}

func TestOSRMRouteThroughPreservesStopOrderAndLegMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/route/v1/driving/77.594600,12.971600;77.600000,12.980000;77.610000,12.990000" {
			t.Errorf("unexpected multi-stop route path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(map[string]interface{}{
			"code": "Ok",
			"routes": []interface{}{map[string]interface{}{
				"distance": 5000.0, "duration": 900.0,
				"geometry": map[string]interface{}{"type": "LineString", "coordinates": [][]float64{{77.5946, 12.9716}, {77.61, 12.99}}},
				"legs": []interface{}{
					map[string]interface{}{"distance": 2000.0, "duration": 300.0},
					map[string]interface{}{"distance": 3000.0, "duration": 600.0},
				},
			}},
		})
	}))
	defer server.Close()

	route, err := OSRMRouteThrough(context.Background(), server.URL, []Coordinate{
		{Latitude: 12.9716, Longitude: 77.5946},
		{Latitude: 12.98, Longitude: 77.6},
		{Latitude: 12.99, Longitude: 77.61},
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.DistanceMeters != 5000 || route.DurationSeconds != 900 || len(route.Legs) != 2 || route.Legs[1].DistanceMeters != 3000 {
		t.Fatalf("unexpected multi-stop route: %+v", route)
	}
}
