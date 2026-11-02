package metro

import (
	"errors"
	"sort"
	"time"
)

type Departure struct {
	StationID string
	LineID    string
	DepartsAt time.Time
}

func NextDepartures(schedule []Departure, stationID, lineID string, after time.Time, limit int) ([]Departure, error) {
	if stationID == "" || lineID == "" || after.IsZero() {
		return nil, errors.New("station, line, and reference time are required")
	}
	if limit < 1 || limit > 20 {
		limit = 5
	}
	result := make([]Departure, 0, limit)
	for _, departure := range schedule {
		if departure.StationID == stationID && departure.LineID == lineID && !departure.DepartsAt.Before(after) {
			result = append(result, departure)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].DepartsAt.Before(result[j].DepartsAt) })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}