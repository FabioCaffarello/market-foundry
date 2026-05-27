package derive

import (
	"bytes"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appingest "internal/application/ingest"
	"internal/domain/instrument"

	"github.com/anthdm/hollywood/actor"
)

// synthetic_source_canary_integration_test.go (H-6.c.1 commit 8)
//
// This test fixes the regression-shape canary established by commit
// 6's BindingTarget.Instrument() at the derive scope level. Commit 6
// proved the boundary helper rejects synthetic sources in isolation
// (internal/application/ingest binding_test.go); this file proves
// that derive's activation pathway honors that rejection — i.e., the
// pre-commit-6 silent-zero shape (H-6.b' commit 37f8ddd) cannot
// re-enter via a synthetic-source binding.
//
// The 6 synthetic sources mirror the canary list from commit 6 (see
// TestBindingTarget_Instrument_RejectsUnknownSource for the full
// rationale of each entry).
//
// Test strategy:
//
//   - Synthetic-source bindings flow through ParseBindingTopic →
//     BindingTarget → Instrument(). The boundary helper rejects.
//   - source_scope_actor.onActivateSampler must skip activation and
//     emit a structured Error log carrying source/symbol/error.
//   - We exercise the activation pathway by wiring a stand-in
//     activator that delegates to BindingTarget.Instrument() with
//     the same shape as source_scope_actor, captured slog output is
//     asserted for canary visibility.
//   - The legitimate-source path is also exercised so an unintended
//     filter-too-aggressive regression would surface as a failure.

// syntheticSourceCanaries enumerates the synthetic sources that the
// derive scope must reject without spawning child actors. Each entry
// is paired with the symbol that the watcher would observe alongside
// it in production traffic.
var syntheticSourceCanaries = []struct {
	name   string
	source string
	symbol string
	// expectInErr is a substring that the rejection error must
	// contain so the canary log message is actionable for operators.
	expectInErr string
}{
	{
		name:        "binance_no_suffix",
		source:      "binance",
		symbol:      "btcusdt",
		expectInErr: "binance",
	},
	{
		name:        "binance_spot_verify_session_fallback",
		source:      "binance_spot",
		symbol:      "btcusdt",
		expectInErr: "binance_spot",
	},
	{
		name:        "derive_synthetic_scope_id",
		source:      "derive",
		symbol:      "btcusdt",
		expectInErr: "derive",
	},
	{
		name:        "clickhouse_back_compat_tag",
		source:      "clickhouse",
		symbol:      "btcusdt",
		expectInErr: "clickhouse",
	},
	{
		name:        "unknown_exchange_explicit_fallback",
		source:      "unknown_exchange",
		symbol:      "btcusdt",
		expectInErr: "unknown_exchange",
	},
	{
		name:        "execute_venue_adapter_37f8ddd_trigger",
		source:      "execute.venue-adapter",
		symbol:      "btcusdt",
		expectInErr: "execute.venue-adapter",
	},
}

// TestSyntheticSourceCanary_RejectsAtBoundary asserts the derive
// scope's contract on the boundary helper: every synthetic source
// must produce an error from BindingTarget.Instrument() AND the
// error must contain the source name so the canary log emission
// in source_scope_actor remains operator-actionable.
//
// Anti-anti-pattern note: a future change that silently maps an
// unknown source to a zero CanonicalInstrument (the 37f8ddd shape)
// would surface here as a nil error, failing the test.
func TestSyntheticSourceCanary_RejectsAtBoundary(t *testing.T) {
	for _, tc := range syntheticSourceCanaries {
		t.Run(tc.name, func(t *testing.T) {
			target := appingest.BindingTarget{Source: tc.source, Symbol: tc.symbol}
			inst, err := target.Instrument()
			if err == nil {
				t.Fatalf("expected error for synthetic source %q; got Instrument=%+v", tc.source, inst)
			}
			if !inst.IsZero() {
				t.Errorf("expected zero Instrument on error; got %+v", inst)
			}
			if !strings.Contains(err.Error(), tc.expectInErr) {
				t.Errorf("error %q does not contain expected canary substring %q (operators filter logs by source)", err.Error(), tc.expectInErr)
			}
		})
	}
}

// canaryActivator is a stand-in actor that mirrors the
// source_scope_actor.onActivateSampler decision-making for the
// boundary check: on activateSamplerMessage, it calls
// msg.Target.Instrument() and either records the rejection (via
// emitted Error log + tracked rejection counter) or records the
// would-be spawn (via tracked activation counter).
//
// This isolates the boundary-rejection contract from the full
// source_scope_actor (which depends on NATS publishers) while
// preserving the message shape and decision sequence.
type canaryActivator struct {
	logger        *slog.Logger
	rejections    atomic.Int32
	activations   atomic.Int32
	lastRejection chan canaryRejection
}

type canaryRejection struct {
	Source string
	Symbol string
	Err    error
}

