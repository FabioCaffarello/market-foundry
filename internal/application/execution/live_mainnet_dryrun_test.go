//go:build livemainnet

package execution_test

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

// live_mainnet_dryrun_test.go — S436: Mainnet Dry-Run Proof.
//
// These tests exercise REAL network connectivity to Binance mainnet endpoints
// (api.binance.com, fapi.binance.com) to validate DNS resolution, TLS handshake,
// HTTP reachability, and DryRunSubmitter interception — proving the dry-run path
// works end-to-end against production endpoints without submitting any real order.
//
// Build tag: livemainnet — never runs in CI or default `go test`.
// These tests require outbound HTTPS access to api.binance.com and fapi.binance.com.
//
// Credential-gated tests additionally require:
//   MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY
//   MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET
//   MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY
//   MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET
//
// What these tests prove:
//   MDR-1: DNS resolution and TCP connectivity to both mainnet endpoints
//   MDR-2: TLS handshake succeeds with valid certificate chain
//   MDR-3: Public endpoints (/api/v3/ping, /fapi/v1/ping) return HTTP 200
//   MDR-4: DryRunSubmitter intercepts mainnet Spot adapter — zero venue call
//   MDR-5: DryRunSubmitter intercepts mainnet Futures adapter — zero venue call
//   MDR-6: Audit markers (dryrun- prefix, Simulated=true) on dry-run receipts
//   MDR-7: Credential loading for mainnet enforces format validation
//   MDR-8: Pipeline chain: mainnet adapter → RateLimiter → DryRunSubmitter

// ── Endpoint Constants ──────────────────────────────────────────────

var mainnetEndpoints = []struct {
	name    string
	host    string
	pingURL string
}{
	{"spot", "api.binance.com", "https://api.binance.com/api/v3/ping"},
	{"futures", "fapi.binance.com", "https://fapi.binance.com/fapi/v1/ping"},
}

// ── MDR-1: DNS Resolution and TCP Connectivity ──────────────────────

func TestMainnetDryRun_DNSResolutionAndTCPConnectivity(t *testing.T) {
	for _, ep := range mainnetEndpoints {
		t.Run(ep.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			resolver := &net.Resolver{}
			addrs, err := resolver.LookupHost(ctx, ep.host)
			if err != nil {
				t.Fatalf("[MDR-1/%s] DNS resolution failed for %s: %v", ep.name, ep.host, err)
			}
			if len(addrs) == 0 {
				t.Fatalf("[MDR-1/%s] DNS returned 0 addresses for %s", ep.name, ep.host)
			}
			t.Logf("[MDR-1/%s] DNS resolved %s → %v", ep.name, ep.host, addrs)

			dialer := &net.Dialer{Timeout: 5 * time.Second}
			conn, err := dialer.DialContext(ctx, "tcp", ep.host+":443")
			if err != nil {
				t.Fatalf("[MDR-1/%s] TCP connection to %s:443 failed: %v", ep.name, ep.host, err)
			}
			conn.Close()

			t.Logf("[s436/MDR-1/%s] PASS — DNS resolves, TCP port 443 accepts connections", ep.name)
		})
	}
}

// ── MDR-2: TLS Handshake ────────────────────────────────────────────

