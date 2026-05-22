package derive

// producer_invariant_test.go — S366: Canonical producer-side invariant tests.
//
// Maps each S359 producer-relevant invariant to a concrete test, proving that
// the derive binary's MeanReversionEntryResolverActor satisfies the strategy
// contract on the producer side. Covers structural (PI), behavioral (BI),
// and transport-readiness (TI) invariants as defined in S365.
//
// Governing questions answered:
//   - DIQ-3: Do unit tests prove each invariant on producer side?
//   - DIQ-4 (partial): Does the published event carry correct metadata?

import (
	"fmt"
	"testing"
	"time"

	domainstrategy "internal/domain/strategy"
)

// ── Structural Invariants (PI) ──────────────────────────────────────

// TestPI1_TypeAlwaysMeanReversionEntry proves PI-1: Strategy.Type is always "mean_reversion_entry".
func TestPI1_TypeAlwaysMeanReversionEntry(t *testing.T) {
	cases := []struct {
		name    string
		outcome string
	}{
		{"triggered", "triggered"},
		{"not_triggered", "not_triggered"},
		{"insufficient", "insufficient"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "pi1-pub-"+tc.name)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "pi1-resolver-"+tc.name)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: tc.outcome,
				DecisionConfidence: "0.7500", Timeframe: 60, Timestamp: windowBase(),
			})

			if tc.outcome == "triggered" || tc.outcome == "not_triggered" || tc.outcome == "insufficient" {
				pub.waitFor(t, 1, 2*time.Second)
				s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
				if s.Type != "mean_reversion_entry" {
					t.Errorf("PI-1 violated: want type mean_reversion_entry, got %s", s.Type)
				}
			}
		})
	}
}

// TestPI2_DirectionIsValid proves PI-2: Direction is one of {long, short, flat}.
func TestPI2_DirectionIsValid(t *testing.T) {
	validDirections := map[domainstrategy.Direction]bool{
		domainstrategy.DirectionLong:  true,
		domainstrategy.DirectionShort: true,
		domainstrategy.DirectionFlat:  true,
	}

	cases := []struct {
		name      string
		outcome   string
		wantDir   domainstrategy.Direction
	}{
		{"triggered_long", "triggered", domainstrategy.DirectionLong},
		{"not_triggered_flat", "not_triggered", domainstrategy.DirectionFlat},
		{"insufficient_flat", "insufficient", domainstrategy.DirectionFlat},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "pi2-pub-"+tc.name)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "pi2-resolver-"+tc.name)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: tc.outcome,
				DecisionConfidence: "0.7500", Timeframe: 60, Timestamp: windowBase(),
			})

			pub.waitFor(t, 1, 2*time.Second)
			s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
			if s.Direction != tc.wantDir {
				t.Errorf("PI-2: want direction %s, got %s", tc.wantDir, s.Direction)
			}
			if !validDirections[s.Direction] {
				t.Errorf("PI-2 violated: direction %q is not in {long, short, flat}", s.Direction)
			}
		})
	}
}

// TestPI3_ConfidenceIsValidDecimal proves PI-3: Confidence is a valid decimal string in [0.0, 1.0].
func TestPI3_ConfidenceIsValidDecimal(t *testing.T) {
	cases := []struct {
		name       string
		outcome    string
		confidence string
		severity   string
		wantConf   string
	}{
		{"triggered_high", "triggered", "0.8500", "high", "0.8500"},
		{"triggered_moderate", "triggered", "0.8500", "moderate", "0.7650"},
		{"triggered_low", "triggered", "0.8500", "low", "0.6800"},
		{"not_triggered", "not_triggered", "0.9000", "", "0.0000"},
		{"insufficient", "insufficient", "0.0000", "", "0.0000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "pi3-pub-"+tc.name)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "pi3-resolver-"+tc.name)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: tc.outcome,
				DecisionConfidence: tc.confidence, DecisionSeverity: tc.severity,
				Timeframe: 60, Timestamp: windowBase(),
			})

			pub.waitFor(t, 1, 2*time.Second)
			s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
			if s.Confidence != tc.wantConf {
				t.Errorf("PI-3: want confidence %s, got %s", tc.wantConf, s.Confidence)
			}
		})
	}
}