func (a *canaryActivator) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started, actor.Stopped, actor.Initialized:
		return
	case activateSamplerMessage:
		_, err := msg.Target.Instrument()
		if err != nil {
			a.rejections.Add(1)
			a.logger.Error("skip sampler activation: invalid binding",
				"source", msg.Target.Source,
				"symbol", msg.Target.Symbol,
				"error", err,
			)
			select {
			case a.lastRejection <- canaryRejection{
				Source: msg.Target.Source,
				Symbol: msg.Target.Symbol,
				Err:    err,
			}:
			default:
			}
			return
		}
		a.activations.Add(1)
	}
}

// TestSyntheticSourceCanary_DeriveActivationFlow asserts the full
// derive activation flow: an activateSamplerMessage carrying a
// synthetic-source BindingTarget reaches an activator that mirrors
// source_scope_actor's onActivateSampler boundary check; the
// activator skips activation (no would-be spawn) and emits a
// structured Error log with the source/symbol/error fields needed
// for monitoring.
//
// Legitimate bindings ("binancef", "btcusdt") exercise the
// activation path so an over-zealous filter would surface as a
// failure.
func TestSyntheticSourceCanary_DeriveActivationFlow(t *testing.T) {
	e := newTestEngine(t)

	for _, tc := range syntheticSourceCanaries {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			activator := &canaryActivator{
				logger:        slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
				lastRejection: make(chan canaryRejection, 1),
			}
			pid := e.Spawn(func() actor.Receiver { return activator }, "canary-activator-"+tc.name)
			defer e.Poison(pid)

			target := appingest.BindingTarget{Source: tc.source, Symbol: tc.symbol}
			e.Send(pid, activateSamplerMessage{Target: target})

			select {
			case rec := <-activator.lastRejection:
				if rec.Source != tc.source {
					t.Errorf("rejection log source = %q, want %q", rec.Source, tc.source)
				}
				if rec.Symbol != tc.symbol {
					t.Errorf("rejection log symbol = %q, want %q", rec.Symbol, tc.symbol)
				}
				if rec.Err == nil {
					t.Errorf("rejection log error must be non-nil for operator visibility")
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("activation did not produce a rejection within 2s — boundary canary BROKEN for source %q (silent-zero regression)", tc.source)
			}

			if activator.activations.Load() != 0 {
				t.Errorf("activations = %d, want 0 (synthetic source must NOT activate)", activator.activations.Load())
			}
			if activator.rejections.Load() != 1 {
				t.Errorf("rejections = %d, want 1", activator.rejections.Load())
			}
			logOut := logBuf.String()
			if !strings.Contains(logOut, "skip sampler activation: invalid binding") {
				t.Errorf("canary log message missing canonical phrase; got:\n%s", logOut)
			}
			if !strings.Contains(logOut, "source="+tc.source) && !strings.Contains(logOut, `source=`+`"`+tc.source+`"`) {
				t.Errorf("canary log must carry source=%q for monitoring filterability; got:\n%s", tc.source, logOut)
			}
		})
	}
}

// TestSyntheticSourceCanary_LegitimateActivationProceeds asserts
// the other side of the canary contract: legitimate venue bindings
// (binances spot + binancef perpetual) MUST proceed past the
// boundary check. A regression that over-rejects would silently
// stop production traffic; this test prevents it.
func TestSyntheticSourceCanary_LegitimateActivationProceeds(t *testing.T) {
	e := newTestEngine(t)

	cases := []struct {
		name         string
		source       string
		symbol       string
		wantContract instrument.ContractType
	}{
		{"binances_spot_btcusdt", "binances", "btcusdt", instrument.ContractSpot},
		{"binancef_perp_btcusdt", "binancef", "btcusdt", instrument.ContractPerpetual},
		{"binancef_perp_ethusdt", "binancef", "ethusdt", instrument.ContractPerpetual},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			activator := &canaryActivator{
				logger:        slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug})),
				lastRejection: make(chan canaryRejection, 1),
			}
			pid := e.Spawn(func() actor.Receiver { return activator }, "canary-activator-legit-"+tc.name)
			defer e.Poison(pid)

			target := appingest.BindingTarget{Source: tc.source, Symbol: tc.symbol}
			inst, instErr := target.Instrument()
			if instErr != nil {
				t.Fatalf("legitimate binding %s.%s unexpectedly rejected: %v", tc.source, tc.symbol, instErr)
			}
			if inst.Contract != tc.wantContract {
				t.Fatalf("legitimate binding contract = %s, want %s", inst.Contract, tc.wantContract)
			}

			e.Send(pid, activateSamplerMessage{Target: target})

			// Allow the activator to process; legitimate path
			// records activation but no rejection.
			deadline := time.After(500 * time.Millisecond)
			for activator.activations.Load() != 1 {
				select {
				case <-deadline:
					t.Fatalf("legitimate activation did not occur within 500ms (activations=%d rejections=%d)", activator.activations.Load(), activator.rejections.Load())
				default:
					time.Sleep(10 * time.Millisecond)
				}
			}

			if activator.rejections.Load() != 0 {
				t.Errorf("rejections = %d, want 0 (legitimate source must NOT be rejected); log:\n%s", activator.rejections.Load(), logBuf.String())
			}
		})
	}
}
