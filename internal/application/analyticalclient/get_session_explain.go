package analyticalclient

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"internal/application/executionclient"
	"internal/domain/execution"
	"internal/shared/problem"
)

// SessionExplainKVReader is the local interface for reading KV execution status.
// It maps to the execution gateway's composite status query.
//
// S455A: Enables the session explain use case to read KV state without
// importing the gateway adapter directly.
type SessionExplainKVReader interface {
	Execute(context.Context, executionclient.ExecutionStatusQuery) (executionclient.ExecutionStatusReply, *problem.Problem)
}

// GetSessionExplainUseCase combines KV latest state and ClickHouse history
// into a single operational explanation with cross-surface consistency checks.
type GetSessionExplainUseCase struct {
	chReader LifecycleHistoryReader
	kvReader SessionExplainKVReader
	logger   *slog.Logger
}

func NewGetSessionExplainUseCase(chReader LifecycleHistoryReader, kvReader SessionExplainKVReader, logger *slog.Logger) *GetSessionExplainUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &GetSessionExplainUseCase{
		chReader: chReader,
		kvReader: kvReader,
		logger:   logger.With("component", "session_explain_usecase"),
	}
}

func (uc *GetSessionExplainUseCase) Execute(ctx context.Context, query SessionExplainQuery) (SessionExplainReply, *problem.Problem) {
	if query.Source == "" {
		return SessionExplainReply{}, problem.New(problem.InvalidArgument, "source is required")
	}
	if query.Symbol == "" {
		return SessionExplainReply{}, problem.New(problem.InvalidArgument, "symbol is required")
	}
	if query.Timeframe <= 0 {
		return SessionExplainReply{}, problem.New(problem.InvalidArgument, "timeframe must be positive")
	}

	if query.Limit <= 0 {
		query.Limit = defaultLimit
	}
	if query.Limit > maxLimit {
		query.Limit = maxLimit
	}

	start := time.Now()
	reply := SessionExplainReply{
		Source:    query.Source,
		Symbol:    query.Symbol,
		Timeframe: query.Timeframe,
	}

	// Phase 1: Read KV latest state (best-effort).
	var kvStatus executionclient.ExecutionStatusReply
	if uc.kvReader != nil {
		kvResult, kvProb := uc.kvReader.Execute(ctx, executionclient.ExecutionStatusQuery{
			Source:    query.Source,
			Symbol:    query.Symbol,
			Timeframe: query.Timeframe,
		})
		if kvProb == nil {
			kvStatus = kvResult
			reply.KVAvailable = true
			reply.KVPropagation = kvResult.Propagation
			if kvResult.Intent != nil {
				reply.KVIntentStatus = string(kvResult.Intent.Status)
			}
			if kvResult.Result != nil {
				reply.KVFillStatus = string(kvResult.Result.Status)
			}
			if kvResult.Rejection != nil {
				reply.KVRejectionStatus = string(kvResult.Rejection.Status)
			}
		} else {
			uc.logger.Warn("kv status unavailable for explain",
				"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
				"problem", kvProb.Code,
			)
		}
	}

	// Phase 2: Read ClickHouse lifecycle history.
	if uc.chReader != nil {
		intents, err := uc.chReader.QueryLifecycleHistory(ctx, query.Source, query.Symbol, query.Timeframe, "", "", 0, 0, query.Limit)
		if err == nil {
			reply.CHAvailable = true
			entries := make([]LifecycleHistoryEntry, 0, len(intents))
			for _, intent := range intents {
				entries = append(entries, intentToLifecycleEntry(intent))
			}
			reply.History = entries

			// Derive CH latest status per type (history is newest-first).
			reply.CHLatestIntentStatus, reply.CHLatestFillStatus, reply.CHLatestRejectionStatus = deriveCHLatestStatuses(intents)
			reply.CHPropagation = deriveCHPropagation(intents)
		} else {
			uc.logger.Warn("clickhouse lifecycle unavailable for explain",
				"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
				"error", err,
			)
		}
	}

	// Phase 3: Cross-surface consistency checks.
	reply.Consistency, reply.Consistent = computeConsistencyChecks(reply, kvStatus)

	// Phase 4: Build explanation.
	reply.Explanation = buildExplanation(reply)

	elapsed := time.Since(start)
	reply.Meta = QueryMeta{
		QueryMs:  elapsed.Milliseconds(),
		RowCount: len(reply.History),
	}

	uc.logger.Info("session explain completed",
		"source", query.Source, "symbol", query.Symbol, "timeframe", query.Timeframe,
		"kv_available", reply.KVAvailable, "ch_available", reply.CHAvailable,
		"consistent", reply.Consistent, "history_rows", len(reply.History),
		"elapsed_ms", elapsed.Milliseconds(),
	)

	return reply, nil
}

