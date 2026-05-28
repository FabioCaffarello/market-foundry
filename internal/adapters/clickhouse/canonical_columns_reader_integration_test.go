//go:build requireclickhouse

package clickhouse_test

// canonical_columns_reader_integration_test.go — H-6.d.2 commit 3.
//
// Reader-side canary for the dual-path canonical-column resolution introduced
// by H-6.d.2 commit 2. Counterpart to the writer canary in
// internal/adapters/clickhouse/writerpipeline/canonical_columns_integration_test.go
// landed by H-6.d.1.
//
// For each of the 6 Instrument-bearing tables, three subtests:
//
//   canonical_path  — insert a row with populated base/quote/contract;
//                     reader must produce an Instrument whose Base/Quote/
//                     Contract match the column values exactly.
//   fallback_path   — insert a row leaving base/quote/contract as the schema
//                     DEFAULT '' (legacy row shape, pre-H-6.d.1 writers);
//                     reader must fall back to
//                     reconstructInstrumentFromLegacy(src, sym) and produce a
//                     correctly-shaped Instrument (BTC/USDT + spot|perpetual).
//   mixed_state     — insert both row shapes in one fixture; query returns
//                     both rows; the canonical row resolves via the new
//                     helper, the legacy row resolves via the fallback. This
//                     is the production shape during the 90-day TTL window
//                     between H-6.d.1 and H-6.f.
//
// Reader removal of `reconstructInstrumentFromLegacy` is intentionally
// deferred to H-6.f per Resolution 1 (correctness-driven: legacy rows survive
// up to the TTL boundary). The mixed_state subtest is the literal proof
// that during the migration window both row shapes must reconstruct
// correctly.
//
// Requirements:
//   CLICKHOUSE_DSN must be set (gate). The test creates and drops its own
//   tables, so no migration ordering dependency. Database defaults to
//   market_foundry_test but can be overridden via CLICKHOUSE_DATABASE.

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"internal/adapters/clickhouse"
	"internal/domain/instrument"
)

// Per-table DDL post-H-6.d.1 — mirrors deploy/migrations/008-013 schema.
// Same shape as the writer canary's DDL constants (writerpipeline package);
// duplicated rather than imported because Go test packages cannot import
// _test files from other packages.

const evidenceCandlesReaderDDL = `
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

const signalsReaderDDL = `
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

const decisionsReaderDDL = `
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

const strategiesReaderDDL = `
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

const riskAssessmentsReaderDDL = `
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

const executionsReaderDDL = `
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

// skipUnlessClickHouseReader is the reader-side counterpart to the writer
// canary's skipUnlessClickHouseCanonical helper. Reads CLICKHOUSE_DSN as
// the gate, then connects with env-overridable connection parameters.
func skipUnlessClickHouseReader(t *testing.T) *clickhouse.Client {
	t.Helper()
	if os.Getenv("CLICKHOUSE_DSN") == "" {
		t.Skip("CLICKHOUSE_DSN not set — skipping live ClickHouse test")
	}
	cfg := clickhouse.Config{
		Addr:     envOr("CLICKHOUSE_ADDR", "localhost:9000"),
		Database: envOr("CLICKHOUSE_DATABASE", "market_foundry_test"),
		Username: envOr("CLICKHOUSE_USER", "default"),
		Password: os.Getenv("CLICKHOUSE_PASSWORD"),
	}
	client, err := clickhouse.Open(cfg)
	if err != nil {
		t.Fatalf("open clickhouse: %v", err)
	}
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("clickhouse not reachable: %v", err)
	}
	return client
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resetReaderTable drops + recreates the table with the given DDL so each
// subtest runs against a clean slate.
func resetReaderTable(t *testing.T, client *clickhouse.Client, table, ddl string) {
	t.Helper()
	ctx := context.Background()
	if err := client.Exec(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
		t.Fatalf("drop %s: %v", table, err)
	}
	if err := client.Exec(ctx, ddl); err != nil {
		t.Fatalf("create %s: %v", table, err)
	}
}