// TestPI4_DecisionsHasExactlyOneEntry proves PI-4: Decisions array has exactly one entry.
func TestPI4_DecisionsHasExactlyOneEntry(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "pi4-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "pi4-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	if len(s.Decisions) != 1 {
		t.Fatalf("PI-4 violated: want exactly 1 decision, got %d", len(s.Decisions))
	}
	d := s.Decisions[0]
	if d.Type != "rsi_oversold" {
		t.Errorf("PI-4: decision type want rsi_oversold, got %s", d.Type)
	}
	if d.Outcome != "triggered" {
		t.Errorf("PI-4: decision outcome want triggered, got %s", d.Outcome)
	}
	if d.Confidence != "0.8000" {
		t.Errorf("PI-4: decision confidence (raw) want 0.8000, got %s", d.Confidence)
	}
}

// TestPI5_FinalAlwaysTrue proves PI-5: Final flag is always true.
func TestPI5_FinalAlwaysTrue(t *testing.T) {
	outcomes := []string{"triggered", "not_triggered", "insufficient"}

	for _, outcome := range outcomes {
		t.Run(outcome, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "pi5-pub-"+outcome)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "pi5-resolver-"+outcome)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: outcome,
				DecisionConfidence: "0.7500", Timeframe: 60, Timestamp: windowBase(),
			})

			pub.waitFor(t, 1, 2*time.Second)
			s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
			if !s.Final {
				t.Errorf("PI-5 violated: Final must be true for outcome %s", outcome)
			}
		})
	}
}

// TestPI6_TimestampIsDecisionTimestamp proves PI-6: Timestamp is the decision timestamp, not time.Now().
func TestPI6_TimestampIsDecisionTimestamp(t *testing.T) {
	decisionTS := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "pi6-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "pi6-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.7500", Timeframe: 60, Timestamp: decisionTS,
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	if !s.Timestamp.Equal(decisionTS) {
		t.Errorf("PI-6 violated: want timestamp %v (decision time), got %v", decisionTS, s.Timestamp)
	}
}

// ── Behavioral Invariants (BI) ──────────────────────────────────────

// TestBI1_ResolutionIsDeterministic proves BI-1: Same input → same output.
func TestBI1_ResolutionIsDeterministic(t *testing.T) {
	ts := windowBase()

	for i := 0; i < 3; i++ {
		e := newTestEngine(t)
		pub := newMsgCollector()
		pubPID := e.Spawn(pub.producer(), "bi1-pub")
		resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
			Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
			StrategyPublisherPID: pubPID,
		}), "bi1-resolver")
		time.Sleep(50 * time.Millisecond)

		e.Send(resolverPID, decisionEvaluatedMessage{
			Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
			DecisionConfidence: "0.8500", DecisionSeverity: "high",
			Timeframe: 60, Timestamp: ts,
		})

		pub.waitFor(t, 1, 2*time.Second)
		s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
		if s.Direction != domainstrategy.DirectionLong {
			t.Errorf("BI-1 iteration %d: direction want long, got %s", i, s.Direction)
		}
		if s.Confidence != "0.8500" {
			t.Errorf("BI-1 iteration %d: confidence want 0.8500, got %s", i, s.Confidence)
		}
		if s.Parameters["target_offset"] != "0.03" {
			t.Errorf("BI-1 iteration %d: target_offset want 0.03, got %s", i, s.Parameters["target_offset"])
		}
		if s.Parameters["stop_offset"] != "0.01" {
			t.Errorf("BI-1 iteration %d: stop_offset want 0.01, got %s", i, s.Parameters["stop_offset"])
		}
	}
}

// TestBI3_UnknownDecisionOutcome_NeverProducesEvent proves BI-3.
func TestBI3_UnknownDecisionOutcome_NeverProducesEvent(t *testing.T) {
	unknownOutcomes := []string{"unknown", "error", "partial", "timeout", ""}

	for _, outcome := range unknownOutcomes {
		t.Run(outcome, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "bi3-pub-"+outcome)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "bi3-resolver-"+outcome)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: outcome,
				DecisionConfidence: "0.5000", Timeframe: 60, Timestamp: windowBase(),
			})

			time.Sleep(200 * time.Millisecond)
			if pub.count() != 0 {
				t.Errorf("BI-3 violated: outcome %q should not produce an event, got %d", outcome, pub.count())
			}
		})
	}
}

