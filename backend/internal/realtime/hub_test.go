package realtime

import "testing"

func TestHubPublishesToSubscriber(t *testing.T) {
	hub := NewHub()
	_, events, unsubscribe := hub.Subscribe(1)
	defer unsubscribe()
	if err := hub.Publish(Event{Type: "trip.matched", Resource: "trip", ResourceID: "t1"}); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Type != "trip.matched" || event.ID == "" {
			t.Fatalf("unexpected event: %+v", event)
		}
	default:
		t.Fatal("expected published event")
	}
}