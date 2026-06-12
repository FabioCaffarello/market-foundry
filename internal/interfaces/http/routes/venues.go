package routes

import (
	"net/http"

	"internal/adapters/exchanges/capabilities"
	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// VenuesFamilyDeps groups the multi-venue introspection surface
// (ADR-0022 R2, H-7.a). Capabilities carries the union of all
// shipping adapters' static declarations, resolved at boot.
type VenuesFamilyDeps struct {
	Capabilities []capabilities.Capabilities
}

// HasAny reports whether at least one venue declaration is wired.
func (v VenuesFamilyDeps) HasAny() bool {
	return len(v.Capabilities) > 0
}

// Venues exposes the capabilities introspection route. Adding a
// route here requires the matching entry in cmd/gateway/boot_test.go
// (CLAUDE.md core protocol #5).
func Venues(deps VenuesFamilyDeps) []webserver.Route {
	handler := handlers.NewVenuesWebHandler(deps.Capabilities)
	return []webserver.Route{
		{
			Method:  http.MethodGet,
			Path:    "/venues/capabilities",
			Handler: handler.Capabilities,
		},
	}
}
