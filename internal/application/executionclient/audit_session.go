package executionclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"internal/domain/execution"
	"internal/shared/problem"
)

// AuditLifecycleReader reads lifecycle entries for the session window.
// Mapped to the LifecycleListQuery — returns per-partition lifecycle summaries.
type AuditLifecycleReader interface {
	Execute(context.Context, LifecycleListQuery) (LifecycleListReply, *problem.Problem)
}

// AuditCHFillReader reads fill records from ClickHouse for fee analysis.
// S485: Accepts since/until bounds for session-scoped queries.
type AuditCHFillReader interface {
	List(ctx context.Context, symbol, execType, status string, limit int, since, until int64) ([]VerifyCHListResult, *problem.Problem)
}

// AuditSessionUseCase assembles the canonical session audit bundle.
//
// S462: This use case orchestrates multiple read surfaces to produce a single
// consolidated view of a session, suitable for human audit and explainability.
// It reads session metadata, runs PO verification, queries lifecycle state,
// and computes fee/activity summaries.
type AuditSessionUseCase struct {
	sessionReader   VerifySessionReader
	verifyUseCase   verifySessionExecutor
	lifecycleReader AuditLifecycleReader
	fillReader      AuditCHFillReader
}

// verifySessionExecutor runs PO verification. Matches VerifySessionUseCase.Execute signature.
type verifySessionExecutor interface {
	Execute(context.Context, SessionVerifyQuery) (SessionVerifyReply, *problem.Problem)
}

func NewAuditSessionUseCase(
	sessionReader VerifySessionReader,
	verifyUseCase verifySessionExecutor,
	lifecycleReader AuditLifecycleReader,
	fillReader AuditCHFillReader,
) *AuditSessionUseCase {
	return &AuditSessionUseCase{
		sessionReader:   sessionReader,
		verifyUseCase:   verifyUseCase,
		lifecycleReader: lifecycleReader,
		fillReader:      fillReader,
	}
}

func (uc *AuditSessionUseCase) Execute(ctx context.Context, query SessionAuditQuery) (SessionAuditReply, *problem.Problem) {
	if query.SessionID == "" {
		return SessionAuditReply{}, problem.New(problem.InvalidArgument, "session_id is required")
	}

	start := time.Now()

	bundle := execution.SessionAuditBundle{
		AssembledAt: start.UTC(),
	}

	// Phase 1: Fetch session metadata.
	if uc.sessionReader != nil {
		reply, prob := uc.sessionReader.Execute(ctx, SessionGetQuery(query))
		if prob != nil {
			return SessionAuditReply{}, problem.New(problem.NotFound, "session not found: "+prob.Message)
		}
		if reply.Session == nil {
			return SessionAuditReply{}, problem.New(problem.NotFound, "session not found: "+query.SessionID)
		}
		bundle.Session = *reply.Session
		bundle.Consistency.SessionFound = true
	} else {
		return SessionAuditReply{}, problem.New(problem.Unavailable, "session reader is unavailable")
	}

	// Phase 2: Run PO verification.
	if uc.verifyUseCase != nil {
		vReply, vProb := uc.verifyUseCase.Execute(ctx, SessionVerifyQuery(query))
		if vProb == nil {
			bundle.Verification = &vReply.Report
			bundle.Consistency.VerificationRan = true
			bundle.Consistency.AllChecksPassed = vReply.Report.AllPassed()
			// S467: Build structured check index for quick scanning.
			bundle.CheckIndex = execution.NewAuditCheckIndex(&vReply.Report)
		}
	}

	// Phase 3: Query lifecycle entries.
	// S467: Use session config segments to filter lifecycle query when possible.
	if uc.lifecycleReader != nil {
		lcQuery := LifecycleListQuery{}
		if len(bundle.Session.Config.Segments) > 0 {
			lcQuery.Source = bundle.Session.Config.Segments[0]
		}
		lcReply, lcProb := uc.lifecycleReader.Execute(ctx, lcQuery)
		if lcProb == nil && len(lcReply.Entries) > 0 {
			bundle.Consistency.LifecycleAvailable = true
			bundle.Lifecycle = convertLifecycleEntries(lcReply.Entries)
		}
	}

	// Phase 4: Compute order activity.
	if bundle.Session.Status.IsTerminal() && len(bundle.Session.SegmentCounters) > 0 {
		bundle.OrderActivity = execution.NewAuditOrderActivityFromCounters(bundle.Session.SegmentCounters)
	} else {
		// Derive from lifecycle if counters not available.
		bundle.OrderActivity = deriveActivityFromLifecycle(bundle.Lifecycle)
	}

	// Phase 5: Compute fee summary (S485: session-scoped).
	bundle.FeeSummary = uc.computeFeeSummary(ctx, bundle.Session)

	// Phase 6: Cross-check counters vs lifecycle.
	bundle.Consistency.CountersMatchActivity = checkCountersMatch(bundle)

	// Phase 7: Compute overall verdict.
	bundle.Consistency.OverallVerdict = computeOverallVerdict(bundle.Consistency)

	// Phase 8: Build explanation.
	bundle.Explanation = buildAuditExplanation(bundle)

	bundle.AssemblyMs = time.Since(start).Milliseconds()

	return SessionAuditReply{Bundle: bundle}, nil
}

