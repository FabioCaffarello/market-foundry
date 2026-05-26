package execution_test

import (
	"regexp"
	"testing"
	"time"

	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
)

// EC-1.1: Same intent → same ID across multiple calls.
func TestClientOrderID_Deterministic(t *testing.T) {
	intent := testBuyIntent(t)

	id1 := appexec.ClientOrderID(intent)
	id2 := appexec.ClientOrderID(intent)
	id3 := appexec.ClientOrderID(intent)

	if id1 != id2 || id2 != id3 {
		t.Fatalf("expected deterministic IDs, got %q, %q, %q", id1, id2, id3)
	}
}

// EC-1.2: Different intents → different IDs.
func TestClientOrderID_Uniqueness(t *testing.T) {
	base := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Side:       domainexec.SideBuy,
		Quantity:   "0.001",
		Status:     domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Final:     true,
		Timestamp: time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
	}

	baseID := appexec.ClientOrderID(base)
	seen := map[string]string{"base": baseID}

	// Vary type.
	v1 := base
	v1.Type = "market_order"
	checkUnique(t, seen, "type", appexec.ClientOrderID(v1))

	// Vary source.
	v2 := base
	v2.Source = "okx"
	checkUnique(t, seen, "source", appexec.ClientOrderID(v2))

	// Vary symbol.
	v3 := base
	v3.Instrument = ethUSDTPerp(t)
	checkUnique(t, seen, "symbol", appexec.ClientOrderID(v3))

	// Vary timeframe.
	v4 := base
	v4.Timeframe = 300
	checkUnique(t, seen, "timeframe", appexec.ClientOrderID(v4))

	// Vary timestamp.
	v5 := base
	v5.Timestamp = base.Timestamp.Add(1 * time.Second)
	checkUnique(t, seen, "timestamp", appexec.ClientOrderID(v5))
}

// EC-1.3: Generated ID conforms to Binance newClientOrderId format constraints.
// Binance: alphanumeric, max 36 characters.
func TestClientOrderID_BinanceFormat(t *testing.T) {
	intent := testBuyIntent(t)
	id := appexec.ClientOrderID(intent)

	if len(id) > 36 {
		t.Fatalf("ID length %d exceeds Binance max 36 chars", len(id))
	}
	if len(id) == 0 {
		t.Fatal("ID must not be empty")
	}

	// Hex chars only (subset of alphanumeric).
	matched, err := regexp.MatchString(`^[0-9a-f]+$`, id)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Fatalf("ID %q contains non-hex characters", id)
	}
}

// EC-1.6: Derivation does not use random or time-varying inputs.
// Verified by calling with the same frozen intent 1000 times.
func TestClientOrderID_NoRandomInputs(t *testing.T) {
	intent := domainexec.ExecutionIntent{
		Type:       "paper_order",
		Source:     "binancef",
		Instrument: btcUSDTPerp(t),
		Timeframe:  60,
		Timestamp:  time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
	}

	expected := appexec.ClientOrderID(intent)
	for i := 0; i < 1000; i++ {
		got := appexec.ClientOrderID(intent)
		if got != expected {
			t.Fatalf("iteration %d: expected %q, got %q — derivation is non-deterministic", i, expected, got)
		}
	}
}

func checkUnique(t *testing.T, seen map[string]string, field, id string) {
	t.Helper()
	for existing, existingID := range seen {
		if id == existingID {
			t.Fatalf("varying %q produced same ID as %q: %s", field, existing, id)
		}
	}
	seen[field] = id
}
