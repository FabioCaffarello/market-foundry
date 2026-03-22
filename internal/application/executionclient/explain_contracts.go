package executionclient

import "internal/domain/execution"

// SourceExplainQuery is the request contract for the source-driven path explanation.
// S361: Queries the composite explainability surface for the source-driven execution path.
type SourceExplainQuery struct {
	Source    string `json:"source"`
	Symbol    string `json:"symbol"`
	Timeframe int    `json:"timeframe"`
}

// SourceExplainReply is the response contract for the source-driven path explanation.
type SourceExplainReply struct {
	Explanation execution.SourcePathExplanation `json:"explanation"`
}
