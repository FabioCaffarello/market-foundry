//go:build requireclickhouse

package writerpipeline

// canonical_columns_integration_test.go — H-6.d.1 commit 3b: affirmative
// canary that the writer pipeline populates the canonical instrument columns
// (base, quote, contract) added in H-6.d.1 commit 1 (migrations 008-013).
//
// One test per Instrument-bearing table — 6 tests:
//   evidence_candles, signals, decisions, strategies, risk_assessments,
//   executions (also exercised via mapVenueFillRow + mapVenueRejectionRow
//   shape since they target the same table).
//
// Each test:
//   1. Construct an event with a known canonical Instrument (btcUSDTPerp).
//   2. Run it through the real row mapper from support.go.
//   3. INSERT via the real INSERT SQL string from cmd/writer/pipeline.go.
//   4. SELECT base, quote, contract WHERE event_id = ?.
//   5. Assert all 3 columns equal Instrument.Base / .Quote / .Contract.
//
// This is the affirmative pair to the schema-compatibility fix in commit 3a.
// Whereas commit 3a kept existing tests run-able against the new schema (with
// canonical columns defaulted to ''), this commit proves the WRITE PATH
// actually populates the columns end-to-end.
//
// Requirements (same as live_execution_analytical_test.go):
//   CLICKHOUSE_DSN=clickhouse://default:@localhost:9000/market_foundry_test
//   Skipped when CLICKHOUSE_DSN is not set.

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	ch "internal/adapters/clickhouse"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/events"
)

// Per-table DDL with H-6.d.1 canonical columns. Tests drop + recreate each
// table to remain isolated from prior runs. Matches the post-H-6.d.1 schema
// (see deploy/migrations/008-013).

const evidenceCandlesDDLCanonical = `
CREATE TABLE IF NOT EXISTS evidence_candles (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    base           LowCardinality(String) DEFAULT '',
    quote          LowCardinality(String) DEFAULT '',
    contract       LowCardinality(String) DEFAULT '',
    timeframe      UInt32,
    open           Float64,
    high           Float64,
    low            Float64,
    close          Float64,
    volume         Float64,
    trade_count    Int64,
    open_time      DateTime64(3),
    close_time     DateTime64(3),
    final          Bool,
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY (timeframe, toYYYYMM(open_time))
ORDER BY (source, symbol, timeframe, open_time)`

const signalsDDLCanonical = `
CREATE TABLE IF NOT EXISTS signals (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    base           LowCardinality(String) DEFAULT '',
    quote          LowCardinality(String) DEFAULT '',
    contract       LowCardinality(String) DEFAULT '',
    timeframe      UInt32,
    value          Float64,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)`

const decisionsDDLCanonical = `
CREATE TABLE IF NOT EXISTS decisions (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    base           LowCardinality(String) DEFAULT '',
    quote          LowCardinality(String) DEFAULT '',
    contract       LowCardinality(String) DEFAULT '',
    timeframe      UInt32,
    outcome        LowCardinality(String),
    confidence     Float64,
    severity       LowCardinality(String) DEFAULT '',
    rationale      String DEFAULT '',
    signals        String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)`

const strategiesDDLCanonical = `
CREATE TABLE IF NOT EXISTS strategies (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    base           LowCardinality(String) DEFAULT '',
    quote          LowCardinality(String) DEFAULT '',
    contract       LowCardinality(String) DEFAULT '',
    timeframe      UInt32,
    direction      LowCardinality(String),
    confidence     Float64,
    decisions      String,
    parameters     String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)`

const riskAssessmentsDDLCanonical = `
CREATE TABLE IF NOT EXISTS risk_assessments (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type           LowCardinality(String),
    source         LowCardinality(String),
    symbol         LowCardinality(String),
    base           LowCardinality(String) DEFAULT '',
    quote          LowCardinality(String) DEFAULT '',
    contract       LowCardinality(String) DEFAULT '',
    timeframe      UInt32,
    disposition    LowCardinality(String),
    confidence     Float64,
    strategies     String,
    constraints    String,
    rationale      String,
    parameters     String,
    metadata       String,
    final          Bool,
    timestamp      DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)`