// TestBI5_FlatDirection_ZeroConfidence proves BI-5: flat direction always has zero confidence.
func TestBI5_FlatDirection_ZeroConfidence(t *testing.T) {
	flatOutcomes := []string{"not_triggered", "insufficient"}

	for _, outcome := range flatOutcomes {
		t.Run(outcome, func(t *testing.T) {
			e := newTestEngine(t)
			pub := newMsgCollector()
			pubPID := e.Spawn(pub.producer(), "bi5-pub-"+outcome)
			resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
				Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
				StrategyPublisherPID: pubPID,
			}), "bi5-resolver-"+outcome)
			time.Sleep(50 * time.Millisecond)

			e.Send(resolverPID, decisionEvaluatedMessage{
				Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: outcome,
				DecisionConfidence: "0.9500", Timeframe: 60, Timestamp: windowBase(),
			})

			pub.waitFor(t, 1, 2*time.Second)
			s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
			if s.Direction != domainstrategy.DirectionFlat {
				t.Errorf("BI-5: direction should be flat, got %s", s.Direction)
			}
			if s.Confidence != "0.0000" {
				t.Errorf("BI-5 violated: flat direction must have 0.0000 confidence, got %s", s.Confidence)
			}
		})
	}
}

// TestBI6_EventMetadata_ConstructedOnceImmutable proves BI-6: event metadata is constructed once.
func TestBI6_EventMetadata_ConstructedOnceImmutable(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "bi6-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "bi6-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
		CorrelationID: "corr-bi6", CausationID: "cause-bi6",
	})

	pub.waitFor(t, 1, 2*time.Second)
	event := pub.messages()[0].(publishStrategyMessage).Event

	// Metadata.ID must be non-empty (UUID).
	if event.Metadata.ID == "" {
		t.Fatal("BI-6: Metadata.ID must be non-empty")
	}
	// CorrelationID must be propagated (not generated).
	if event.Metadata.CorrelationID != "corr-bi6" {
		t.Errorf("BI-6: CorrelationID want corr-bi6, got %s", event.Metadata.CorrelationID)
	}
	// CausationID must be propagated from decision fan-out.
	if event.Metadata.CausationID != "cause-bi6" {
		t.Errorf("BI-6: CausationID want cause-bi6, got %s", event.Metadata.CausationID)
	}
}

// ── Transport-Readiness Invariants (TI) ─────────────────────────────

// TestTI_DeduplicationKey_DeterministicFromStrategy proves TI-2: dedup key is deterministic.
func TestTI_DeduplicationKey_DeterministicFromStrategy(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "ti2-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "ti2-resolver")
	time.Sleep(50 * time.Millisecond)

	ts := windowBase()
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: ts,
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy

	// P4.1.10: dedup key precision is nanoseconds (was seconds);
	// prevents silent JetStream dedup drops under rapid same-second
	// publishes.
	wantKey := fmt.Sprintf("strat:mean_reversion_entry:binancef:btcusdt:60:%d", ts.UnixNano())
	gotKey := s.DeduplicationKey()
	if gotKey != wantKey {
		t.Errorf("TI-2: dedup key want %s, got %s", wantKey, gotKey)
	}
}

// TestTI_CorrelationIDAndCausationID_PassedToEvent proves TI-4: IDs flow to event metadata.
func TestTI_CorrelationIDAndCausationID_PassedToEvent(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "ti4-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "ti4-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
		CorrelationID: "trace-ti4", CausationID: "parent-ti4",
	})

	pub.waitFor(t, 1, 2*time.Second)
	event := pub.messages()[0].(publishStrategyMessage).Event

	if event.Metadata.CorrelationID != "trace-ti4" {
		t.Errorf("TI-4: CorrelationID want trace-ti4, got %s", event.Metadata.CorrelationID)
	}
	if event.Metadata.CausationID != "parent-ti4" {
		t.Errorf("TI-4: CausationID want parent-ti4, got %s", event.Metadata.CausationID)
	}
}

// TestTI_EventValidation_GatesPublish proves TI: Validate() gates publication.
func TestTI_EventValidation_GatesPublish(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "ti-val-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "ti-val-resolver")
	time.Sleep(50 * time.Millisecond)

	// Valid decision → produces valid event.
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
	})

	pub.waitFor(t, 1, 2*time.Second)
	event := pub.messages()[0].(publishStrategyMessage).Event
	if prob := event.Strategy.Validate(); prob != nil {
		t.Errorf("TI validation: event should be valid, got %s", prob.Message)
	}
}

