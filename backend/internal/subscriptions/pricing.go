package subscriptions

import "errors"

type Plan struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PricePaise       int64  `json:"pricePaise"`
	MonthlyRideLimit int    `json:"monthlyRideLimit"`
	DiscountBPS      int64  `json:"discountBps"`
}

func (plan Plan) Validate() error {
	if plan.ID == "" || plan.Name == "" || plan.PricePaise < 0 || plan.MonthlyRideLimit < 0 || plan.DiscountBPS < 0 || plan.DiscountBPS > 10000 {
		return errors.New("subscription plan fields are invalid")
	}
	return nil
}

func ApplyDiscount(fareCents int64, plan Plan, ridesUsed int) (int64, error) {
	if err := plan.Validate(); err != nil {
		return 0, err
	}
	if fareCents < 0 || ridesUsed < 0 {
		return 0, errors.New("fare and ride count must be non-negative")
	}
	if plan.MonthlyRideLimit > 0 && ridesUsed >= plan.MonthlyRideLimit {
		return fareCents, nil
	}
	discount := fareCents/10000*plan.DiscountBPS + fareCents%10000*plan.DiscountBPS/10000
	return fareCents - discount, nil
}
