package audit

import (
	"testing"
	"time"
)

func TestEventValidation(t *testing.T) {
	event := Event{ID: "e1", ActorID: "u1", Action: "trip.cancel", Resource: "trip", ResourceID: "t1", OccurredAt: time.Now()}
	if err := event.Validate(); err != nil {
		t.Fatal(err)
	}
	event.Metadata = map[string]string{"": "bad"}
	if err := event.Validate(); err == nil {
		t.Fatal("expected invalid metadata rejection")
	}
}