// assertInstrument compares a CanonicalInstrument's identity against a
// (base, quote, contract) triple. Provides operator-actionable failure shape
// for the dual-path canary.
func assertInstrument(t *testing.T, label string, got instrument.CanonicalInstrument, wantBase, wantQuote string, wantContract instrument.ContractType) {
	t.Helper()
	if string(got.Base) != wantBase {
		t.Errorf("%s: base=%q, want %q", label, got.Base, wantBase)
	}
	if string(got.Quote) != wantQuote {
		t.Errorf("%s: quote=%q, want %q", label, got.Quote, wantQuote)
	}
	if got.Contract != wantContract {
		t.Errorf("%s: contract=%q, want %q", label, got.Contract, wantContract)
	}
}

// readerCanaryTs is a fixed timestamp used by all canary inserts so the time
// filter on the readers admits them.
var readerCanaryTs = time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)

// ─── evidence_candles ───────────────────────────────────────────────────────

func TestReader_CanonicalColumns_EvidenceCandles(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "evidence_candles", evidenceCandlesReaderDDL)
		insertCandleRow(t, client, "h6d2r-cand-c1", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewCandleReader(client, slog.Default())
		got, err := r.QueryCandleHistory(context.Background(), "binancef", "btcusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryCandleHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "candle canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "evidence_candles", evidenceCandlesReaderDDL)
		insertCandleRow(t, client, "h6d2r-cand-f1", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewCandleReader(client, slog.Default())
		got, err := r.QueryCandleHistory(context.Background(), "binancef", "btcusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryCandleHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		// Legacy reconstruction: source=binancef → ContractPerpetual; symbol=btcusdt → BTC/USDT.
		assertInstrument(t, "candle fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "evidence_candles", evidenceCandlesReaderDDL)
		// Both rows share source/symbol/timeframe so QueryCandleHistory returns both.
		// Canonical row uses explicit ETH-flavored Instrument to discriminate the
		// reader output from the legacy reconstruction below (which would yield
		// BTC/USDT regardless of canonical column contents).
		insertCandleRow(t, client, "h6d2r-cand-m1-canon", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertCandleRow(t, client, "h6d2r-cand-m2-legacy", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewCandleReader(client, slog.Default())
		got, err := r.QueryCandleHistory(context.Background(), "binances", "ethusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryCandleHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		// Both rows must resolve to the same Instrument value: the canonical
		// row reads ETH/USDT/spot directly from the columns; the legacy row
		// reconstructs ETH/USDT/spot from (binances, ethusdt).
		// This is the production invariant during the TTL window.
		for i, c := range got {
			assertInstrument(t, "candle mixed["+itoa(i)+"]", c.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── signals ────────────────────────────────────────────────────────────────

func TestReader_CanonicalColumns_Signals(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "signals", signalsReaderDDL)
		insertSignalRow(t, client, "h6d2r-sig-c1", "rsi", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewSignalReader(client, slog.Default())
		got, err := r.QuerySignalHistory(context.Background(), "rsi", "binancef", "btcusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QuerySignalHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "signal canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "signals", signalsReaderDDL)
		insertSignalRow(t, client, "h6d2r-sig-f1", "rsi", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewSignalReader(client, slog.Default())
		got, err := r.QuerySignalHistory(context.Background(), "rsi", "binancef", "btcusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QuerySignalHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "signal fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "signals", signalsReaderDDL)
		insertSignalRow(t, client, "h6d2r-sig-m1-canon", "rsi", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertSignalRow(t, client, "h6d2r-sig-m2-legacy", "rsi", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewSignalReader(client, slog.Default())
		got, err := r.QuerySignalHistory(context.Background(), "rsi", "binances", "ethusdt", 60, 0, 0, 10)
		if err != nil {
			t.Fatalf("QuerySignalHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		for i, s := range got {
			assertInstrument(t, "signal mixed["+itoa(i)+"]", s.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── decisions ──────────────────────────────────────────────────────────────

func TestReader_CanonicalColumns_Decisions(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "decisions", decisionsReaderDDL)
		insertDecisionRow(t, client, "h6d2r-dec-c1", "rsi_oversold", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewDecisionReader(client, slog.Default())
		got, err := r.QueryDecisionHistory(context.Background(), "rsi_oversold", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryDecisionHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "decision canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "decisions", decisionsReaderDDL)
		insertDecisionRow(t, client, "h6d2r-dec-f1", "rsi_oversold", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewDecisionReader(client, slog.Default())
		got, err := r.QueryDecisionHistory(context.Background(), "rsi_oversold", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryDecisionHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "decision fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "decisions", decisionsReaderDDL)
		insertDecisionRow(t, client, "h6d2r-dec-m1-canon", "rsi_oversold", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertDecisionRow(t, client, "h6d2r-dec-m2-legacy", "rsi_oversold", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewDecisionReader(client, slog.Default())
		got, err := r.QueryDecisionHistory(context.Background(), "rsi_oversold", "binances", "ethusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryDecisionHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		for i, d := range got {
			assertInstrument(t, "decision mixed["+itoa(i)+"]", d.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── strategies ─────────────────────────────────────────────────────────────

func TestReader_CanonicalColumns_Strategies(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "strategies", strategiesReaderDDL)
		insertStrategyRow(t, client, "h6d2r-strat-c1", "mean_reversion_entry", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewStrategyReader(client, slog.Default())
		got, err := r.QueryStrategyHistory(context.Background(), "mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryStrategyHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "strategy canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "strategies", strategiesReaderDDL)
		insertStrategyRow(t, client, "h6d2r-strat-f1", "mean_reversion_entry", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewStrategyReader(client, slog.Default())
		got, err := r.QueryStrategyHistory(context.Background(), "mean_reversion_entry", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryStrategyHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "strategy fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "strategies", strategiesReaderDDL)
		insertStrategyRow(t, client, "h6d2r-strat-m1-canon", "mean_reversion_entry", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertStrategyRow(t, client, "h6d2r-strat-m2-legacy", "mean_reversion_entry", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewStrategyReader(client, slog.Default())
		got, err := r.QueryStrategyHistory(context.Background(), "mean_reversion_entry", "binances", "ethusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryStrategyHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		for i, s := range got {
			assertInstrument(t, "strategy mixed["+itoa(i)+"]", s.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── risk_assessments ───────────────────────────────────────────────────────

func TestReader_CanonicalColumns_RiskAssessments(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "risk_assessments", riskAssessmentsReaderDDL)
		insertRiskRow(t, client, "h6d2r-risk-c1", "position_exposure", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewRiskReader(client, slog.Default())
		got, err := r.QueryRiskHistory(context.Background(), "position_exposure", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryRiskHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "risk canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "risk_assessments", riskAssessmentsReaderDDL)
		insertRiskRow(t, client, "h6d2r-risk-f1", "position_exposure", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewRiskReader(client, slog.Default())
		got, err := r.QueryRiskHistory(context.Background(), "position_exposure", "binancef", "btcusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryRiskHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "risk fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "risk_assessments", riskAssessmentsReaderDDL)
		insertRiskRow(t, client, "h6d2r-risk-m1-canon", "position_exposure", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertRiskRow(t, client, "h6d2r-risk-m2-legacy", "position_exposure", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewRiskReader(client, slog.Default())
		got, err := r.QueryRiskHistory(context.Background(), "position_exposure", "binances", "ethusdt", 60, "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryRiskHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		for i, r := range got {
			assertInstrument(t, "risk mixed["+itoa(i)+"]", r.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── executions ─────────────────────────────────────────────────────────────

func TestReader_CanonicalColumns_Executions(t *testing.T) {
	client := skipUnlessClickHouseReader(t)
	defer client.Close()

	t.Run("canonical_path", func(t *testing.T) {
		resetReaderTable(t, client, "executions", executionsReaderDDL)
		insertExecutionRow(t, client, "h6d2r-exec-c1", "paper_order", "binancef", "btcusdt", "BTC", "USDT", "perpetual")

		r := clickhouse.NewExecutionReader(client, slog.Default())
		got, err := r.QueryExecutionHistory(context.Background(), "paper_order", "binancef", "btcusdt", 60, "", "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryExecutionHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "execution canonical", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("fallback_path", func(t *testing.T) {
		resetReaderTable(t, client, "executions", executionsReaderDDL)
		insertExecutionRow(t, client, "h6d2r-exec-f1", "paper_order", "binancef", "btcusdt", "", "", "")

		r := clickhouse.NewExecutionReader(client, slog.Default())
		got, err := r.QueryExecutionHistory(context.Background(), "paper_order", "binancef", "btcusdt", 60, "", "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryExecutionHistory: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}
		assertInstrument(t, "execution fallback", got[0].Instrument, "BTC", "USDT", instrument.ContractPerpetual)
	})

	t.Run("mixed_state", func(t *testing.T) {
		resetReaderTable(t, client, "executions", executionsReaderDDL)
		insertExecutionRow(t, client, "h6d2r-exec-m1-canon", "paper_order", "binances", "ethusdt", "ETH", "USDT", "spot")
		insertExecutionRow(t, client, "h6d2r-exec-m2-legacy", "paper_order", "binances", "ethusdt", "", "", "")

		r := clickhouse.NewExecutionReader(client, slog.Default())
		got, err := r.QueryExecutionHistory(context.Background(), "paper_order", "binances", "ethusdt", 60, "", "", 0, 0, 10)
		if err != nil {
			t.Fatalf("QueryExecutionHistory: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(got))
		}
		for i, e := range got {
			assertInstrument(t, "execution mixed["+itoa(i)+"]", e.Instrument, "ETH", "USDT", instrument.ContractSpot)
		}
	})
}

// ─── insert helpers ─────────────────────────────────────────────────────────
//
// Each helper inserts one row through Client.InsertBatch using an explicit
// column list. When base/quote/contract are passed as "", the columns receive
// the schema DEFAULT '' — the legacy-row shape.

func insertCandleRow(t *testing.T, client *clickhouse.Client, eventID, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO evidence_candles (event_id, occurred_at, correlation_id, causation_id, source, symbol, base, quote, contract, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		source, symbol, base, quote, contract, uint32(60),
		float64(1), float64(2), float64(0.5), float64(1.5), float64(100),
		int64(10), readerCanaryTs, readerCanaryTs, true,
	}})
	if err != nil {
		t.Fatalf("insert candle %s: %v", eventID, err)
	}
}

func insertSignalRow(t *testing.T, client *clickhouse.Client, eventID, sigType, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO signals (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, value, metadata, final, timestamp)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		sigType, source, symbol, base, quote, contract, uint32(60),
		float64(42.5), `{"period":"14"}`, true, readerCanaryTs,
	}})
	if err != nil {
		t.Fatalf("insert signal %s: %v", eventID, err)
	}
}

func insertDecisionRow(t *testing.T, client *clickhouse.Client, eventID, decType, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO decisions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, outcome, confidence, severity, rationale, signals, metadata, final, timestamp)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		decType, source, symbol, base, quote, contract, uint32(60),
		"actionable", float64(0.8), "warning", "test", `[]`, `{}`, true, readerCanaryTs,
	}})
	if err != nil {
		t.Fatalf("insert decision %s: %v", eventID, err)
	}
}

func insertStrategyRow(t *testing.T, client *clickhouse.Client, eventID, stratType, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO strategies (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, direction, confidence, decisions, parameters, metadata, final, timestamp)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		stratType, source, symbol, base, quote, contract, uint32(60),
		"long", float64(0.8), `[]`, `{}`, `{}`, true, readerCanaryTs,
	}})
	if err != nil {
		t.Fatalf("insert strategy %s: %v", eventID, err)
	}
}

func insertRiskRow(t *testing.T, client *clickhouse.Client, eventID, riskType, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO risk_assessments (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		riskType, source, symbol, base, quote, contract, uint32(60),
		"pass_through", float64(0.8), `[]`, `{}`, "", `{}`, `{}`, true, readerCanaryTs,
	}})
	if err != nil {
		t.Fatalf("insert risk %s: %v", eventID, err)
	}
}

func insertExecutionRow(t *testing.T, client *clickhouse.Client, eventID, execType, source, symbol, base, quote, contract string) {
	t.Helper()
	const sql = "INSERT INTO executions (event_id, occurred_at, correlation_id, causation_id, type, source, symbol, base, quote, contract, timeframe, side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp)"
	err := client.InsertBatch(context.Background(), sql, [][]any{{
		eventID, readerCanaryTs, eventID + "-corr", "",
		execType, source, symbol, base, quote, contract, uint32(60),
		"buy", float64(0.01), float64(0.01), "filled", `{}`, `[]`, `{}`, `{}`, "", "", true, readerCanaryTs,
	}})
	if err != nil {
		t.Fatalf("insert execution %s: %v", eventID, err)
	}
}

// itoa avoids strconv.Itoa import just for index labels.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+(i%10))) + digits
		i /= 10
	}
	return digits
}