// deriveCHLatestStatuses extracts the most recent status per event type from the
// ClickHouse history (which is newest-first). Returns the first occurrence of each type.
func deriveCHLatestStatuses(intents []execution.ExecutionIntent) (intent, fill, rejection string) {
	for _, i := range intents {
		switch i.Type {
		case "paper_order":
			if intent == "" {
				intent = string(i.Status)
			}
		case "venue_market_order":
			if fill == "" {
				fill = string(i.Status)
			}
		case "venue_rejection":
			if rejection == "" {
				rejection = string(i.Status)
			}
		}
	}
	return
}

// deriveCHPropagation derives the effective propagation from ClickHouse history
// using the same logic as KV's DeriveEffectivePropagation but from historical data.
func deriveCHPropagation(intents []execution.ExecutionIntent) string {
	var latestIntent, latestFill, latestRejection *execution.ExecutionIntent
	for idx := range intents {
		i := &intents[idx]
		switch i.Type {
		case "paper_order":
			if latestIntent == nil {
				latestIntent = i
			}
		case "venue_market_order":
			if latestFill == nil {
				latestFill = i
			}
		case "venue_rejection":
			if latestRejection == nil {
				latestRejection = i
			}
		}
	}
	return executionclient.DeriveEffectivePropagation(latestIntent, latestFill, latestRejection)
}

// computeConsistencyChecks compares KV and ClickHouse state and returns findings.
func computeConsistencyChecks(reply SessionExplainReply, kvStatus executionclient.ExecutionStatusReply) ([]ConsistencyCheck, bool) {
	var checks []ConsistencyCheck
	consistent := true

	if !reply.KVAvailable && !reply.CHAvailable {
		checks = append(checks, ConsistencyCheck{
			Surface: "both",
			Field:   "availability",
			Status:  "unavailable",
			Detail:  "Neither KV nor ClickHouse returned data for this partition key.",
		})
		return checks, false
	}

	if !reply.KVAvailable {
		checks = append(checks, ConsistencyCheck{
			Surface: "kv",
			Field:   "availability",
			Status:  "unavailable",
			Detail:  "KV store did not respond. Only ClickHouse data available.",
		})
		consistent = false
	}

	if !reply.CHAvailable {
		checks = append(checks, ConsistencyCheck{
			Surface: "clickhouse",
			Field:   "availability",
			Status:  "unavailable",
			Detail:  "ClickHouse did not respond. Only KV data available.",
		})
		consistent = false
	}

	if reply.KVAvailable && reply.CHAvailable {
		// Check intent status consistency.
		checks = append(checks, compareField("intent_status", reply.KVIntentStatus, reply.CHLatestIntentStatus, &consistent))

		// Check fill status consistency.
		checks = append(checks, compareField("fill_status", reply.KVFillStatus, reply.CHLatestFillStatus, &consistent))

		// Check rejection status consistency.
		checks = append(checks, compareField("rejection_status", reply.KVRejectionStatus, reply.CHLatestRejectionStatus, &consistent))

		// Check propagation consistency.
		checks = append(checks, compareField("propagation", reply.KVPropagation, reply.CHPropagation, &consistent))

		// Check quantity precision (KV stores strings, CH stores Float64).
		if kvStatus.Intent != nil && reply.CHLatestIntentStatus != "" {
			checks = append(checks, ConsistencyCheck{
				Surface: "cross",
				Field:   "quantity_precision",
				Status:  "known_limitation",
				Detail:  "KV stores string quantities; ClickHouse stores Float64. Trailing zeros may differ (e.g., '0.50' vs '0.5'). This is a known representation difference, not data loss.",
			})
		}

		// Check correlation ID semantics.
		if len(reply.History) > 0 {
			checks = append(checks, ConsistencyCheck{
				Surface: "clickhouse",
				Field:   "correlation_id_semantics",
				Status:  "known_limitation",
				Detail:  "ClickHouse stores both event-envelope correlation_id and intent exec_correlation_id. The read path maps exec_correlation_id to ExecutionIntent.CorrelationID. Both IDs exist in the same row.",
			})
		}
	}

	return checks, consistent
}

