package ingest

import (
	"fmt"
	"strings"

	"internal/domain/instrument"
	"internal/shared/problem"
)

// BindingTarget represents a parsed ingestion binding topic.
// Convention: topic = "{source}.{symbol}" (e.g., "binancef.btcusdt").
type BindingTarget struct {
	Source string
	Symbol string
}

// Key returns the unique identifier for this binding target.
func (b BindingTarget) Key() string {
	return b.Source + "." + b.Symbol
}

// venueSourceContract is the canonical registry of venue source
// identifiers and the ContractType they map to. Sources here are
// production routing identifiers used by configctl ingestion
// bindings (parsed via ParseBindingTopic); synthetic sources used
// as internal labels (e.g., "derive.signal-publisher.binancef") are
// intentionally absent and MUST NOT be added — they signal a code
// path that should pass-through an upstream Instrument rather than
// reconstruct via this registry.
//
// Adding a venue here is a deliberate act (auditable in git):
// new venue support requires adapter, registry, and test coverage.
// H-6.c.1 establishes this registry as the single canonical
// reconstruction point at the (source, symbol) → CanonicalInstrument
// boundary. The duplicated per-package `instrumentFromBinding`
// helpers in application/{signal,decision,strategy,risk} are
// deleted in commits 7a-7d once derive actors switch to
// BindingTarget.Instrument() pass-through.
var venueSourceContract = map[string]instrument.ContractType{
	"binances": instrument.ContractSpot,
	"binancef": instrument.ContractPerpetual,
}

// Instrument resolves the BindingTarget into a canonical
// CanonicalInstrument. Returns an error when:
//   - Source is empty.
//   - Symbol is empty.
//   - Source is not a recognized venue identifier (only those
//     declared in venueSourceContract are accepted).
//   - Symbol does not end in "USDT" (current routing path is
//     USDT-quoted only; non-USDT symbols are rejected at the
//     Binance adapters per H-6.a).
//   - The underlying instrument.New rejects the parsed components.
//
// Callers MUST NOT discard the returned error: silent-zero
// CanonicalInstrument is the exact regression-shape that caused
// commit 37f8ddd in H-6.b' (the strategy_consumer fix). The
// `check instruments` analyzer (H-6.c.1) lists the per-package
// `instrumentFromBinding` helpers as forbidden anti-patterns;
// this method is their canonical replacement.
//
// Symbol matching is case-insensitive (matches the lowercase
// convention from ParseBindingTopic); the parsed Base asset is
// upper-cased per CanonicalInstrument convention.
func (b BindingTarget) Instrument() (instrument.CanonicalInstrument, error) {
	if b.Source == "" {
		return instrument.CanonicalInstrument{}, fmt.Errorf("binding target source is empty")
	}
	if b.Symbol == "" {
		return instrument.CanonicalInstrument{}, fmt.Errorf("binding target symbol is empty for source %q", b.Source)
	}
	contract, ok := venueSourceContract[b.Source]
	if !ok {
		return instrument.CanonicalInstrument{}, fmt.Errorf("binding target source %q is not a recognized venue identifier (registered: binances, binancef)", b.Source)
	}
	upper := strings.ToUpper(strings.TrimSpace(b.Symbol))
	const quote = "USDT"
	if !strings.HasSuffix(upper, quote) || len(upper) <= len(quote) {
		return instrument.CanonicalInstrument{}, fmt.Errorf("binding target symbol %q does not end in USDT (current routing path is USDT-quoted only)", b.Symbol)
	}
	base := upper[:len(upper)-len(quote)]
	inst, prob := instrument.New(base, quote, contract)
	if prob != nil {
		return instrument.CanonicalInstrument{}, fmt.Errorf("build canonical instrument from binding (%s, %s): %s", b.Source, b.Symbol, prob.Message)
	}
	return inst, nil
}

// ParseBindingTopic extracts source and symbol from a binding topic.
func ParseBindingTopic(topic string) (BindingTarget, *problem.Problem) {
	parts := strings.SplitN(topic, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return BindingTarget{}, problem.New(problem.InvalidArgument,
			"binding topic must follow the format 'source.symbol', got: "+topic,
		)
	}

	return BindingTarget{
		Source: strings.ToLower(parts[0]),
		Symbol: strings.ToLower(parts[1]),
	}, nil
}
