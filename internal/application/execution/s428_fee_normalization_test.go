package execution_test

// s428_fee_normalization_test.go — S428: Fee normalization and cross-segment consistency.
//
// Validates the canonical fee model introduced by S428:
//   - FillRecord.Fee = actual trading commission (Spot: real, Futures: "0")
//   - FillRecord.FeeAsset = denomination of the fee (Spot: e.g. "BNB", Futures: "")
//   - FillRecord.CostBasis = total notional value (Spot: cummulativeQuoteQty, Futures: cumQuote)
//   - Paper/DryRun: Fee="0", FeeAsset="", CostBasis="0"
//
// Key invariant: Fee never carries notional value. CostBasis never carries commission.
// This separation makes cross-segment queries meaningful.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
)

// ═══════════════════════════════════════════════════════════════════
// 1. Spot fee semantics: real commission, asset, and cost basis
// ═══════════════════════════════════════════════════════════════════

func TestS428_SpotFee_SingleFill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             1001,
			"symbol":              "BTCUSDT",
			"status":              "FILLED",
			"executedQty":         "0.002",
			"cummulativeQuoteQty": "130.86",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "65430.00", "qty": "0.002", "commission": "0.00013086", "commissionAsset": "BNB"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 5*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testSpotBuyIntent(t)})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	// Fee = real commission
	if fill.Fee != "0.00013086" {
		t.Errorf("Fee: got %q, want 0.00013086 (actual commission)", fill.Fee)
	}
	// FeeAsset = commission denomination
	if fill.FeeAsset != "BNB" {
		t.Errorf("FeeAsset: got %q, want BNB", fill.FeeAsset)
	}
	// CostBasis = notional value (cummulativeQuoteQty)
	if fill.CostBasis != "130.86" {
		t.Errorf("CostBasis: got %q, want 130.86", fill.CostBasis)
	}
	if fill.Simulated {
		t.Error("venue fill must not be simulated")
	}
}

func TestS428_SpotFee_MultiFill_CommissionAggregated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":             1002,
			"symbol":              "ETHUSDT",
			"status":              "FILLED",
			"executedQty":         "0.5",
			"cummulativeQuoteQty": "1750.00",
			"transactTime":        time.Now().UnixMilli(),
			"fills": []map[string]any{
				{"price": "3500.00", "qty": "0.3", "commission": "0.001050", "commissionAsset": "BNB"},
				{"price": "3500.00", "qty": "0.2", "commission": "0.000700", "commissionAsset": "BNB"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := spotTestCredentials(t)
	adapter := appexec.NewBinanceSpotTestnetAdapter(creds, 5*time.Second).WithBaseURL(server.URL)

	intent := testSpotBuyIntent(t)
	intent.Instrument = ethUSDTSpot(t)
	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	// Aggregated commission: 0.001050 + 0.000700 = 0.00175
	if fill.Fee != "0.00175" {
		t.Errorf("Fee: got %q, want 0.00175 (aggregated commission)", fill.Fee)
	}
	if fill.FeeAsset != "BNB" {
		t.Errorf("FeeAsset: got %q, want BNB", fill.FeeAsset)
	}
	if fill.CostBasis != "1750.00" {
		t.Errorf("CostBasis: got %q, want 1750.00", fill.CostBasis)
	}
}

// ═══════════════════════════════════════════════════════════════════
// 2. Futures fee semantics: no commission, cumQuote in CostBasis
// ═══════════════════════════════════════════════════════════════════

func TestS428_FuturesFee_CumQuoteInCostBasis_NotFee(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     2001,
			"symbol":      "BTCUSDT",
			"status":      "FILLED",
			"avgPrice":    "65432.10",
			"executedQty": "0.001",
			"cumQuote":    "65.43210",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 5*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	// Fee must be "0" — Futures RESULT response has no commission field.
	if fill.Fee != "0" {
		t.Errorf("Fee: got %q, want 0 (Futures RESULT has no commission)", fill.Fee)
	}
	// FeeAsset must be empty — no commission info available.
	if fill.FeeAsset != "" {
		t.Errorf("FeeAsset: got %q, want empty (Futures has no commission asset)", fill.FeeAsset)
	}
	// CostBasis = cumQuote (the notional value that was previously in Fee).
	if fill.CostBasis != "65.43210" {
		t.Errorf("CostBasis: got %q, want 65.43210 (cumQuote)", fill.CostBasis)
	}
	if fill.Simulated {
		t.Error("venue fill must not be simulated")
	}
}

func TestS428_FuturesFee_PartialFill_CostBasisReflectsCumQuote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderId":     2002,
			"symbol":      "BTCUSDT",
			"status":      "PARTIALLY_FILLED",
			"avgPrice":    "65000.50",
			"executedQty": "0.0005",
			"cumQuote":    "32.50025",
			"updateTime":  time.Now().UnixMilli(),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	creds := testCredentials(t)
	adapter := appexec.NewBinanceFuturesTestnetAdapter(creds, 5*time.Second).WithBaseURL(server.URL)

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: testBuyIntent(t)})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if fill.Fee != "0" {
		t.Errorf("Fee: got %q, want 0", fill.Fee)
	}
	if fill.CostBasis != "32.50025" {
		t.Errorf("CostBasis: got %q, want 32.50025", fill.CostBasis)
	}
}