func compareField(field, kvVal, chVal string, consistent *bool) ConsistencyCheck {
	if kvVal == chVal {
		return ConsistencyCheck{
			Surface: "cross",
			Field:   field,
			Status:  "consistent",
			KVValue: kvVal,
			CHValue: chVal,
		}
	}
	// Both empty is consistent (no data in either surface).
	if kvVal == "" && chVal == "" {
		return ConsistencyCheck{
			Surface: "cross",
			Field:   field,
			Status:  "consistent",
			Detail:  "No data in either surface.",
		}
	}
	// One has data and the other doesn't — might be timing.
	if kvVal == "" || chVal == "" {
		*consistent = false
		return ConsistencyCheck{
			Surface: "cross",
			Field:   field,
			Status:  "divergent",
			KVValue: kvVal,
			CHValue: chVal,
			Detail:  "One surface has data while the other does not. This may be a timing difference if the event is recent.",
		}
	}
	*consistent = false
	return ConsistencyCheck{
		Surface: "cross",
		Field:   field,
		Status:  "divergent",
		KVValue: kvVal,
		CHValue: chVal,
		Detail:  fmt.Sprintf("KV reports '%s' but ClickHouse reports '%s'.", kvVal, chVal),
	}
}

// buildExplanation constructs a human-readable lifecycle narrative.
func buildExplanation(reply SessionExplainReply) string {
	var parts []string
	key := fmt.Sprintf("%s.%s.%d", reply.Source, reply.Symbol, reply.Timeframe)

	if !reply.KVAvailable && !reply.CHAvailable {
		return fmt.Sprintf("No data found for partition %s in either KV or ClickHouse.", key)
	}

	// Use whichever propagation is available (prefer KV as it's the live view).
	prop := reply.KVPropagation
	if prop == "" || prop == "none" {
		prop = reply.CHPropagation
	}

	switch prop {
	case "filled":
		parts = append(parts, fmt.Sprintf("Order for %s reached terminal state: filled.", key))
	case "rejected":
		parts = append(parts, fmt.Sprintf("Order for %s was rejected by the venue.", key))
	case "cancelled":
		parts = append(parts, fmt.Sprintf("Order for %s was cancelled.", key))
	case "partially_filled":
		parts = append(parts, fmt.Sprintf("Order for %s is partially filled (non-terminal).", key))
	case "submitted", "sent", "accepted":
		parts = append(parts, fmt.Sprintf("Order for %s is in-progress (status: %s).", key, prop))
	case "none", "":
		parts = append(parts, fmt.Sprintf("No lifecycle state found for %s.", key))
	default:
		parts = append(parts, fmt.Sprintf("Order for %s has status: %s.", key, prop))
	}

	// Add history summary.
	if len(reply.History) > 0 {
		typeCount := map[string]int{}
		for _, e := range reply.History {
			typeCount[e.Type]++
		}
		var typeSummary []string
		for t, c := range typeCount {
			typeSummary = append(typeSummary, fmt.Sprintf("%d %s", c, t))
		}
		parts = append(parts, fmt.Sprintf("ClickHouse has %d events: %s.", len(reply.History), strings.Join(typeSummary, ", ")))
	}

	// Add consistency note.
	if reply.Consistent {
		parts = append(parts, "KV and ClickHouse are consistent.")
	} else if reply.KVAvailable && reply.CHAvailable {
		parts = append(parts, "Cross-surface divergences detected — see consistency checks.")
	}

	return strings.Join(parts, " ")
}
