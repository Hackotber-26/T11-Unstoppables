package fares

import "testing"

func TestQuoteFareUsesCentsAndPoolDiscount(t *testing.T) {
	corridor, err := QuoteFare(1500, "corridor")
	if err != nil {
		t.Fatal(err)
	}
	if corridor.BaseCents != 250 || corridor.DistanceCents != 225 || corridor.DiscountCents != 71 || corridor.TotalCents != 404 {
		t.Fatalf("unexpected corridor quote: %+v", corridor)
	}
	solo, err := QuoteFare(1500, "solo")
	if err != nil {
		t.Fatal(err)
	}
	if solo.TotalCents != 475 {
		t.Fatalf("unexpected solo quote: %+v", solo)
	}
}

func TestQuoteFareRejectsUnsupportedInputs(t *testing.T) {
	if _, err := QuoteFare(-1, "solo"); err == nil {
		t.Fatal("expected negative distance to fail")
	}
	if _, err := QuoteFare(1, "unknown"); err == nil {
		t.Fatal("expected unknown pool mode to fail")
	}
}