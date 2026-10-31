package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"t11-unstoppables/internal/models"
)

type memoryUsers struct {
	user models.User
	err  error
}

func (m memoryUsers) CreateRider(context.Context, string, string, string, string) (models.User, error) {
	return m.user, m.err
}

func (m memoryUsers) UserByEmail(context.Context, string) (models.User, error) {
	return m.user, m.err
}

func TestSessionTokenSignatureAndExpiry(t *testing.T) {
	service, err := NewService(memoryUsers{}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }
	session, err := service.newSession(models.User{ID: "user-1", Role: "rider"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ParseToken(session.AccessToken)
	if err != nil || claims.Subject != "user-1" {
		t.Fatalf("expected valid claims, got %#v, %v", claims, err)
	}
	service.now = func() time.Time { return time.Unix(claims.Expires, 0).UTC() }
	if _, err := service.ParseToken(session.AccessToken); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestRegistrationRejectsWeakInput(t *testing.T) {
	service, err := NewService(memoryUsers{}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RegisterRider(context.Background(), "Rider", "rider@example.com", "", "short"); !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected registration validation error, got %v", err)
	}
}