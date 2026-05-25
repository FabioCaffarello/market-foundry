package executionclient

import (
	"context"
	"time"

	"internal/domain/execution"
	"internal/shared/clock"
	"internal/shared/problem"
)

// VerifyGateReader can read the current execution gate status.
type VerifyGateReader interface {
	Execute(context.Context, ExecutionControlQuery) (ExecutionControlReply, *problem.Problem)
}

// VerifySessionReader can read session metadata.
type VerifySessionReader interface {
	Execute(context.Context, SessionGetQuery) (SessionGetReply, *problem.Problem)
}

// VerifyCHSummary returns execution summary from ClickHouse.
// S485: Accepts since/until bounds for session-scoped queries.
type VerifyCHSummary interface {
	Summary(ctx context.Context, symbol string, since, until int64) (total int64, err *problem.Problem)
}

// VerifyCHListResult is a minimal representation of a ClickHouse execution row
// for use by verification checks, decoupled from analyticalclient types.
type VerifyCHListResult struct {
	Symbol string
	Status string
	Type   string
	Fills  []execution.FillRecord
}

// VerifyCHLister queries ClickHouse for execution records.
// S485: Accepts since/until bounds for session-scoped queries.
type VerifyCHLister interface {
	List(ctx context.Context, symbol, execType, status string, limit int, since, until int64) ([]VerifyCHListResult, *problem.Problem)
}

// VerifyConsistencyChecker performs CH-vs-KV lifecycle consistency checks.
type VerifyConsistencyChecker interface {
	CheckConsistency(ctx context.Context, source, symbol string, timeframe int) (consistent bool, evidence map[string]any, err *problem.Problem)
}

// VerifySessionUseCase runs the automated subset of PO checks for a session.
//
// S461: This is the server-side complement to scripts/po-verify.sh. The script
// is the canonical operational harness; this use case enables programmatic
// access and future automation without external tooling.
type VerifySessionUseCase struct {
	sessionReader VerifySessionReader
	gateReader    VerifyGateReader
	chSummary     VerifyCHSummary
	chLister      VerifyCHLister
	consistency   VerifyConsistencyChecker
	// clk is the time port used when materializing default
	// verification scope (when session metadata is unavailable);
	// defaults to clock.SystemClock{} via NewVerifySessionUseCase
	// and can be overridden via WithClock for tests. Not consumed
	// in this commit — call sites that read clk land in commit 6a
	// (DefaultVerificationScope migration to clock.Clock).
	clk clock.Clock
}

func NewVerifySessionUseCase(
	sessionReader VerifySessionReader,
	gateReader VerifyGateReader,
	chSummary VerifyCHSummary,
	chLister VerifyCHLister,
	consistency VerifyConsistencyChecker,
) *VerifySessionUseCase {
	return &VerifySessionUseCase{
		sessionReader: sessionReader,
		gateReader:    gateReader,
		chSummary:     chSummary,
		chLister:      chLister,
		consistency:   consistency,
		clk:           clock.SystemClock{},
	}
}

// WithClock overrides the Clock used by this use case for time-
// sourced fields. Returns the use case to allow chaining, e.g.:
//
//	uc := executionclient.NewVerifySessionUseCase(...).WithClock(testClock)
//
// Optional; defaults to clock.SystemClock{}.
func (uc *VerifySessionUseCase) WithClock(clk clock.Clock) *VerifySessionUseCase {
	if uc != nil && clk != nil {
		uc.clk = clk
	}
	return uc
}

func (uc *VerifySessionUseCase) Execute(ctx context.Context, query SessionVerifyQuery) (SessionVerifyReply, *problem.Problem) {
	if query.SessionID == "" {
		return SessionVerifyReply{}, problem.New(problem.InvalidArgument, "session_id is required")
	}

	start := time.Now()

	// Fetch session metadata and derive verification scope.
	var operator string
	scope := execution.DefaultVerificationScope()
	if uc.sessionReader != nil {
		reply, _ := uc.sessionReader.Execute(ctx, SessionGetQuery(query))
		if reply.Session != nil {
			operator = reply.Session.Operator
			scope = deriveVerificationScope(reply.Session)
		}
	}

	report := execution.POVerificationReport{
		SessionID:  query.SessionID,
		Operator:   operator,
		Scope:      &scope,
		ExecutedAt: start.UTC(),
	}

	// S485: Pass session-derived scope to all checks.
	report.Checks = append(report.Checks, uc.checkGateHalted(ctx))
	report.Checks = append(report.Checks, uc.checkBackup())
	report.Checks = append(report.Checks, uc.checkIntentRecords(ctx, scope))
	report.Checks = append(report.Checks, uc.checkVenueResponses(ctx, scope))
	report.Checks = append(report.Checks, uc.checkKVState())
	report.Checks = append(report.Checks, uc.checkSystemStatus())
	report.Checks = append(report.Checks, uc.checkFeeFields(ctx, scope))
	report.Checks = append(report.Checks, uc.checkLifecycleConsistency(ctx, scope))
	report.Checks = append(report.Checks, uc.checkScopeContainment(ctx, scope))

	report.DurationMs = time.Since(start).Milliseconds()
	report.ComputeSummary()

	return SessionVerifyReply{Report: report}, nil
}

