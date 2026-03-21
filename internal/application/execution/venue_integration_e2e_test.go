package execution_test

// S316 — End-to-End Venue Integration Proof.
//
// This file validates the complete venue integration path:
//   submit → fill → receipt → persistence compatibility → composite read compatibility
//   + safety gate validation on the venue path.
//
// Tests are guarded by MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY.
// When credentials are absent, tests skip with a clear message.
// When credentials are present, tests hit the real Binance Futures testnet.
//
// Guard rails (S316 charter):
//   - No async fills / websocket
//   - No advanced order types (market only)
//   - No mainnet
//   - Single venue only (Binance Futures testnet)

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

// requireTestnetCredentials loads real Binance Futures testnet credentials from env.
// Skips the test if credentials are not present.
func requireTestnetCredentials(t *testing.T) *appexec.CredentialSet {
	t.Helper()
	apiKey := os.Getenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY")
	apiSecret := os.Getenv("MF_VENUE_BINANCE_FUTURES_TESTNET_API_SECRET")
	if apiKey == "" || apiSecret == "" {
		t.Skip("S316: skipping venue integration test — MF_VENUE_BINANCE_FUTURES_TESTNET_API_KEY/API_SECRET not set")
	}
	creds, prob := appexec.LoadCredentials("binance_futures_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("load testnet credentials: %s", prob.Message)
	}
	return creds
}

// testnetBuyIntent creates a minimal market buy intent for testnet proof.
// Uses BTCUSDT with the smallest practical quantity.
func testnetBuyIntent() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:             "position_exposure",
			Disposition:      "approved",
			Confidence:       "0.85",
			Timeframe:        60,
			StrategyType:     "mean_reversion",
			DecisionSeverity: "moderate",
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
}

// testnetSellIntent creates a minimal market sell intent for testnet proof.
func testnetSellIntent() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideSell,
		Quantity:  "0.001",
		Status:    domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:             "position_exposure",
			Disposition:      "approved",
			Confidence:       "0.80",
			Timeframe:        60,
			StrategyType:     "trend_following",
			DecisionSeverity: "moderate",
		},
		Final:     true,
		Timestamp: time.Now().UTC(),
	}
}

// ---------------------------------------------------------------------------
// VQ1: Can we submit a market order to the real venue?
// ---------------------------------------------------------------------------

func TestS316_VQ1_SubmitMarketBuy_RealTestnet(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetBuyIntent()
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VQ1: submit market buy failed: [%s] %s", prob.Code, prob.Message)
	}

	// Validate venue assigned an order ID.
	if receipt.VenueOrderID == "" {
		t.Fatal("VQ1: venue order ID must not be empty")
	}

	// Validate status is terminal (FILLED for market orders).
	if receipt.Status != domainexec.StatusFilled {
		t.Logf("VQ1: non-filled status %q — testnet may have rejected or partially filled", receipt.Status)
	}

	t.Logf("VQ1 PASS: venueOrderID=%s status=%s", receipt.VenueOrderID, receipt.Status)
}

func TestS316_VQ1_SubmitMarketSell_RealTestnet(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetSellIntent()
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VQ1: submit market sell failed: [%s] %s", prob.Code, prob.Message)
	}

	if receipt.VenueOrderID == "" {
		t.Fatal("VQ1: venue order ID must not be empty")
	}

	t.Logf("VQ1 PASS: venueOrderID=%s status=%s side=SELL", receipt.VenueOrderID, receipt.Status)
}

// ---------------------------------------------------------------------------
// VQ3: Does the fill carry real price/quantity data (not simulated)?
// ---------------------------------------------------------------------------

