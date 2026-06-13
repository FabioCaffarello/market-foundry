package writerpipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	natsdecision "internal/adapters/nats/natsdecision"
	natsevidence "internal/adapters/nats/natsevidence"
	natsexecution "internal/adapters/nats/natsexecution"
	natsinsights "internal/adapters/nats/natsinsights"
	natskit "internal/adapters/nats/natskit"
	natsrisk "internal/adapters/nats/natsrisk"
	natssignal "internal/adapters/nats/natssignal"
	natsstrategy "internal/adapters/nats/natsstrategy"
	"internal/domain/decision"
	"internal/domain/evidence"
	"internal/domain/execution"
	"internal/domain/insights"
	"internal/domain/risk"
	"internal/domain/signal"
	"internal/domain/strategy"
	"internal/shared/healthz"
)

type RowEmitter func([]any)

type ConsumerStarter func(
	natsURL string,
	spec natskit.ConsumerSpec,
	emitRow RowEmitter,
	tracker *healthz.Tracker,
	logger *slog.Logger,
) (io.Closer, error)

func NewCandleStarter(reg natsevidence.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsevidence.NewCandleConsumer(natsURL, spec, reg,
			func(event evidence.CandleSampledEvent) {
				recordEvent(tracker)
				emitRow(mapCandleRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func NewSignalStarter(reg natssignal.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natssignal.NewConsumer(natsURL, spec, reg,
			func(event signal.SignalGeneratedEvent) {
				recordEvent(tracker)
				emitRow(mapSignalRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func NewDecisionStarter(reg natsdecision.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsdecision.NewConsumer(natsURL, spec, reg,
			func(event decision.DecisionEvaluatedEvent) {
				recordEvent(tracker)
				emitRow(mapDecisionRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func NewStrategyStarter(reg natsstrategy.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsstrategy.NewConsumer(natsURL, spec, reg,
			func(event strategy.StrategyResolvedEvent) {
				recordEvent(tracker)
				emitRow(mapStrategyRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func NewRiskStarter(reg natsrisk.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsrisk.NewConsumer(natsURL, spec, reg,
			func(event risk.RiskAssessedEvent) {
				recordEvent(tracker)
				emitRow(mapRiskRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func NewExecutionStarter(reg natsexecution.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsexecution.NewConsumer(natsURL, spec, reg,
			func(event execution.PaperOrderSubmittedEvent) {
				recordEvent(tracker)
				emitRow(mapExecutionRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

// NewVenueFillStarter creates a ConsumerStarter for venue order fill events.
// S317: closes the persistence round-trip gap — venue fills flow from
// EXECUTION_FILL_EVENTS stream to the executions ClickHouse table.
func NewVenueFillStarter(reg natsexecution.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsexecution.NewFillConsumer(natsURL, spec, reg,
			func(event execution.VenueOrderFilledEvent) {
				recordEvent(tracker)
				emitRow(mapVenueFillRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

// NewVenueRejectionStarter creates a ConsumerStarter for venue order rejection events.
// S411: closes the ClickHouse persistence gap (RG-1) — venue rejections flow from
// EXECUTION_REJECTION_EVENTS stream to the executions ClickHouse table.
// Rejection-specific fields (code, reason, venue details) are embedded into the
// metadata JSON column, matching the pattern used by the KV projection actor.
func NewVenueRejectionStarter(reg natsexecution.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsexecution.NewRejectionConsumer(natsURL, spec, reg,
			func(event execution.VenueOrderRejectedEvent) {
				recordEvent(tracker)
				emitRow(mapVenueRejectionRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

// NewVolumeProfileStarter creates a ConsumerStarter for insights volume
// profile events (H-8.a.1). Insights is a family-specific codegen layer
// (like evidence): the VolumeProfile is its own event type, so this
// starter binds the natsinsights volume-profile consumer to the writer's
// single-row emitter. Single-writer (ADR-0008): the writer owns the
// insights_volume_profile table; the store side owns the KV-latest bucket.
func NewVolumeProfileStarter(reg natsinsights.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsinsights.NewVolumeProfileConsumer(natsURL, spec, reg,
			func(event insights.VolumeProfileSampledEvent) {
				recordEvent(tracker)
				emitRow(mapVolumeProfileRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

// NewTPOStarter creates a ConsumerStarter for insights TPO events
// (H-8.b.1). Family-specific insights layer (like volume_profile);
// reuses the natsinsights TPO consumer with the writer-tpo spec.
func NewTPOStarter(reg natsinsights.Registry) ConsumerStarter {
	return func(
		natsURL string,
		spec natskit.ConsumerSpec,
		emitRow RowEmitter,
		tracker *healthz.Tracker,
		logger *slog.Logger,
	) (io.Closer, error) {
		consumer := natsinsights.NewTPOConsumer(natsURL, spec, reg,
			func(event insights.TPOProfileSampledEvent) {
				recordEvent(tracker)
				emitRow(mapTPOProfileRow(event))
			},
			logger,
		)
		return consumer, consumer.Start()
	}
}

func recordEvent(tracker *healthz.Tracker) {
	if tracker == nil {
		return
	}
	tracker.RecordEvent()
	tracker.Counter("events_received").Add(1)
}

// mapCandleRow maps a CandleSampledEvent to ClickHouse evidence_candles row values.
// Column order matches DDL: event_id, occurred_at, correlation_id, causation_id,
// source, symbol, base, quote, contract, timeframe, open, high, low, close, volume,
// trade_count, open_time, close_time, final.
//
// H-6.d.1 commit 2: base/quote/contract are sourced from the canonical
// Instrument (already migrated in H-6.b); the legacy symbol column is
// preserved alongside as the venue-native display string via VenueSymbol().
func mapCandleRow(e evidence.CandleSampledEvent) []any {
	m := e.Metadata
	c := e.Candle
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		c.Source,
		c.VenueSymbol(),
		string(c.Instrument.Base),
		string(c.Instrument.Quote),
		string(c.Instrument.Contract),
		uint32(c.Timeframe),
		parseFloat(c.Open),
		parseFloat(c.High),
		parseFloat(c.Low),
		parseFloat(c.Close),
		parseFloat(c.Volume),
		c.TradeCount,
		c.OpenTime,
		c.CloseTime,
		c.Final,
	}
}

// mapSignalRow maps a SignalGeneratedEvent to ClickHouse signals row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, base, quote, contract, timeframe, value, metadata,
// final, timestamp.
//
// H-6.d.1 commit 2: see mapCandleRow for the canonical-column rationale.
func mapSignalRow(e signal.SignalGeneratedEvent) []any {
	m := e.Metadata
	s := e.Signal
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		s.Type,
		s.Source,
		s.VenueSymbol(),
		string(s.Instrument.Base),
		string(s.Instrument.Quote),
		string(s.Instrument.Contract),
		uint32(s.Timeframe),
		parseFloat(s.Value),
		marshalJSON(s.Metadata),
		s.Final,
		s.Timestamp,
	}
}

// mapDecisionRow maps a DecisionEvaluatedEvent to ClickHouse decisions row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, base, quote, contract, timeframe, outcome, confidence,
// severity, rationale, signals, metadata, final, timestamp.
//
// H-6.d.1 commit 2: see mapCandleRow for the canonical-column rationale.
func mapDecisionRow(e decision.DecisionEvaluatedEvent) []any {
	m := e.Metadata
	d := e.Decision
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		d.Type,
		d.Source,
		d.VenueSymbol(),
		string(d.Instrument.Base),
		string(d.Instrument.Quote),
		string(d.Instrument.Contract),
		uint32(d.Timeframe),
		string(d.Outcome),
		parseFloat(d.Confidence),
		string(d.Severity),
		d.Rationale,
		marshalJSON(d.Signals),
		marshalJSON(d.Metadata),
		d.Final,
		d.Timestamp,
	}
}

// mapStrategyRow maps a StrategyResolvedEvent to ClickHouse strategies row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, base, quote, contract, timeframe, direction, confidence,
// decisions, parameters, metadata, final, timestamp.
//
// H-6.d.1 commit 2: see mapCandleRow for the canonical-column rationale.
func mapStrategyRow(e strategy.StrategyResolvedEvent) []any {
	m := e.Metadata
	s := e.Strategy
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		s.Type,
		s.Source,
		s.VenueSymbol(),
		string(s.Instrument.Base),
		string(s.Instrument.Quote),
		string(s.Instrument.Contract),
		uint32(s.Timeframe),
		string(s.Direction),
		parseFloat(s.Confidence),
		marshalJSON(s.Decisions),
		marshalJSON(s.Parameters),
		marshalJSON(s.Metadata),
		s.Final,
		s.Timestamp,
	}
}

// mapRiskRow maps a RiskAssessedEvent to ClickHouse risk_assessments row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, base, quote, contract, timeframe, disposition, confidence,
// strategies, constraints, rationale, parameters, metadata, final, timestamp.
//
// H-6.d.1 commit 2: see mapCandleRow for the canonical-column rationale.
func mapRiskRow(e risk.RiskAssessedEvent) []any {
	m := e.Metadata
	r := e.RiskAssessment
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		r.Type,
		r.Source,
		r.VenueSymbol(),
		string(r.Instrument.Base),
		string(r.Instrument.Quote),
		string(r.Instrument.Contract),
		uint32(r.Timeframe),
		string(r.Disposition),
		parseFloat(r.Confidence),
		marshalJSON(r.Strategies),
		marshalJSON(r.Constraints),
		r.Rationale,
		marshalJSON(r.Parameters),
		marshalJSON(r.Metadata),
		r.Final,
		r.Timestamp,
	}
}

// mapExecutionRow maps a PaperOrderSubmittedEvent to ClickHouse executions row values.
// Column order: event_id, occurred_at, correlation_id, causation_id,
// type, source, symbol, base, quote, contract, timeframe, side, quantity,
// filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id,
// exec_causation_id, final, timestamp.
//
// H-6.d.1 commit 2: see mapCandleRow for the canonical-column rationale.
func mapExecutionRow(e execution.PaperOrderSubmittedEvent) []any {
	m := e.Metadata
	x := e.ExecutionIntent
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		x.Type,
		x.Source,
		x.VenueSymbol(),
		string(x.Instrument.Base),
		string(x.Instrument.Quote),
		string(x.Instrument.Contract),
		uint32(x.Timeframe),
		string(x.Side),
		parseFloat(x.Quantity),
		parseFloat(x.FilledQuantity),
		string(x.Status),
		marshalJSON(x.Risk),
		marshalJSON(x.Fills),
		marshalJSON(x.Parameters),
		marshalJSON(x.Metadata),
		x.CorrelationID,
		x.CausationID,
		x.Final,
		x.Timestamp,
	}
}

// mapVenueFillRow maps a VenueOrderFilledEvent to ClickHouse executions row values.
// Uses the same column order as mapExecutionRow — both event types target the same table.
// The execution_intent inside the fill event carries the updated state (status=filled,
// real fills, filled_quantity) from the venue adapter.
func mapVenueFillRow(e execution.VenueOrderFilledEvent) []any {
	m := e.Metadata
	x := e.ExecutionIntent
	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		x.Type,
		x.Source,
		x.VenueSymbol(),
		string(x.Instrument.Base),
		string(x.Instrument.Quote),
		string(x.Instrument.Contract),
		uint32(x.Timeframe),
		string(x.Side),
		parseFloat(x.Quantity),
		parseFloat(x.FilledQuantity),
		string(x.Status),
		marshalJSON(x.Risk),
		marshalJSON(x.Fills),
		marshalJSON(x.Parameters),
		marshalJSON(x.Metadata),
		x.CorrelationID,
		x.CausationID,
		x.Final,
		x.Timestamp,
	}
}

// mapVenueRejectionRow maps a VenueOrderRejectedEvent to ClickHouse executions row values.
// Uses the same column order as mapExecutionRow — all execution lifecycle events target the same table.
// S411: Rejection-specific fields (rejection_code, rejection_reason, venue details) are merged
// into the intent's Metadata map before serialization, so they survive the ClickHouse round-trip
// and are queryable via the metadata JSON column. This mirrors the KV projection approach from S407.
func mapVenueRejectionRow(e execution.VenueOrderRejectedEvent) []any {
	m := e.Metadata
	x := e.ExecutionIntent

	// Embed rejection audit fields into metadata for ClickHouse persistence.
	enrichedMeta := make(map[string]string, len(x.Metadata)+3)
	for k, v := range x.Metadata {
		enrichedMeta[k] = v
	}
	if e.RejectionCode != "" {
		enrichedMeta["rejection_code"] = e.RejectionCode
	}
	if e.RejectionReason != "" {
		enrichedMeta["rejection_reason"] = e.RejectionReason
	}
	for k, v := range e.VenueDetails {
		enrichedMeta["venue_detail."+k] = fmt.Sprintf("%v", v)
	}

	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		x.Type,
		x.Source,
		x.VenueSymbol(),
		string(x.Instrument.Base),
		string(x.Instrument.Quote),
		string(x.Instrument.Contract),
		uint32(x.Timeframe),
		string(x.Side),
		parseFloat(x.Quantity),
		parseFloat(x.FilledQuantity),
		string(x.Status),
		marshalJSON(x.Risk),
		marshalJSON(x.Fills),
		marshalJSON(x.Parameters),
		marshalJSON(enrichedMeta),
		x.CorrelationID,
		x.CausationID,
		x.Final,
		x.Timestamp,
	}
}

// mapVolumeProfileRow maps an insights volume profile event to a single
// ClickHouse row (H-8.a.1). The per-window price buckets are emitted as
// three PARALLEL, index-aligned Array(String) slices
// (bucket_price_level[i] ↔ bucket_buy_volume[i] ↔ bucket_sell_volume[i]),
// preserving the codegen 1-event→1-row contract. Decimal values stay as
// String to keep the big.Rat-deterministic binning exact. Column order
// matches the insertSQL in cmd/writer/pipeline.go.
func mapVolumeProfileRow(e insights.VolumeProfileSampledEvent) []any {
	m := e.Metadata
	vp := e.VolumeProfile

	priceLevels := make([]string, len(vp.Buckets))
	buyVolumes := make([]string, len(vp.Buckets))
	sellVolumes := make([]string, len(vp.Buckets))
	for i, b := range vp.Buckets {
		priceLevels[i] = b.PriceLevel
		buyVolumes[i] = b.BuyVolume
		sellVolumes[i] = b.SellVolume
	}

	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		vp.Source,
		vp.VenueSymbol(),
		string(vp.Instrument.Base),
		string(vp.Instrument.Quote),
		string(vp.Instrument.Contract),
		uint32(vp.Timeframe),
		vp.BucketSize,
		priceLevels,
		buyVolumes,
		sellVolumes,
		vp.TradeCount,
		vp.Overload.Label(),
		vp.OpenTime,
		vp.CloseTime,
		vp.Final,
	}
}

// mapTPOProfileRow maps an insights TPO event to a single ClickHouse row
// (H-8.b.1). Periods and price levels become two sets of PARALLEL,
// index-aligned arrays (period_letter/high/low; level_price/letters/count),
// preserving the codegen 1-event→1-row contract. Decimals stay as String;
// level counts are int32. Column order matches the insertSQL in
// cmd/writer/pipeline.go.
func mapTPOProfileRow(e insights.TPOProfileSampledEvent) []any {
	m := e.Metadata
	tp := e.TPOProfile

	periodLetters := make([]string, len(tp.Periods))
	periodHighs := make([]string, len(tp.Periods))
	periodLows := make([]string, len(tp.Periods))
	for i, p := range tp.Periods {
		periodLetters[i] = p.Letter
		periodHighs[i] = p.HighPrice
		periodLows[i] = p.LowPrice
	}

	levelPrices := make([]string, len(tp.Levels))
	levelLetters := make([]string, len(tp.Levels))
	levelCounts := make([]int32, len(tp.Levels))
	for i, l := range tp.Levels {
		levelPrices[i] = l.PriceLevel
		levelLetters[i] = l.Letters
		levelCounts[i] = int32(l.Count)
	}

	return []any{
		m.ID,
		m.OccurredAt,
		m.CorrelationID,
		m.CausationID,
		tp.Source,
		tp.VenueSymbol(),
		string(tp.Instrument.Base),
		string(tp.Instrument.Quote),
		string(tp.Instrument.Contract),
		uint32(tp.Timeframe),
		tp.BucketSize,
		uint32(tp.PeriodSeconds),
		periodLetters,
		periodHighs,
		periodLows,
		levelPrices,
		levelLetters,
		levelCounts,
		tp.TradeCount,
		tp.Overload.Label(),
		tp.POCPrice,
		tp.ValueAreaHigh,
		tp.ValueAreaLow,
		tp.InitialBalanceHigh,
		tp.InitialBalanceLow,
		tp.RangeHigh,
		tp.RangeLow,
		tp.OpenTime,
		tp.CloseTime,
		tp.Final,
	}
}

func parseFloat(s string) float64 {
	if s == "" {
		slog.Warn("parseFloat: empty string, defaulting to 0")
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		slog.Warn("parseFloat: fallback to 0", "input", s, "error", err)
		return 0
	}
	return f
}

func marshalJSON(v any) string {
	if v == nil {
		return "{}"
	}
	data, err := json.Marshal(v)
	if err != nil {
		slog.Warn("marshalJSON: fallback to empty object", "type", fmt.Sprintf("%T", v), "error", err)
		return "{}"
	}
	return string(data)
}
