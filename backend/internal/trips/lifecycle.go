package trips

import "errors"

type Status string
type Actor string

const (
	Requested  Status = "requested"
	Matched    Status = "matched"
	InProgress Status = "in_progress"
	Completed  Status = "completed"
	Cancelled  Status = "cancelled"
	Rider      Actor  = "rider"
	Driver     Actor  = "driver"
	Operator   Actor  = "operator"
)

func Transition(current, next Status, actor Actor) (Status, error) {
	allowed := false
	switch current {
	case Requested:
		allowed = next == Matched && actor == Operator || next == Cancelled && (actor == Rider || actor == Operator)
	case Matched:
		allowed = next == InProgress && actor == Driver || next == Cancelled && actor == Operator
	case InProgress:
		allowed = next == Completed && actor == Driver
	}
	if !allowed {
		return current, errors.New("trip transition is not allowed for this actor")
	}
	return next, nil
}