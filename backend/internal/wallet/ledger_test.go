package wallet

import "testing"

func TestBalanceIsIdempotentAndCannotOverdraw(t *testing.T) {
	balance, err := Balance([]Entry{{ID: "credit-1", AmountCents: 1000, Type: "credit"}, {ID: "credit-1", AmountCents: 1000, Type: "credit"}, {ID: "debit-1", AmountCents: -250, Type: "debit"}})
	if err != nil || balance != 750 {
		t.Fatalf("unexpected balance %d: %v", balance, err)
	}
	if _, err := Balance([]Entry{{ID: "debit", AmountCents: -1, Type: "debit"}}); err == nil {
		t.Fatal("expected overdraft rejection")
	}
}