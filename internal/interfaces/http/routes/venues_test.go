package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"internal/adapters/exchanges/binancef"
	"internal/adapters/exchanges/binances"
	"internal/application/ports"

	"github.com/julienschmidt/httprouter"
)

func TestVenuesRoutesServeCapabilitiesUnion(t *testing.T) {
	t.Parallel()

	routes := Venues(VenuesFamilyDeps{
		Capabilities: []ports.Capabilities{
			binances.Capabilities(),
			binancef.Capabilities(),
		},
	})
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	router := httprouter.New()
	for _, route := range routes {
		router.HandlerFunc(route.Method, route.Path, route.Handler)
	}

	req := httptest.NewRequest(http.MethodGet, "/venues/capabilities", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body struct {
		Venues []ports.Capabilities `json:"venues"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(body.Venues) != 2 {
		t.Fatalf("expected 2 venue declarations, got %d", len(body.Venues))
	}
	seen := map[string]bool{}
	for _, v := range body.Venues {
		seen[string(v.Venue)] = true
	}
	if !seen["binance"] || !seen["binancef"] {
		t.Errorf("expected binance + binancef declarations, got %v", seen)
	}
	if !strings.Contains(rec.Body.String(), "observation.trade") {
		t.Error("declared event type observation.trade missing from payload")
	}
}

func TestVenuesFamilyDeps_HasAny(t *testing.T) {
	t.Parallel()

	if (VenuesFamilyDeps{}).HasAny() {
		t.Error("empty deps must report HasAny=false (route not registered)")
	}
	deps := VenuesFamilyDeps{Capabilities: []ports.Capabilities{binances.Capabilities()}}
	if !deps.HasAny() {
		t.Error("non-empty deps must report HasAny=true")
	}
}
