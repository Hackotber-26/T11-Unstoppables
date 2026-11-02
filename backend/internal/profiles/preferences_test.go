package profiles

import "testing"

func TestPreferencesAndNameNormalization(t *testing.T) {
	if err := (Preferences{MaxWalkMeters: 1200, MaxWaitSeconds: 300, MaxDetourPercent: 20}).Validate(); err != nil {
		t.Fatal(err)
	}
	name, err := NormalizeDisplayName("  Alex   Morgan ")
	if err != nil || name != "Alex Morgan" {
		t.Fatalf("unexpected normalized name %q: %v", name, err)
	}
}