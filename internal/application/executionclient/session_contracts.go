package executionclient

import "internal/domain/execution"

// SessionGetQuery is the request contract for retrieving a session by ID.
type SessionGetQuery struct {
	SessionID string `json:"session_id"`
}

// SessionGetReply is the response contract for a session get query.
type SessionGetReply struct {
	Session *execution.Session `json:"session"`
}

// SessionListQuery is the request contract for listing all sessions.
type SessionListQuery struct{}

// SessionListReply is the response contract for the session list query.
type SessionListReply struct {
	Sessions []execution.Session `json:"sessions"`
	Total    int                 `json:"total"`
}

// SessionVerifyQuery is the request contract for running PO verification on a session.
// S461: Automated post-operation verification pipeline.
type SessionVerifyQuery struct {
	SessionID string `json:"session_id"`
}

// SessionVerifyReply is the response contract for a session PO verification run.
type SessionVerifyReply struct {
	Report execution.POVerificationReport `json:"report"`
}

// SessionAuditQuery is the request contract for the consolidated session audit bundle.
// S462: Returns a single response combining session metadata, PO verification,
// lifecycle state, order activity, fees, and consistency assessment.
type SessionAuditQuery struct {
	SessionID string `json:"session_id"`
}

// SessionAuditReply is the response contract for the session audit bundle.
type SessionAuditReply struct {
	Bundle execution.SessionAuditBundle `json:"bundle"`
}

// SessionBatchAuditQuery is the request contract for batch audit of multiple sessions.
// S467: When SessionIDs is empty, all terminal sessions are audited.
// When StatusFilter is set, only sessions with matching status are included.
type SessionBatchAuditQuery struct {
	SessionIDs   []string `json:"session_ids,omitempty"`
	StatusFilter string   `json:"status_filter,omitempty"`
}

// SessionBatchAuditReply is the response contract for batch session audit.
type SessionBatchAuditReply struct {
	Result execution.BatchAuditResult `json:"result"`
}
