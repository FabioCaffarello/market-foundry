package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"internal/domain/evidence"
	"internal/domain/instrument"
)

// CandleReader queries evidence_candles from ClickHouse and maps rows back to
// domain types. It implements analyticalclient.CandleReader — the analytical
// read-path counterpart to cmd/writer/mappers.go on the write path.
//
// Ownership: adapter layer. This reader owns the storage↔domain translation
// for candle reads. The gateway consumes it through the CandleReader interface
// defined in internal/application/analyticalclient.
type CandleReader struct {
	client *Client
	logger *slog.Logger
}

// NewCandleReader creates a CandleReader backed by the given ClickHouse client.
func NewCandleReader(client *Client, logger *slog.Logger) *CandleReader {
	if logger == nil {
		logger = slog.Default()
	}
	return &CandleReader{client: client, logger: logger.With("component", "candle_reader")}
}

// QueryCandleHistory queries evidence_candles from ClickHouse with filters.
// Results are ordered newest-first (DESC by open_time).
func (r *CandleReader) QueryCandleHistory(ctx context.Context, source, symbol string, timeframe int, since, until int64, limit int) ([]evidence.EvidenceCandle, error) {
	query, args := BuildCandleQuery(source, symbol, timeframe, since, until, limit)

	start := time.Now()
	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		elapsed := time.Since(start)
		r.logger.Error("query failed",
			"source", source, "symbol", symbol, "timeframe", timeframe,
			"elapsed_ms", elapsed.Milliseconds(), "error", err,
		)
		return nil, fmt.Errorf("query candle history: %w", err)
	}
	defer rows.Close()

	var candles []evidence.EvidenceCandle
	for rows.Next() {
		var (
			src       string
			sym       string
			tf        uint32
			open      float64
			high      float64
			low       float64
			close     float64
			volume    float64
			tradeCnt  int64
			openTime  time.Time
			closeTime time.Time
			final     bool
		)

		if err := rows.Scan(&src, &sym, &tf, &open, &high, &low, &close, &volume, &tradeCnt, &openTime, &closeTime, &final); err != nil {
			r.logger.Error("scan failed",
				"source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
			)
			return nil, fmt.Errorf("scan candle row: %w", err)
		}

		inst, instErr := reconstructInstrumentFromLegacy(src, sym)
		if instErr != nil {
			r.logger.Warn("instrument reconstruction failed; emitting zero instrument",
				"source", src, "symbol", sym, "error", instErr,
			)
		}
		candles = append(candles, evidence.EvidenceCandle{
			Source:     src,
			Instrument: inst,
			Timeframe:  int(tf),
			Open:       FormatFloat(open),
			High:       FormatFloat(high),
			Low:        FormatFloat(low),
			Close:      FormatFloat(close),
			Volume:     FormatFloat(volume),
			TradeCount: tradeCnt,
			OpenTime:   openTime,
			CloseTime:  closeTime,
			Final:      final,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("row iteration failed",
			"source", source, "symbol", symbol, "timeframe", timeframe, "error", err,
		)
		return nil, fmt.Errorf("iterate candle rows: %w", err)
	}

	elapsed := time.Since(start)
	r.logger.Debug("query completed",
		"source", source, "symbol", symbol, "timeframe", timeframe,
		"rows", len(candles), "elapsed_ms", elapsed.Milliseconds(),
	)

	return candles, nil
}

// BuildCandleQuery constructs a parameterized SELECT for evidence_candles.
// Exported for testing without requiring a live ClickHouse connection.
func BuildCandleQuery(source, symbol string, timeframe int, since, until int64, limit int) (string, []any) {
	return BuildQuery(
		"source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final",
		"evidence_candles",
		"source = ? AND symbol = ? AND timeframe = ?",
		[]any{source, symbol, uint32(timeframe)},
		nil,
		"open_time", since, until, "open_time", limit,
	)
}

// FormatFloat converts a float64 to a decimal string, preserving reasonable precision.
// Exported for consistency between write-path mappers and read-path readers.
func FormatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// reconstructInstrumentFromLegacy maps a legacy `(source, venue_native_symbol)`
// row pair to a CanonicalInstrument. The current ClickHouse schema stores only
// venue-native lowercase symbols (e.g. "btcusdt"); the source column carries the
// venue identity ("binances" → spot; "binancef" → perpetual). This helper is
// the storage→domain boundary translator until H-6.d (criterion #4b) adds
// canonical columns to the schema, at which point the reader will read
// `base`/`quote`/`contract` directly and this helper is removed.
//
// TRANSITORY (H-6.b–H-6.d).
//
// Known limitations:
//   - Lossy for delivery futures: `binancef` rows currently cannot tell us
//     whether a contract is `usdtfutures` (with expiry) or `perpetual`. The
//     helper returns `perpetual` for all binancef rows; delivery contracts
//     are not in the current routing path.
//   - Non-USDT quotes are rejected: the H-6.a Binance adapters reject non-USDT
//     symbols at ingest, so the column is expected to always end in "usdt".
func reconstructInstrumentFromLegacy(source, venueNative string) (instrument.CanonicalInstrument, error) {
	upper := strings.ToUpper(strings.TrimSpace(venueNative))
	if upper == "" {
		return instrument.CanonicalInstrument{}, fmt.Errorf("empty venue-native symbol for source %q", source)
	}
	const quote = "USDT"
	if !strings.HasSuffix(upper, quote) || len(upper) <= len(quote) {
		return instrument.CanonicalInstrument{}, fmt.Errorf("symbol %q does not end in USDT (current routing path only supports USDT-quoted instruments)", venueNative)
	}
	base := upper[:len(upper)-len(quote)]

	var contract instrument.ContractType
	switch source {
	case "binances":
		contract = instrument.ContractSpot
	case "binancef":
		contract = instrument.ContractPerpetual
	default:
		return instrument.CanonicalInstrument{}, fmt.Errorf("unknown source %q (expected binances or binancef in H-6.b routing path)", source)
	}

	inst, prob := instrument.New(base, quote, contract)
	if prob != nil {
		return instrument.CanonicalInstrument{}, fmt.Errorf("build canonical instrument: %s", prob.Message)
	}
	return inst, nil
}
