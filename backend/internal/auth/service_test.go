package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"t11-unstoppables/internal/models"
)

type memoryUsers struct {
	user models.User
	err  error
}

func (m memoryUsers) CreateAccount(context.Context, string, string, string, string, string, string) (models.User, error) {
	return m.user, m.err
}

func (memoryUsers) CreateInvitation(context.Context, string, string, string, time.Time) error {
	return nil
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

func TestRegistrationRequiresRoleSpecificInvitation(t *testing.T) {
	service, err := NewService(memoryUsers{}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Register(context.Background(), "Driver", "driver@example.com", "", "a-password-long-enough", "driver", "")
	if !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("expected driver invitation requirement, got %v", err)
	}
	_, err = service.Register(context.Background(), "Admin", "admin@example.com", "", "a-password-long-enough", "admin", "")
	if !errors.Is(err, ErrInvalidInvitation) {
		t.Fatalf("expected admin invitation requirement, got %v", err)
	}
	_, err = service.Register(context.Background(), "Admin", "admin@example.com", "", "a-password-long-enough", "superadmin", "")
	if !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected unsupported role to be rejected, got %v", err)
	}
}

func TestLoginEnforcesSelectedWorkspaceRole(t *testing.T) {
	password := "correct-horse-battery"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewService(memoryUsers{
		user: models.User{ID: "rider-1", Role: "rider", PasswordHash: string(passwordHash)},
	}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.LoginAs(context.Background(), "rider@example.com", password, "driver"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected a role mismatch to fail as invalid credentials, got %v", err)
	}
	if _, err := service.LoginAs(context.Background(), "rider@example.com", password, "rider"); err != nil {
		t.Fatalf("expected correct role sign-in to succeed, got %v", err)
	}
}

func TestInvitationGenerationAndRoleValidation(t *testing.T) {
	service, err := NewService(memoryUsers{}, "test-secret-with-more-than-thirty-two-characters")
	if err != nil {
		t.Fatal(err)
	}
	code, expiresAt, err := service.CreateInvitation(context.Background(), "driver", "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) < 32 || !expiresAt.After(time.Now()) {
		t.Fatalf("expected a strong invitation and future expiry, got %q and %v", code, expiresAt)
	}
	if _, _, err := service.CreateInvitation(context.Background(), "rider", "admin-1"); !errors.Is(err, ErrInvalidRegistration) {
		t.Fatalf("expected rider invitations to be rejected, got %v", err)
	}
}