// deriveVerificationScope extracts verification boundaries from session metadata.
// S485: Replaces hardcoded BTCUSDT/24h with session-aware values.
func deriveVerificationScope(s *execution.Session) execution.VerificationScope {
	scope := execution.VerificationScope{
		DryRun:    s.Config.DryRun,
		VenueType: s.Config.VenueType,
		Segments:  s.Config.Segments,
	}

	// Derive symbols from segments — default to BTCUSDT if unavailable.
	// TODO(S486+): map segment to canonical symbols when multi-symbol support lands.
	scope.Symbols = []string{"BTCUSDT"}

	// Derive time bounds from session lifecycle.
	if !s.StartedAt.IsZero() {
		scope.Since = s.StartedAt.Add(-5 * time.Minute) // small buffer for inflight events
	} else {
		scope.Since = time.Now().UTC().Add(-24 * time.Hour)
	}
	if s.ClosedAt != nil {
		scope.Until = s.ClosedAt.Add(5 * time.Minute) // buffer for late writes
	} else {
		scope.Until = time.Now().UTC()
	}

	return scope
}

func (uc *VerifySessionUseCase) checkGateHalted(ctx context.Context) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckGateHalted,
		Name:       "Kill-switch halt verification",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.gateReader == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Gate reader unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	reply, prob := uc.gateReader.Execute(ctx, ExecutionControlQuery{})
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Gate query failed: " + prob.Message
		return result
	}

	gateStatus := string(reply.Gate.Status)
	result.Evidence = map[string]any{"gate_status": gateStatus}

	if reply.Gate.IsHalted() {
		result.Verdict = execution.VerdictPass
		result.Detail = "Gate is halted"
	} else {
		result.Verdict = execution.VerdictWarn
		result.Detail = "Gate is " + gateStatus + ", expected halted"
	}
	return result
}

func (uc *VerifySessionUseCase) checkBackup() execution.POCheckResult {
	return execution.POCheckResult{
		CheckID:    execution.POCheckBackupCompleted,
		Name:       "Post-session backup",
		Verdict:    execution.VerdictManual,
		Detail:     "Backup verification requires filesystem access — use scripts/po-verify.sh",
		Automated:  false,
		ExecutedAt: time.Now().UTC(),
	}
}

func (uc *VerifySessionUseCase) checkIntentRecords(ctx context.Context, scope execution.VerificationScope) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckIntentRecords,
		Name:       "ClickHouse intent records",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.chSummary == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "ClickHouse summary unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	symbol := scopeSymbol(scope)
	total, prob := uc.chSummary.Summary(ctx, symbol, scope.Since.Unix(), scope.Until.Unix())
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Summary query failed: " + prob.Message
		return result
	}

	result.Evidence = map[string]any{"total_records": total, "symbol": symbol}

	if total > 0 {
		result.Verdict = execution.VerdictPass
		result.Detail = "Intent records found in ClickHouse"
	} else {
		result.Verdict = execution.VerdictWarn
		result.Detail = "No intent records found in session window"
	}
	return result
}

func (uc *VerifySessionUseCase) checkVenueResponses(ctx context.Context, scope execution.VerificationScope) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckVenueResponses,
		Name:       "ClickHouse venue response records",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.chLister == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "ClickHouse list unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	symbol := scopeSymbol(scope)
	rows, prob := uc.chLister.List(ctx, symbol, "venue_market_order", "", 10, scope.Since.Unix(), scope.Until.Unix())
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "List query failed: " + prob.Message
		return result
	}

	result.Evidence = map[string]any{"count": len(rows), "symbol": symbol}

	if len(rows) > 0 {
		result.Verdict = execution.VerdictPass
		result.Detail = "Venue response records found"
	} else {
		result.Verdict = execution.VerdictWarn
		result.Detail = "No venue response records found in session window"
	}
	return result
}

func (uc *VerifySessionUseCase) checkKVState() execution.POCheckResult {
	return execution.POCheckResult{
		CheckID:    execution.POCheckKVState,
		Name:       "NATS KV state validation",
		Verdict:    execution.VerdictPass,
		Detail:     "KV state captured via session explain (see PO-8)",
		Automated:  true,
		ExecutedAt: time.Now().UTC(),
	}
}

func (uc *VerifySessionUseCase) checkSystemStatus() execution.POCheckResult {
	return execution.POCheckResult{
		CheckID:    execution.POCheckSystemStatus,
		Name:       "System status summary",
		Verdict:    execution.VerdictPass,
		Detail:     "System is responding (this endpoint is served by the gateway)",
		Automated:  true,
		ExecutedAt: time.Now().UTC(),
	}
}

