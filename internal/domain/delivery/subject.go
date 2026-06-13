// Package delivery is the bounded context for push delivery of foundry
// events to connected clients over a transport (WebSocket, in the
// gateway). It is pure domain: subscription state and NATS subject
// matching, with no knowledge of the transport, the actor system, or
// NATS itself.
//
// Delivery is read-only transport (ADR-0028 I1): a Subscription names
// what a client wants to receive; it is never a directive to act. The
// effective stream scope is bounded by what the delivery consumer reads
// (insights events — ADR-0027 decision-support), not by this package.
package delivery

import "strings"

// SubjectMatches reports whether the concrete NATS subject `subject`
// matches the subscription `pattern`, using NATS subject wildcard
// semantics:
//
//   - subjects are dot-separated tokens;
//   - `*` matches exactly one (non-empty) token;
//   - `>` matches one or more trailing tokens and must be the final
//     token of the pattern.
//
// A pattern with no wildcards matches only the identical subject.
// `*` and `>` are wildcards only when they are an entire token (a token
// like `fo*o` is matched literally, per NATS). Empty inputs never
// match. The function is pure and allocates only the two token splits.
func SubjectMatches(pattern, subject string) bool {
	if pattern == "" || subject == "" {
		return false
	}
	pTokens := strings.Split(pattern, ".")
	sTokens := strings.Split(subject, ".")
	for i, pt := range pTokens {
		if pt == ">" {
			// `>` is terminal and matches at least one remaining token.
			return i == len(pTokens)-1 && i < len(sTokens)
		}
		if i >= len(sTokens) {
			return false
		}
		if pt == "*" {
			if sTokens[i] == "" {
				return false
			}
			continue
		}
		if pt != sTokens[i] {
			return false
		}
	}
	// All pattern tokens matched literally/`*`; lengths must be equal
	// (no `>` consumed the tail).
	return len(pTokens) == len(sTokens)
}
