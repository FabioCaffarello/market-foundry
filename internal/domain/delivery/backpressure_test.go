package delivery

import "testing"

func TestParseBackpressurePolicy(t *testing.T) {
	cases := map[string]BackpressurePolicy{
		"":            DropNewest,
		"drop_newest": DropNewest,
		"drop-newest": DropNewest,
		"DropNewest":  DropNewest,
		"drop_oldest": DropOldest,
		"drop-oldest": DropOldest,
		"DropOldest":  DropOldest,
	}
	for in, want := range cases {
		got, prob := ParseBackpressurePolicy(in)
		if prob != nil {
			t.Fatalf("ParseBackpressurePolicy(%q) unexpected problem: %v", in, prob)
		}
		if got != want {
			t.Fatalf("ParseBackpressurePolicy(%q) = %v, want %v", in, got, want)
		}
	}

	if _, prob := ParseBackpressurePolicy("priority_drop"); prob == nil {
		t.Fatal("expected a problem for an unknown/deferred policy")
	}
}

func TestBackpressurePolicyStringRoundtrip(t *testing.T) {
	for _, p := range []BackpressurePolicy{DropNewest, DropOldest} {
		got, prob := ParseBackpressurePolicy(p.String())
		if prob != nil {
			t.Fatalf("round-trip parse of %q failed: %v", p.String(), prob)
		}
		if got != p {
			t.Fatalf("round-trip %v → %q → %v", p, p.String(), got)
		}
		if prob := p.Validate(); prob != nil {
			t.Fatalf("Validate(%v) unexpected problem: %v", p, prob)
		}
	}

	if prob := BackpressurePolicy(99).Validate(); prob == nil {
		t.Fatal("expected Validate to reject an unknown policy value")
	}
}
