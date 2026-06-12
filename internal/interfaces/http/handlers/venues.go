package handlers

import (
	"net/http"

	"internal/application/ports"
)

// VenuesWebHandler serves the multi-venue capabilities introspection
// surface (ADR-0022 R2). The declarations are static — resolved once
// at boot from each shipping adapter's Capabilities(); capabilities
// change only on deploy, so no refresh path exists by design.
type VenuesWebHandler struct {
	capabilities []ports.Capabilities
}

// NewVenuesWebHandler builds the handler over the union of all
// shipping adapters' declarations (wired in cmd/gateway).
func NewVenuesWebHandler(caps []ports.Capabilities) *VenuesWebHandler {
	return &VenuesWebHandler{capabilities: caps}
}

// Capabilities returns the union of all adapters' Capabilities() as
// JSON: {"venues": [...]}. Consumers (operators, the future Odin
// client, monitoring) discover at runtime what each venue declares
// and MUST tolerate absence of undeclared event types (ADR-0022 R3).
func (h *VenuesWebHandler) Capabilities(w http.ResponseWriter, _ *http.Request) {
	writeJSONResponse(w, http.StatusOK, map[string]any{
		"venues": h.capabilities,
	})
}
