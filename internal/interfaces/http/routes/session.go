package routes

import (
	"net/http"

	"internal/interfaces/http/handlers"
	"internal/shared/webserver"
)

// Session registers HTTP routes for session metadata query endpoints.
// S460: Exposes canonical session entity as queryable HTTP surface.
// S461: Adds PO verification endpoint.
// S462: Adds audit bundle endpoint.
// S491: Adds unified operational report endpoint.
func Session(deps SessionFamilyDeps) []webserver.Route {
	handler := handlers.NewSessionWebHandler(deps.GetSession, deps.ListSessions, deps.VerifySession, deps.AuditSession, deps.BatchAuditSession, deps.UnifiedReport)

	var routes []webserver.Route

	if deps.ListSessions != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/list",
			Handler: handler.ListSessions,
		})
	}

	// S467: Batch audit registered as a fixed path before wildcards.
	if deps.BatchAuditSession != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/batch-audit",
			Handler: handler.BatchAuditSessions,
		})
	}

	// S461: Verify must be registered BEFORE the wildcard :id route.
	if deps.VerifySession != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/:id/verify",
			Handler: handler.VerifySession,
		})
	}

	// S462: Audit must be registered BEFORE the wildcard :id route.
	if deps.AuditSession != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/:id/audit",
			Handler: handler.AuditSession,
		})
	}

	// S491: Unified report must be registered BEFORE the wildcard :id route.
	if deps.UnifiedReport != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/:id/report",
			Handler: handler.UnifiedReport,
		})
	}

	if deps.GetSession != nil {
		routes = append(routes, webserver.Route{
			Method:  http.MethodGet,
			Path:    "/session/:id",
			Handler: handler.GetSession,
		})
	}

	return routes
}
