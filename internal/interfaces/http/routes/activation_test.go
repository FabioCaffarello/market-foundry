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

type getActivationSurfaceUseCaseStub struct {
	reply executionclient.ActivationSurfaceReply
	prob  *problem.Problem
}

func (s getActivationSurfaceUseCaseStub) Execute(_ context.Context, _ executionclient.ActivationSurfaceQuery) (executionclient.ActivationSurfaceReply, *problem.Problem) {
	return s.reply, s.prob
}

func TestActivationRoutesRegisterHandler(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	surface := execution.ActivationSurface{
		Adapter: execution.AdapterVenue,
		Gate: execution.ControlGate{
			Status:    execution.GateHalted,
			Reason:    "operator halt",
			UpdatedAt: now,
			UpdatedBy: "admin",
		},
		Credentials: execution.CredentialPresent,
		Effective:   execution.ModeVenueHalted,
		ObservedAt:  now,
	}

	routes := Activation(ActivationFamilyDeps{
		GetActivationSurface: getActivationSurfaceUseCaseStub{
			reply: executionclient.ActivationSurfaceReply{Surface: surface},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/activation/surface", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /activation/surface: expected %d, got %d", http.StatusOK, rec.Code)
	}

	var reply executionclient.ActivationSurfaceReply
	if err := json.NewDecoder(rec.Body).Decode(&reply); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if reply.Surface.Effective != execution.ModeVenueHalted {
		t.Fatalf("expected effective mode %q, got %q", execution.ModeVenueHalted, reply.Surface.Effective)
	}

	if reply.Surface.Gate.Status != execution.GateHalted {
		t.Fatalf("expected gate status %q, got %q", execution.GateHalted, reply.Surface.Gate.Status)
	}

	if reply.Surface.Adapter != execution.AdapterVenue {
		t.Fatalf("expected adapter %q, got %q", execution.AdapterVenue, reply.Surface.Adapter)
	}

	if reply.Surface.Credentials != execution.CredentialPresent {
		t.Fatalf("expected credentials %q, got %q", execution.CredentialPresent, reply.Surface.Credentials)
	}

	if reply.Surface.Gate.Reason != "operator halt" {
		t.Fatalf("expected gate reason %q, got %q", "operator halt", reply.Surface.Gate.Reason)
	}

	if reply.Surface.Gate.UpdatedBy != "admin" {
		t.Fatalf("expected gate updated_by %q, got %q", "admin", reply.Surface.Gate.UpdatedBy)
	}
}

func TestActivationSurfaceReturnsAllEffectiveModes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		adapter   execution.AdapterState
		gate      execution.GateStatus
		creds     execution.CredentialState
		effective execution.EffectiveMode
	}{
		{"paper", execution.AdapterPaper, execution.GateActive, execution.CredentialPresent, execution.ModePaper},
		{"venue_halted", execution.AdapterVenue, execution.GateHalted, execution.CredentialPresent, execution.ModeVenueHalted},
		{"venue_live", execution.AdapterVenue, execution.GateActive, execution.CredentialPresent, execution.ModeVenueLive},
		{"venue_degraded", execution.AdapterVenue, execution.GateActive, execution.CredentialAbsent, execution.ModeVenueDegraded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			surface := execution.NewActivationSurface(
				tc.adapter,
				execution.ControlGate{Status: tc.gate, UpdatedAt: time.Now().UTC()},
				tc.creds,
			)

			routes := Activation(ActivationFamilyDeps{
				GetActivationSurface: getActivationSurfaceUseCaseStub{
					reply: executionclient.ActivationSurfaceReply{Surface: surface},
				},
			})

			router := httprouter.New()
			for _, route := range routes {
				router.HandlerFunc(route.Method, route.Path, route.Handler)
			}

			req := httptest.NewRequest(http.MethodGet, "/activation/surface", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
			}

			var reply executionclient.ActivationSurfaceReply
			if err := json.NewDecoder(rec.Body).Decode(&reply); err != nil {
				t.Fatalf("decode: %v", err)
			}

			if reply.Surface.Effective != tc.effective {
				t.Fatalf("expected effective %q, got %q", tc.effective, reply.Surface.Effective)
			}
		})
	}
}

func TestDefaultRoutesIncludesActivationWhenProvided(t *testing.T) {
	t.Parallel()

	routes := DefaultRoutes(Dependencies{
		CreateDraft:    createDraftUseCaseStub{},
		GetConfig:      getConfigUseCaseStub{},
		GetActive:      getActiveUseCaseStub{},
		ListConfigs:    listConfigsUseCaseStub{},
		ValidateDraft:  validateDraftUseCaseStub{},
		ValidateConfig: validateConfigUseCaseStub{},
		CompileConfig:  compileConfigUseCaseStub{},
		ActivateConfig: activateConfigUseCaseStub{},
		Activation: ActivationFamilyDeps{
			GetActivationSurface: getActivationSurfaceUseCaseStub{},
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/activation/surface", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected activation surface route to be registered, got %d", rec.Code)
	}
}

func TestDefaultRoutesOmitsActivationWhenNil(t *testing.T) {
	t.Parallel()

	routes := DefaultRoutes(Dependencies{
		CreateDraft:    createDraftUseCaseStub{},
		GetConfig:      getConfigUseCaseStub{},
		GetActive:      getActiveUseCaseStub{},
		ListConfigs:    listConfigsUseCaseStub{},
		ValidateDraft:  validateDraftUseCaseStub{},
		ValidateConfig: validateConfigUseCaseStub{},
		CompileConfig:  compileConfigUseCaseStub{},
		ActivateConfig: activateConfigUseCaseStub{},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/activation/surface", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected activation route to be absent (404), got %d", rec.Code)
	}
}

func TestActivationSurfaceUnavailableWhenUseCaseNil(t *testing.T) {
	t.Parallel()

	routes := Activation(ActivationFamilyDeps{
		GetActivationSurface: getActivationSurfaceUseCaseStub{
			prob: problem.New(problem.Unavailable, "activation surface gateway is unavailable"),
		},
	})

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/activation/surface", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}
