// Package lineage defines the canonical decision lineage and causality model
// for the market-foundry pipeline. It provides stage constants, chain validation,
// and invariant checks that formalize how signal, decision, strategy, risk, and
// execution stages relate causally.
//
// S470: This package does NOT own domain data — it provides a cross-cutting
// vocabulary and validation layer over the existing domain types and event metadata.
package lineage

import (
	"fmt"

	"internal/shared/problem"
)

// Stage represents a named pipeline stage in the causal chain.
type Stage string

const (
	StageSignal    Stage = "signal"
	StageDecision  Stage = "decision"
	StageStrategy  Stage = "strategy"
	StageRisk      Stage = "risk"
	StageExecution Stage = "execution"
)

// StageOrder defines the canonical ordering of pipeline stages.
// Each stage causally depends on the preceding one.
var StageOrder = []Stage{
	StageSignal,
	StageDecision,
	StageStrategy,
	StageRisk,
	StageExecution,
}

// StageIndex returns the zero-based position of a stage in the causal chain.
// Returns -1 if the stage is not recognized.
func StageIndex(s Stage) int {
	for i, st := range StageOrder {
		if st == s {
			return i
		}
	}
	return -1
}

// ChainLink represents one stage's contribution to a causal chain.
// EventID is the event that produced this stage's output.
// CausationID is the EventID of the immediately preceding stage.
// CorrelationID is the chain-wide trace identifier (set once at signal).
type ChainLink struct {
	Stage         Stage  `json:"stage"`
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id"`
	CausationID   string `json:"causation_id"`
}

// Chain is an ordered sequence of ChainLinks representing a complete or partial
// causal chain through the pipeline. Links must be in stage order.
type Chain struct {
	Links []ChainLink `json:"links"`
}

// ValidateChain checks that a Chain satisfies the lineage invariants:
//   - Links are in correct stage order (no gaps, no reversals)
//   - All links share the same CorrelationID
//   - Each link's CausationID matches the previous link's EventID
//   - No empty EventIDs
//
// Returns nil if the chain is valid.
func ValidateChain(chain Chain) *problem.Problem {
	if len(chain.Links) == 0 {
		return problem.New(problem.InvalidArgument, "lineage chain must have at least one link")
	}

	var issues []problem.ValidationIssue

	correlationID := chain.Links[0].CorrelationID
	if correlationID == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "links[0].correlation_id",
			Message: "must not be empty",
		})
	}

	prevEventID := ""
	prevStageIdx := -1

	for i, link := range chain.Links {
		prefix := fmt.Sprintf("links[%d]", i)

		if link.EventID == "" {
			issues = append(issues, problem.ValidationIssue{
				Field:   prefix + ".event_id",
				Message: "must not be empty",
			})
		}

		idx := StageIndex(link.Stage)
		if idx == -1 {
			issues = append(issues, problem.ValidationIssue{
				Field:   prefix + ".stage",
				Message: fmt.Sprintf("unknown stage %q", link.Stage),
			})
		} else if idx <= prevStageIdx {
			issues = append(issues, problem.ValidationIssue{
				Field:   prefix + ".stage",
				Message: fmt.Sprintf("stage %q must come after %q in causal order", link.Stage, chain.Links[i-1].Stage),
			})
		}
		prevStageIdx = idx

		if link.CorrelationID != correlationID {
			issues = append(issues, problem.ValidationIssue{
				Field:   prefix + ".correlation_id",
				Message: fmt.Sprintf("must match chain correlation_id %q, got %q", correlationID, link.CorrelationID),
			})
		}

		if i > 0 && prevEventID != "" && link.CausationID != prevEventID {
			issues = append(issues, problem.ValidationIssue{
				Field:   prefix + ".causation_id",
				Message: fmt.Sprintf("must match previous link event_id %q, got %q", prevEventID, link.CausationID),
			})
		}

		prevEventID = link.EventID
	}

	if len(issues) == 0 {
		return nil
	}
	return problem.Validation(problem.InvalidArgument, "lineage chain validation failed", issues...)
}

// IsComplete reports whether a chain contains all five pipeline stages.
func IsComplete(chain Chain) bool {
	if len(chain.Links) != len(StageOrder) {
		return false
	}
	for i, link := range chain.Links {
		if link.Stage != StageOrder[i] {
			return false
		}
	}
	return true
}

// MissingStages returns the stages not present in the chain.
func MissingStages(chain Chain) []Stage {
	present := make(map[Stage]bool, len(chain.Links))
	for _, link := range chain.Links {
		present[link.Stage] = true
	}
	var missing []Stage
	for _, s := range StageOrder {
		if !present[s] {
			missing = append(missing, s)
		}
	}
	return missing
}
