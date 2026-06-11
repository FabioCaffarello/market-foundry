package executionclient_test

import (
	"context"
	"testing"
	"time"

	"internal/application/executionclient"
	"internal/domain/execution"
)

func TestVerifySession_DerivesScope_ClosedSession(t *testing.T) {
	now := time.Now().UTC()
	closed := now
	session := &execution.Session{
		SessionID: "session_20260324_120000",
		Operator:  "test",
		Status:    execution.SessionClosed,
		StartedAt: now.Add(-2 * time.Hour),
		ClosedAt:  &closed,
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance_spot",
			DryRun:    true,
			Segments:  []string{"spot"},
		},
	}

	uc := executionclient.NewVerifySessionUseCase(
		&stubSessionReader{session: session},
		&stubGateReader{status: execution.GateHalted},
		&stubCHSummary{total: 3},
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{Symbol: "BTCUSDT", Type: "venue_market_order", Status: "filled",
				Fills: []execution.FillRecord{{Fee: "0.001", FeeAsset: "BNB"}}},
		}},
		nil,
	)

	reply, prob := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: session.SessionID})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	// S485: Report should carry session-derived scope.
	if reply.Report.Scope == nil {
		t.Fatal("expected scope in report")
	}

	scope := reply.Report.Scope
	if scope.VenueType != "binance_spot" {
		t.Errorf("expected venue_type binance_spot, got %s", scope.VenueType)
	}
	if !scope.DryRun {
		t.Error("expected dry_run=true in scope")
	}
	if len(scope.Segments) != 1 || scope.Segments[0] != "spot" {
		t.Errorf("expected segments [spot], got %v", scope.Segments)
	}

	// Time bounds should be derived from session.
	expectedSince := session.StartedAt.Add(-5 * time.Minute)
	expectedUntil := closed.Add(5 * time.Minute)
	if scope.Since.Before(expectedSince.Add(-time.Second)) || scope.Since.After(expectedSince.Add(time.Second)) {
		t.Errorf("scope.Since %v not close to expected %v", scope.Since, expectedSince)
	}
	if scope.Until.Before(expectedUntil.Add(-time.Second)) || scope.Until.After(expectedUntil.Add(time.Second)) {
		t.Errorf("scope.Until %v not close to expected %v", scope.Until, expectedUntil)
	}
}

func TestVerifySession_DerivesScope_NilSession(t *testing.T) {
	// When session reader is nil, should use default scope (24h/BTCUSDT).
	uc := executionclient.NewVerifySessionUseCase(nil, nil, nil, nil, nil)
	reply, prob := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob)
	}

	if reply.Report.Scope == nil {
		t.Fatal("expected scope in report even without session")
	}
	if len(reply.Report.Scope.Symbols) != 1 || reply.Report.Scope.Symbols[0] != "btcusdt" {
		t.Errorf("expected default symbol btcusdt, got %v", reply.Report.Scope.Symbols)
	}
}

func TestVerifySession_ScopeContainment_UsesAllowedSymbols(t *testing.T) {
	now := time.Now().UTC()
	closed := now
	session := &execution.Session{
		SessionID: "session_20260324_120000",
		Status:    execution.SessionClosed,
		StartedAt: now.Add(-1 * time.Hour),
		ClosedAt:  &closed,
		Config: execution.SessionConfigSnapshot{
			VenueType: "binance_spot",
			Segments:  []string{"spot"},
		},
	}

	// BTCUSDT is allowed (derived from scope), ETHUSDT is out-of-scope.
	uc := executionclient.NewVerifySessionUseCase(
		&stubSessionReader{session: session},
		nil, nil,
		&stubCHLister{rows: []executionclient.VerifyCHListResult{
			{Symbol: "BTCUSDT", Type: "venue_market_order"},
			{Symbol: "ETHUSDT", Type: "venue_market_order"},
		}},
		nil,
	)

	reply, _ := uc.Execute(context.Background(), executionclient.SessionVerifyQuery{SessionID: "test"})

	po9 := reply.Report.Checks[8]
	if po9.CheckID != execution.POCheckScopeContainment {
		t.Fatalf("expected PO-9, got %s", po9.CheckID)
	}
	if po9.Verdict != execution.VerdictFail {
		t.Errorf("PO-9 should fail with out-of-scope ETHUSDT, got %s", po9.Verdict)
	}
	if po9.Evidence["out_of_scope"] != 1 {
		t.Errorf("expected 1 out_of_scope, got %v", po9.Evidence["out_of_scope"])
	}
}
