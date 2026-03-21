//go:build requireclickhouse

package clickhouse_test

// composite_reader_integration_test.go — S296: Composite execution read model proof.
//
// This test exercises the composite reader against a live ClickHouse instance.
// It writes deterministic events across all 5 tables with known correlation_ids
// and causation_id chains, then validates:
//
//   CRI-1: Single-chain lookup by correlation_id reconstructs all 5 stages.
//   CRI-2: Causal metadata (event_id, correlation_id, causation_id, occurred_at) is preserved.
//   CRI-3: Domain fields survive the composite round-trip (same as individual readers).
//   CRI-4: Partial chains (risk-rejected, no execution) are correctly represented.
//   CRI-5: Batch lookup returns chains ordered by execution timestamp DESC.
//   CRI-6: Missing correlation_id returns an empty chain (stage_count=0).
//
// Requirements:
//   CLICKHOUSE_DSN=clickhouse://default:@localhost:9000/market_foundry_test
//   Skipped when CLICKHOUSE_DSN is not set.

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"internal/adapters/clickhouse"
)

const allTablesDDL = `
CREATE TABLE IF NOT EXISTS signals (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    timeframe           UInt32,
    value               Float64,
    metadata            String,
    final               Bool,
    timestamp           DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp);

CREATE TABLE IF NOT EXISTS decisions (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    timeframe           UInt32,
    outcome             LowCardinality(String),
    confidence          Float64,
    severity            LowCardinality(String),
    rationale           String,
    signals             String,
    metadata            String,
    final               Bool,
    timestamp           DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp);

CREATE TABLE IF NOT EXISTS strategies (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    timeframe           UInt32,
    direction           LowCardinality(String),
    confidence          Float64,
    decisions           String,
    parameters          String,
    metadata            String,
    final               Bool,
    timestamp           DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp);

CREATE TABLE IF NOT EXISTS risk_assessments (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
    timeframe           UInt32,
    disposition         LowCardinality(String),
    confidence          Float64,
    strategies          String,
    constraints         String,
    rationale           String,
    parameters          String,
    metadata            String,
    final               Bool,
    timestamp           DateTime64(3),
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp);

CREATE TABLE IF NOT EXISTS executions (
    event_id       String,
    occurred_at    DateTime64(3),
    correlation_id String DEFAULT '',
    causation_id   String DEFAULT '',
    type                LowCardinality(String),
    source              LowCardinality(String),
    symbol              LowCardinality(String),
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
    ingested_at    DateTime64(3) DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (source, symbol, timeframe, type, timestamp)
`

func setupAllTables(t *testing.T, client *clickhouse.Client) {
	t.Helper()
	ctx := context.Background()
	for _, table := range []string{"signals", "decisions", "strategies", "risk_assessments", "executions"} {
		rows, err := client.Query(ctx, "DROP TABLE IF EXISTS "+table)
		if err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
		rows.Close()
	}

	// Execute DDL statements one at a time (ClickHouse does not support multi-statement).
	ddls := []string{
		`CREATE TABLE IF NOT EXISTS signals (event_id String, occurred_at DateTime64(3), correlation_id String DEFAULT '', causation_id String DEFAULT '', type LowCardinality(String), source LowCardinality(String), symbol LowCardinality(String), timeframe UInt32, value Float64, metadata String, final Bool, timestamp DateTime64(3), ingested_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (source, symbol, timeframe, type, timestamp)`,
		`CREATE TABLE IF NOT EXISTS decisions (event_id String, occurred_at DateTime64(3), correlation_id String DEFAULT '', causation_id String DEFAULT '', type LowCardinality(String), source LowCardinality(String), symbol LowCardinality(String), timeframe UInt32, outcome LowCardinality(String), confidence Float64, severity LowCardinality(String), rationale String, signals String, metadata String, final Bool, timestamp DateTime64(3), ingested_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (source, symbol, timeframe, type, timestamp)`,
		`CREATE TABLE IF NOT EXISTS strategies (event_id String, occurred_at DateTime64(3), correlation_id String DEFAULT '', causation_id String DEFAULT '', type LowCardinality(String), source LowCardinality(String), symbol LowCardinality(String), timeframe UInt32, direction LowCardinality(String), confidence Float64, decisions String, parameters String, metadata String, final Bool, timestamp DateTime64(3), ingested_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (source, symbol, timeframe, type, timestamp)`,
		`CREATE TABLE IF NOT EXISTS risk_assessments (event_id String, occurred_at DateTime64(3), correlation_id String DEFAULT '', causation_id String DEFAULT '', type LowCardinality(String), source LowCardinality(String), symbol LowCardinality(String), timeframe UInt32, disposition LowCardinality(String), confidence Float64, strategies String, constraints String, rationale String, parameters String, metadata String, final Bool, timestamp DateTime64(3), ingested_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (source, symbol, timeframe, type, timestamp)`,
		`CREATE TABLE IF NOT EXISTS executions (event_id String, occurred_at DateTime64(3), correlation_id String DEFAULT '', causation_id String DEFAULT '', type LowCardinality(String), source LowCardinality(String), symbol LowCardinality(String), timeframe UInt32, side LowCardinality(String), quantity Float64, filled_quantity Float64, status LowCardinality(String), risk String, fills String, parameters String, metadata String, exec_correlation_id String DEFAULT '', exec_causation_id String DEFAULT '', final Bool, timestamp DateTime64(3), ingested_at DateTime64(3) DEFAULT now64(3)) ENGINE = MergeTree() PARTITION BY toYYYYMM(timestamp) ORDER BY (source, symbol, timeframe, type, timestamp)`,
	}
	for _, ddl := range ddls {
		rows, err := client.Query(ctx, ddl)
		if err != nil {
			t.Fatalf("create table: %v", err)
		}
		rows.Close()
	}
}