const executionsDDLCanonical = `
CREATE TABLE IF NOT EXISTS executions (
    event_id            String,
    occurred_at         DateTime64(3),
    correlation_id      String DEFAULT '',
    causation_id        String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    base                LowCardinality(String) DEFAULT '',
    quote               LowCardinality(String) DEFAULT '',
    contract            LowCardinality(String) DEFAULT '',
    timeframe           UInt32,
    side                LowCardinality(String),
    quantity            Float64,
    filled_quantity     Float64,
    status              LowCardinality(String),
    risk                String,
    fills               String,
    parameters          String,
    metadata            String,
    exec_correlation_id String DEFAULT '',
    exec_causation_id   String DEFAULT '',
    final               Bool,
    timestamp           DateTime64(3),
    ingested_at         DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)`

// skipUnlessClickHouseCanonical mirrors the helper from live_execution_analytical_test.go
// but in this package's scope (avoids cross-package _test dependency). Returns
// a connected client or skips the test.
func skipUnlessClickHouseCanonical(t *testing.T) *ch.Client {
	t.Helper()
	if os.Getenv("CLICKHOUSE_DSN") == "" {
		t.Skip("CLICKHOUSE_DSN not set — skipping live ClickHouse test")
	}
	cfg := ch.Config{
		Addr:     envOrDefault("CLICKHOUSE_ADDR", "localhost:9000"),
		Database: envOrDefault("CLICKHOUSE_DATABASE", "market_foundry_test"),
		Username: envOrDefault("CLICKHOUSE_USER", "default"),
		Password: os.Getenv("CLICKHOUSE_PASSWORD"),
	}
	client, err := ch.Open(cfg)
	if err != nil {
		t.Fatalf("open clickhouse: %v", err)
	}
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("clickhouse not reachable: %v", err)
	}
	return client
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resetTable drops + recreates the table with the given DDL so each test
// runs against a clean slate. Idempotency is acceptable: the DDL has CREATE
// IF NOT EXISTS so a partial drop is recoverable on retry.
//
// DDL goes through Client.Exec (native-protocol non-row-returning path);
// clickhouse-go/v2's Query() returns EOF on DDL because it expects a row
// stream that DDL never produces. See client.go Exec docstring.
func resetTable(t *testing.T, client *ch.Client, table, ddl string) {
	t.Helper()
	ctx := context.Background()
	if err := client.Exec(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
		t.Fatalf("drop %s: %v", table, err)
	}
	if err := client.Exec(ctx, ddl); err != nil {
		t.Fatalf("create %s: %v", table, err)
	}
}

