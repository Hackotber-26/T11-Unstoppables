package incidents

import "testing"

func TestIncidentEscalationLifecycle(t *testing.T) {
	status, err := Transition(Open, Review)
	if err != nil || status != Review {
		t.Fatalf("expected review status, got %q, %v", status, err)
	}
	if _, err := Transition(Open, Closed); err == nil {
		t.Fatal("incident cannot skip review and resolution")
	}
}