package payments

import "testing"

func TestPaymentLifecycle(t *testing.T) {
	if _, err := Transition(Pending, Authorized); err != nil {
		t.Fatal(err)
	}
	if _, err := Transition(Pending, Refunded); err == nil {
		t.Fatal("pending payment cannot be refunded")
	}
}