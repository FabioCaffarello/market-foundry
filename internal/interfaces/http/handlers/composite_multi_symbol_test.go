package handlers

// S302: Multi-symbol deterministic scenario tests for HTTP handler layer.
//
// These tests validate that the HTTP composite endpoints correctly route
// multi-symbol requests and return symbol-consistent responses. They complement
// the use case layer tests in analyticalclient/multi_symbol_scenario_test.go.

import (
	"strings"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/risk"
)

// ---------------------------------------------------------------------------
// S302-HTTP-1: Sequential symbol queries return correct chains
// ---------------------------------------------------------------------------

func TestS302_HTTP_SequentialSymbolChainQueries(t *testing.T) {
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	dispositions := map[string]string{
		"btcusdt": "approved",
		"ethusdt": "rejected",
		"solusdt": "modified",
	}

	for _, sym := range symbols {
		t.Run("chain_"+sym, func(t *testing.T) {
			disp := dispositions[sym]
			hasExec := disp != "rejected"
			stageCount := 5
			if !hasExec {
				stageCount = 4
			}

			chains := []analyticalclient.CompositeExecutionChain{{
				CorrelationID: "s302-http-" + sym,
				StageCount:    stageCount,
				ChainComplete: hasExec,
				Signal:        &analyticalclient.SignalWithTrace{OccurredAt: time.Now()},
				Decision:      &analyticalclient.DecisionWithTrace{OccurredAt: time.Now()},
				Strategy:      &analyticalclient.StrategyWithTrace{OccurredAt: time.Now()},
				Risk:          &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
				Attribution: &analyticalclient.RiskAttribution{
					Disposition: disp,
					Rationale:   "test rationale for " + sym,
					ActiveConstraints: risk.Constraints{
						MaxPositionSize: "0.10",
					},
				},
			}}
			if hasExec {
				chains[0].Execution = &analyticalclient.ExecutionWithTrace{OccurredAt: time.Now()}
			} else {
				chains[0].MissingStages = []string{"execution"}
			}

			uc := &stubCompositeUseCase{
				reply: analyticalclient.CompositeChainReply{
					Chains: chains,
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 3, ChainCount: 1},
				},
			}
			handler := newTestHandler(uc, nil, nil)

			req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=s302-http-"+sym+"&base="+strings.TrimSuffix(sym, "usdt")+"&quote=usdt&contract=perpetual", nil)
			w := httptest.NewRecorder()
			handler.GetChain(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			var resp compositeChainResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Chains) != 1 {
				t.Fatalf("expected 1 chain, got %d", len(resp.Chains))
			}
			if resp.Chains[0].Attribution == nil {
				t.Fatal("expected attribution")
			}
			if resp.Chains[0].Attribution.Disposition != disp {
				t.Errorf("attribution.disposition=%q, want %q", resp.Chains[0].Attribution.Disposition, disp)
			}
			if resp.Chains[0].StageCount != stageCount {
				t.Errorf("stage_count=%d, want %d", resp.Chains[0].StageCount, stageCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// S302-HTTP-2: Funnel results are symbol-scoped
// ---------------------------------------------------------------------------

func TestS302_HTTP_FunnelPerSymbol(t *testing.T) {
	type funnelCase struct {
		symbol   string
		sigCount int64
		exeCount int64
	}
	cases := []funnelCase{
		{symbol: "btcusdt", sigCount: 50, exeCount: 40},
		{symbol: "ethusdt", sigCount: 30, exeCount: 10},
		{symbol: "solusdt", sigCount: 15, exeCount: 12},
	}

	for _, tc := range cases {
		t.Run("funnel_"+tc.symbol, func(t *testing.T) {
			uc := &stubFunnelUseCase{
				reply: analyticalclient.PipelineFunnelReply{
					Stages: []analyticalclient.StageFunnelCount{
						{Stage: "signal", Count: tc.sigCount},
						{Stage: "decision", Count: tc.sigCount - 5},
						{Stage: "strategy", Count: tc.sigCount - 8},
						{Stage: "risk", Count: tc.exeCount + 2},
						{Stage: "execution", Count: tc.exeCount},
					},
					Source: "clickhouse",
					Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 2, ChainCount: 5},
				},
			}
			handler := newTestHandler(nil, uc, nil)

			req := httptest.NewRequest(http.MethodGet,
				"/analytical/composite/funnel?type=rsi&source=binancef&base="+strings.TrimSuffix(tc.symbol, "usdt")+"&quote=usdt&contract=perpetual"+"&timeframe=60", nil)
			w := httptest.NewRecorder()
			handler.GetFunnel(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			var resp pipelineFunnelResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Stages) != 5 {
				t.Fatalf("expected 5 stages, got %d", len(resp.Stages))
			}
			if resp.Stages[0].Count != tc.sigCount {
				t.Errorf("[%s] signal count=%d, want %d", tc.symbol, resp.Stages[0].Count, tc.sigCount)
			}
			if resp.Stages[4].Count != tc.exeCount {
				t.Errorf("[%s] execution count=%d, want %d", tc.symbol, resp.Stages[4].Count, tc.exeCount)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// S302-HTTP-3: Disposition breakdown per symbol
// ---------------------------------------------------------------------------

func TestS302_HTTP_DispositionsPerSymbol(t *testing.T) {
	type dispCase struct {
		symbol   string
		approved int64
		rejected int64
		modified int64
	}
	cases := []dispCase{
		{symbol: "btcusdt", approved: 40, rejected: 5, modified: 5},
		{symbol: "ethusdt", approved: 10, rejected: 15, modified: 5},
		{symbol: "solusdt", approved: 8, rejected: 0, modified: 2},
	}

	for _, tc := range cases {
		t.Run("dispositions_"+tc.symbol, func(t *testing.T) {
			total := tc.approved + tc.rejected + tc.modified
			disps := []analyticalclient.DispositionCount{}
			if tc.approved > 0 {
				disps = append(disps, analyticalclient.DispositionCount{
					Disposition: "approved", Count: tc.approved,
					Percentage: float64(tc.approved) / float64(total) * 100,
				})
			}
			if tc.rejected > 0 {
				disps = append(disps, analyticalclient.DispositionCount{
					Disposition: "rejected", Count: tc.rejected,
					Percentage: float64(tc.rejected) / float64(total) * 100,
				})
			}
			if tc.modified > 0 {
				disps = append(disps, analyticalclient.DispositionCount{
					Disposition: "modified", Count: tc.modified,
					Percentage: float64(tc.modified) / float64(total) * 100,
				})
			}

			uc := &stubDispositionUseCase{
				reply: analyticalclient.DispositionBreakdownReply{
					Dispositions: disps,
					Total:        total,
					Source:       "clickhouse",
					Meta:         analyticalclient.CompositeQueryMeta{TotalMs: 2, ChainCount: int(total)},
				},
			}
			handler := newTestHandler(nil, nil, uc)

			req := httptest.NewRequest(http.MethodGet,
				"/analytical/composite/dispositions?type=position_exposure&source=binancef&base="+strings.TrimSuffix(tc.symbol, "usdt")+"&quote=usdt&contract=perpetual"+"&timeframe=60", nil)
			w := httptest.NewRecorder()
			handler.GetDispositions(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			var resp dispositionBreakdownResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Total != total {
				t.Errorf("[%s] total=%d, want %d", tc.symbol, resp.Total, total)
			}

			// Verify the dominant disposition per symbol.
			var dominant analyticalclient.DispositionCount
			for _, d := range resp.Dispositions {
				if d.Count > dominant.Count {
					dominant = d
				}
			}
			switch tc.symbol {
			case "btcusdt":
				if dominant.Disposition != "approved" {
					t.Errorf("btcusdt dominant=%q, want approved", dominant.Disposition)
				}
			case "ethusdt":
				if dominant.Disposition != "rejected" {
					t.Errorf("ethusdt dominant=%q, want rejected", dominant.Disposition)
				}
			case "solusdt":
				if dominant.Disposition != "approved" {
					t.Errorf("solusdt dominant=%q, want approved", dominant.Disposition)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Response types (shared with composite_test.go via same package)
// ---------------------------------------------------------------------------
// compositeChainResponse, pipelineFunnelResponse, dispositionBreakdownResponse
// are already defined in composite_test.go via the same package scope.
