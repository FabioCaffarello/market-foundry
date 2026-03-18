package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/interfaces/http/webserver"
)

// Evidence registers HTTP routes for evidence query endpoints, grouped by projection family.
// Each family conditionally registers its routes based on use case availability.
// Adding a new evidence type means adding one use case field to EvidenceFamilyDeps
// and one route block here.
func Evidence(deps EvidenceFamilyDeps) []webserver.Route {
	handler := handlers.NewEvidenceWebHandler(deps.GetLatestCandle, deps.GetCandleHistory, deps.GetLatestTradeBurst, deps.GetLatestVolume)

	var routes []webserver.Route

	// --- Candle family ---
	if deps.GetLatestCandle != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/evidence/candles/latest",
			Handler: handler.GetLatestCandle,
		})
	}
	if deps.GetCandleHistory != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/evidence/candles/history",
			Handler: handler.GetCandleHistory,
		})
	}

	// --- TradeBurst family ---
	if deps.GetLatestTradeBurst != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/evidence/tradeburst/latest",
			Handler: handler.GetLatestTradeBurst,
		})
	}

	// --- Volume family ---
	if deps.GetLatestVolume != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/evidence/volume/latest",
			Handler: handler.GetLatestVolume,
		})
	}

	return routes
}
