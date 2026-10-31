package drivers

import "errors"

type Status string

const (
	Offline   Status = "offline"
	Available Status = "available"
	Busy      Status = "busy"
)

var transitions = map[Status]map[Status]bool{
	Offline:   {Available: true},
	Available: {Offline: true, Busy: true},
	Busy:      {Available: true, Offline: true},
}

func Transition(current, next Status) (Status, error) {
	if !transitions[current][next] {
		return current, errors.New("invalid driver availability transition")
	}
	return next, nil
}