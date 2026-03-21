package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/analyticalclient"
	"internal/domain/risk"
	"internal/shared/problem"
)

// stubCompositeUseCase implements getCompositeChainUseCase for handler testing.
type stubCompositeUseCase struct {
	reply analyticalclient.CompositeChainReply
	prob  *problem.Problem
}

func (s *stubCompositeUseCase) Execute(_ context.Context, _ analyticalclient.CompositeChainQuery) (analyticalclient.CompositeChainReply, *problem.Problem) {
	return s.reply, s.prob
}

// stubFunnelUseCase implements getPipelineFunnelUseCase for handler testing.
type stubFunnelUseCase struct {
	reply analyticalclient.PipelineFunnelReply
	prob  *problem.Problem
}

func (s *stubFunnelUseCase) Execute(_ context.Context, _ analyticalclient.PipelineFunnelQuery) (analyticalclient.PipelineFunnelReply, *problem.Problem) {
	return s.reply, s.prob
}

// stubDispositionUseCase implements getDispositionBreakdownUseCase for handler testing.
type stubDispositionUseCase struct {
	reply analyticalclient.DispositionBreakdownReply
	prob  *problem.Problem
}

func (s *stubDispositionUseCase) Execute(_ context.Context, _ analyticalclient.DispositionBreakdownQuery) (analyticalclient.DispositionBreakdownReply, *problem.Problem) {
	return s.reply, s.prob
}

func newTestHandler(chain *stubCompositeUseCase, funnel *stubFunnelUseCase, disp *stubDispositionUseCase) *CompositeWebHandler {
	return NewCompositeWebHandler(CompositeHandlerDeps{
		GetCompositeChain:      chain,
		GetPipelineFunnel:      funnel,
		GetDispositionBreakdown: disp,
	})
}

// --- Chain endpoint tests (S297) ---

func TestCompositeGetChain_Success(t *testing.T) {
	uc := &stubCompositeUseCase{
		reply: analyticalclient.CompositeChainReply{
			Chains: []analyticalclient.CompositeExecutionChain{
				{
					CorrelationID: "test-corr-001",
					StageCount:    5,
					ChainComplete: true,
					Signal:        &analyticalclient.SignalWithTrace{OccurredAt: time.Now()},
					Decision:      &analyticalclient.DecisionWithTrace{OccurredAt: time.Now()},
					Strategy:      &analyticalclient.StrategyWithTrace{OccurredAt: time.Now()},
					Risk:          &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
					Execution:     &analyticalclient.ExecutionWithTrace{OccurredAt: time.Now()},
					Attribution: &analyticalclient.RiskAttribution{
						Disposition: "approved",
						Rationale:   "within limits",
					},
				},
			},
			Source: "clickhouse",
			Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 5, ChainCount: 1},
		},
	}
	handler := newTestHandler(uc, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=test-corr-001&symbol=btcusdt", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp compositeChainResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Chains) != 1 {
		t.Fatalf("expected 1 chain, got %d", len(resp.Chains))
	}
	if resp.Chains[0].CorrelationID != "test-corr-001" {
		t.Errorf("expected correlation_id test-corr-001, got %s", resp.Chains[0].CorrelationID)
	}
	if resp.Chains[0].Attribution == nil {
		t.Error("expected attribution to be present")
	}
	if resp.Source != "clickhouse" {
		t.Errorf("expected source clickhouse, got %s", resp.Source)
	}
	if w.Header().Get("Server-Timing") == "" {
		t.Error("expected Server-Timing header")
	}
}

