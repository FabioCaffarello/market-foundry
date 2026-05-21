//go:build livemainnet

package execution_test

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

// s441_authenticated_mainnet_proof_test.go — S441: Authenticated Mainnet Proof & Sustained Soak.
//
// These tests exercise REAL AUTHENTICATED calls to Binance mainnet endpoints
// (api.binance.com, fapi.binance.com) using valid API credentials to prove:
//   - HMAC-SHA256 signing works correctly against production
//   - Credential resolution produces valid, accepted keys
//   - Endpoint selection routes to the correct mainnet base URL
//   - The account exists, is reachable, and returns meaningful metadata
//   - Sustained operation remains stable over a defined soak window
//   - DryRunSubmitter remains intact throughout — zero order submission
//
// Build tag: livemainnet — never runs in CI or default `go test`.
// Requires outbound HTTPS and valid credentials:
//   MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY
//   MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET
//   MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY
//   MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET
//
// SAFETY INVARIANT: No test in this file submits real orders. All order paths
// are intercepted by DryRunSubmitter. AccountStatus() is a GET-only endpoint.
//
// Test matrix:
//   AMP-1: Authenticated Spot account status (GET /api/v3/account)
//   AMP-2: Authenticated Futures account status (GET /fapi/v2/account)
//   AMP-3: DryRunSubmitter remains intact after authenticated calls
//   AMP-4: Pipeline chain with authenticated adapter + DryRunSubmitter
//   AMP-5: Sustained soak — repeated authenticated calls over 5-minute window
//   AMP-6: Soak stability — DryRunSubmitter interception verified throughout soak

// ── Helpers ──────────────────────────────────────────────────────────

func requireSpotMainnetCreds(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	key := os.Getenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY")
	secret := os.Getenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET")
	if key == "" || secret == "" {
		t.Skip("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY/SECRET not set — skipping authenticated test")
	}
	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load failed: %s", prob.Message)
	}
	return creds
}

func requireFuturesMainnetCreds(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	key := os.Getenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY")
	secret := os.Getenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET")
	if key == "" || secret == "" {
		t.Skip("MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY/SECRET not set — skipping authenticated test")
	}
	creds, prob := appexec.LoadCredentials("binance_futures_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load failed: %s", prob.Message)
	}
	return creds
}

// soakDuration controls the sustained soak window. Default: 5 minutes.
// Set MF_SOAK_DURATION to override (e.g., "30s" for quick validation).
func soakDuration() time.Duration {
	if v := os.Getenv("MF_SOAK_DURATION"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}
	return 5 * time.Minute
}

// ── AMP-1: Authenticated Spot Account Status ────────────────────────

