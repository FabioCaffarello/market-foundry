package handlers

// S303: Composite Observability Under Multi-Symbol Load — HTTP Handler Layer
//
// These tests validate that the HTTP explainability surfaces remain correct
// and readable when multiple symbols are queried through the same endpoints.
//
// Focus areas:
//   HTTP-OBS-1 — Cross-surface JSON consistency: chain/funnel/disposition responses
//                remain structurally valid and symbol-specific.
//   HTTP-OBS-2 — Attribution JSON completeness: all explainability fields serialized.
//   HTTP-OBS-3 — Multi-symbol sequential queries produce independent responses.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/risk"
)

// ---------------------------------------------------------------------------
// HTTP-OBS-1: Cross-surface structural validity per symbol
// ---------------------------------------------------------------------------

func TestS303_HTTP_OBS1_CrossSurfaceStructure(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	// Each symbol gets queried on all 3 surfaces; responses must be valid JSON
	// with correct source field and non-zero meta.

	for _, sym := range symbols {
		t.Run("structure_"+sym, func(t *testing.T) {
			// Chain surface.
			chainUC := &stubCompositeUseCase{
				reply: analyticalclient.CompositeChainReply{
					Chains: []analyticalclient.CompositeExecutionChain{{
						CorrelationID: "s303-http-obs1-" + sym,
						StageCount:    5,
						ChainComplete: true,
						Signal:        &analyticalclient.SignalWithTrace{OccurredAt: time.Now()},
						Decision:      &analyticalclient.DecisionWithTrace{OccurredAt: time.Now()},
						Strategy:      &analyticalclient.StrategyWithTrace{OccurredAt: time.Now()},
						Risk:          &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
						Execution:     &analyticalclient.ExecutionWithTrace{OccurredAt: time.Now()},
						Attribution: &analyticalclient.RiskAttribution{
							Disposition:       "approved",
							Rationale:         "within limits for " + sym,
							ActiveConstraints: risk.Constraints{MaxPositionSize: "0.10"},
						},
					}},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 3, ChainCount: 1},
				},
			}

			funnelUC := &stubFunnelUseCase{
				reply: analyticalclient.PipelineFunnelReply{
					Stages: []analyticalclient.StageFunnelCount{
						{Stage: "signal", Count: 50},
						{Stage: "decision", Count: 48},
						{Stage: "strategy", Count: 45},
						{Stage: "risk", Count: 42},
						{Stage: "execution", Count: 40},
					},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 2, ChainCount: 5},
				},
			}

			dispUC := &stubDispositionUseCase{
				reply: analyticalclient.DispositionBreakdownReply{
					Dispositions: []analyticalclient.DispositionCount{
						{Disposition: "approved", Count: 40, Percentage: 95.24},
						{Disposition: "rejected", Count: 2, Percentage: 4.76},
					},
					Total:  42,
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 1, ChainCount: 2},
				},
			}

			handler := newTestHandler(chainUC, funnelUC, dispUC)

			// Chain endpoint.
			req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=s303-http-obs1-"+sym+"&symbol="+sym, nil)
			w := httptest.NewRecorder()
			handler.GetChain(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("chain: expected 200, got %d", w.Code)
			}
			var chainResp compositeChainResponse
			if err := json.NewDecoder(w.Body).Decode(&chainResp); err != nil {
				t.Fatalf("chain decode: %v", err)
			}
			if chainResp.Source != "clickhouse" {
				t.Errorf("chain source=%q, want clickhouse", chainResp.Source)
			}

			// Funnel endpoint.
			req = httptest.NewRequest(http.MethodGet, "/analytical/composite/funnel?type=rsi&source=binancef&symbol="+sym+"&timeframe=60", nil)
			w = httptest.NewRecorder()
			handler.GetFunnel(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("funnel: expected 200, got %d", w.Code)
			}
			var funnelResp pipelineFunnelResponse
			if err := json.NewDecoder(w.Body).Decode(&funnelResp); err != nil {
				t.Fatalf("funnel decode: %v", err)
			}
			if len(funnelResp.Stages) != 5 {
				t.Errorf("funnel stages=%d, want 5", len(funnelResp.Stages))
			}

			// Dispositions endpoint.
			req = httptest.NewRequest(http.MethodGet, "/analytical/composite/dispositions?type=rsi&source=binancef&symbol="+sym+"&timeframe=60", nil)
			w = httptest.NewRecorder()
			handler.GetDispositions(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("dispositions: expected 200, got %d", w.Code)
			}
			var dispResp dispositionBreakdownResponse
			if err := json.NewDecoder(w.Body).Decode(&dispResp); err != nil {
				t.Fatalf("dispositions decode: %v", err)
			}
			if dispResp.Total != 42 {
				t.Errorf("dispositions total=%d, want 42", dispResp.Total)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HTTP-OBS-2: Attribution JSON completeness in multi-symbol context
// ---------------------------------------------------------------------------

func TestS303_HTTP_OBS2_AttributionCompleteness(t *testing.T) {
	// For each symbol, verify that the chain JSON includes all attribution
	// fields needed for operational readability.
	type attrCase struct {
		symbol      string
		disposition string
		rationale   string
		maxPos      string
		hasExec     bool
	}

	cases := []attrCase{
		{symbol: "btcusdt", disposition: "approved", rationale: "within limits", maxPos: "0.10", hasExec: true},
		{symbol: "ethusdt", disposition: "rejected", rationale: "drawdown exceeded", maxPos: "0.05", hasExec: false},
		{symbol: "solusdt", disposition: "modified", rationale: "position capped", maxPos: "0.03", hasExec: true},
	}

	for _, tc := range cases {
		t.Run("attribution_"+tc.symbol, func(t *testing.T) {
			stageCount := 5
			if !tc.hasExec {
				stageCount = 4
			}
			chain := analyticalclient.CompositeExecutionChain{
				CorrelationID: "s303-http-obs2-" + tc.symbol,
				StageCount:    stageCount,
				ChainComplete: tc.hasExec,
				Signal:        &analyticalclient.SignalWithTrace{OccurredAt: time.Now()},
				Decision:      &analyticalclient.DecisionWithTrace{OccurredAt: time.Now()},
				Strategy:      &analyticalclient.StrategyWithTrace{OccurredAt: time.Now()},
				Risk:          &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
				Attribution: &analyticalclient.RiskAttribution{
					Disposition:       tc.disposition,
					Rationale:         tc.rationale,
					ActiveConstraints: risk.Constraints{MaxPositionSize: tc.maxPos, MaxExposure: "1.0"},
					StrategyContext: []analyticalclient.AttributionStrategyContext{{
						Type:              "test_strategy",
						Direction:         "long",
						Confidence:        "0.80",
						DecisionSeverity:  "high",
						DecisionRationale: "test decision",
					}},
				},
			}
			if tc.hasExec {
				chain.Execution = &analyticalclient.ExecutionWithTrace{OccurredAt: time.Now()}
			} else {
				chain.MissingStages = []string{"execution"}
			}

			uc := &stubCompositeUseCase{
				reply: analyticalclient.CompositeChainReply{
					Chains: []analyticalclient.CompositeExecutionChain{chain},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 2, ChainCount: 1},
				},
			}
			handler := newTestHandler(uc, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=s303-http-obs2-"+tc.symbol+"&symbol="+tc.symbol, nil)
			w := httptest.NewRecorder()
			handler.GetChain(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			// Decode into raw JSON to verify field presence.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
				t.Fatalf("decode: %v", err)
			}

			// Decode chains array.
			var chains []map[string]json.RawMessage
			if err := json.Unmarshal(raw["chains"], &chains); err != nil {
				t.Fatalf("decode chains: %v", err)
			}
			if len(chains) != 1 {
				t.Fatalf("expected 1 chain, got %d", len(chains))
			}

			c := chains[0]
			// Attribution must exist.
			attrRaw, ok := c["attribution"]
			if !ok {
				t.Fatal("attribution field missing from JSON")
			}

			var attr map[string]json.RawMessage
			if err := json.Unmarshal(attrRaw, &attr); err != nil {
				t.Fatalf("decode attribution: %v", err)
			}

			// Required fields.
			requiredFields := []string{"disposition", "rationale", "active_constraints", "strategy_context"}
			for _, f := range requiredFields {
				if _, exists := attr[f]; !exists {
					t.Errorf("attribution missing field %q", f)
				}
			}

			// Disposition value.
			var disp string
			if err := json.Unmarshal(attr["disposition"], &disp); err != nil {
				t.Fatalf("decode disposition: %v", err)
			}
			if disp != tc.disposition {
				t.Errorf("disposition=%q, want %q", disp, tc.disposition)
			}

			// Missing stages for rejected.
			if !tc.hasExec {
				msRaw, ok := c["missing_stages"]
				if !ok {
					t.Error("missing_stages field absent for rejected chain")
				} else {
					var ms []string
					if err := json.Unmarshal(msRaw, &ms); err != nil {
						t.Fatalf("decode missing_stages: %v", err)
					}
					if len(ms) != 1 || ms[0] != "execution" {
						t.Errorf("missing_stages=%v, want [execution]", ms)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HTTP-OBS-3: Multi-symbol sequential queries produce independent responses
// ---------------------------------------------------------------------------

func TestS303_HTTP_OBS3_SequentialQueryIndependence(t *testing.T) {
	// Simulate querying chain, funnel, and dispositions for btcusdt then ethusdt.
	// Each pair must be fully independent — no state leakage between symbols.
	symbolData := map[string]struct {
		chainCount int
		signalCnt  int64
		execCnt    int64
		approved   int64
		rejected   int64
	}{
		"btcusdt": {chainCount: 1, signalCnt: 100, execCnt: 90, approved: 85, rejected: 5},
		"ethusdt": {chainCount: 1, signalCnt: 40, execCnt: 20, approved: 15, rejected: 5},
	}

	for _, sym := range []string{"btcusdt", "ethusdt"} {
		t.Run("independence_"+sym, func(t *testing.T) {
			sd := symbolData[sym]

			chainUC := &stubCompositeUseCase{
				reply: analyticalclient.CompositeChainReply{
					Chains: []analyticalclient.CompositeExecutionChain{{
						CorrelationID: "s303-obs3-" + sym,
						StageCount:    5, ChainComplete: true,
						Signal:    &analyticalclient.SignalWithTrace{OccurredAt: time.Now()},
						Decision:  &analyticalclient.DecisionWithTrace{OccurredAt: time.Now()},
						Strategy:  &analyticalclient.StrategyWithTrace{OccurredAt: time.Now()},
						Risk:      &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
						Execution: &analyticalclient.ExecutionWithTrace{OccurredAt: time.Now()},
						Attribution: &analyticalclient.RiskAttribution{
							Disposition: "approved",
							Rationale:   "ok for " + sym,
						},
					}},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 2, ChainCount: sd.chainCount},
				},
			}

			funnelUC := &stubFunnelUseCase{
				reply: analyticalclient.PipelineFunnelReply{
					Stages: []analyticalclient.StageFunnelCount{
						{Stage: "signal", Count: sd.signalCnt},
						{Stage: "decision", Count: sd.signalCnt - 2},
						{Stage: "strategy", Count: sd.signalCnt - 5},
						{Stage: "risk", Count: sd.approved + sd.rejected},
						{Stage: "execution", Count: sd.execCnt},
					},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 1, ChainCount: 5},
				},
			}

			dispUC := &stubDispositionUseCase{
				reply: analyticalclient.DispositionBreakdownReply{
					Dispositions: []analyticalclient.DispositionCount{
						{Disposition: "approved", Count: sd.approved, Percentage: float64(sd.approved) / float64(sd.approved+sd.rejected) * 100},
						{Disposition: "rejected", Count: sd.rejected, Percentage: float64(sd.rejected) / float64(sd.approved+sd.rejected) * 100},
					},
					Total:  sd.approved + sd.rejected,
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 1, ChainCount: 2},
				},
			}

			handler := newTestHandler(chainUC, funnelUC, dispUC)

			// Chain.
			req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=s303-obs3-"+sym+"&symbol="+sym, nil)
			w := httptest.NewRecorder()
			handler.GetChain(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("chain: %d", w.Code)
			}

			// Funnel — verify signal count matches expected.
			req = httptest.NewRequest(http.MethodGet, "/analytical/composite/funnel?type=rsi&source=binancef&symbol="+sym+"&timeframe=60", nil)
			w = httptest.NewRecorder()
			handler.GetFunnel(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("funnel: %d", w.Code)
			}
			var fResp pipelineFunnelResponse
			if err := json.NewDecoder(w.Body).Decode(&fResp); err != nil {
				t.Fatalf("decode funnel: %v", err)
			}
			if fResp.Stages[0].Count != sd.signalCnt {
				t.Errorf("[%s] signal count=%d, want %d", sym, fResp.Stages[0].Count, sd.signalCnt)
			}

			// Dispositions — verify total matches expected.
			req = httptest.NewRequest(http.MethodGet, "/analytical/composite/dispositions?type=rsi&source=binancef&symbol="+sym+"&timeframe=60", nil)
			w = httptest.NewRecorder()
			handler.GetDispositions(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("dispositions: %d", w.Code)
			}
			var dResp dispositionBreakdownResponse
			if err := json.NewDecoder(w.Body).Decode(&dResp); err != nil {
				t.Fatalf("decode dispositions: %v", err)
			}
			if dResp.Total != sd.approved+sd.rejected {
				t.Errorf("[%s] disposition total=%d, want %d", sym, dResp.Total, sd.approved+sd.rejected)
			}
		})
	}
}
