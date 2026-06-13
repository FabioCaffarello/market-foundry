package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// InsightsFamilyDeps groups insights read use cases (PROGRAM-0005 /
// ADR-0027 — decision-support, read-only).
type InsightsFamilyDeps struct {
	GetLatestVolumeProfile handlersGetLatestVolumeProfileUseCase
	GetLatestTPOProfile    handlersGetLatestTPOProfileUseCase
}

// HasAny reports whether at least one insights use case is wired.
func (i InsightsFamilyDeps) HasAny() bool {
	return i.GetLatestVolumeProfile != nil || i.GetLatestTPOProfile != nil
}

// Insights exposes the insights read routes. Adding a route here
// requires the matching entry in cmd/gateway/boot_test.go
// (CLAUDE.md core protocol #5).
func Insights(deps InsightsFamilyDeps) []webserver.Route {
	handler := handlers.NewInsightsWebHandler(deps.GetLatestVolumeProfile, deps.GetLatestTPOProfile)
	return []webserver.Route{
		{
			Method:  http.MethodGet,
			Path:    "/insights/volume-profile/latest",
			Handler: handler.GetLatestVolumeProfile,
		},
		{
			Method:  http.MethodGet,
			Path:    "/insights/tpo/latest",
			Handler: handler.GetLatestTPOProfile,
		},
	}
}
