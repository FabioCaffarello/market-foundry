package routes

import (
	"log/slog"
	"net/http"

	"internal/application/ports"
	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// DeliveryFamilyDeps wires the WebSocket delivery endpoint (ADR-0028 /
// PROGRAM-0006). Hub is the gateway's grip on the delivery actor
// subsystem (an application port — interfaces never imports actors/,
// ADR-0005); nil when NATS is unavailable (the route is then omitted).
type DeliveryFamilyDeps struct {
	Hub ports.DeliveryHub
}

// HasAny reports whether delivery is wired.
func (d DeliveryFamilyDeps) HasAny() bool { return d.Hub != nil }

// Delivery exposes the WebSocket delivery route. Adding a route here
// requires the matching entry in cmd/gateway/boot_test.go (CLAUDE.md
// core protocol #5).
func Delivery(deps DeliveryFamilyDeps, logger *slog.Logger) []webserver.Route {
	handler := handlers.NewDeliveryWebHandler(deps.Hub, logger)
	return []webserver.Route{
		{
			Method:  http.MethodGet,
			Path:    "/ws",
			Handler: handler.Connect,
		},
	}
}
