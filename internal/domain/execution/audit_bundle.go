package execution

import (
	"fmt"
	"strings"
	"time"

	"internal/domain/instrument"
)

// SessionAuditBundle is the canonical consolidated view of an operational session
// for human audit and explainability. It connects session metadata, automated
// checks, per-partition lifecycle state, order activity, fees, and artifacts
// into a single queryable response.
//
// S462: This type is the primary deliverable of the session audit bundle stage.
// It answers "what happened in this session?" without requiring multiple endpoint
// round trips or manual correlation.
type SessionAuditBundle struct {
	// Session metadata (from NATS KV).
	Session Session `json:"session"`

	// PO verification report (automated checks).
	Verification *POVerificationReport `json:"verification,omitempty"`

	// Per-partition lifecycle summary during the session window.
	Lifecycle []AuditLifecycleEntry `json:"lifecycle"`

	// Aggregated order activity counters across all partitions.
	OrderActivity AuditOrderActivity `json:"order_activity"`

	// Aggregated fee summary from fills observed in the session window.
	FeeSummary AuditFeeSummary `json:"fee_summary"`

	// Session-level consistency assessment.
	Consistency AuditConsistency `json:"consistency"`

	// S467: Structured per-check verdict index for quick scanning.
	CheckIndex AuditCheckIndex `json:"check_index"`

	// Human-readable summary of the session.
	Explanation string `json:"explanation"`

	// When this bundle was assembled.
	AssembledAt time.Time `json:"assembled_at"`
	AssemblyMs  int64     `json:"assembly_ms"`
}

// AuditLifecycleEntry captures the lifecycle state of a single partition key
// within the session's time window.
//
// Per ADR-0021, the canonical instrument identity is carried in the
// Instrument field. Migrated from Symbol string in H-6.b'.
type AuditLifecycleEntry struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`

	// Latest status per event type within the session window.
	IntentStatus    string `json:"intent_status"`
	FillStatus      string `json:"fill_status"`
	RejectionStatus string `json:"rejection_status"`

	// Effective propagation.
	Propagation string `json:"propagation"`

	// Counters within the session window.
	IntentCount    int `json:"intent_count"`
	FillCount      int `json:"fill_count"`
	RejectionCount int `json:"rejection_count"`
}

// VenueSymbol returns the lowercase venue-native symbol form.
//
// TRANSITORY ADAPTER (H-6.b' → sunset H-6.f). See ADR-0021.
func (e AuditLifecycleEntry) VenueSymbol() string {
	return strings.ToLower(string(e.Instrument.Base) + string(e.Instrument.Quote))
}

// AuditOrderActivity aggregates order counts across all partitions
// within the session window.
type AuditOrderActivity struct {
	TotalIntents    int `json:"total_intents"`
	TotalFills      int `json:"total_fills"`
	TotalRejections int `json:"total_rejections"`
	TotalErrors     int `json:"total_errors"`

	// Derived from session counters (authoritative) when available.
	FromSessionCounters bool `json:"from_session_counters"`
}

// AuditFeeSummary aggregates fee information from fills within the session window.
type AuditFeeSummary struct {
	TotalFillRecords int      `json:"total_fill_records"`
	FillsWithFee     int      `json:"fills_with_fee"`
	FillsWithoutFee  int      `json:"fills_without_fee"`
	SimulatedFills   int      `json:"simulated_fills"`
	FeeAssets        []string `json:"fee_assets,omitempty"`
	FeeCoverageRatio string   `json:"fee_coverage_ratio"` // e.g., "5/5" or "3/5"
}

// AuditConsistency summarizes the cross-surface consistency findings
// at the session level.
type AuditConsistency struct {
	SessionFound          bool   `json:"session_found"`
	VerificationRan       bool   `json:"verification_ran"`
	LifecycleAvailable    bool   `json:"lifecycle_available"`
	CountersMatchActivity bool   `json:"counters_match_activity"`
	AllChecksPassed       bool   `json:"all_checks_passed"`
	OverallVerdict        string `json:"overall_verdict"` // "consistent", "degraded", "inconsistent"
}

// AuditCheckIndex provides a structured per-check verdict map for quick
// scanning without parsing the full verification report.
// S467: Enables operators to spot specific check failures at a glance.
type AuditCheckIndex struct {
	Verdicts map[string]string `json:"verdicts"` // check_id -> verdict
	Failed   []string          `json:"failed,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

// NewAuditCheckIndex builds the check index from a verification report.
func NewAuditCheckIndex(report *POVerificationReport) AuditCheckIndex {
	idx := AuditCheckIndex{Verdicts: make(map[string]string)}
	if report == nil {
		return idx
	}
	for _, c := range report.Checks {
		idx.Verdicts[string(c.CheckID)] = string(c.Verdict)
		switch c.Verdict {
		case VerdictFail:
			idx.Failed = append(idx.Failed, string(c.CheckID))
		case VerdictWarn:
			idx.Warnings = append(idx.Warnings, string(c.CheckID))
		}
	}
	return idx
}