func (uc *AuditSessionUseCase) computeFeeSummary(ctx context.Context, session execution.Session) execution.AuditFeeSummary {
	if uc.fillReader == nil {
		return execution.AuditFeeSummary{FeeCoverageRatio: "0/0"}
	}

	// S485: Use session time bounds instead of hardcoded 24h.
	var since, until int64
	if !session.StartedAt.IsZero() {
		since = session.StartedAt.Add(-5 * time.Minute).Unix()
	} else {
		since = time.Now().UTC().Add(-24 * time.Hour).Unix()
	}
	if session.ClosedAt != nil {
		until = session.ClosedAt.Add(5 * time.Minute).Unix()
	}

	rows, prob := uc.fillReader.List(ctx, "", "", "filled", 100, since, until)
	if prob != nil || len(rows) == 0 {
		return execution.AuditFeeSummary{FeeCoverageRatio: "0/0"}
	}

	var allFills []execution.FillRecord
	for _, row := range rows {
		allFills = append(allFills, row.Fills...)
	}

	return execution.NewAuditFeeSummary(allFills)
}

func convertLifecycleEntries(entries []LifecycleEntry) []execution.AuditLifecycleEntry {
	result := make([]execution.AuditLifecycleEntry, 0, len(entries))
	for _, e := range entries {
		ale := execution.AuditLifecycleEntry{
			Source:          e.Source,
			Instrument:      instrumentFromBinding(e.Source, e.Symbol),
			Timeframe:       e.Timeframe,
			IntentStatus:    e.IntentStatus,
			FillStatus:      e.FillStatus,
			RejectionStatus: e.RejectionStatus,
			Propagation:     e.Propagation,
		}
		if e.IntentStatus != "" {
			ale.IntentCount = 1
		}
		if e.FillStatus != "" {
			ale.FillCount = 1
		}
		if e.RejectionStatus != "" {
			ale.RejectionCount = 1
		}
		result = append(result, ale)
	}
	return result
}

func deriveActivityFromLifecycle(entries []execution.AuditLifecycleEntry) execution.AuditOrderActivity {
	activity := execution.AuditOrderActivity{FromSessionCounters: false}
	for _, e := range entries {
		activity.TotalIntents += e.IntentCount
		activity.TotalFills += e.FillCount
		activity.TotalRejections += e.RejectionCount
	}
	return activity
}

func checkCountersMatch(bundle execution.SessionAuditBundle) bool {
	if !bundle.OrderActivity.FromSessionCounters {
		return true // no authoritative counters to check against
	}
	if !bundle.Consistency.LifecycleAvailable {
		return true // no lifecycle to compare
	}
	lcActivity := deriveActivityFromLifecycle(bundle.Lifecycle)
	return bundle.OrderActivity.TotalIntents >= lcActivity.TotalIntents &&
		bundle.OrderActivity.TotalFills >= lcActivity.TotalFills
}

func computeOverallVerdict(c execution.AuditConsistency) string {
	if !c.SessionFound {
		return "inconsistent"
	}
	if c.VerificationRan && c.AllChecksPassed && c.CountersMatchActivity {
		return "consistent"
	}
	if !c.VerificationRan || !c.LifecycleAvailable {
		return "degraded"
	}
	if !c.AllChecksPassed {
		return "inconsistent"
	}
	return "consistent"
}

func buildAuditExplanation(bundle execution.SessionAuditBundle) string {
	var parts []string

	s := bundle.Session
	parts = append(parts, fmt.Sprintf("Session %s (%s) ran from %s",
		s.SessionID, string(s.Status), s.StartedAt.UTC().Format(time.RFC3339)))
	if s.ClosedAt != nil {
		parts = append(parts, fmt.Sprintf("to %s (%s).",
			s.ClosedAt.UTC().Format(time.RFC3339), s.Duration().Truncate(time.Second)))
	} else {
		parts = append(parts, "(still open).")
	}

	parts = append(parts, fmt.Sprintf("Config: %s, dry_run=%t, segments=%s.",
		s.Config.VenueType, s.Config.DryRun, strings.Join(s.Config.Segments, ",")))

	parts = append(parts, fmt.Sprintf("Activation: adapter=%s, effective=%s.",
		string(s.Activation.Adapter), string(s.Activation.Effective)))

	oa := bundle.OrderActivity
	parts = append(parts, fmt.Sprintf("Activity: %d intents, %d fills, %d rejections, %d errors.",
		oa.TotalIntents, oa.TotalFills, oa.TotalRejections, oa.TotalErrors))

	fs := bundle.FeeSummary
	if fs.TotalFillRecords > 0 {
		parts = append(parts, fmt.Sprintf("Fees: %s coverage (%d simulated).",
			fs.FeeCoverageRatio, fs.SimulatedFills))
	}

	if bundle.Verification != nil {
		v := bundle.Verification
		parts = append(parts, fmt.Sprintf("Verification: %d/%d passed, %d failed, %d warnings.",
			v.Summary.Passed, v.Summary.Total, v.Summary.Failed, v.Summary.Warnings))
		// S467: Include specific failed/warned check IDs for quick triage.
		if len(bundle.CheckIndex.Failed) > 0 {
			parts = append(parts, fmt.Sprintf("Failed checks: %s.", strings.Join(bundle.CheckIndex.Failed, ", ")))
		}
		if len(bundle.CheckIndex.Warnings) > 0 {
			parts = append(parts, fmt.Sprintf("Warned checks: %s.", strings.Join(bundle.CheckIndex.Warnings, ", ")))
		}
	}

	parts = append(parts, fmt.Sprintf("Overall: %s.", bundle.Consistency.OverallVerdict))

	return strings.Join(parts, " ")
}
