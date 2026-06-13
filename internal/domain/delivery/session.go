package delivery

// SessionID identifies one delivery session — one connected client
// (one WebSocket connection in the gateway).
type SessionID string

// Session holds the subscription state for one connected delivery
// client. It is owned by exactly one SessionActor in the outer layer,
// so its methods mutate in place without internal synchronization — the
// actor model provides the single-owner guarantee (analogous to the
// single-writer invariant, ADR-0008, applied to in-memory session
// state). The type is pure domain: it knows subscriptions and matching,
// nothing about the transport or NATS.
type Session struct {
	id   SessionID
	subs map[string]Subscription
}

// NewSession returns an empty session with the given id.
func NewSession(id SessionID) *Session {
	return &Session{id: id, subs: make(map[string]Subscription)}
}

// ID returns the session id.
func (s *Session) ID() SessionID { return s.id }

// Subscribe adds a subscription. It is idempotent by pattern: returns
// false if the pattern was already present (no duplicate stored).
func (s *Session) Subscribe(sub Subscription) bool {
	if _, ok := s.subs[sub.pattern]; ok {
		return false
	}
	s.subs[sub.pattern] = sub
	return true
}

// Unsubscribe removes a subscription by pattern. Returns false if the
// pattern was not present.
func (s *Session) Unsubscribe(pattern string) bool {
	if _, ok := s.subs[pattern]; !ok {
		return false
	}
	delete(s.subs, pattern)
	return true
}

// SubscriptionCount returns how many subscriptions the session holds.
func (s *Session) SubscriptionCount() int { return len(s.subs) }

// Subscriptions returns a copy of the current subscription patterns
// (order unspecified). Mutating the returned slice does not affect the
// session.
func (s *Session) Subscriptions() []string {
	out := make([]string, 0, len(s.subs))
	for p := range s.subs {
		out = append(out, p)
	}
	return out
}

// Matches reports whether any of the session's subscriptions matches the
// concrete NATS subject — i.e. whether an event published on `subject`
// should be delivered to this client.
func (s *Session) Matches(subject string) bool {
	for _, sub := range s.subs {
		if sub.Matches(subject) {
			return true
		}
	}
	return false
}
