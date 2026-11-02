package main

import (
	"strings"
	"testing"
)

func TestBootstrapCredentialsRequireValidAddressAndStrongPassword(t *testing.T) {
	t.Setenv("INITIAL_ADMIN_EMAIL", " Admin@Example.com ")
	t.Setenv("INITIAL_ADMIN_NAME", "Initial Admin")
	t.Setenv("INITIAL_ADMIN_PASSWORD", strings.Repeat("x", 20))
	email, password, name, err := bootstrapCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if email != "admin@example.com" || name != "Initial Admin" || password != strings.Repeat("x", 20) {
		t.Fatalf("unexpected bootstrap credentials: email=%q name=%q", email, name)
	}

	t.Setenv("INITIAL_ADMIN_EMAIL", "not-an-email")
	if _, _, _, err := bootstrapCredentials(); err == nil {
		t.Fatal("expected invalid email to be rejected")
	}
	t.Setenv("INITIAL_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("INITIAL_ADMIN_PASSWORD", "short")
	if _, _, _, err := bootstrapCredentials(); err == nil {
		t.Fatal("expected short password to be rejected")
	}
}