// ── S359 Contract Invariant Coverage ────────────────────────────────

// TestINV1_TypeIdentity proves INV-1: type is mean_reversion_entry.
// (Covered by PI-1; included for explicit S359 mapping.)
func TestINV1_TypeIdentity(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "inv1-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "inv1-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	if s.Type != "mean_reversion_entry" {
		t.Errorf("INV-1: type must be mean_reversion_entry, got %s", s.Type)
	}
}

// TestINV3_CausationChain proves INV-3: causation chain links to upstream decision.
func TestINV3_CausationChain(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "inv3-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "inv3-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: windowBase(),
		CorrelationID: "trace-inv3", CausationID: "decision-event-001",
	})

	pub.waitFor(t, 1, 2*time.Second)
	event := pub.messages()[0].(publishStrategyMessage).Event

	// CorrelationID must be propagated (never generated).
	if event.Metadata.CorrelationID != "trace-inv3" {
		t.Errorf("INV-3: CorrelationID must propagate, got %s", event.Metadata.CorrelationID)
	}
	// CausationID must equal the decision event's ID.
	if event.Metadata.CausationID != "decision-event-001" {
		t.Errorf("INV-3: CausationID must be decision event ID, got %s", event.Metadata.CausationID)
	}
	// Metadata.ID must be fresh (not equal to CausationID).
	if event.Metadata.ID == event.Metadata.CausationID {
		t.Error("INV-3: Metadata.ID must be fresh, not equal to CausationID")
	}
}

// TestINV5_TimestampMonotonicity proves INV-5: timestamp is from decision (source-derived).
func TestINV5_TimestampMonotonicity(t *testing.T) {
	decTS := time.Date(2025, 7, 1, 10, 30, 0, 0, time.UTC)

	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "inv5-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "inv5-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: decTS,
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	if !s.Timestamp.Equal(decTS) {
		t.Errorf("INV-5: timestamp must be decision timestamp %v, got %v", decTS, s.Timestamp)
	}
}

// TestINV7_FlatMeansNoExecution proves INV-7: flat direction = no execution intended.
func TestINV7_FlatMeansNoExecution(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "inv7-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "inv7-resolver")
	time.Sleep(50 * time.Millisecond)

	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "not_triggered",
		DecisionConfidence: "0.9000", Timeframe: 60, Timestamp: windowBase(),
	})

	pub.waitFor(t, 1, 2*time.Second)
	s := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	if s.Direction != domainstrategy.DirectionFlat {
		t.Errorf("INV-7: direction should be flat, got %s", s.Direction)
	}
	if s.Confidence != "0.0000" {
		t.Errorf("INV-7: flat confidence should be 0.0000, got %s", s.Confidence)
	}
	if s.Parameters != nil {
		t.Errorf("INV-7: flat should have nil parameters, got %v", s.Parameters)
	}
}

// TestINV11_DeduplicationKeyUniqueness proves INV-11: dedup key uniqueness per event.
func TestINV11_DeduplicationKeyUniqueness(t *testing.T) {
	e := newTestEngine(t)
	pub := newMsgCollector()
	pubPID := e.Spawn(pub.producer(), "inv11-pub")
	resolverPID := e.Spawn(NewMeanReversionEntryResolverActor(StrategyResolverConfig{
		Source: "binancef", Symbol: "btcusdt", Timeframe: 60 * time.Second,
		StrategyPublisherPID: pubPID,
	}), "inv11-resolver")
	time.Sleep(50 * time.Millisecond)

	base := windowBase()
	// Send two decisions with different timestamps → different dedup keys.
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: base,
	})
	e.Send(resolverPID, decisionEvaluatedMessage{
		Symbol: "btcusdt", DecisionType: "rsi_oversold", DecisionOutcome: "triggered",
		DecisionConfidence: "0.8000", Timeframe: 60, Timestamp: base.Add(time.Minute),
	})

	pub.waitFor(t, 2, 2*time.Second)
	s0 := pub.messages()[0].(publishStrategyMessage).Event.Strategy
	s1 := pub.messages()[1].(publishStrategyMessage).Event.Strategy

	k0 := s0.DeduplicationKey()
	k1 := s1.DeduplicationKey()
	if k0 == k1 {
		t.Errorf("INV-11 violated: dedup keys must differ for different timestamps, both %s", k0)
	}
}

