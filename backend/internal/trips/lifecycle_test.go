package trips

import "testing"

func TestTripLifecycleRequiresCorrectActor(t *testing.T) {
	if _, err := Transition(Requested, Matched, Operator); err != nil {
		t.Fatal(err)
	}
	if _, err := Transition(Matched, InProgress, Rider); err == nil {
		t.Fatal("rider cannot start driver's trip")
	}
}