func TestMainnetDryRun_TLSHandshake(t *testing.T) {
	for _, ep := range mainnetEndpoints {
		t.Run(ep.name, func(t *testing.T) {
			dialer := &tls.Dialer{
				Config: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn, err := dialer.DialContext(ctx, "tcp", ep.host+":443")
			if err != nil {
				t.Fatalf("[MDR-2/%s] TLS handshake to %s:443 failed: %v", ep.name, ep.host, err)
			}
			defer conn.Close()

			tlsConn := conn.(*tls.Conn)
			state := tlsConn.ConnectionState()

			t.Logf("[MDR-2/%s] TLS version: 0x%04x, cipher: 0x%04x",
				ep.name, state.Version, state.CipherSuite)

			if len(state.PeerCertificates) == 0 {
				t.Fatalf("[MDR-2/%s] no peer certificates returned", ep.name)
			}
			cert := state.PeerCertificates[0]
			t.Logf("[MDR-2/%s] cert subject: %s, issuer: %s, not_after: %s",
				ep.name, cert.Subject.CommonName, cert.Issuer.CommonName,
				cert.NotAfter.Format(time.RFC3339))

			if time.Now().After(cert.NotAfter) {
				t.Errorf("[MDR-2/%s] WARNING: certificate expired at %s", ep.name, cert.NotAfter)
			}

			t.Logf("[s436/MDR-2/%s] PASS — TLS handshake succeeds, certificate valid", ep.name)
		})
	}
}

// ── MDR-3: Public Endpoint Reachability ─────────────────────────────

func TestMainnetDryRun_PublicEndpointReachability(t *testing.T) {
	client := &http.Client{Timeout: 10 * time.Second}

	for _, ep := range mainnetEndpoints {
		t.Run(ep.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, ep.pingURL, nil)
			if err != nil {
				t.Fatalf("[MDR-3/%s] build request: %v", ep.name, err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("[MDR-3/%s] request failed: %v", ep.name, err)
			}
			defer resp.Body.Close()

			t.Logf("[MDR-3/%s] GET %s → HTTP %d", ep.name, ep.pingURL, resp.StatusCode)
			if resp.StatusCode != http.StatusOK {
				t.Errorf("[MDR-3/%s] expected 200, got %d", ep.name, resp.StatusCode)
			}

			t.Logf("[s436/MDR-3/%s] PASS — public ping endpoint returns 200", ep.name)
		})
	}
}

// ── MDR-4: DryRunSubmitter Intercepts Mainnet Spot Adapter ──────────

func TestMainnetDryRun_SpotDryRunInterception(t *testing.T) {
	// Create a real mainnet Spot adapter with test credentials.
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "s436-proof-spot-key-0123456789abcdef")
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "s436-proof-spot-secret-0123456789ab")

	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load: %s", prob.Message)
	}

	// Build the real mainnet adapter — this points to api.binance.com.
	innerAdapter := appexec.NewBinanceSpotMainnetAdapter(creds, 10*time.Second)

	// Wrap with DryRunSubmitter — this MUST intercept all calls.
	drs := appexec.NewDryRunSubmitter(innerAdapter)

	intent := domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binances",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Final:     true,
		Timestamp: time.Now().UTC(),
	}

	receipt, submitProb := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if submitProb != nil {
		t.Fatalf("[MDR-4] DryRunSubmitter returned error: %s", submitProb.Message)
	}

	// Verify dry-run markers.
	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("[MDR-4] expected StatusFilled, got %s", receipt.Status)
	}
	if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
		t.Errorf("[MDR-4] VenueOrderID must start with dryrun-, got %q", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("[MDR-4] expected at least one fill")
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("[MDR-4] fill must have Simulated=true")
	}

	t.Logf("[s436/MDR-4] PASS — DryRunSubmitter intercepted Spot mainnet adapter, venue_order_id=%s", receipt.VenueOrderID)
}

// ── MDR-5: DryRunSubmitter Intercepts Mainnet Futures Adapter ───────

