package drivers

import "testing"

func TestAvailabilityTransitions(t *testing.T) {
	if _, err := Transition(Offline, Available); err != nil {
		t.Fatal(err)
	}
	if _, err := Transition(Offline, Busy); err == nil {
		t.Fatal("expected invalid transition rejection")
	}
}