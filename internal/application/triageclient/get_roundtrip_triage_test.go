package triageclient

import (
	"context"
	"testing"

	"internal/application/analyticalclient"
	"internal/domain/instrument"
	"internal/domain/pairing"
	"internal/shared/problem"
)

// stubRoundTripReviewer implements RoundTripReviewer for unit tests by
// returning a canned RoundTripReviewReply. It lets the test set the
// embedded pairing.RoundTrip on each review precisely (including the
// canonical Instrument identity) so the projection at line 80 of
// get_roundtrip_triage.go can be exercised directly.
type stubRoundTripReviewer struct {
	reply analyticalclient.RoundTripReviewReply
	prob  *problem.Problem
}

func (s *stubRoundTripReviewer) Execute(_ context.Context, _ analyticalclient.RoundTripReviewQuery) (analyticalclient.RoundTripReviewReply, *problem.Problem) {
	return s.reply, s.prob
}

// btcUSDTSpotForTriage constructs the canonical BTC/USDT-spot instrument
// for fixtures in this package. Mirrors the helper convention used in
// internal/domain/pairing/*_test.go.
func btcUSDTSpotForTriage(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractSpot)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

// TestGetRoundTripTriage_ProjectsVenueSymbolFromInstrument covers the
// boundary projection introduced by H-6.b” commit 3 pull-forward:
// pairing.RoundTrip carries the canonical Instrument; the triage layer
// presents a venue-native string via review.VenueSymbol() at the
// projection site (get_roundtrip_triage.go:80). The test asserts the
// projection produces the expected lowercase venue-native form when
// the embedded RoundTrip.Instrument is populated correctly.
//
// This closes the test-coverage gap flagged in pre-flight 7 of H-6.b”
// (Decision #5β). Before this test, no unit covered get_roundtrip_triage
// and no integration test exercised the path — a silent zero-Instrument
// regression would have shipped invisible to make verify.
func TestGetRoundTripTriage_ProjectsVenueSymbolFromInstrument(t *testing.T) {
	btcSpot := btcUSDTSpotForTriage(t)

	reviewer := &stubRoundTripReviewer{
		reply: analyticalclient.RoundTripReviewReply{
			Reviews: []analyticalclient.RoundTripReviewItem{
				{
					RoundTrip: pairing.RoundTrip{
						Instrument: btcSpot,
						Source:     "binance_spot",
						State:      pairing.StatePaired,
					},
				},
			},
		},
	}

	uc := NewGetRoundTripTriageUseCase(reviewer)
	reply, prob := uc.Execute(context.Background(), RoundTripTriageQuery{
		Source:     "binance_spot",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Items) != 1 {
		t.Fatalf("expected 1 triage item, got %d", len(reply.Items))
	}
	got := reply.Items[0].Symbol
	if got != "btcusdt" {
		t.Errorf("triage item Symbol = %q, want %q (canonical Instrument lowered via VenueSymbol)", got, "btcusdt")
	}
}

// TestGetRoundTripTriage_ZeroInstrumentProducesEmptyString documents
// the regression-detection contract at the projection boundary: when
// the upstream review carries a zero-valued pairing.RoundTrip.Instrument
// (e.g., a future bug where some construction site forgets to populate
// it), VenueSymbol() returns the empty string and the triage item's
// Symbol is visibly "". This is intentional: an empty Symbol in the
// JSON wire shape is observable by operators and downstream consumers,
// matching the regression-detection canary established for H-6.b' /
// commit 37f8ddd (where a silent zero Instrument leaked to wire as an
// empty symbol).
//
// Decision #5β of H-6.b”: make zero-Instrument observable rather than
// silent.
func TestGetRoundTripTriage_ZeroInstrumentProducesEmptyString(t *testing.T) {
	reviewer := &stubRoundTripReviewer{
		reply: analyticalclient.RoundTripReviewReply{
			Reviews: []analyticalclient.RoundTripReviewItem{
				{
					RoundTrip: pairing.RoundTrip{
						// Instrument intentionally left zero —
						// simulates upstream regression.
						Source: "binance_spot",
						State:  pairing.StatePaired,
					},
				},
			},
		},
	}

	uc := NewGetRoundTripTriageUseCase(reviewer)
	reply, prob := uc.Execute(context.Background(), RoundTripTriageQuery{
		Source:     "binance_spot",
		Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual},
		Timeframe:  60,
	})

	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}
	if len(reply.Items) != 1 {
		t.Fatalf("expected 1 triage item, got %d", len(reply.Items))
	}
	got := reply.Items[0].Symbol
	if got != "" {
		t.Errorf("triage item Symbol = %q, want empty string (zero Instrument must surface as empty venue-native form, not as a silent default)", got)
	}
}
