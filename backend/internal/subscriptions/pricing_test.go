package subscriptions

import "testing"

func TestApplyDiscountHonorsRideLimit(t *testing.T) {
	plan := Plan{ID: "commuter", Name: "Commuter", PriceCents: 1000, MonthlyRideLimit: 10, DiscountBPS: 2000}
	price, err := ApplyDiscount(500, plan, 2)
	if err != nil || price != 400 {
		t.Fatalf("unexpected discounted price %d: %v", price, err)
	}
	price, err = ApplyDiscount(500, plan, 10)
	if err != nil || price != 500 {
		t.Fatalf("limit should disable discount: %d, %v", price, err)
	}
}