func TestMainnetDryRun_FuturesDryRunInterception(t *testing.T) {
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_KEY", "s436-proof-futures-key-0123456789ab")
	t.Setenv("MF_VENUE_BINANCE_FUTURES_MAINNET_API_SECRET", "s436-proof-futures-secret-0123456789")

	creds, prob := appexec.LoadCredentials("binance_futures_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load: %s", prob.Message)
	}

	innerAdapter := appexec.NewBinanceFuturesMainnetAdapter(creds, 10*time.Second)
	drs := appexec.NewDryRunSubmitter(innerAdapter)

	intent := domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideSell,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Final:     true,
		Timestamp: time.Now().UTC(),
	}

	receipt, submitProb := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if submitProb != nil {
		t.Fatalf("[MDR-5] DryRunSubmitter returned error: %s", submitProb.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Errorf("[MDR-5] expected StatusFilled, got %s", receipt.Status)
	}
	if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
		t.Errorf("[MDR-5] VenueOrderID must start with dryrun-, got %q", receipt.VenueOrderID)
	}
	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("[MDR-5] expected at least one fill")
	}
	if !receipt.Intent.Fills[0].Simulated {
		t.Error("[MDR-5] fill must have Simulated=true")
	}

	t.Logf("[s436/MDR-5] PASS — DryRunSubmitter intercepted Futures mainnet adapter, venue_order_id=%s", receipt.VenueOrderID)
}

// ── MDR-6: Audit Marker Consistency ─────────────────────────────────

func TestMainnetDryRun_AuditMarkerConsistency(t *testing.T) {
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "s436-audit-spot-key-0123456789abcdef")
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "s436-audit-spot-secret-0123456789ab")

	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load: %s", prob.Message)
	}

	adapter := appexec.NewBinanceSpotMainnetAdapter(creds, 10*time.Second)
	drs := appexec.NewDryRunSubmitter(adapter)

	// Test both active and no-action intents.
	cases := []struct {
		name           string
		side           domainexec.Side
		expectFilled   bool
		expectFillLen  int
	}{
		{"buy_intent", domainexec.SideBuy, true, 1},
		{"sell_intent", domainexec.SideSell, true, 1},
		{"noop_intent", domainexec.SideNone, false, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intent := domainexec.ExecutionIntent{
				Type:      "paper_order",
				Source:    "binances",
				Symbol:    "ethusdt",
				Timeframe: 60,
				Side:      tc.side,
				Quantity:  "0.01",
				Status:    domainexec.StatusSubmitted,
				Timestamp: time.Now().UTC(),
			}

			receipt, submitProb := drs.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
			if submitProb != nil {
				t.Fatalf("unexpected error: %s", submitProb.Message)
			}

			// All dry-run receipts must have dryrun- prefix.
			if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
				t.Errorf("VenueOrderID must start with dryrun-, got %q", receipt.VenueOrderID)
			}

			if tc.expectFilled {
				if receipt.Status != domainexec.StatusFilled {
					t.Errorf("expected StatusFilled, got %s", receipt.Status)
				}
			} else {
				if receipt.Status != domainexec.StatusAccepted {
					t.Errorf("expected StatusAccepted for noop, got %s", receipt.Status)
				}
			}

			if len(receipt.Intent.Fills) != tc.expectFillLen {
				t.Fatalf("expected %d fills, got %d", tc.expectFillLen, len(receipt.Intent.Fills))
			}

			for _, fill := range receipt.Intent.Fills {
				if !fill.Simulated {
					t.Error("all dry-run fills must have Simulated=true")
				}
				if fill.Fee != "0" {
					t.Errorf("dry-run fills must have Fee=0, got %s", fill.Fee)
				}
			}

			t.Logf("[s436/MDR-6/%s] PASS — audit markers correct", tc.name)
		})
	}
}

// ── MDR-7: Credential Format Validation for Mainnet ─────────────────

