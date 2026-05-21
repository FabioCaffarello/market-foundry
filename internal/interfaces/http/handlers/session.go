package handlers

import (
	"context"
	"net/http"
	"strings"

	"internal/application/executionclient"
	"internal/shared/problem"
)

type getSessionUseCase interface {
	Execute(context.Context, executionclient.SessionGetQuery) (executionclient.SessionGetReply, *problem.Problem)
}

type listSessionsUseCase interface {
	Execute(context.Context, executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem)
}

// S461: verifySessionUseCase runs automated PO checks for a session.
type verifySessionUseCase interface {
	Execute(context.Context, executionclient.SessionVerifyQuery) (executionclient.SessionVerifyReply, *problem.Problem)
}

// S462: auditSessionUseCase assembles the consolidated audit bundle.
type auditSessionUseCase interface {
	Execute(context.Context, executionclient.SessionAuditQuery) (executionclient.SessionAuditReply, *problem.Problem)
}

// S467: batchAuditSessionUseCase audits multiple sessions in a single call.
type batchAuditSessionUseCase interface {
	Execute(context.Context, executionclient.SessionBatchAuditQuery) (executionclient.SessionBatchAuditReply, *problem.Problem)
}

// S491: unifiedReportUseCase generates the unified operational report.
type unifiedReportUseCase interface {
	Execute(context.Context, executionclient.SessionUnifiedReportQuery) (executionclient.SessionUnifiedReportReply, *problem.Problem)
}

// SessionWebHandler handles HTTP requests for session metadata queries.
// S460: Exposes session entity as a queryable HTTP surface.
// S461: Adds PO verification endpoint.
// S462: Adds audit bundle endpoint.
// S467: Adds batch audit endpoint.
// S491: Adds unified operational report endpoint.
type SessionWebHandler struct {
	getSession        getSessionUseCase
	listSessions      listSessionsUseCase
	verifySession     verifySessionUseCase
	auditSession      auditSessionUseCase
	batchAuditSession batchAuditSessionUseCase
	unifiedReport     unifiedReportUseCase // S491
}

func NewSessionWebHandler(getSession getSessionUseCase, listSessions listSessionsUseCase, verifySession verifySessionUseCase, auditSession auditSessionUseCase, batchAudit batchAuditSessionUseCase, unifiedReport unifiedReportUseCase) *SessionWebHandler {
	return &SessionWebHandler{getSession: getSession, listSessions: listSessions, verifySession: verifySession, auditSession: auditSession, batchAuditSession: batchAudit, unifiedReport: unifiedReport}
}

// GetSession handles GET /session/:id
func (h *SessionWebHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.getSession == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session query is unavailable"))
		return
	}

	sessionID := pathParam(r, "id")
	if sessionID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "session id path parameter is required"))
		return
	}

	result, prob := h.getSession.Execute(r.Context(), executionclient.SessionGetQuery{SessionID: sessionID})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// VerifySession handles GET /session/:id/verify
// S461: Runs automated PO checks and returns structured verification report.
func (h *SessionWebHandler) VerifySession(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.verifySession == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session verification is unavailable"))
		return
	}

	sessionID := pathParam(r, "id")
	if sessionID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "session id path parameter is required"))
		return
	}

	result, prob := h.verifySession.Execute(r.Context(), executionclient.SessionVerifyQuery{SessionID: sessionID})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// AuditSession handles GET /session/:id/audit
// S462: Returns the consolidated session audit bundle for human review.
func (h *SessionWebHandler) AuditSession(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.auditSession == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session audit is unavailable"))
		return
	}

	sessionID := pathParam(r, "id")
	if sessionID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "session id path parameter is required"))
		return
	}

	result, prob := h.auditSession.Execute(r.Context(), executionclient.SessionAuditQuery{SessionID: sessionID})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ListSessions handles GET /session-list
func (h *SessionWebHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.listSessions == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "session list query is unavailable"))
		return
	}

	result, prob := h.listSessions.Execute(r.Context(), executionclient.SessionListQuery{})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// BatchAuditSessions handles GET /session-batch-audit
// S467: Audits all terminal sessions (or a filtered set) in a single call.
// Query parameters:
//   - status: optional status filter (e.g., "closed", "halted")
//   - ids: optional comma-separated session IDs
func (h *SessionWebHandler) BatchAuditSessions(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.batchAuditSession == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "batch audit is unavailable"))
		return
	}

	query := executionclient.SessionBatchAuditQuery{
		StatusFilter: r.URL.Query().Get("status"),
	}

	if idsParam := r.URL.Query().Get("ids"); idsParam != "" {
		ids := splitCommaSeparated(idsParam)
		if len(ids) > 0 {
			query.SessionIDs = ids
		}
	}

	result, prob := h.batchAuditSession.Execute(r.Context(), query)
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// UnifiedReport handles GET /session/:id/report
// S491: Returns the unified operational report for a session.
func (h *SessionWebHandler) UnifiedReport(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.unifiedReport == nil {
		writeProblemResponse(w, problem.New(problem.Unavailable, "unified report is unavailable"))
		return
	}

	sessionID := pathParam(r, "id")
	if sessionID == "" {
		writeProblemResponse(w, problem.New(problem.InvalidArgument, "session id path parameter is required"))
		return
	}

	result, prob := h.unifiedReport.Execute(r.Context(), executionclient.SessionUnifiedReportQuery{
		SessionID:   sessionID,
		GeneratedBy: "http-request",
	})
	if prob != nil {
		writeProblemResponse(w, prob)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func splitCommaSeparated(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
