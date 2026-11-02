package notifications

import (
	"errors"
	"strings"
	"time"
)

type Channel string

const (
	InApp Channel = "in_app"
	Email Channel = "email"
)

type Notification struct {
	ID             string    `json:"id"`
	RecipientID    string    `json:"recipientId"`
	EventType      string    `json:"eventType"`
	Channel        Channel   `json:"channel"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	IdempotencyKey string    `json:"idempotencyKey"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (n Notification) Validate() error {
	if n.ID == "" || n.RecipientID == "" || n.EventType == "" || n.IdempotencyKey == "" || strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Body) == "" {
		return errors.New("notification identity, recipient, content, and idempotency key are required")
	}
	if n.Channel != InApp && n.Channel != Email {
		return errors.New("unsupported notification channel")
	}
	if len(n.Title) > 160 || len(n.Body) > 4000 {
		return errors.New("notification content exceeds limits")
	}
	return nil
}