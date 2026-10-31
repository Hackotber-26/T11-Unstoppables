package payments

import "errors"

type Status string

const (
	Pending    Status = "pending"
	Authorized Status = "authorized"
	Captured   Status = "captured"
	Failed     Status = "failed"
	Voided     Status = "voided"
	Refunded   Status = "refunded"
)

func Transition(current, next Status) (Status, error) {
	valid := map[Status]map[Status]bool{
		Pending:    {Authorized: true, Failed: true},
		Authorized: {Captured: true, Voided: true},
		Captured:   {Refunded: true},
	}
	if !valid[current][next] {
		return current, errors.New("invalid payment transition")
	}
	return next, nil
}