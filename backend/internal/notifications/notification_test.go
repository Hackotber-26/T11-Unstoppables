package notifications

import (
	"testing"
	"time"
)

func TestNotificationValidation(t *testing.T) {
	notification := Notification{ID: "n1", RecipientID: "u1", EventType: "trip.matched", Channel: InApp, Title: "Trip matched", Body: "Your trip is matched", IdempotencyKey: "trip-1-match", CreatedAt: time.Now()}
	if err := notification.Validate(); err != nil {
		t.Fatal(err)
	}
	notification.Channel = "sms"
	if err := notification.Validate(); err == nil {
		t.Fatal("expected unsupported channel rejection")
	}
}