package riders

import "testing"

func TestProfileRejectsMalformedEmail(t *testing.T) {
	profile := Profile{ID: "r1", Name: "Alex", Email: "alex@@example.com"}
	if err := profile.Validate(); err == nil {
		t.Fatal("expected malformed email rejection")
	}
}