package riskclient

import "internal/domain/risk"

// RiskLatestQuery is the request contract for querying the latest risk assessment of a given type.
type RiskLatestQuery struct {
	Type      string `json:"type"`
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
}

// RiskLatestReply is the response contract for the latest risk assessment query.
// RiskAssessment is always present in JSON output (null when not found) — no omitempty.
type RiskLatestReply struct {
	RiskAssessment *risk.RiskAssessment `json:"risk_assessment"`
}