func TestCompositeGetChain_MissingCorrelationID(t *testing.T) {
	handler := newTestHandler(&stubCompositeUseCase{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?symbol=btcusdt", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompositeGetChain_MissingSymbol(t *testing.T) {
	handler := newTestHandler(&stubCompositeUseCase{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=test-001", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing symbol, got %d", w.Code)
	}
}

func TestCompositeGetChain_UseCaseError(t *testing.T) {
	uc := &stubCompositeUseCase{
		prob: problem.New(problem.Unavailable, "reader down"),
	}
	handler := newTestHandler(uc, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=test-001&symbol=btcusdt", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestCompositeGetChain_NilHandler(t *testing.T) {
	var handler *CompositeWebHandler
	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=test-001&symbol=btcusdt", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// --- Batch chains tests (S297) ---

func TestCompositeGetChains_Success(t *testing.T) {
	uc := &stubCompositeUseCase{
		reply: analyticalclient.CompositeChainReply{
			Chains: []analyticalclient.CompositeExecutionChain{
				{CorrelationID: "batch-001", StageCount: 5, ChainComplete: true},
				{CorrelationID: "batch-002", StageCount: 4, ChainComplete: false, MissingStages: []string{"execution"}},
			},
			Source: "clickhouse",
			Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 42, ChainCount: 2},
		},
	}
	handler := newTestHandler(uc, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chains?source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetChains(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp compositeChainResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Chains) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(resp.Chains))
	}
	if resp.Meta.ChainCount != 2 {
		t.Errorf("expected chain_count 2, got %d", resp.Meta.ChainCount)
	}
}

func TestCompositeGetChains_MissingTimeframe(t *testing.T) {
	handler := newTestHandler(&stubCompositeUseCase{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chains?source=binance&symbol=BTCUSDT", nil)
	w := httptest.NewRecorder()
	handler.GetChains(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompositeGetChains_InvalidLimit(t *testing.T) {
	handler := newTestHandler(&stubCompositeUseCase{}, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chains?source=binance&symbol=BTCUSDT&timeframe=60&limit=abc", nil)
	w := httptest.NewRecorder()
	handler.GetChains(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompositeGetChains_NilHandler(t *testing.T) {
	var handler *CompositeWebHandler
	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chains?source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetChains(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// --- Funnel endpoint tests (S298) ---

func TestCompositeGetFunnel_Success(t *testing.T) {
	uc := &stubFunnelUseCase{
		reply: analyticalclient.PipelineFunnelReply{
			Stages: []analyticalclient.StageFunnelCount{
				{Stage: "signal", Count: 100},
				{Stage: "decision", Count: 80},
				{Stage: "strategy", Count: 60},
				{Stage: "risk", Count: 55},
				{Stage: "execution", Count: 50},
			},
			Source: "clickhouse",
			Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 10, ChainCount: 5},
		},
	}
	handler := newTestHandler(nil, uc, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/funnel?type=ema_crossover&source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetFunnel(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp pipelineFunnelResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Stages) != 5 {
		t.Fatalf("expected 5 stages, got %d", len(resp.Stages))
	}
	if resp.Stages[0].Stage != "signal" || resp.Stages[0].Count != 100 {
		t.Errorf("expected signal:100, got %s:%d", resp.Stages[0].Stage, resp.Stages[0].Count)
	}
}

func TestCompositeGetFunnel_MissingType(t *testing.T) {
	handler := newTestHandler(nil, &stubFunnelUseCase{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/funnel?source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetFunnel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompositeGetFunnel_NilHandler(t *testing.T) {
	var handler *CompositeWebHandler
	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/funnel?type=ema&source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetFunnel(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// --- Disposition endpoint tests (S298) ---

func TestCompositeGetDispositions_Success(t *testing.T) {
	uc := &stubDispositionUseCase{
		reply: analyticalclient.DispositionBreakdownReply{
			Dispositions: []analyticalclient.DispositionCount{
				{Disposition: "approved", Count: 80, Percentage: 80.0},
				{Disposition: "rejected", Count: 15, Percentage: 15.0},
				{Disposition: "modified", Count: 5, Percentage: 5.0},
			},
			Total:  100,
			Source: "clickhouse",
			Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 3, ChainCount: 3},
		},
	}
	handler := newTestHandler(nil, nil, uc)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/dispositions?type=position_exposure&source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetDispositions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp dispositionBreakdownResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Dispositions) != 3 {
		t.Fatalf("expected 3 dispositions, got %d", len(resp.Dispositions))
	}
	if resp.Total != 100 {
		t.Errorf("expected total 100, got %d", resp.Total)
	}
}

func TestCompositeGetDispositions_MissingType(t *testing.T) {
	handler := newTestHandler(nil, nil, &stubDispositionUseCase{})

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/dispositions?source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetDispositions(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCompositeGetDispositions_NilHandler(t *testing.T) {
	var handler *CompositeWebHandler
	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/dispositions?type=pos&source=binance&symbol=BTCUSDT&timeframe=60", nil)
	w := httptest.NewRecorder()
	handler.GetDispositions(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

// --- Attribution tests (S298) ---

func TestCompositeGetChain_WithRejectionAttribution(t *testing.T) {
	uc := &stubCompositeUseCase{
		reply: analyticalclient.CompositeChainReply{
			Chains: []analyticalclient.CompositeExecutionChain{
				{
					CorrelationID: "rejected-001",
					StageCount:    4,
					ChainComplete: false,
					MissingStages: []string{"execution"},
					Risk:          &analyticalclient.RiskWithTrace{OccurredAt: time.Now()},
					Attribution: &analyticalclient.RiskAttribution{
						Disposition: "rejected",
						Rationale:   "position size exceeds max",
						ActiveConstraints: risk.Constraints{
							MaxPositionSize: "0.01",
							MaxExposure:     "0.05",
						},
						StrategyContext: []analyticalclient.AttributionStrategyContext{
							{Type: "mean_reversion_entry", Direction: "long", Confidence: "0.72", DecisionSeverity: "high"},
						},
					},
				},
			},
			Source: "clickhouse",
			Meta:   analyticalclient.CompositeQueryMeta{TotalMs: 4, ChainCount: 1},
		},
	}
	handler := newTestHandler(uc, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/analytical/composite/chain?correlation_id=rejected-001&symbol=btcusdt", nil)
	w := httptest.NewRecorder()
	handler.GetChain(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp compositeChainResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	chain := resp.Chains[0]
	if chain.Attribution == nil {
		t.Fatal("expected attribution to be present")
	}
	if chain.Attribution.Disposition != "rejected" {
		t.Errorf("expected disposition rejected, got %s", chain.Attribution.Disposition)
	}
	if chain.Attribution.Rationale != "position size exceeds max" {
		t.Errorf("unexpected rationale: %s", chain.Attribution.Rationale)
	}
	if chain.Attribution.ActiveConstraints.MaxPositionSize != "0.01" {
		t.Errorf("expected max_position_size 0.01, got %s", chain.Attribution.ActiveConstraints.MaxPositionSize)
	}
	if len(chain.Attribution.StrategyContext) != 1 {
		t.Fatalf("expected 1 strategy context, got %d", len(chain.Attribution.StrategyContext))
	}
	if chain.Attribution.StrategyContext[0].DecisionSeverity != "high" {
		t.Errorf("expected decision_severity high, got %s", chain.Attribution.StrategyContext[0].DecisionSeverity)
	}
}
