package audit

import (
	"errors"
	"strings"
	"time"
)

type Event struct {
	ID         string            `json:"id"`
	ActorID    string            `json:"actorId"`
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	ResourceID string            `json:"resourceId"`
	OccurredAt time.Time         `json:"occurredAt"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (event Event) Validate() error {
	if strings.TrimSpace(event.ID) == "" || strings.TrimSpace(event.ActorID) == "" || strings.TrimSpace(event.Action) == "" || strings.TrimSpace(event.Resource) == "" || strings.TrimSpace(event.ResourceID) == "" {
		return errors.New("audit event requires id, actor, action, resource, and resource id")
	}
	if event.OccurredAt.IsZero() {
		return errors.New("audit event timestamp is required")
	}
	for key, value := range event.Metadata {
		if strings.TrimSpace(key) == "" || len(key) > 80 || len(value) > 500 {
			return errors.New("audit metadata key or value is invalid")
		}
	}
	return nil
}