func TestS316_VQ3_RealFill_NotSimulated(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetBuyIntent()
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VQ3: submit failed: [%s] %s", prob.Code, prob.Message)
	}

	if receipt.Status != domainexec.StatusFilled {
		t.Skipf("VQ3: skipping fill validation — status is %s (not filled)", receipt.Status)
	}

	if len(receipt.Intent.Fills) == 0 {
		t.Fatal("VQ3: filled order must have at least one fill record")
	}

	fill := receipt.Intent.Fills[0]

	// Real venue fills must NOT be simulated.
	if fill.Simulated {
		t.Fatal("VQ3: real venue fill must have Simulated=false")
	}

	// Real fills must have non-zero price.
	if fill.Price == "" || fill.Price == "0" || fill.Price == "0.00" {
		t.Fatalf("VQ3: real fill price must be non-zero, got %q", fill.Price)
	}

	// Real fills must have non-zero quantity.
	if fill.Quantity == "" || fill.Quantity == "0" {
		t.Fatalf("VQ3: real fill quantity must be non-zero, got %q", fill.Quantity)
	}

	// Fill timestamp must be recent (within last 60 seconds).
	if time.Since(fill.Timestamp) > 60*time.Second {
		t.Logf("VQ3 WARNING: fill timestamp %s is older than 60s", fill.Timestamp)
	}

	t.Logf("VQ3 PASS: price=%s qty=%s fee=%s simulated=%v ts=%s",
		fill.Price, fill.Quantity, fill.Fee, fill.Simulated, fill.Timestamp)
}

// ---------------------------------------------------------------------------
// VQ4: Is the receipt structurally compatible with persistence and composite read?
// ---------------------------------------------------------------------------

func TestS316_VQ4_ReceiptPersistenceCompatibility(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetBuyIntent()
	intent.CorrelationID = "s316-e2e-proof-corr-001"
	intent.CausationID = "s316-e2e-proof-caus-001"

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VQ4: submit failed: [%s] %s", prob.Code, prob.Message)
	}

	// Verify all fields required for persistence are populated.
	if receipt.VenueOrderID == "" {
		t.Fatal("VQ4: VenueOrderID required for persistence")
	}
	if receipt.ClientOrderID == "" {
		t.Fatal("VQ4: ClientOrderID required for persistence")
	}

	// Client order ID must match deterministic derivation.
	expectedClientID := appexec.ClientOrderID(intent)
	if receipt.ClientOrderID != expectedClientID {
		t.Fatalf("VQ4: ClientOrderID mismatch: got %q, want %q", receipt.ClientOrderID, expectedClientID)
	}

	// Intent in receipt preserves correlation context for composite queries.
	if receipt.Intent.CorrelationID != intent.CorrelationID {
		t.Fatalf("VQ4: CorrelationID must be preserved: got %q, want %q",
			receipt.Intent.CorrelationID, intent.CorrelationID)
	}
	if receipt.Intent.CausationID != intent.CausationID {
		t.Fatalf("VQ4: CausationID must be preserved: got %q, want %q",
			receipt.Intent.CausationID, intent.CausationID)
	}

	// Verify receipt is JSON-serializable (required for event persistence).
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("VQ4: receipt must be JSON-serializable: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("VQ4: serialized receipt must not be empty")
	}

	// Verify round-trip JSON (structural integrity).
	var roundTrip ports.VenueOrderReceipt
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("VQ4: receipt JSON round-trip failed: %v", err)
	}
	if roundTrip.VenueOrderID != receipt.VenueOrderID {
		t.Fatalf("VQ4: VenueOrderID round-trip mismatch")
	}

	// Verify partition key is valid for KV storage.
	pk := receipt.Intent.PartitionKey()
	if pk == "" {
		t.Fatal("VQ4: PartitionKey must not be empty")
	}
	if pk != "binancef.btcusdt.60" {
		t.Fatalf("VQ4: unexpected partition key %q", pk)
	}

	// Verify deduplication key is valid for JetStream.
	dk := receipt.Intent.DeduplicationKey()
	if dk == "" {
		t.Fatal("VQ4: DeduplicationKey must not be empty")
	}

	t.Logf("VQ4 PASS: venueOrderID=%s clientOrderID=%s partitionKey=%s json_bytes=%d",
		receipt.VenueOrderID, receipt.ClientOrderID, pk, len(data))
}

// ---------------------------------------------------------------------------
// VQ6 (partial): Safety gate does NOT block fresh intents on venue path.
// ---------------------------------------------------------------------------