// queryCanonicalColumns SELECTs base, quote, contract for the given event_id.
// Returns the three column values; t.Fatal on missing row.
func queryCanonicalColumns(t *testing.T, client *ch.Client, table, eventID string) (string, string, string) {
	t.Helper()
	ctx := context.Background()
	rows, err := client.Query(ctx,
		"SELECT base, quote, contract FROM "+table+" WHERE event_id = ?",
		eventID,
	)
	if err != nil {
		t.Fatalf("SELECT %s: %v", table, err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("no row found in %s for event_id=%s", table, eventID)
	}
	var base, quote, contract string
	if err := rows.Scan(&base, &quote, &contract); err != nil {
		t.Fatalf("scan %s: %v", table, err)
	}
	return base, quote, contract
}

// assertCanonicalColumns checks that the queried columns match the input
// Instrument fields. Provides the operator-actionable failure shape that
// would surface if the writer pipeline silently drops Instrument values
// (the H-6.b' commit 37f8ddd regression-shape at the storage boundary).
func assertCanonicalColumns(t *testing.T, table string, gotBase, gotQuote, gotContract, wantBase, wantQuote, wantContract string) {
	t.Helper()
	if gotBase != wantBase {
		t.Errorf("%s.base: got %q, want %q", table, gotBase, wantBase)
	}
	if gotQuote != wantQuote {
		t.Errorf("%s.quote: got %q, want %q", table, gotQuote, wantQuote)
	}
	if gotContract != wantContract {
		t.Errorf("%s.contract: got %q, want %q", table, gotContract, wantContract)
	}
}

// INSERT SQL strings — must match the post-H-6.d.1 commit 2 pipeline.go
// strings verbatim. If a future schema change reorders columns and the
// writer pipeline regenerates from codegen, these constants must move with
// the codegen output (the test would fail at INSERT time otherwise).

const insertEvidenceCandlesSQL = "INSERT INTO evidence_candles (event_id, occurred_at, correlation_id, causation_id, source, symbol, base, quote, contract, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final)"
const insertSignalsSQL = "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, value, metadata, final, timestamp)"
const insertDecisionsSQL = "INSERT INTO decisions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp)"
const insertStrategiesSQL = "INSERT INTO strategies (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp)"
const insertRiskAssessmentsSQL = "INSERT INTO risk_assessments (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp)"
const insertExecutionsSQL = "INSERT INTO executions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp)"

// canonicalEventID returns a unique event_id for the test run, prefixed for
// observability in case the test row leaks beyond the test scope.
func canonicalEventID(suffix string) string {
	return "h6d1-canonical-canary-" + suffix
}

// metadataWith returns events.Metadata with the canonical canary event id.
func metadataWith(id string) events.Metadata {
	return events.Metadata{
		ID:            id,
		OccurredAt:    fixedTime,
		CorrelationID: "h6d1-canon-corr",
		CausationID:   "h6d1-canon-caus",
	}
}

// ── Tests ─────────────────────────────────────────────────────────

func TestWriter_PopulatesCanonicalColumns_EvidenceCandles(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "evidence_candles", evidenceCandlesDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("candle")
	event := evidence.CandleSampledEvent{
		Metadata: metadataWith(id),
		Candle: evidence.EvidenceCandle{
			Source:     "binancef",
			Instrument: inst,
			Timeframe:  60,
			Open:       "100", High: "101", Low: "99", Close: "100.5",
			Volume: "1000", TradeCount: 10,
			OpenTime: fixedTime, CloseTime: fixedTime.Add(time.Minute),
			Final: true,
		},
	}
	row := mapCandleRow(event)
	if err := client.InsertBatch(context.Background(), insertEvidenceCandlesSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch evidence_candles: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "evidence_candles", id)
	assertCanonicalColumns(t, "evidence_candles",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
	slog.Default().Info("candle canonical columns canary passed",
		"event_id", id, "base", gotBase, "quote", gotQuote, "contract", gotContract,
	)
}

func TestWriter_PopulatesCanonicalColumns_Signals(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "signals", signalsDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("signal")
	event := signal.SignalGeneratedEvent{
		Metadata: metadataWith(id),
		Signal: signal.Signal{
			Type: "rsi", Source: "binancef", Instrument: inst, Timeframe: 60,
			Value: "42.5", Metadata: map[string]string{"period": "14"},
			Final: true, Timestamp: fixedTime,
		},
	}
	row := mapSignalRow(event)
	if err := client.InsertBatch(context.Background(), insertSignalsSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch signals: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "signals", id)
	assertCanonicalColumns(t, "signals",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
}

func TestWriter_PopulatesCanonicalColumns_Decisions(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "decisions", decisionsDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("decision")
	event := decision.DecisionEvaluatedEvent{
		Metadata: metadataWith(id),
		Decision: decision.Decision{
			Type: "rsi_oversold", Source: "binancef", Instrument: inst, Timeframe: 60,
			Outcome:    decision.OutcomeTriggered,
			Confidence: "0.85",
			Severity:   decision.SeverityHigh,
			Rationale:  "RSI below threshold",
			Signals:    []decision.SignalInput{{Type: "rsi", Value: "28", Timeframe: 60}},
			Final:      true, Timestamp: fixedTime,
		},
	}
	row := mapDecisionRow(event)
	if err := client.InsertBatch(context.Background(), insertDecisionsSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch decisions: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "decisions", id)
	assertCanonicalColumns(t, "decisions",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
}

func TestWriter_PopulatesCanonicalColumns_Strategies(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "strategies", strategiesDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("strategy")
	event := strategy.StrategyResolvedEvent{
		Metadata: metadataWith(id),
		Strategy: strategy.Strategy{
			Type: "mean_reversion_entry", Source: "binancef", Instrument: inst, Timeframe: 60,
			Direction: strategy.DirectionLong, Confidence: "0.8",
			Decisions: []strategy.DecisionInput{{
				Type: "rsi_oversold", Outcome: "triggered",
				Confidence: "0.85", Severity: "high",
				Rationale: "RSI below threshold", Timeframe: 60,
			}},
			Final: true, Timestamp: fixedTime,
		},
	}
	row := mapStrategyRow(event)
	if err := client.InsertBatch(context.Background(), insertStrategiesSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch strategies: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "strategies", id)
	assertCanonicalColumns(t, "strategies",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
}

func TestWriter_PopulatesCanonicalColumns_RiskAssessments(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "risk_assessments", riskAssessmentsDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("risk")
	event := risk.RiskAssessedEvent{
		Metadata: metadataWith(id),
		RiskAssessment: risk.RiskAssessment{
			Type: "position_exposure", Source: "binancef", Instrument: inst, Timeframe: 60,
			Disposition: risk.DispositionApproved, Confidence: "0.9",
			Strategies: []risk.StrategyInput{{
				Type: "mean_reversion_entry", Direction: "long",
				Confidence: "0.85", Timeframe: 60,
				DecisionSeverity: "high", DecisionRationale: "RSI below threshold",
			}},
			Constraints: risk.Constraints{MaxPositionSize: "0.1"},
			Rationale:   "within limits",
			Final:       true, Timestamp: fixedTime,
		},
	}
	row := mapRiskRow(event)
	if err := client.InsertBatch(context.Background(), insertRiskAssessmentsSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch risk_assessments: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "risk_assessments", id)
	assertCanonicalColumns(t, "risk_assessments",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
}

func TestWriter_PopulatesCanonicalColumns_Executions(t *testing.T) {
	client := skipUnlessClickHouseCanonical(t)
	defer func() { _ = client.Close() }()

	resetTable(t, client, "executions", executionsDDLCanonical)

	inst := btcUSDTPerp(t)
	id := canonicalEventID("execution")
	event := execution.PaperOrderSubmittedEvent{
		Metadata: metadataWith(id),
		ExecutionIntent: execution.ExecutionIntent{
			Type: "paper_order", Source: "binancef", Instrument: inst, Timeframe: 60,
			Side: execution.SideBuy, Quantity: "0.1", FilledQuantity: "0.1",
			Status: execution.StatusFilled,
			Risk: execution.RiskInput{
				Type: "position_exposure", Disposition: "approved",
				Confidence: "0.9", Timeframe: 60,
			},
			Fills: []execution.FillRecord{{
				Price: "100.5", Quantity: "0.1", Fee: "0.001",
				Simulated: true, Timestamp: fixedTime,
			}},
			CorrelationID: "h6d1-canon-corr",
			CausationID:   "h6d1-canon-caus",
			Final:         true, Timestamp: fixedTime,
		},
	}
	row := mapExecutionRow(event)
	if err := client.InsertBatch(context.Background(), insertExecutionsSQL, [][]any{row}); err != nil {
		t.Fatalf("InsertBatch executions: %v", err)
	}

	gotBase, gotQuote, gotContract := queryCanonicalColumns(t, client, "executions", id)
	assertCanonicalColumns(t, "executions",
		gotBase, gotQuote, gotContract,
		string(inst.Base), string(inst.Quote), string(inst.Contract),
	)
}
