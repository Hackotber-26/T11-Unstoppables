package wallet

import "errors"

type Entry struct {
	ID          string
	AmountCents int64
	Type        string
}

func Balance(entries []Entry) (int64, error) {
	seen := make(map[string]struct{}, len(entries))
	var balance int64
	for _, entry := range entries {
		if entry.ID == "" || entry.AmountCents == 0 {
			return 0, errors.New("ledger entry requires id and non-zero amount")
		}
		if _, ok := seen[entry.ID]; ok {
			continue
		}
		seen[entry.ID] = struct{}{}
		switch entry.Type {
		case "credit", "refund":
			if entry.AmountCents < 0 {
				return 0, errors.New("credit and refund entries must be positive")
			}
		case "debit":
			if entry.AmountCents > 0 {
				return 0, errors.New("debit entries must be negative")
			}
		case "adjustment":
		default:
			return 0, errors.New("unknown ledger entry type")
		}
		if entry.AmountCents > 0 && balance > int64(^uint64(0)>>1)-entry.AmountCents || entry.AmountCents < 0 && balance < -int64(^uint64(0)>>1)-1-entry.AmountCents {
			return 0, errors.New("wallet balance overflow")
		}
		balance += entry.AmountCents
		if balance < 0 {
			return 0, errors.New("ledger entries overdraw the wallet")
		}
	}
	return balance, nil
}