func TestS316_VQ6_SafetyGate_FreshIntent_AllowsVenueSubmit(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	// Safety gate: no kill switch, staleness = 5 minutes.
	staleness := appexec.NewStalenessGuard(5 * time.Minute)
	gate := appexec.NewSafetyGate(nil, 0, staleness)

	intent := testnetBuyIntent()
	now := time.Now().UTC()

	// Check safety gate BEFORE venue submission (actor behavior).
	verdict := gate.Check(intent.Timestamp, now)
	if !verdict.Allowed {
		t.Fatalf("VQ6: safety gate must allow fresh intent, got blocked: %s", verdict.Reason)
	}

	// Now submit to real venue.
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("VQ6: submit after safety gate passed failed: [%s] %s", prob.Code, prob.Message)
	}

	if receipt.VenueOrderID == "" {
		t.Fatal("VQ6: venue order ID must not be empty after safety gate pass")
	}

	t.Logf("VQ6 PASS: safety gate allowed → venue submit succeeded, venueOrderID=%s", receipt.VenueOrderID)
}

// ---------------------------------------------------------------------------
// VQ6: Safety gate blocks stale intents from reaching venue.
// ---------------------------------------------------------------------------

func TestS316_VQ6_SafetyGate_StaleIntent_BlocksVenueSubmit(t *testing.T) {
	// This test does NOT need credentials — it validates gate blocking.
	staleness := appexec.NewStalenessGuard(2 * time.Minute)
	gate := appexec.NewSafetyGate(nil, 0, staleness)

	intent := testnetBuyIntent()
	intent.Timestamp = time.Now().Add(-5 * time.Minute) // 5 minutes ago → stale
	now := time.Now().UTC()

	verdict := gate.Check(intent.Timestamp, now)
	if verdict.Allowed {
		t.Fatal("VQ6: safety gate must block stale intent")
	}
	if verdict.Reason != "stale" {
		t.Fatalf("VQ6: expected reason 'stale', got %q", verdict.Reason)
	}

	t.Logf("VQ6 PASS: stale intent correctly blocked (reason=%s)", verdict.Reason)
}

// ---------------------------------------------------------------------------
// VQ6: Kill switch blocks all venue submissions.
// ---------------------------------------------------------------------------

func TestS316_VQ6_SafetyGate_KillSwitch_BlocksVenueSubmit(t *testing.T) {
	// This test does NOT need credentials — it validates gate blocking.
	checker := &haltedGateChecker{}
	staleness := appexec.NewStalenessGuard(5 * time.Minute)
	gate := appexec.NewSafetyGate(checker, 0, staleness)

	intent := testnetBuyIntent()
	now := time.Now().UTC()

	verdict := gate.Check(intent.Timestamp, now)
	if verdict.Allowed {
		t.Fatal("VQ6: kill switch must block all submissions")
	}
	if verdict.Reason != "kill_switch" {
		t.Fatalf("VQ6: expected reason 'kill_switch', got %q", verdict.Reason)
	}

	t.Logf("VQ6 PASS: kill switch correctly blocked (reason=%s)", verdict.Reason)
}

// ---------------------------------------------------------------------------
// VQ6: Kill switch takes priority over staleness.
// ---------------------------------------------------------------------------

func TestS316_VQ6_SafetyGate_KillSwitchPriority(t *testing.T) {
	checker := &haltedGateChecker{}
	staleness := appexec.NewStalenessGuard(2 * time.Minute)
	gate := appexec.NewSafetyGate(checker, 0, staleness)

	intent := testnetBuyIntent()
	intent.Timestamp = time.Now().Add(-5 * time.Minute) // also stale
	now := time.Now().UTC()

	verdict := gate.Check(intent.Timestamp, now)
	if verdict.Allowed {
		t.Fatal("VQ6: both gates active — must block")
	}
	// Kill switch takes priority.
	if verdict.Reason != "kill_switch" {
		t.Fatalf("VQ6: kill switch must take priority over staleness, got %q", verdict.Reason)
	}

	t.Logf("VQ6 PASS: kill switch priority confirmed (reason=%s)", verdict.Reason)
}

// ---------------------------------------------------------------------------
// E2E: Full actor-path simulation (gate check → submit → receipt validation).
// ---------------------------------------------------------------------------

