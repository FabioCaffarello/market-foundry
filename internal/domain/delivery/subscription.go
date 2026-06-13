package delivery

import (
	"strings"

	"internal/shared/problem"
)

// Subscription is a validated NATS subject pattern a client wants to
// receive on. Delivery is read-only transport (ADR-0028 I1): a
// Subscription declares what to receive, never a directive to act.
type Subscription struct {
	pattern string
}

// NewSubscription validates `pattern` as a legal NATS subject pattern
// and returns the Subscription. Rules (NATS subject grammar):
//
//   - non-empty;
//   - dot-separated, no empty tokens (no leading/trailing/double dots);
//   - `>` only as the final token;
//   - `*` and `>` only as whole tokens (e.g. `fo*o` is rejected).
//
// Validation is pure; it does not constrain the subject namespace — the
// effective delivery scope (insights events, ADR-0027/0028 I3) is
// bounded by the consumer that feeds the router, not by this type. A
// subscription outside that namespace simply never matches a delivered
// subject.
func NewSubscription(pattern string) (Subscription, *problem.Problem) {
	if pattern == "" {
		return Subscription{}, problem.New(problem.InvalidArgument, "delivery: subscription pattern is empty")
	}
	tokens := strings.Split(pattern, ".")
	for i, t := range tokens {
		switch {
		case t == "":
			return Subscription{}, problem.New(problem.InvalidArgument, "delivery: subscription pattern has an empty token")
		case strings.Contains(t, ">") && t != ">":
			return Subscription{}, problem.New(problem.InvalidArgument, "delivery: '>' must be a whole token")
		case t == ">" && i != len(tokens)-1:
			return Subscription{}, problem.New(problem.InvalidArgument, "delivery: '>' must be the final token")
		case strings.Contains(t, "*") && t != "*":
			return Subscription{}, problem.New(problem.InvalidArgument, "delivery: '*' must be a whole token")
		}
	}
	return Subscription{pattern: pattern}, nil
}

// Pattern returns the raw subject pattern.
func (s Subscription) Pattern() string { return s.pattern }

// Matches reports whether this subscription matches a concrete NATS
// subject.
func (s Subscription) Matches(subject string) bool {
	return SubjectMatches(s.pattern, subject)
}