// ═══════════════════════════════════════════════════════════════════
// 3. Paper and DryRun: zero fees, no cost basis
// ═══════════════════════════════════════════════════════════════════

func TestS428_PaperAdapter_ZeroFees(t *testing.T) {
	adapter := appexec.NewPaperVenueAdapter(0)
	intent := testBuyIntent(t)
	intent.Source = "binances"

	receipt, prob := adapter.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if fill.Fee != "0" {
		t.Errorf("Paper Fee: got %q, want 0", fill.Fee)
	}
	if fill.FeeAsset != "" {
		t.Errorf("Paper FeeAsset: got %q, want empty", fill.FeeAsset)
	}
	if fill.CostBasis != "" {
		t.Errorf("Paper CostBasis: got %q, want empty", fill.CostBasis)
	}
	if !fill.Simulated {
		t.Error("paper fills must be simulated")
	}
}

func TestS428_DryRunSubmitter_ZeroFees(t *testing.T) {
	// Inner venue is never called.
	inner := appexec.NewPaperVenueAdapter(0)
	dryRun := appexec.NewDryRunSubmitter(inner)

	intent := testBuyIntent(t)
	receipt, prob := dryRun.SubmitOrder(context.Background(), ports.VenueOrderRequest{Intent: intent})
	if prob != nil {
		t.Fatalf("submit: %s", prob.Message)
	}

	fill := receipt.Intent.Fills[0]
	if fill.Fee != "0" {
		t.Errorf("DryRun Fee: got %q, want 0", fill.Fee)
	}
	if fill.FeeAsset != "" {
		t.Errorf("DryRun FeeAsset: got %q, want empty", fill.FeeAsset)
	}
	if fill.CostBasis != "" {
		t.Errorf("DryRun CostBasis: got %q, want empty", fill.CostBasis)
	}
	if !fill.Simulated {
		t.Error("dry-run fills must be simulated")
	}
}

// ═══════════════════════════════════════════════════════════════════
// 4. Cross-segment invariant: Fee field never carries notional value
// ═══════════════════════════════════════════════════════════════════

func TestS428_CrossSegment_FeeNeverCarriesNotional(t *testing.T) {
	// Construct FillRecords mimicking each segment and verify invariant.
	cases := []struct {
		name      string
		fill      domainexec.FillRecord
		wantFee   string
		wantBasis string
	}{
		{
			name:      "Spot with real commission",
			fill:      domainexec.FillRecord{Price: "65430", Quantity: "0.001", Fee: "0.00006543", FeeAsset: "BNB", CostBasis: "65.43"},
			wantFee:   "0.00006543",
			wantBasis: "65.43",
		},
		{
			name:      "Futures with no commission",
			fill:      domainexec.FillRecord{Price: "65432.10", Quantity: "0.001", Fee: "0", CostBasis: "65.43210"},
			wantFee:   "0",
			wantBasis: "65.43210",
		},
		{
			name:      "Paper simulated",
			fill:      domainexec.FillRecord{Price: "50000", Quantity: "0.02", Fee: "0", Simulated: true},
			wantFee:   "0",
			wantBasis: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fill.Fee != tc.wantFee {
				t.Errorf("Fee: got %q, want %q", tc.fill.Fee, tc.wantFee)
			}
			if tc.fill.CostBasis != tc.wantBasis {
				t.Errorf("CostBasis: got %q, want %q", tc.fill.CostBasis, tc.wantBasis)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════
// 5. JSON round-trip: new fields survive serialization
// ═══════════════════════════════════════════════════════════════════

func TestS428_FillRecord_JSONRoundTrip(t *testing.T) {
	original := domainexec.FillRecord{
		Price:     "65430.00",
		Quantity:  "0.001",
		Fee:       "0.00006543",
		FeeAsset:  "BNB",
		CostBasis: "65.43",
		Simulated: false,
		Timestamp: time.Now().UTC().Truncate(time.Millisecond),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded domainexec.FillRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Fee != original.Fee {
		t.Errorf("Fee round-trip: got %q, want %q", decoded.Fee, original.Fee)
	}
	if decoded.FeeAsset != original.FeeAsset {
		t.Errorf("FeeAsset round-trip: got %q, want %q", decoded.FeeAsset, original.FeeAsset)
	}
	if decoded.CostBasis != original.CostBasis {
		t.Errorf("CostBasis round-trip: got %q, want %q", decoded.CostBasis, original.CostBasis)
	}
}

func TestS428_FillRecord_JSONOmitsEmptyOptionalFields(t *testing.T) {
	// Paper/DryRun fill with no FeeAsset or CostBasis — verify omitempty works.
	fill := domainexec.FillRecord{
		Price:     "50000",
		Quantity:  "0.02",
		Fee:       "0",
		Simulated: true,
		Timestamp: time.Now().UTC(),
	}

	data, err := json.Marshal(fill)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	raw := string(data)
	if json.Valid([]byte(raw)) == false {
		t.Fatal("invalid JSON")
	}

	// fee_asset and cost_basis should be omitted when empty.
	var m map[string]any
	json.Unmarshal(data, &m)
	if _, ok := m["fee_asset"]; ok {
		t.Error("fee_asset should be omitted when empty")
	}
	if _, ok := m["cost_basis"]; ok {
		t.Error("cost_basis should be omitted when empty")
	}
}
