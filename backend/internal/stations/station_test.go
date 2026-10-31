package stations

import "testing"

func TestNearestActiveStation(t *testing.T) {
	station, distance, err := Nearest([]Station{
		{ID: "inactive", Name: "Inactive", Latitude: 40, Longitude: -73, Active: false},
		{ID: "near", Name: "Near", Latitude: 40.001, Longitude: -73, Active: true},
		{ID: "far", Name: "Far", Latitude: 41, Longitude: -73, Active: true},
	}, 40, -73)
	if err != nil || station.ID != "near" || distance <= 0 {
		t.Fatalf("unexpected nearest station: %+v, %f, %v", station, distance, err)
	}
}