package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"

	"github.com/julienschmidt/httprouter"
)

type getSourceExplanationUseCaseStub struct {
	reply executionclient.SourceExplainReply
	prob  *problem.Problem
}

func (s getSourceExplanationUseCaseStub) Execute(_ context.Context, _ executionclient.SourceExplainQuery) (executionclient.SourceExplainReply, *problem.Problem) {
	return s.reply, s.prob
}

func TestSourceExplainRouteRegistered(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	explanation := execution.SourcePathExplanation{
		SourcePath:   "strategy_consumer.mean_reversion_entry",
		StrategyType: "mean_reversion_entry",
		Activation: execution.ActivationSurface{
			Adapter:     execution.AdapterPaper,
			Gate:        execution.ControlGate{Status: execution.GateActive, UpdatedAt: now},
			Credentials: execution.CredentialPresent,
			Effective:   execution.ModePaper,
			ObservedAt:  now,
		},
		Gate: execution.ControlGate{Status: execution.GateActive, UpdatedAt: now},
		Config: execution.SourcePathConfig{
			MaxPositionPct:  "0.01",
			MinConfidence:   "0.50",
			StalenessMaxAge: "120s",
			RiskType:        "pass_through",
		},
		Propagation: "none",
		ObservedAt:  now,
	}

	routes := SourceExplain(SourceExplainFamilyDeps{
		GetSourceExplanation: getSourceExplanationUseCaseStub{
			reply: executionclient.SourceExplainReply{Explanation: explanation},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/execution-source-explain", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /execution-source-explain: expected %d, got %d", http.StatusOK, rec.Code)
	}

	var reply executionclient.SourceExplainReply
	if err := json.NewDecoder(rec.Body).Decode(&reply); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if reply.Explanation.SourcePath != "strategy_consumer.mean_reversion_entry" {
		t.Fatalf("expected source_path %q, got %q", "strategy_consumer.mean_reversion_entry", reply.Explanation.SourcePath)
	}

	if reply.Explanation.Config.MinConfidence != "0.50" {
		t.Fatalf("expected min_confidence %q, got %q", "0.50", reply.Explanation.Config.MinConfidence)
	}

	if reply.Explanation.Activation.Effective != execution.ModePaper {
		t.Fatalf("expected effective mode %q, got %q", execution.ModePaper, reply.Explanation.Activation.Effective)
	}
}

func TestSourceExplainRouteUnavailable(t *testing.T) {
	t.Parallel()

	routes := SourceExplain(SourceExplainFamilyDeps{
		GetSourceExplanation: getSourceExplanationUseCaseStub{
			prob: problem.New(problem.Unavailable, "source explanation gateway is unavailable"),
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/execution-source-explain", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestSourceExplainRouteOmittedWhenNil(t *testing.T) {
	t.Parallel()

	routes := SourceExplain(SourceExplainFamilyDeps{})

	if len(routes) != 0 {
		t.Fatalf("expected no routes when use case is nil, got %d", len(routes))
	}
}