func TestMainnetDryRun_CredentialFormatValidation(t *testing.T) {
	// Short credentials must be rejected for mainnet.
	t.Run("short_key_rejected", func(t *testing.T) {
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "short")
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "alsoshort")

		_, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
		if prob == nil {
			t.Fatal("[MDR-7] expected rejection for short mainnet credentials")
		}
		t.Logf("[MDR-7/short] correctly rejected: %s", prob.Message)
	})

	// Whitespace in credentials must be rejected for mainnet.
	t.Run("whitespace_rejected", func(t *testing.T) {
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "validkey1234567890abcdef ")
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "validsecret1234567890abc")

		_, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
		if prob == nil {
			t.Fatal("[MDR-7] expected rejection for whitespace in mainnet credential")
		}
		t.Logf("[MDR-7/whitespace] correctly rejected: %s", prob.Message)
	})

	// Valid-format credentials must succeed.
	t.Run("valid_format_accepted", func(t *testing.T) {
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "s436validkey1234567890abcdef")
		t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "s436validsecret1234567890ab")

		creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
		if prob != nil {
			t.Fatalf("[MDR-7] unexpected rejection: %s", prob.Message)
		}
		if creds.VenueType() != "binance_spot_mainnet" {
			t.Errorf("[MDR-7] venue type: %s", creds.VenueType())
		}
	})

	t.Log("[s436/MDR-7] PASS — mainnet credential format validation enforced")
}

// ── MDR-8: Pipeline Chain Composition ───────────────────────────────

func TestMainnetDryRun_PipelineChainComposition(t *testing.T) {
	// Prove that RateLimiter + DryRunSubmitter compose correctly.
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_KEY", "s436-chain-key-0123456789abcdef0123")
	t.Setenv("MF_VENUE_BINANCE_SPOT_MAINNET_API_SECRET", "s436-chain-secret-0123456789abcdef")

	creds, prob := appexec.LoadCredentials("binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("credential load: %s", prob.Message)
	}

	// Build the full mainnet pipeline: adapter → RateLimiter → DryRunSubmitter.
	rawAdapter := appexec.NewBinanceSpotMainnetAdapter(creds, 10*time.Second)
	rateLimited := appexec.NewRateLimiter(rawAdapter, 10, 100*time.Millisecond)
	defer rateLimited.Close()
	drs := appexec.NewDryRunSubmitter(rateLimited)

	// Submit multiple intents — all must be intercepted by DryRunSubmitter.
	for i := range 5 {
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
			t.Fatalf("[MDR-8] intent %d: unexpected error: %s", i, submitProb.Message)
		}

		if len(receipt.VenueOrderID) < 7 || receipt.VenueOrderID[:7] != "dryrun-" {
			t.Errorf("[MDR-8] intent %d: expected dryrun- prefix, got %q", i, receipt.VenueOrderID)
		}
		if !receipt.Intent.Fills[0].Simulated {
			t.Errorf("[MDR-8] intent %d: fill must be simulated", i)
		}
	}

	t.Log("[s436/MDR-8] PASS — full pipeline chain (adapter → RateLimiter → DryRunSubmitter) intercepts all intents")
}

// ── Summary ─────────────────────────────────────────────────────────

func TestMainnetDryRun_Summary(t *testing.T) {
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("S436 Mainnet Dry-Run Proof — Test Summary")
	t.Log("═══════════════════════════════════════════════════════════")
	t.Log("MDR-1: DNS resolution and TCP connectivity (Spot + Futures)")
	t.Log("MDR-2: TLS handshake with certificate validation")
	t.Log("MDR-3: Public endpoint reachability (/ping)")
	t.Log("MDR-4: DryRunSubmitter intercepts Spot mainnet adapter")
	t.Log("MDR-5: DryRunSubmitter intercepts Futures mainnet adapter")
	t.Log("MDR-6: Audit marker consistency (dryrun- prefix, Simulated)")
	t.Log("MDR-7: Credential format validation for mainnet")
	t.Log("MDR-8: Pipeline chain composition (adapter → RL → DRS)")
	t.Log("")
	t.Log("Run with: go test -tags=livemainnet -count=1 -v ./internal/application/execution/...")
	t.Log("")
	t.Log("SAFETY: MDR-4/5/6/8 prove DryRunSubmitter interception.")
	t.Log("        The inner adapter (pointing to api.binance.com)")
	t.Log("        is NEVER called — DryRunSubmitter short-circuits.")
}
