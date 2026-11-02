package pooling

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type Request struct {
	ID              string
	OriginID        string
	Destination     string
	RequestedAt     time.Time
	PartySize       int
	MaxWaitSeconds  int
	MaxDetourMeters float64
}

type Pool struct {
	OriginID    string    `json:"originId"`
	Destination string    `json:"destination"`
	Requests    []Request `json:"requests"`
}

func Form(requests []Request, maxPoolSize int, destinationSimilarity bool) ([]Pool, error) {
	if maxPoolSize < 2 || maxPoolSize > 12 {
		return nil, errors.New("maxPoolSize must be between 2 and 12")
	}
	ordered := append([]Request(nil), requests...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].OriginID != ordered[j].OriginID {
			return ordered[i].OriginID < ordered[j].OriginID
		}
		return ordered[i].RequestedAt.Before(ordered[j].RequestedAt)
	})
	groups := make([]Pool, 0)
	current := make([]Request, 0, maxPoolSize)
	currentSeats := 0
	flush := func() {
		if len(current) > 1 {
			groups = append(groups, Pool{OriginID: current[0].OriginID, Destination: current[0].Destination, Requests: append([]Request(nil), current...)})
		}
		current = current[:0]
		currentSeats = 0
	}
	for _, request := range ordered {
		if request.ID == "" || request.OriginID == "" || request.Destination == "" || request.RequestedAt.IsZero() || request.PartySize < 1 || request.PartySize > maxPoolSize || request.MaxWaitSeconds < 0 || request.MaxDetourMeters < 0 {
			return nil, errors.New("pool request contains invalid values")
		}
		compatible := len(current) > 0 && current[0].OriginID == request.OriginID
		if compatible {
			oldest := current[0]
			compatible = request.RequestedAt.Sub(oldest.RequestedAt).Seconds() <= float64(min(oldest.MaxWaitSeconds, request.MaxWaitSeconds))
		}
		if compatible && destinationSimilarity {
			compatible = strings.EqualFold(strings.TrimSpace(current[0].Destination), strings.TrimSpace(request.Destination))
		}
		if compatible {
			compatible = currentSeats+request.PartySize <= maxPoolSize
		}
		if !compatible || len(current) >= maxPoolSize {
			flush()
		}
		current = append(current, request)
		currentSeats += request.PartySize
	}
	flush()
	return groups, nil
}