func TestAuthenticatedMainnet_SpotAccountStatus(t *testing.T) {
	creds := requireSpotMainnetCreds(t)
	adapter := appexec.NewBinanceSpotMainnetAdapter(creds, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, prob := adapter.AccountStatus(ctx)
	if prob != nil {
		t.Fatalf("[AMP-1] authenticated Spot account status failed: %s", prob.Message)
	}

	if info.HTTPStatus != 200 {
		t.Errorf("[AMP-1] expected HTTP 200, got %d", info.HTTPStatus)
	}
	t.Logf("[AMP-1] Spot account: canTrade=%v, accountType=%s, balances=%d",
		info.CanTrade, info.AccountType, info.BalanceCount)

	t.Log("[s441/AMP-1] PASS — authenticated Spot mainnet account status returns valid response")
}

// ── AMP-2: Authenticated Futures Account Status ─────────────────────

func TestAuthenticatedMainnet_FuturesAccountStatus(t *testing.T) {
	creds := requireFuturesMainnetCreds(t)
	adapter := appexec.NewBinanceFuturesMainnetAdapter(creds, 10*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, prob := adapter.AccountStatus(ctx)
	if prob != nil {
		t.Fatalf("[AMP-2] authenticated Futures account status failed: %s", prob.Message)
	}

	if info.HTTPStatus != 200 {
		t.Errorf("[AMP-2] expected HTTP 200, got %d", info.HTTPStatus)
	}
	t.Logf("[AMP-2] Futures account: canTrade=%v, feeTier=%d, assets=%d, positions=%d",
		info.CanTrade, info.FeeTier, info.AssetCount, info.PositionCount)

	t.Log("[s441/AMP-2] PASS — authenticated Futures mainnet account status returns valid response")
}

// ── AMP-3: DryRunSubmitter Remains Intact After Auth Calls ──────────

func TestAuthenticatedMainnet_DryRunIntactAfterAuth(t *testing.T) {
	creds := requireSpotMainnetCreds(t)
	rawAdapter := appexec.NewBinanceSpotMainnetAdapter(creds, 10*time.Second)

	// First: make an authenticated call to prove connectivity.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, prob := rawAdapter.AccountStatus(ctx)
	if prob != nil {
		t.Fatalf("[AMP-3] pre-check auth call failed: %s", prob.Message)
	}
	t.Logf("[AMP-3] auth call succeeded (HTTP %d)", info.HTTPStatus)

	// Second: verify DryRunSubmitter still intercepts everything.
	drs := appexec.NewDryRunSubmitter(rawAdapter)
	intent := domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binances",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Timestamp: time.Now().UTC(),
	}

	receipt, submitProb := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if submitProb != nil {
		t.Fatalf("[AMP-3] DryRunSubmitter error: %s", submitProb.Message)
	}
	if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
		t.Errorf("[AMP-3] expected dryrun- prefix, got %q", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) == 0 || !receipt.Intent.Fills[0].Simulated {
		t.Error("[AMP-3] fill must be Simulated=true")
	}

	t.Log("[s441/AMP-3] PASS — DryRunSubmitter intercepts after authenticated call, no real order submitted")
}

// ── AMP-4: Pipeline Chain with Authenticated Adapter ────────────────

func TestAuthenticatedMainnet_PipelineChainAuth(t *testing.T) {
	spotCreds := requireSpotMainnetCreds(t)
	futuresCreds := requireFuturesMainnetCreds(t)

	// Spot pipeline: adapter → RateLimiter → DryRunSubmitter.
	spotAdapter := appexec.NewBinanceSpotMainnetAdapter(spotCreds, 10*time.Second)
	spotRL := appexec.NewRateLimiter(spotAdapter, 10, 100*time.Millisecond)
	defer spotRL.Close()
	spotDRS := appexec.NewDryRunSubmitter(spotRL)

	// Futures pipeline: adapter → RateLimiter → DryRunSubmitter.
	futuresAdapter := appexec.NewBinanceFuturesMainnetAdapter(futuresCreds, 10*time.Second)
	futuresRL := appexec.NewRateLimiter(futuresAdapter, 10, 100*time.Millisecond)
	defer futuresRL.Close()
	futuresDRS := appexec.NewDryRunSubmitter(futuresRL)

	// Prove auth works on both raw adapters.
	ctx := context.Background()
	spotInfo, prob := spotAdapter.AccountStatus(ctx)
	if prob != nil {
		t.Fatalf("[AMP-4/spot] auth failed: %s", prob.Message)
	}
	t.Logf("[AMP-4/spot] auth OK (HTTP %d)", spotInfo.HTTPStatus)

	futuresInfo, fProb := futuresAdapter.AccountStatus(ctx)
	if fProb != nil {
		t.Fatalf("[AMP-4/futures] auth failed: %s", fProb.Message)
	}
	t.Logf("[AMP-4/futures] auth OK (HTTP %d)", futuresInfo.HTTPStatus)

	// Prove DryRunSubmitter intercepts on both pipelines.
	for _, tc := range []struct {
		name   string
		drs    ports.VenuePort
		source string
	}{
		{"spot", spotDRS, "binances"},
		{"futures", futuresDRS, "binancef"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			intent := domainexec.ExecutionIntent{
				Type:      "paper_order",
				Source:    tc.source,
				Symbol:    "btcusdt",
				Timeframe: 60,
				Side:      domainexec.SideBuy,
				Quantity:  "0.001",
				Status:    domainexec.StatusSubmitted,
				Timestamp: time.Now().UTC(),
			}
			receipt, submitProb := tc.drs.SubmitOrder(ctx, ports.VenueOrderRequest{Intent: intent})
			if submitProb != nil {
				t.Fatalf("DryRunSubmitter error: %s", submitProb.Message)
			}
			if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
				t.Errorf("expected dryrun- prefix, got %q", receipt.VenueOrderID)
			}
			if len(receipt.Intent.Fills) == 0 || !receipt.Intent.Fills[0].Simulated {
				t.Error("fill must be Simulated=true")
			}
		})
	}

	t.Log("[s441/AMP-4] PASS — full pipeline chain with authenticated adapters, DryRunSubmitter intact")
}