// insertCompositeFixtureForSymbol inserts a full 5-stage chain for the given symbol.
// This enables multi-symbol isolation testing (S301).
func insertCompositeFixtureForSymbol(t *testing.T, client *clickhouse.Client, corrID, symbol string, ts time.Time) {
	t.Helper()
	ctx := context.Background()

	err := client.InsertBatch(ctx, "INSERT INTO signals", [][]any{{
		"sig-" + corrID, ts, corrID, "",
		"rsi", "binance", symbol, uint32(60),
		42.5, `{"period":"14"}`, true, ts,
	}})
	if err != nil {
		t.Fatalf("insert signal (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO decisions", [][]any{{
		"dec-" + corrID, ts.Add(time.Millisecond), corrID, "sig-" + corrID,
		"rsi_oversold", "binance", symbol, uint32(60),
		"triggered", 0.85, "high", "RSI below 30",
		`[{"type":"rsi","value":"42.5"}]`, `{}`, true, ts.Add(time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert decision (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO strategies", [][]any{{
		"str-" + corrID, ts.Add(2 * time.Millisecond), corrID, "dec-" + corrID,
		"mean_reversion_entry", "binance", symbol, uint32(60),
		"long", 0.80,
		`[{"type":"rsi_oversold","outcome":"triggered"}]`, `{"take_profit":"0.02"}`, `{}`, true, ts.Add(2 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert strategy (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO risk_assessments", [][]any{{
		"rsk-" + corrID, ts.Add(3 * time.Millisecond), corrID, "str-" + corrID,
		"position_exposure", "binance", symbol, uint32(60),
		"approved", 0.75,
		`[{"type":"mean_reversion_entry","direction":"long"}]`,
		`{"max_position_pct":"0.05","current_exposure":"0.02"}`,
		"within limits",
		`{}`, `{}`, true, ts.Add(3 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert risk (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO executions", [][]any{{
		"exc-" + corrID, ts.Add(4 * time.Millisecond), corrID, "rsk-" + corrID,
		"paper_order", "binance", symbol, uint32(60),
		"buy", 0.1, 0.0, "submitted",
		`{"disposition":"approved","strategy_type":"mean_reversion_entry"}`, `[]`,
		`{}`, `{}`,
		corrID, "rsk-" + corrID,
		true, ts.Add(4 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert execution (%s): %v", symbol, err)
	}
}

func insertCompositeFixture(t *testing.T, client *clickhouse.Client, corrID string, ts time.Time) {
	t.Helper()
	ctx := context.Background()

	// Signal
	err := client.InsertBatch(ctx, "INSERT INTO signals", [][]any{{
		"sig-" + corrID, ts, corrID, "",
		"rsi", "binance", "btcusdt", uint32(60),
		42.5, `{"period":"14"}`, true, ts,
	}})
	if err != nil {
		t.Fatalf("insert signal: %v", err)
	}

	// Decision
	err = client.InsertBatch(ctx, "INSERT INTO decisions", [][]any{{
		"dec-" + corrID, ts.Add(time.Millisecond), corrID, "sig-" + corrID,
		"rsi_oversold", "binance", "btcusdt", uint32(60),
		"triggered", 0.85, "high", "RSI below 30",
		`[{"type":"rsi","value":"42.5"}]`, `{}`, true, ts.Add(time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert decision: %v", err)
	}

	// Strategy
	err = client.InsertBatch(ctx, "INSERT INTO strategies", [][]any{{
		"str-" + corrID, ts.Add(2 * time.Millisecond), corrID, "dec-" + corrID,
		"mean_reversion_entry", "binance", "btcusdt", uint32(60),
		"long", 0.80,
		`[{"type":"rsi_oversold","outcome":"triggered"}]`, `{"take_profit":"0.02"}`, `{}`, true, ts.Add(2 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert strategy: %v", err)
	}

	// Risk
	err = client.InsertBatch(ctx, "INSERT INTO risk_assessments", [][]any{{
		"rsk-" + corrID, ts.Add(3 * time.Millisecond), corrID, "str-" + corrID,
		"position_exposure", "binance", "btcusdt", uint32(60),
		"approved", 0.75,
		`[{"type":"mean_reversion_entry","direction":"long"}]`,
		`{"max_position_pct":"0.05","current_exposure":"0.02"}`,
		"within limits",
		`{}`, `{}`, true, ts.Add(3 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert risk: %v", err)
	}

	// Execution
	err = client.InsertBatch(ctx, "INSERT INTO executions", [][]any{{
		"exc-" + corrID, ts.Add(4 * time.Millisecond), corrID, "rsk-" + corrID,
		"paper_order", "binance", "btcusdt", uint32(60),
		"buy", 0.1, 0.0, "submitted",
		`{"disposition":"approved","strategy_type":"mean_reversion_entry"}`, `[]`,
		`{}`, `{}`,
		corrID, "rsk-" + corrID,
		true, ts.Add(4 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert execution: %v", err)
	}
}

func insertPartialFixture(t *testing.T, client *clickhouse.Client, corrID string, ts time.Time) {
	t.Helper()
	ctx := context.Background()

	// Signal + Decision + Strategy + Risk (rejected) — no Execution.
	err := client.InsertBatch(ctx, "INSERT INTO signals", [][]any{{
		"sig-" + corrID, ts, corrID, "",
		"rsi", "binance", "btcusdt", uint32(60),
		28.0, `{"period":"14"}`, true, ts,
	}})
	if err != nil {
		t.Fatalf("insert signal: %v", err)
	}

	err = client.InsertBatch(ctx, "INSERT INTO decisions", [][]any{{
		"dec-" + corrID, ts.Add(time.Millisecond), corrID, "sig-" + corrID,
		"rsi_oversold", "binance", "btcusdt", uint32(60),
		"triggered", 0.90, "high", "RSI deeply oversold",
		`[{"type":"rsi","value":"28.0"}]`, `{}`, true, ts.Add(time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert decision: %v", err)
	}

	err = client.InsertBatch(ctx, "INSERT INTO strategies", [][]any{{
		"str-" + corrID, ts.Add(2 * time.Millisecond), corrID, "dec-" + corrID,
		"mean_reversion_entry", "binance", "btcusdt", uint32(60),
		"long", 0.85,
		`[{"type":"rsi_oversold","outcome":"triggered"}]`, `{"take_profit":"0.02"}`, `{}`, true, ts.Add(2 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert strategy: %v", err)
	}

	err = client.InsertBatch(ctx, "INSERT INTO risk_assessments", [][]any{{
		"rsk-" + corrID, ts.Add(3 * time.Millisecond), corrID, "str-" + corrID,
		"position_exposure", "binance", "btcusdt", uint32(60),
		"rejected", 0.10,
		`[{"type":"mean_reversion_entry","direction":"long"}]`,
		`{"max_position_pct":"0.05","current_exposure":"0.06"}`,
		"drawdown limit exceeded",
		`{}`, `{}`, true, ts.Add(3 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert risk: %v", err)
	}
}

// CRI-1: Full chain reconstruction by correlation_id.
func TestCompositeReader_FullChain(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	corrID := "s296-full-001"
	ts := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	insertCompositeFixture(t, client, corrID, ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())
	chain, err := reader.QueryChainByCorrelationID(context.Background(), corrID, "btcusdt")
	if err != nil {
		t.Fatalf("query chain: %v", err)
	}

	if chain.CorrelationID != corrID {
		t.Errorf("correlation_id: got %q, want %q", chain.CorrelationID, corrID)
	}
	if !chain.ChainComplete {
		t.Error("expected chain_complete=true")
	}
	if chain.StageCount != 5 {
		t.Errorf("stage_count: got %d, want 5", chain.StageCount)
	}

	// CRI-2: Causal metadata preserved.
	if chain.Signal.EventID != "sig-"+corrID {
		t.Errorf("signal.event_id: got %q, want %q", chain.Signal.EventID, "sig-"+corrID)
	}
	if chain.Signal.CorrelationID != corrID {
		t.Errorf("signal.correlation_id: got %q, want %q", chain.Signal.CorrelationID, corrID)
	}
	if chain.Decision.CausationID != "sig-"+corrID {
		t.Errorf("decision.causation_id: got %q, want %q", chain.Decision.CausationID, "sig-"+corrID)
	}
	if chain.Strategy.CausationID != "dec-"+corrID {
		t.Errorf("strategy.causation_id: got %q, want %q", chain.Strategy.CausationID, "dec-"+corrID)
	}
	if chain.Risk.CausationID != "str-"+corrID {
		t.Errorf("risk.causation_id: got %q, want %q", chain.Risk.CausationID, "str-"+corrID)
	}
	if chain.Execution.EventCausationID != "rsk-"+corrID {
		t.Errorf("execution.event_causation_id: got %q, want %q", chain.Execution.EventCausationID, "rsk-"+corrID)
	}

	// CRI-3: Domain fields round-trip.
	if chain.Signal.Type != "rsi" {
		t.Errorf("signal.type: got %q, want %q", chain.Signal.Type, "rsi")
	}
	if chain.Decision.Outcome != "triggered" {
		t.Errorf("decision.outcome: got %q, want %q", chain.Decision.Outcome, "triggered")
	}
	if chain.Decision.Severity != "high" {
		t.Errorf("decision.severity: got %q, want %q", chain.Decision.Severity, "high")
	}
	if chain.Strategy.Direction != "long" {
		t.Errorf("strategy.direction: got %q, want %q", chain.Strategy.Direction, "long")
	}
	if chain.Risk.Disposition != "approved" {
		t.Errorf("risk.disposition: got %q, want %q", chain.Risk.Disposition, "approved")
	}
	if chain.Execution.Side != "buy" {
		t.Errorf("execution.side: got %q, want %q", chain.Execution.Side, "buy")
	}
	if chain.Execution.Status != "submitted" {
		t.Errorf("execution.status: got %q, want %q", chain.Execution.Status, "submitted")
	}
}

// CRI-4: Partial chain — risk rejected, no execution.
func TestCompositeReader_PartialChain(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	corrID := "s296-partial-001"
	ts := time.Date(2026, 3, 21, 13, 0, 0, 0, time.UTC)
	insertPartialFixture(t, client, corrID, ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())
	chain, err := reader.QueryChainByCorrelationID(context.Background(), corrID, "btcusdt")
	if err != nil {
		t.Fatalf("query chain: %v", err)
	}

	if chain.ChainComplete {
		t.Error("expected chain_complete=false for partial chain")
	}
	if chain.StageCount != 4 {
		t.Errorf("stage_count: got %d, want 4", chain.StageCount)
	}
	if chain.Execution != nil {
		t.Error("expected execution=nil for risk-rejected chain")
	}
	if chain.Risk.Disposition != "rejected" {
		t.Errorf("risk.disposition: got %q, want %q", chain.Risk.Disposition, "rejected")
	}
	if len(chain.MissingStages) != 1 || chain.MissingStages[0] != "execution" {
		t.Errorf("missing_stages: got %v, want [execution]", chain.MissingStages)
	}
}

// CRI-5: Batch lookup returns chains ordered by execution timestamp DESC.
func TestCompositeReader_BatchLookup(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts1 := time.Date(2026, 3, 21, 14, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 3, 21, 14, 1, 0, 0, time.UTC)
	insertCompositeFixture(t, client, "s296-batch-001", ts1)
	insertCompositeFixture(t, client, "s296-batch-002", ts2)

	reader := clickhouse.NewCompositeReader(client, slog.Default())
	chains, err := reader.QueryChainsBatch(context.Background(), "binance", "btcusdt", 60, 0, 0, 10)
	if err != nil {
		t.Fatalf("batch query: %v", err)
	}

	if len(chains) < 2 {
		t.Fatalf("expected at least 2 chains, got %d", len(chains))
	}

	// First chain should be the newer one (ts2).
	if chains[0].CorrelationID != "s296-batch-002" {
		t.Errorf("first chain should be s296-batch-002 (newest), got %q", chains[0].CorrelationID)
	}
	if chains[1].CorrelationID != "s296-batch-001" {
		t.Errorf("second chain should be s296-batch-001, got %q", chains[1].CorrelationID)
	}
}

// CRI-6: Missing correlation_id returns empty chain.
func TestCompositeReader_MissingCorrelation(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	reader := clickhouse.NewCompositeReader(client, slog.Default())
	chain, err := reader.QueryChainByCorrelationID(context.Background(), "nonexistent-corr-id", "btcusdt")
	if err != nil {
		t.Fatalf("query chain: %v", err)
	}

	if chain.StageCount != 0 {
		t.Errorf("expected stage_count=0, got %d", chain.StageCount)
	}
	if chain.ChainComplete {
		t.Error("expected chain_complete=false for nonexistent correlation")
	}
}

// ---------------------------------------------------------------------------
// S301: Multi-Symbol Isolation Tests
// ---------------------------------------------------------------------------

// S301-ISO-1: Single-chain lookup with symbol filter returns ONLY the requested symbol's data.
// Inserts chains for btcusdt, ethusdt, solusdt with the SAME correlation_id prefix pattern
// and verifies that querying with symbol=btcusdt never returns ethusdt or solusdt data.
func TestCompositeReader_SymbolIsolation_SingleChain(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 15, 0, 0, 0, time.UTC)
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	for _, sym := range symbols {
		insertCompositeFixtureForSymbol(t, client, "s301-iso-"+sym, sym, ts)
	}

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Query each symbol independently — must get exactly 1 chain with correct symbol.
	for _, sym := range symbols {
		chain, err := reader.QueryChainByCorrelationID(context.Background(), "s301-iso-"+sym, sym)
		if err != nil {
			t.Fatalf("query chain for %s: %v", sym, err)
		}
		if chain.StageCount != 5 {
			t.Errorf("[%s] expected stage_count=5, got %d", sym, chain.StageCount)
		}
		if chain.Signal.Symbol != sym {
			t.Errorf("[%s] signal.symbol=%q, want %q", sym, chain.Signal.Symbol, sym)
		}
		if chain.Decision.Symbol != sym {
			t.Errorf("[%s] decision.symbol=%q, want %q", sym, chain.Decision.Symbol, sym)
		}
		if chain.Strategy.Symbol != sym {
			t.Errorf("[%s] strategy.symbol=%q, want %q", sym, chain.Strategy.Symbol, sym)
		}
		if chain.Risk.Symbol != sym {
			t.Errorf("[%s] risk.symbol=%q, want %q", sym, chain.Risk.Symbol, sym)
		}
		if chain.Execution.Symbol != sym {
			t.Errorf("[%s] execution.symbol=%q, want %q", sym, chain.Execution.Symbol, sym)
		}
	}
}

// S301-ISO-2: Cross-symbol query returns zero results.
// Queries btcusdt's correlation_id but with symbol=ethusdt — must return empty chain.
func TestCompositeReader_SymbolIsolation_CrossSymbolBlocked(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 16, 0, 0, 0, time.UTC)
	insertCompositeFixtureForSymbol(t, client, "s301-cross-btc", "btcusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s301-cross-eth", "ethusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Query btcusdt's correlation_id with wrong symbol — must return 0 stages.
	chain, err := reader.QueryChainByCorrelationID(context.Background(), "s301-cross-btc", "ethusdt")
	if err != nil {
		t.Fatalf("cross-symbol query: %v", err)
	}
	if chain.StageCount != 0 {
		t.Errorf("cross-symbol contamination detected: stage_count=%d, expected 0", chain.StageCount)
	}

	// And vice versa.
	chain, err = reader.QueryChainByCorrelationID(context.Background(), "s301-cross-eth", "btcusdt")
	if err != nil {
		t.Fatalf("cross-symbol query: %v", err)
	}
	if chain.StageCount != 0 {
		t.Errorf("cross-symbol contamination detected: stage_count=%d, expected 0", chain.StageCount)
	}
}

// S301-ISO-3: Batch lookup scopes correctly to requested symbol.
// Inserts chains for 3 symbols, batch-queries for btcusdt only — must return only btcusdt chains.
func TestCompositeReader_SymbolIsolation_BatchScoping(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 17, 0, 0, 0, time.UTC)
	insertCompositeFixtureForSymbol(t, client, "s301-batch-btc-1", "btcusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s301-batch-btc-2", "btcusdt", ts.Add(time.Minute))
	insertCompositeFixtureForSymbol(t, client, "s301-batch-eth-1", "ethusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s301-batch-sol-1", "solusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Batch for btcusdt — must get exactly 2 chains, all btcusdt.
	chains, err := reader.QueryChainsBatch(context.Background(), "binance", "btcusdt", 60, 0, 0, 10)
	if err != nil {
		t.Fatalf("batch query btcusdt: %v", err)
	}
	if len(chains) != 2 {
		t.Fatalf("expected 2 btcusdt chains, got %d", len(chains))
	}
	for i, ch := range chains {
		if ch.Signal != nil && ch.Signal.Symbol != "btcusdt" {
			t.Errorf("chain[%d] signal.symbol=%q, expected btcusdt", i, ch.Signal.Symbol)
		}
		if ch.Execution != nil && ch.Execution.Symbol != "btcusdt" {
			t.Errorf("chain[%d] execution.symbol=%q, expected btcusdt", i, ch.Execution.Symbol)
		}
	}

	// Batch for ethusdt — must get exactly 1 chain.
	chains, err = reader.QueryChainsBatch(context.Background(), "binance", "ethusdt", 60, 0, 0, 10)
	if err != nil {
		t.Fatalf("batch query ethusdt: %v", err)
	}
	if len(chains) != 1 {
		t.Errorf("expected 1 ethusdt chain, got %d", len(chains))
	}

	// Batch for solusdt — must get exactly 1 chain.
	chains, err = reader.QueryChainsBatch(context.Background(), "binance", "solusdt", 60, 0, 0, 10)
	if err != nil {
		t.Fatalf("batch query solusdt: %v", err)
	}
	if len(chains) != 1 {
		t.Errorf("expected 1 solusdt chain, got %d", len(chains))
	}
}

// S301-ISO-4: Funnel counts are symbol-scoped under multi-symbol data.
func TestCompositeReader_SymbolIsolation_Funnel(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 18, 0, 0, 0, time.UTC)
	// Insert 2 btcusdt chains, 1 ethusdt chain.
	insertCompositeFixtureForSymbol(t, client, "s301-funnel-btc-1", "btcusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s301-funnel-btc-2", "btcusdt", ts.Add(time.Minute))
	insertCompositeFixtureForSymbol(t, client, "s301-funnel-eth-1", "ethusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Funnel for btcusdt — each stage should have count=2.
	stages, err := reader.QueryPipelineFunnel(context.Background(), "rsi", "binance", "btcusdt", 60, 0, 0)
	if err != nil {
		t.Fatalf("funnel btcusdt: %v", err)
	}
	for _, s := range stages {
		if s.Count != 2 {
			t.Errorf("btcusdt funnel %s: expected count=2, got %d", s.Stage, s.Count)
		}
	}

	// Funnel for ethusdt — each stage should have count=1.
	stages, err = reader.QueryPipelineFunnel(context.Background(), "rsi", "binance", "ethusdt", 60, 0, 0)
	if err != nil {
		t.Fatalf("funnel ethusdt: %v", err)
	}
	for _, s := range stages {
		if s.Count != 1 {
			t.Errorf("ethusdt funnel %s: expected count=1, got %d", s.Stage, s.Count)
		}
	}
}

// S301-ISO-5: Disposition breakdown is symbol-scoped under multi-symbol data.
func TestCompositeReader_SymbolIsolation_Dispositions(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 19, 0, 0, 0, time.UTC)
	// btcusdt: 1 approved chain.
	insertCompositeFixtureForSymbol(t, client, "s301-disp-btc-1", "btcusdt", ts)
	// ethusdt: 1 approved chain.
	insertCompositeFixtureForSymbol(t, client, "s301-disp-eth-1", "ethusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Disposition for btcusdt — should have 1 approved.
	disps, err := reader.QueryDispositionBreakdown(context.Background(), "position_exposure", "binance", "btcusdt", 60, 0, 0)
	if err != nil {
		t.Fatalf("disposition btcusdt: %v", err)
	}
	if len(disps) != 1 {
		t.Fatalf("expected 1 disposition for btcusdt, got %d", len(disps))
	}
	if disps[0].Disposition != "approved" || disps[0].Count != 1 {
		t.Errorf("btcusdt disposition: got %s:%d, want approved:1", disps[0].Disposition, disps[0].Count)
	}

	// Disposition for ethusdt — should have 1 approved (independent).
	disps, err = reader.QueryDispositionBreakdown(context.Background(), "position_exposure", "binance", "ethusdt", 60, 0, 0)
	if err != nil {
		t.Fatalf("disposition ethusdt: %v", err)
	}
	if len(disps) != 1 {
		t.Fatalf("expected 1 disposition for ethusdt, got %d", len(disps))
	}
	if disps[0].Count != 1 {
		t.Errorf("ethusdt disposition count: got %d, want 1", disps[0].Count)
	}
}

// ---------------------------------------------------------------------------
// S302: Multi-Symbol Deterministic Scenario Pack
// ---------------------------------------------------------------------------

// insertPartialFixtureForSymbol inserts a 4-stage chain (risk=rejected, no execution) for the given symbol.
func insertPartialFixtureForSymbol(t *testing.T, client *clickhouse.Client, corrID, symbol string, ts time.Time) {
	t.Helper()
	ctx := context.Background()

	err := client.InsertBatch(ctx, "INSERT INTO signals", [][]any{{
		"sig-" + corrID, ts, corrID, "",
		"rsi", "binance", symbol, uint32(60),
		28.0, `{"period":"14"}`, true, ts,
	}})
	if err != nil {
		t.Fatalf("insert signal (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO decisions", [][]any{{
		"dec-" + corrID, ts.Add(time.Millisecond), corrID, "sig-" + corrID,
		"rsi_oversold", "binance", symbol, uint32(60),
		"triggered", 0.90, "high", "RSI deeply oversold",
		`[{"type":"rsi","value":"28.0"}]`, `{}`, true, ts.Add(time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert decision (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO strategies", [][]any{{
		"str-" + corrID, ts.Add(2 * time.Millisecond), corrID, "dec-" + corrID,
		"mean_reversion_entry", "binance", symbol, uint32(60),
		"long", 0.85,
		`[{"type":"rsi_oversold","outcome":"triggered"}]`, `{"take_profit":"0.02"}`, `{}`, true, ts.Add(2 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert strategy (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO risk_assessments", [][]any{{
		"rsk-" + corrID, ts.Add(3 * time.Millisecond), corrID, "str-" + corrID,
		"position_exposure", "binance", symbol, uint32(60),
		"rejected", 0.10,
		`[{"type":"mean_reversion_entry","direction":"long"}]`,
		`{"max_position_pct":"0.05","current_exposure":"0.06"}`,
		"drawdown limit exceeded",
		`{}`, `{}`, true, ts.Add(3 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert risk (%s): %v", symbol, err)
	}
}

// insertModifiedFixtureForSymbol inserts a 5-stage chain (risk=modified) for the given symbol.
func insertModifiedFixtureForSymbol(t *testing.T, client *clickhouse.Client, corrID, symbol string, ts time.Time) {
	t.Helper()
	ctx := context.Background()

	err := client.InsertBatch(ctx, "INSERT INTO signals", [][]any{{
		"sig-" + corrID, ts, corrID, "",
		"bollinger", "binance", symbol, uint32(60),
		1.95, `{"period":"20"}`, true, ts,
	}})
	if err != nil {
		t.Fatalf("insert signal (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO decisions", [][]any{{
		"dec-" + corrID, ts.Add(time.Millisecond), corrID, "sig-" + corrID,
		"squeeze_breakout", "binance", symbol, uint32(60),
		"triggered", 0.70, "low", "Squeeze detected",
		`[{"type":"bollinger","value":"1.95"}]`, `{}`, true, ts.Add(time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert decision (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO strategies", [][]any{{
		"str-" + corrID, ts.Add(2 * time.Millisecond), corrID, "dec-" + corrID,
		"squeeze_breakout_entry", "binance", symbol, uint32(60),
		"short", 0.65,
		`[{"type":"squeeze_breakout","outcome":"triggered"}]`, `{"stop_pct":"0.015"}`, `{}`, true, ts.Add(2 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert strategy (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO risk_assessments", [][]any{{
		"rsk-" + corrID, ts.Add(3 * time.Millisecond), corrID, "str-" + corrID,
		"position_exposure", "binance", symbol, uint32(60),
		"modified", 0.60,
		`[{"type":"squeeze_breakout_entry","direction":"short"}]`,
		`{"max_position_pct":"0.03","current_exposure":"0.02"}`,
		"position size reduced to 3%",
		`{}`, `{}`, true, ts.Add(3 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert risk (%s): %v", symbol, err)
	}
	err = client.InsertBatch(ctx, "INSERT INTO executions", [][]any{{
		"exc-" + corrID, ts.Add(4 * time.Millisecond), corrID, "rsk-" + corrID,
		"paper_order", "binance", symbol, uint32(60),
		"sell", 5.0, 0.0, "submitted",
		`{"disposition":"modified","strategy_type":"squeeze_breakout_entry"}`, `[]`,
		`{}`, `{}`,
		corrID, "rsk-" + corrID,
		true, ts.Add(4 * time.Millisecond),
	}})
	if err != nil {
		t.Fatalf("insert execution (%s): %v", symbol, err)
	}
}

// S302-SC1-INT: Three symbols with approved chains — full chain reconstruction per symbol.
func TestCompositeReader_S302_SC1_SimultaneousApproved(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 20, 0, 0, 0, time.UTC)
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}
	for _, sym := range symbols {
		insertCompositeFixtureForSymbol(t, client, "s302-sc1-"+sym, sym, ts)
	}

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	for _, sym := range symbols {
		t.Run("chain_"+sym, func(t *testing.T) {
			chain, err := reader.QueryChainByCorrelationID(context.Background(), "s302-sc1-"+sym, sym)
			if err != nil {
				t.Fatalf("query %s: %v", sym, err)
			}
			if chain.StageCount != 5 {
				t.Errorf("[%s] stage_count=%d, want 5", sym, chain.StageCount)
			}
			if !chain.ChainComplete {
				t.Errorf("[%s] expected chain_complete=true", sym)
			}
			// Verify all stages belong to this symbol.
			if chain.Signal.Symbol != sym {
				t.Errorf("[%s] signal.symbol=%q", sym, chain.Signal.Symbol)
			}
			if chain.Execution.Symbol != sym {
				t.Errorf("[%s] execution.symbol=%q", sym, chain.Execution.Symbol)
			}
			// Verify causal chain integrity.
			if chain.Decision.CausationID != "sig-s302-sc1-"+sym {
				t.Errorf("[%s] decision.causation_id=%q", sym, chain.Decision.CausationID)
			}
			if chain.Execution.EventCausationID != "rsk-s302-sc1-"+sym {
				t.Errorf("[%s] execution.causation_id=%q", sym, chain.Execution.EventCausationID)
			}
		})
	}
}

// S302-SC2-INT: Mixed dispositions across symbols.
// btcusdt=approved(full), ethusdt=rejected(partial), solusdt=modified(full).
func TestCompositeReader_S302_SC2_MixedDispositions(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 21, 0, 0, 0, time.UTC)
	insertCompositeFixtureForSymbol(t, client, "s302-sc2-btcusdt", "btcusdt", ts)
	insertPartialFixtureForSymbol(t, client, "s302-sc2-ethusdt", "ethusdt", ts)
	insertModifiedFixtureForSymbol(t, client, "s302-sc2-solusdt", "solusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// btcusdt: approved, full chain
	t.Run("btcusdt_approved", func(t *testing.T) {
		chain, err := reader.QueryChainByCorrelationID(context.Background(), "s302-sc2-btcusdt", "btcusdt")
		if err != nil {
			t.Fatalf("query btcusdt: %v", err)
		}
		if chain.StageCount != 5 {
			t.Errorf("btcusdt stage_count=%d, want 5", chain.StageCount)
		}
		if chain.Risk.Disposition != "approved" {
			t.Errorf("btcusdt risk.disposition=%q, want approved", chain.Risk.Disposition)
		}
		if chain.Execution == nil {
			t.Error("btcusdt: expected execution stage")
		}
	})

	// ethusdt: rejected, partial chain (no execution)
	t.Run("ethusdt_rejected", func(t *testing.T) {
		chain, err := reader.QueryChainByCorrelationID(context.Background(), "s302-sc2-ethusdt", "ethusdt")
		if err != nil {
			t.Fatalf("query ethusdt: %v", err)
		}
		if chain.StageCount != 4 {
			t.Errorf("ethusdt stage_count=%d, want 4", chain.StageCount)
		}
		if chain.Risk.Disposition != "rejected" {
			t.Errorf("ethusdt risk.disposition=%q, want rejected", chain.Risk.Disposition)
		}
		if chain.Execution != nil {
			t.Error("ethusdt: expected no execution for rejected")
		}
		if len(chain.MissingStages) != 1 || chain.MissingStages[0] != "execution" {
			t.Errorf("ethusdt missing_stages=%v, want [execution]", chain.MissingStages)
		}
	})

	// solusdt: modified, full chain
	t.Run("solusdt_modified", func(t *testing.T) {
		chain, err := reader.QueryChainByCorrelationID(context.Background(), "s302-sc2-solusdt", "solusdt")
		if err != nil {
			t.Fatalf("query solusdt: %v", err)
		}
		if chain.StageCount != 5 {
			t.Errorf("solusdt stage_count=%d, want 5", chain.StageCount)
		}
		if chain.Risk.Disposition != "modified" {
			t.Errorf("solusdt risk.disposition=%q, want modified", chain.Risk.Disposition)
		}
		if chain.Execution == nil {
			t.Fatal("solusdt: expected execution for modified")
		}
		if chain.Execution.Side != "sell" {
			t.Errorf("solusdt execution.side=%q, want sell", chain.Execution.Side)
		}
	})
}

// S302-SC3-INT: Funnel and disposition aggregate independence across symbols.
func TestCompositeReader_S302_SC3_AggregateIndependence(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 22, 0, 0, 0, time.UTC)

	// btcusdt: 2 approved + 1 rejected = 3 chains
	insertCompositeFixtureForSymbol(t, client, "s302-sc3-btc-1", "btcusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s302-sc3-btc-2", "btcusdt", ts.Add(time.Minute))
	insertPartialFixtureForSymbol(t, client, "s302-sc3-btc-3", "btcusdt", ts.Add(2*time.Minute))

	// ethusdt: 1 approved + 1 modified = 2 chains
	insertCompositeFixtureForSymbol(t, client, "s302-sc3-eth-1", "ethusdt", ts)
	insertModifiedFixtureForSymbol(t, client, "s302-sc3-eth-2", "ethusdt", ts.Add(time.Minute))

	// solusdt: 1 approved only
	insertCompositeFixtureForSymbol(t, client, "s302-sc3-sol-1", "solusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	// Funnel independence: btcusdt signals=3, ethusdt signals=2, solusdt signals=1
	t.Run("funnel_btcusdt", func(t *testing.T) {
		stages, err := reader.QueryPipelineFunnel(context.Background(), "rsi", "binance", "btcusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("funnel btcusdt: %v", err)
		}
		for _, s := range stages {
			if s.Stage == "signal" && s.Count != 3 {
				t.Errorf("btcusdt funnel signal: count=%d, want 3", s.Count)
			}
			// btcusdt has 2 executions (2 approved), 1 rejected has no execution
			if s.Stage == "execution" && s.Count != 2 {
				t.Errorf("btcusdt funnel execution: count=%d, want 2", s.Count)
			}
		}
	})

	t.Run("funnel_ethusdt", func(t *testing.T) {
		// ethusdt signals use different types, so query "rsi" for the approved one
		// and "bollinger" for the modified one. For funnel totals we need a type
		// that matches. The approved fixture uses "rsi" type.
		stages, err := reader.QueryPipelineFunnel(context.Background(), "rsi", "binance", "ethusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("funnel ethusdt: %v", err)
		}
		for _, s := range stages {
			if s.Stage == "signal" && s.Count != 1 {
				t.Errorf("ethusdt funnel signal (rsi): count=%d, want 1", s.Count)
			}
		}
	})

	t.Run("funnel_solusdt", func(t *testing.T) {
		stages, err := reader.QueryPipelineFunnel(context.Background(), "rsi", "binance", "solusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("funnel solusdt: %v", err)
		}
		for _, s := range stages {
			if s.Stage == "signal" && s.Count != 1 {
				t.Errorf("solusdt funnel signal: count=%d, want 1", s.Count)
			}
		}
	})

	// Disposition independence
	t.Run("dispositions_btcusdt", func(t *testing.T) {
		disps, err := reader.QueryDispositionBreakdown(context.Background(), "position_exposure", "binance", "btcusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("dispositions btcusdt: %v", err)
		}
		// btcusdt: 2 approved + 1 rejected = 3 total
		total := int64(0)
		for _, d := range disps {
			total += d.Count
		}
		if total != 3 {
			t.Errorf("btcusdt total dispositions=%d, want 3", total)
		}
		// Find approved count
		for _, d := range disps {
			if d.Disposition == "approved" && d.Count != 2 {
				t.Errorf("btcusdt approved=%d, want 2", d.Count)
			}
			if d.Disposition == "rejected" && d.Count != 1 {
				t.Errorf("btcusdt rejected=%d, want 1", d.Count)
			}
		}
	})

	t.Run("dispositions_ethusdt", func(t *testing.T) {
		disps, err := reader.QueryDispositionBreakdown(context.Background(), "position_exposure", "binance", "ethusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("dispositions ethusdt: %v", err)
		}
		total := int64(0)
		for _, d := range disps {
			total += d.Count
		}
		if total != 2 {
			t.Errorf("ethusdt total dispositions=%d, want 2", total)
		}
	})

	t.Run("dispositions_solusdt", func(t *testing.T) {
		disps, err := reader.QueryDispositionBreakdown(context.Background(), "position_exposure", "binance", "solusdt", 60, 0, 0)
		if err != nil {
			t.Fatalf("dispositions solusdt: %v", err)
		}
		if len(disps) != 1 || disps[0].Count != 1 {
			t.Errorf("solusdt dispositions unexpected: %v", disps)
		}
	})
}

// S302-SC4-INT: Batch query returns correct count per symbol with mixed data.
func TestCompositeReader_S302_SC4_BatchCountPerSymbol(t *testing.T) {
	client := skipUnlessClickHouse(t)
	defer client.Close()
	setupAllTables(t, client)

	ts := time.Date(2026, 3, 21, 23, 0, 0, 0, time.UTC)

	// btcusdt: 3 chains at different timestamps
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-btc-1", "btcusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-btc-2", "btcusdt", ts.Add(time.Minute))
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-btc-3", "btcusdt", ts.Add(2*time.Minute))

	// ethusdt: 2 chains
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-eth-1", "ethusdt", ts)
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-eth-2", "ethusdt", ts.Add(time.Minute))

	// solusdt: 1 chain
	insertCompositeFixtureForSymbol(t, client, "s302-sc4-sol-1", "solusdt", ts)

	reader := clickhouse.NewCompositeReader(client, slog.Default())

	expected := map[string]int{"btcusdt": 3, "ethusdt": 2, "solusdt": 1}
	for sym, want := range expected {
		t.Run("batch_"+sym, func(t *testing.T) {
			chains, err := reader.QueryChainsBatch(context.Background(), "binance", sym, 60, 0, 0, 10)
			if err != nil {
				t.Fatalf("batch %s: %v", sym, err)
			}
			if len(chains) != want {
				t.Errorf("[%s] batch count=%d, want %d", sym, len(chains), want)
			}
			// Verify ordering: newest first.
			for i := 1; i < len(chains); i++ {
				if chains[i-1].Execution != nil && chains[i].Execution != nil {
					if chains[i-1].Execution.OccurredAt.Before(chains[i].Execution.OccurredAt) {
						t.Errorf("[%s] chains not ordered DESC: chain[%d] before chain[%d]", sym, i-1, i)
					}
				}
			}
			// All chains belong to the queried symbol.
			for i, ch := range chains {
				if ch.Signal != nil && ch.Signal.Symbol != sym {
					t.Errorf("[%s] chain[%d].signal.symbol=%q", sym, i, ch.Signal.Symbol)
				}
			}
		})
	}
}
