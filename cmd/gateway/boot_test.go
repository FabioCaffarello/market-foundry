package main

import (
	"net/http"
	"testing"

	"internal/shared/settings"
	"internal/shared/webserver"
)

// TestGatewayRouteRegistrationDoesNotPanic registers the full set of gateway
// HTTP routes via the same webserver.RegisterRoutes path used in production.
// httprouter panics at registration time when a static segment competes with
// a wildcard at the same trie position; that class of bug only surfaces at
// boot, not at compile time. This test is the CI-side regression guard.
//
// If this test panics, the message printed by httprouter identifies the
// conflicting segments. Resolve by renaming or removing one side, then
// update the route list below to match the new production wiring.
//
// Sync rule: whenever a new route is added in internal/interfaces/http/routes/*.go,
// add the (method, path) pair below.
func TestGatewayRouteRegistrationDoesNotPanic(t *testing.T) {
	server := webserver.NewWebServer(settings.HTTPConfig{Addr: ":0"})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("route registration panicked: %v", r)
		}
	}()

	noop := func(_ http.ResponseWriter, _ *http.Request) {}

	routes := []webserver.Route{
		{Method: http.MethodGet, Path: "/healthz", Handler: noop},
		{Method: http.MethodGet, Path: "/readyz", Handler: noop},
		{Method: http.MethodGet, Path: "/metrics", Handler: noop},

		{Method: http.MethodPost, Path: "/configctl/configs", Handler: noop},
		{Method: http.MethodGet, Path: "/configctl/config-versions", Handler: noop},
		{Method: http.MethodGet, Path: "/configctl/config-versions/:id", Handler: noop},
		{Method: http.MethodGet, Path: "/configctl/configs/active", Handler: noop},
		{Method: http.MethodPost, Path: "/configctl/configs/validate", Handler: noop},
		{Method: http.MethodPost, Path: "/configctl/config-versions/:id/validate", Handler: noop},
		{Method: http.MethodPost, Path: "/configctl/config-versions/:id/compile", Handler: noop},
		{Method: http.MethodPost, Path: "/configctl/config-versions/:id/activate", Handler: noop},

		{Method: http.MethodGet, Path: "/evidence/candles/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/evidence/candles/history", Handler: noop},
		{Method: http.MethodGet, Path: "/evidence/tradeburst/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/evidence/volume/latest", Handler: noop},

		{Method: http.MethodGet, Path: "/signal/:type/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/decision/:type/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/strategy/:type/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/risk/:type/latest", Handler: noop},

		{Method: http.MethodGet, Path: "/execution/:type/latest", Handler: noop},
		{Method: http.MethodGet, Path: "/execution/:type", Handler: noop},
		{Method: http.MethodPut, Path: "/execution/:type", Handler: noop},

		{Method: http.MethodGet, Path: "/activation/surface", Handler: noop},

		{Method: http.MethodGet, Path: "/execution-source-explain", Handler: noop},

		{Method: http.MethodGet, Path: "/analytical/evidence/candles", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/signal/history", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/decision/history", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/strategy/history", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/risk/history", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/execution/history", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/execution/lifecycle", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/execution/list", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/execution/summary", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/execution/explain", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/chain", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/chains", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/funnel", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/dispositions", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/decision/review", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/decision/reviews", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/decision/effectiveness", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/decision/effectiveness/batch", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/decision/effectiveness/summary", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing/chain", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing/review", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing/review/chain", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing/cross-session", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/composite/pairing/continuity-review", Handler: noop},

		{Method: http.MethodGet, Path: "/analytical/triage/sessions", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/triage/decisions", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/triage/roundtrips", Handler: noop},
		{Method: http.MethodGet, Path: "/analytical/triage/overview", Handler: noop},

		{Method: http.MethodGet, Path: "/session-list", Handler: noop},
		{Method: http.MethodGet, Path: "/session-batch-audit", Handler: noop},
		{Method: http.MethodGet, Path: "/session/:id/verify", Handler: noop},
		{Method: http.MethodGet, Path: "/session/:id/audit", Handler: noop},
		{Method: http.MethodGet, Path: "/session/:id/report", Handler: noop},
		{Method: http.MethodGet, Path: "/session/:id", Handler: noop},

		{Method: http.MethodGet, Path: "/monitoring/state", Handler: noop},

		{Method: http.MethodGet, Path: "/venues/capabilities", Handler: noop},
	}

	server.RegisterRoutes(routes)
}