// ── AMP-5: Sustained Soak — Repeated Authenticated Calls ────────────

func TestAuthenticatedMainnet_SustainedSoak(t *testing.T) {
	spotCreds := requireSpotMainnetCreds(t)
	futuresCreds := requireFuturesMainnetCreds(t)

	spotAdapter := appexec.NewBinanceSpotMainnetAdapter(spotCreds, 10*time.Second)
	futuresAdapter := appexec.NewBinanceFuturesMainnetAdapter(futuresCreds, 10*time.Second)

	duration := soakDuration()
	// Interval between calls: 15 seconds (4 calls/min per segment = 8 total/min).
	// Well within Binance's rate limit of 1200 req/min.
	interval := 15 * time.Second

	t.Logf("[AMP-5] starting sustained soak: duration=%s, interval=%s", duration, interval)

	var spotOK, spotFail, futuresOK, futuresFail atomic.Int64
	var maxSpotLatency, maxFuturesLatency atomic.Int64

	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial call immediately.
	doSpotCall := func() {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, prob := spotAdapter.AccountStatus(ctx)
		latencyMs := time.Since(start).Milliseconds()
		if prob != nil {
			spotFail.Add(1)
			t.Logf("[AMP-5/spot] FAIL at %s: %s (latency=%dms)", time.Now().Format(time.RFC3339), prob.Message, latencyMs)
		} else {
			spotOK.Add(1)
			for {
				cur := maxSpotLatency.Load()
				if latencyMs <= cur || maxSpotLatency.CompareAndSwap(cur, latencyMs) {
					break
				}
			}
		}
	}

	doFuturesCall := func() {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, prob := futuresAdapter.AccountStatus(ctx)
		latencyMs := time.Since(start).Milliseconds()
		if prob != nil {
			futuresFail.Add(1)
			t.Logf("[AMP-5/futures] FAIL at %s: %s (latency=%dms)", time.Now().Format(time.RFC3339), prob.Message, latencyMs)
		} else {
			futuresOK.Add(1)
			for {
				cur := maxFuturesLatency.Load()
				if latencyMs <= cur || maxFuturesLatency.CompareAndSwap(cur, latencyMs) {
					break
				}
			}
		}
	}

	// Initial round.
	doSpotCall()
	doFuturesCall()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				goto done
			}
			doSpotCall()
			doFuturesCall()
		}
	}

done:
	totalSpot := spotOK.Load() + spotFail.Load()
	totalFutures := futuresOK.Load() + futuresFail.Load()

	t.Logf("[AMP-5] Soak results after %s:", duration)
	t.Logf("  Spot:    %d/%d OK (max_latency=%dms, failures=%d)", spotOK.Load(), totalSpot, maxSpotLatency.Load(), spotFail.Load())
	t.Logf("  Futures: %d/%d OK (max_latency=%dms, failures=%d)", futuresOK.Load(), totalFutures, maxFuturesLatency.Load(), futuresFail.Load())

	// Accept up to 5% failure rate (network jitter, transient venue issues).
	spotFailRate := float64(spotFail.Load()) / float64(max(totalSpot, 1))
	futuresFailRate := float64(futuresFail.Load()) / float64(max(totalFutures, 1))

	if spotFailRate > 0.05 {
		t.Errorf("[AMP-5] Spot failure rate %.1f%% exceeds 5%% threshold", spotFailRate*100)
	}
	if futuresFailRate > 0.05 {
		t.Errorf("[AMP-5] Futures failure rate %.1f%% exceeds 5%% threshold", futuresFailRate*100)
	}

	t.Log("[s441/AMP-5] PASS — sustained authenticated soak completed within tolerance")
}