// NewAuditOrderActivityFromCounters creates order activity from session counters
// when the session has terminal state and counters are available.
func NewAuditOrderActivityFromCounters(counters []SessionSegmentCounters) AuditOrderActivity {
	activity := AuditOrderActivity{FromSessionCounters: true}
	for _, c := range counters {
		activity.TotalIntents += int(c.Processed)
		activity.TotalFills += int(c.Filled)
		activity.TotalRejections += int(c.Rejected)
		activity.TotalErrors += int(c.Errors)
	}
	return activity
}

// NewAuditFeeSummary computes fee summary from a set of fill records.
func NewAuditFeeSummary(fills []FillRecord) AuditFeeSummary {
	summary := AuditFeeSummary{
		TotalFillRecords: len(fills),
	}

	assetSet := map[string]struct{}{}
	for _, f := range fills {
		if f.Simulated {
			summary.SimulatedFills++
		}
		if f.Fee != "" && f.Fee != "0" {
			summary.FillsWithFee++
			if f.FeeAsset != "" {
				assetSet[f.FeeAsset] = struct{}{}
			}
		} else {
			summary.FillsWithoutFee++
		}
	}

	for asset := range assetSet {
		summary.FeeAssets = append(summary.FeeAssets, asset)
	}

	if summary.TotalFillRecords > 0 {
		summary.FeeCoverageRatio = fmt.Sprintf("%d/%d", summary.FillsWithFee, summary.TotalFillRecords)
	} else {
		summary.FeeCoverageRatio = "0/0"
	}

	return summary
}

// BatchCheckAggregation captures per-check verdict distribution across sessions.
// S485: Enables operators to see which PO checks fail most across sessions.
type BatchCheckAggregation struct {
	CheckID   string `json:"check_id"`
	PassCount int    `json:"pass_count"`
	FailCount int    `json:"fail_count"`
	WarnCount int    `json:"warn_count"`
	SkipCount int    `json:"skip_count"`
}

// BatchAuditSummary aggregates verdicts across multiple session audit bundles
// for quick operational review. S467: Enables batch audit without requiring
// the operator to inspect each session individually.
type BatchAuditSummary struct {
	TotalSessions    int                     `json:"total_sessions"`
	Consistent       int                     `json:"consistent"`
	Degraded         int                     `json:"degraded"`
	Inconsistent     int                     `json:"inconsistent"`
	Errored          int                     `json:"errored"`
	CheckAggregation []BatchCheckAggregation `json:"check_aggregation,omitempty"` // S485: per-check verdict distribution
}

// BatchAuditEntry pairs a session audit bundle with an optional error
// for sessions that failed to assemble.
type BatchAuditEntry struct {
	SessionID string              `json:"session_id"`
	Bundle    *SessionAuditBundle `json:"bundle,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// BatchAuditResult is the top-level batch audit response.
// It provides a per-session breakdown and an aggregate summary
// so operators can triage quickly by verdict.
type BatchAuditResult struct {
	Entries     []BatchAuditEntry `json:"entries"`
	Summary     BatchAuditSummary `json:"summary"`
	AssembledAt time.Time         `json:"assembled_at"`
	AssemblyMs  int64             `json:"assembly_ms"`
}

// ComputeBatchSummary derives aggregate verdicts from the entries.
// S485: Also computes per-check aggregation for operational triage.
func ComputeBatchSummary(entries []BatchAuditEntry) BatchAuditSummary {
	s := BatchAuditSummary{TotalSessions: len(entries)}

	// S485: Track per-check verdicts across all sessions.
	checkMap := make(map[string]*BatchCheckAggregation)

	for _, e := range entries {
		if e.Error != "" {
			s.Errored++
			continue
		}
		if e.Bundle == nil {
			s.Errored++
			continue
		}
		switch e.Bundle.Consistency.OverallVerdict {
		case "consistent":
			s.Consistent++
		case "degraded":
			s.Degraded++
		default:
			s.Inconsistent++
		}

		// S485: Aggregate per-check verdicts.
		if e.Bundle.Verification != nil {
			for _, c := range e.Bundle.Verification.Checks {
				id := string(c.CheckID)
				agg, ok := checkMap[id]
				if !ok {
					agg = &BatchCheckAggregation{CheckID: id}
					checkMap[id] = agg
				}
				switch c.Verdict {
				case VerdictPass:
					agg.PassCount++
				case VerdictFail:
					agg.FailCount++
				case VerdictWarn:
					agg.WarnCount++
				case VerdictSkip, VerdictManual:
					agg.SkipCount++
				}
			}
		}
	}

	// Convert map to ordered slice using canonical check order.
	for _, checkID := range AllPOChecks() {
		if agg, ok := checkMap[string(checkID)]; ok {
			s.CheckAggregation = append(s.CheckAggregation, *agg)
		}
	}

	return s
}
