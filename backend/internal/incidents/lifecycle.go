package incidents

import (
	"errors"
	"time"
)

type Severity string
type Status string

const (
	Low      Severity = "low"
	Medium   Severity = "medium"
	High     Severity = "high"
	Critical Severity = "critical"
	Open     Status   = "open"
	Review   Status   = "under_review"
	Resolved Status   = "resolved"
	Closed   Status   = "closed"
)

type Incident struct {
	ID          string    `json:"id"`
	TripID      string    `json:"tripId,omitempty"`
	ReporterID  string    `json:"reporterId"`
	Severity    Severity  `json:"severity"`
	Status      Status    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

func Transition(current, next Status) (Status, error) {
	allowed := map[Status]map[Status]bool{
		Open:   {Review: true},
		Review: {Resolved: true, Open: true},
		Resolved: {Closed: true, Review: true},
	}
	if !allowed[current][next] {
		return current, errors.New("invalid incident status transition")
	}
	return next, nil
}

func Validate(incident Incident) error {
	if incident.ID == "" || incident.ReporterID == "" || incident.Description == "" {
		return errors.New("incident id, reporter, and description are required")
	}
	if incident.Severity != Low && incident.Severity != Medium && incident.Severity != High && incident.Severity != Critical {
		return errors.New("invalid incident severity")
	}
	if incident.Status == "" {
		return errors.New("incident status is required")
	}
	return nil
}