// ── AMP-6: Soak Stability — DryRunSubmitter Throughout ──────────────

func TestAuthenticatedMainnet_SoakDryRunStability(t *testing.T) {
	spotCreds := requireSpotMainnetCreds(t)
	rawAdapter := appexec.NewBinanceSpotMainnetAdapter(spotCreds, 10*time.Second)
	rl := appexec.NewRateLimiter(rawAdapter, 10, 100*time.Millisecond)
	defer rl.Close()
	drs := appexec.NewDryRunSubmitter(rl)

	// Use shorter soak for DryRunSubmitter stability (it doesn't hit the network).
	duration := soakDuration()
	if duration > 2*time.Minute {
		duration = 2 * time.Minute
	}
	interval := 5 * time.Second

	t.Logf("[AMP-6] DryRunSubmitter stability soak: duration=%s, interval=%s", duration, interval)

	// Intersperse: auth call → DRS submission → auth call → DRS submission.
	deadline := time.Now().Add(duration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var authCalls, drsCalls int
	var drsFailures int

	for time.Now().Before(deadline) {
		// Auth call (real network).
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, prob := rawAdapter.AccountStatus(ctx)
		cancel()
		if prob != nil {
			t.Logf("[AMP-6] auth call failed (non-fatal): %s", prob.Message)
		} else {
			authCalls++
		}

		// DRS interception (no network).
		intent := domainexec.ExecutionIntent{
			Type:      "paper_order",
			Source:    "binances",
			Symbol:    "btcusdt",
			Timeframe: 60,
			Side:      domainexec.SideBuy,
			Quantity:  "0.001",
			Status:    domainexec.StatusSubmitted,
			Timestamp: time.Now().UTC(),
		}
		receipt, submitProb := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
		drsCalls++
		if submitProb != nil {
			drsFailures++
			t.Errorf("[AMP-6] DryRunSubmitter failed: %s", submitProb.Message)
			continue
		}
		if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
			drsFailures++
			t.Errorf("[AMP-6] VenueOrderID missing dryrun- prefix: %q", receipt.VenueOrderID)
		}
		if len(receipt.Intent.Fills) > 0 && !receipt.Intent.Fills[0].Simulated {
			drsFailures++
			t.Error("[AMP-6] fill must be Simulated=true")
		}

		<-ticker.C
	}

	t.Logf("[AMP-6] Completed: authCalls=%d, drsCalls=%d, drsFailures=%d", authCalls, drsCalls, drsFailures)

	if drsFailures > 0 {
		t.Errorf("[AMP-6] DryRunSubmitter had %d failures — interception must be 100%% reliable", drsFailures)
	}

	t.Log("[s441/AMP-6] PASS — DryRunSubmitter interception remained 100%% reliable throughout soak")
}

// ── Summary ─────────────────────────────────────────────────────────

func TestAuthenticatedMainnet_Summary(t *testing.T) {
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("S441 Authenticated Mainnet Proof & Sustained Soak — Summary")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("AMP-1: Authenticated Spot account status (GET /api/v3/account)")
	t.Log("AMP-2: Authenticated Futures account status (GET /fapi/v2/account)")
	t.Log("AMP-3: DryRunSubmitter intact after authenticated calls")
	t.Log("AMP-4: Full pipeline chain with authenticated adapters")
	t.Log("AMP-5: Sustained soak — repeated auth calls over time window")
	t.Log("AMP-6: DryRunSubmitter stability throughout soak")
	t.Log("")
	t.Log("Run with: go test -tags=livemainnet -count=1 -v -run TestAuthenticatedMainnet ./internal/application/execution/...")
	t.Log("Override soak duration: MF_SOAK_DURATION=30s go test -tags=livemainnet ...")
	t.Log("")
	t.Log("SAFETY: AccountStatus() is GET-only (read). DryRunSubmitter blocks all order submission.")
}
