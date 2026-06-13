package delivery

import "testing"

func TestSubjectMatches(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		subject string
		want    bool
	}{
		// literal
		{"literal equal", "insights.events.volumeprofile.sampled.btc_usdt_spot", "insights.events.volumeprofile.sampled.btc_usdt_spot", true},
		{"literal differ", "insights.events.volumeprofile.sampled.btc_usdt_spot", "insights.events.volumeprofile.sampled.eth_usdt_spot", false},
		{"literal length differ short", "insights.events", "insights.events.volumeprofile", false},
		{"literal length differ long", "insights.events.volumeprofile", "insights.events", false},

		// single-token wildcard *
		{"star one token", "insights.events.*.sampled.btc_usdt_spot", "insights.events.volumeprofile.sampled.btc_usdt_spot", true},
		{"star tail token", "insights.events.volumeprofile.sampled.*", "insights.events.volumeprofile.sampled.btc_usdt_spot", true},
		{"star does not span multiple", "insights.events.*", "insights.events.volumeprofile.sampled", false},
		{"star requires a token", "insights.events.*", "insights.events", false},

		// multi-token wildcard >
		{"gt matches many", "insights.events.>", "insights.events.volumeprofile.sampled.btc_usdt_spot", true},
		{"gt matches one", "insights.events.>", "insights.events.x", true},
		{"gt requires at least one", "insights.events.>", "insights.events", false},
		{"gt at root", ">", "insights.events.volumeprofile", true},
		{"gt non-terminal never matches", "insights.>.sampled", "insights.events.sampled", false}, // '>' is wildcard only as final token; non-terminal → no match
		{"combo star and gt", "insights.events.*.>", "insights.events.volumeprofile.sampled.btc_usdt_spot", true},

		// empties
		{"empty pattern", "", "insights.events.x", false},
		{"empty subject", "insights.events.>", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SubjectMatches(c.pattern, c.subject); got != c.want {
				t.Fatalf("SubjectMatches(%q, %q) = %v, want %v", c.pattern, c.subject, got, c.want)
			}
		})
	}
}

// TestSubjectMatches_GtTerminalSemantics pins that '>' is honored as a
// wildcard only as the final token. A non-terminal '>' is not expanded
// to span tokens; the matcher stops and reports no match (the safe
// behavior — NewSubscription rejects such patterns up front anyway).
func TestSubjectMatches_GtTerminalSemantics(t *testing.T) {
	if SubjectMatches("insights.>.sampled", "insights.events.foo.sampled") {
		t.Fatal("non-terminal '>' should not behave as a multi-token wildcard")
	}
}

func TestNewSubscription(t *testing.T) {
	valid := []string{
		"insights.events.>",
		"insights.events.volumeprofile.sampled.>",
		"insights.events.*.sampled.btc_usdt_spot",
		"insights.events.volumeprofile.sampled.btc_usdt_spot",
		">",
	}
	for _, p := range valid {
		t.Run("valid/"+p, func(t *testing.T) {
			sub, prob := NewSubscription(p)
			if prob != nil {
				t.Fatalf("NewSubscription(%q) unexpected problem: %v", p, prob)
			}
			if sub.Pattern() != p {
				t.Fatalf("Pattern() = %q, want %q", sub.Pattern(), p)
			}
		})
	}

	invalid := []string{
		"",                     // empty
		"insights..events",     // empty token
		".insights.events",     // leading dot
		"insights.events.",     // trailing dot
		"insights.>.sampled",   // '>' not final
		"insights.events.fo>o", // '>' not whole token
		"insights.events.fo*o", // '*' not whole token
	}
	for _, p := range invalid {
		t.Run("invalid/"+p, func(t *testing.T) {
			if _, prob := NewSubscription(p); prob == nil {
				t.Fatalf("NewSubscription(%q) expected a problem, got nil", p)
			}
		})
	}
}

func TestSessionLifecycle(t *testing.T) {
	s := NewSession("sess-1")
	if s.ID() != "sess-1" {
		t.Fatalf("ID() = %q, want sess-1", s.ID())
	}
	if s.SubscriptionCount() != 0 {
		t.Fatalf("new session should have 0 subscriptions, got %d", s.SubscriptionCount())
	}

	vp, prob := NewSubscription("insights.events.volumeprofile.sampled.>")
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if !s.Subscribe(vp) {
		t.Fatal("first Subscribe should report true (added)")
	}
	if s.Subscribe(vp) {
		t.Fatal("duplicate Subscribe should report false (idempotent)")
	}
	if s.SubscriptionCount() != 1 {
		t.Fatalf("count after dup = %d, want 1", s.SubscriptionCount())
	}

	if !s.Matches("insights.events.volumeprofile.sampled.btc_usdt_spot") {
		t.Fatal("session should match a subscribed volume-profile subject")
	}
	if s.Matches("insights.events.tpo.sampled.btc_usdt_spot") {
		t.Fatal("session should NOT match an unsubscribed family")
	}

	if !s.Unsubscribe("insights.events.volumeprofile.sampled.>") {
		t.Fatal("Unsubscribe of a present pattern should report true")
	}
	if s.Unsubscribe("insights.events.volumeprofile.sampled.>") {
		t.Fatal("Unsubscribe of an absent pattern should report false")
	}
	if s.Matches("insights.events.volumeprofile.sampled.btc_usdt_spot") {
		t.Fatal("session should not match after unsubscribe")
	}
}

func TestSessionSubscriptionsCopy(t *testing.T) {
	s := NewSession("sess-2")
	sub, _ := NewSubscription("insights.events.>")
	s.Subscribe(sub)
	got := s.Subscriptions()
	if len(got) != 1 || got[0] != "insights.events.>" {
		t.Fatalf("Subscriptions() = %v, want [insights.events.>]", got)
	}
	// Mutating the returned slice must not affect the session.
	got[0] = "mutated"
	if !s.Matches("insights.events.x") {
		t.Fatal("mutating the returned slice must not affect session state")
	}
}
