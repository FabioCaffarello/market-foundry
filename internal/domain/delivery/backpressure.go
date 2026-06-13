package delivery

import "internal/shared/problem"

// BackpressurePolicy selects how a session's bounded outbound buffer
// sheds load when it is full and the client cannot keep up (ADR-0028
// I4). Both policies keep the buffer strictly bounded — they differ only
// in WHICH frame is dropped, never in whether the bound holds.
//
// PriorityDrop (a third policy named in the PROGRAM-0006 PRD) is
// intentionally NOT implemented here: insights are equally-advisory
// decision-support (ADR-0027), so there is no principled priority
// ordering among the families to drop by. It is revisited only if
// delivery ever carries streams of genuinely heterogeneous priority.
type BackpressurePolicy int

const (
	// DropNewest discards the incoming frame when the buffer is full —
	// it favors already-queued (older) data. The default.
	DropNewest BackpressurePolicy = iota
	// DropOldest evicts the oldest queued frame to make room for the
	// incoming one — it favors freshness (most recent market structure).
	DropOldest
)

// ParseBackpressurePolicy maps a config token to a policy. The empty
// string yields the default (DropNewest).
func ParseBackpressurePolicy(s string) (BackpressurePolicy, *problem.Problem) {
	switch s {
	case "", "drop_newest", "drop-newest", "DropNewest":
		return DropNewest, nil
	case "drop_oldest", "drop-oldest", "DropOldest":
		return DropOldest, nil
	default:
		return DropNewest, problem.New(problem.InvalidArgument, "delivery: unknown backpressure policy: "+s)
	}
}

// String returns the canonical config token (also the metric label value).
func (p BackpressurePolicy) String() string {
	switch p {
	case DropOldest:
		return "drop_oldest"
	default:
		return "drop_newest"
	}
}

// Validate reports whether the policy is a recognized value.
func (p BackpressurePolicy) Validate() *problem.Problem {
	switch p {
	case DropNewest, DropOldest:
		return nil
	default:
		return problem.New(problem.InvalidArgument, "delivery: invalid backpressure policy")
	}
}