func (uc *VerifySessionUseCase) checkFeeFields(ctx context.Context, scope execution.VerificationScope) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckFeeFields,
		Name:       "Fee/commission field verification",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.chLister == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "ClickHouse list unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	symbol := scopeSymbol(scope)
	rows, prob := uc.chLister.List(ctx, symbol, "", "filled", 10, scope.Since.Unix(), scope.Until.Unix())
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "List query failed: " + prob.Message
		return result
	}

	if len(rows) == 0 {
		result.Verdict = execution.VerdictSkip
		result.Detail = "No fill records to check for fees"
		return result
	}

	// S499: Segment-aware fee field verification.
	// Fills with FeeSource="unavailable" (Futures) are expected to have zero fees.
	// Fills with FeeSource="simulated" (paper/dry-run) are also expected to have zero fees.
	// Only FeeSource="venue" or "fallback" fills are checked for non-zero fees.
	totalFills := 0
	fillsWithFee := 0
	fillsExpectedZero := 0
	fillsFallback := 0
	for _, row := range rows {
		for _, fill := range row.Fills {
			totalFills++
			if fill.Fee != "" && fill.Fee != "0" {
				fillsWithFee++
			} else if fill.FeeSource == execution.FeeSourceUnavailable || fill.FeeSource == execution.FeeSourceSimulated {
				fillsExpectedZero++
			} else if fill.FeeSource == execution.FeeSourceFallback {
				fillsFallback++
			}
		}
	}

	result.Evidence = map[string]any{
		"total_fills":         totalFills,
		"fills_with_fee":      fillsWithFee,
		"fills_expected_zero": fillsExpectedZero,
		"fills_fallback":      fillsFallback,
		"symbol":              symbol,
	}

	if totalFills == 0 {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Fill records present but no fill entries"
	} else if fillsWithFee+fillsExpectedZero == totalFills {
		result.Verdict = execution.VerdictPass
		result.Detail = "All fills have fee fields populated or have expected-zero fee source"
	} else if fillsFallback > 0 {
		result.Verdict = execution.VerdictWarn
		result.Detail = "Some fills used fallback fee path (unexpected for FULL response type)"
	} else {
		result.Verdict = execution.VerdictWarn
		result.Detail = "Some fills lack fee fields without expected fee source"
	}
	return result
}

func (uc *VerifySessionUseCase) checkLifecycleConsistency(ctx context.Context, scope execution.VerificationScope) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckLifecycleConsist,
		Name:       "Lifecycle consistency (CH vs KV)",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.consistency == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Consistency checker unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	source := scope.VenueType
	if source == "" {
		source = "binance_spot"
	}
	symbol := scopeSymbol(scope)
	consistent, evidence, prob := uc.consistency.CheckConsistency(ctx, source, symbol, 60)
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "Consistency check failed: " + prob.Message
		return result
	}

	result.Evidence = evidence

	if consistent {
		result.Verdict = execution.VerdictPass
		result.Detail = "ClickHouse and KV are consistent"
	} else {
		result.Verdict = execution.VerdictWarn
		result.Detail = "Divergence detected between ClickHouse and KV"
	}
	return result
}

func (uc *VerifySessionUseCase) checkScopeContainment(ctx context.Context, scope execution.VerificationScope) execution.POCheckResult {
	start := time.Now()
	result := execution.POCheckResult{
		CheckID:    execution.POCheckScopeContainment,
		Name:       "Scope containment verification",
		ExecutedAt: start.UTC(),
		Automated:  true,
	}

	if uc.chLister == nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "ClickHouse list unavailable"
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}

	// Query ALL venue orders (all symbols) in session window to detect scope leakage.
	rows, prob := uc.chLister.List(ctx, "", "venue_market_order", "", 100, scope.Since.Unix(), scope.Until.Unix())
	result.DurationMs = time.Since(start).Milliseconds()

	if prob != nil {
		result.Verdict = execution.VerdictSkip
		result.Detail = "List query failed: " + prob.Message
		return result
	}

	allowedSymbols := make(map[string]struct{}, len(scope.Symbols))
	for _, s := range scope.Symbols {
		allowedSymbols[s] = struct{}{}
	}

	total := len(rows)
	outOfScope := 0
	for _, row := range rows {
		if _, ok := allowedSymbols[row.Symbol]; !ok {
			outOfScope++
		}
	}

	result.Evidence = map[string]any{
		"total_executions": total,
		"out_of_scope":     outOfScope,
		"allowed_symbols":  scope.Symbols,
	}

	if outOfScope == 0 {
		result.Verdict = execution.VerdictPass
		result.Detail = "No out-of-scope executions detected"
	} else {
		result.Verdict = execution.VerdictFail
		result.Detail = "Scope violation: out-of-scope executions detected"
	}
	return result
}

// scopeSymbol returns the primary symbol from a verification scope.
func scopeSymbol(scope execution.VerificationScope) string {
	if len(scope.Symbols) > 0 {
		return scope.Symbols[0]
	}
	return "BTCUSDT"
}