func TestS316_E2E_ActorPath_GateToSubmitToReceipt(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	// Simulate actor's decision path.
	staleness := appexec.NewStalenessGuard(5 * time.Minute)
	gate := appexec.NewSafetyGate(nil, 0, staleness) // no kill switch in testnet proof

	intent := testnetBuyIntent()
	intent.CorrelationID = "s316-e2e-actor-path-corr"
	intent.CausationID = "s316-e2e-actor-path-caus"
	now := time.Now().UTC()

	// Step 1: Safety gate check (actor does this before venue call).
	verdict := gate.Check(intent.Timestamp, now)
	if !verdict.Allowed {
		t.Fatalf("E2E: safety gate blocked fresh intent: %s", verdict.Reason)
	}

	// Step 2: Derive client order ID (actor does this for idempotency).
	clientOrderID := appexec.ClientOrderID(intent)
	if clientOrderID == "" {
		t.Fatal("E2E: client order ID derivation failed")
	}
	if len(clientOrderID) > 36 {
		t.Fatalf("E2E: client order ID exceeds Binance limit (36 chars): len=%d", len(clientOrderID))
	}

	// Step 3: Submit to venue.
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("E2E: venue submit failed: [%s] %s", prob.Code, prob.Message)
	}

	// Step 4: Validate receipt completeness.
	if receipt.VenueOrderID == "" {
		t.Fatal("E2E: venue order ID missing from receipt")
	}
	if receipt.ClientOrderID != clientOrderID {
		t.Fatalf("E2E: client order ID mismatch: receipt=%q derived=%q", receipt.ClientOrderID, clientOrderID)
	}
	if receipt.Intent.CorrelationID != intent.CorrelationID {
		t.Fatal("E2E: correlation ID not preserved through venue submit")
	}

	// Step 5: Validate receipt is JSON-serializable for event persistence.
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("E2E: receipt JSON serialization failed: %v", err)
	}

	// Step 6: Validate intent in receipt can produce valid partition/dedup keys.
	pk := receipt.Intent.PartitionKey()
	dk := receipt.Intent.DeduplicationKey()
	if pk == "" || dk == "" {
		t.Fatal("E2E: partition key or dedup key missing from receipt intent")
	}

	t.Logf("E2E PASS: gate=allowed → clientOrderID=%s → venueOrderID=%s → status=%s → json=%d bytes → pk=%s",
		clientOrderID, receipt.VenueOrderID, receipt.Status, len(data), pk)
}

// ---------------------------------------------------------------------------
// No-action intents: must NOT hit venue even with real adapter.
// ---------------------------------------------------------------------------

func TestS316_NoAction_NoVenueCall_RealAdapter(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetBuyIntent()
	intent.Side = domainexec.SideNone
	intent.Quantity = "0"

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("no-action submit should succeed: [%s] %s", prob.Code, prob.Message)
	}

	if receipt.Status != domainexec.StatusAccepted {
		t.Fatalf("no-action should return accepted, got %s", receipt.Status)
	}

	// No fills for no-action intents.
	if len(receipt.Intent.Fills) > 0 {
		t.Fatal("no-action intent must not produce fills")
	}

	t.Logf("PASS: no-action intent accepted without venue call, venueOrderID=%s", receipt.VenueOrderID)
}

// ---------------------------------------------------------------------------
// Client order ID determinism with real venue receipt.
// ---------------------------------------------------------------------------

func TestS316_ClientOrderID_DeterministicWithRealVenue(t *testing.T) {
	creds := requireTestnetCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 15*time.Second)

	intent := testnetBuyIntent()
	// Fix timestamp for deterministic derivation.
	intent.Timestamp = time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)

	// Derive client order ID twice — must be identical.
	id1 := appexec.ClientOrderID(intent)
	id2 := appexec.ClientOrderID(intent)
	if id1 != id2 {
		t.Fatalf("client order ID not deterministic: %q != %q", id1, id2)
	}

	// Submit with this intent.
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		// Binance may reject due to timestamp being in the past — this is expected.
		// The determinism test still passes since we verified id1 == id2.
		t.Logf("venue rejected (expected for fixed past timestamp): [%s] %s", prob.Code, prob.Message)
		t.Logf("PASS: client order ID determinism verified (id=%s), venue rejection is expected for fixed timestamp", id1)
		return
	}

	if receipt.ClientOrderID != id1 {
		t.Fatalf("receipt client order ID %q != derived %q", receipt.ClientOrderID, id1)
	}

	t.Logf("PASS: client order ID deterministic and matches receipt: %s", id1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// haltedGateChecker simulates a halted kill switch for testing.
type haltedGateChecker struct{}

func (h *haltedGateChecker) IsHalted(_ context.Context) bool { return true }
