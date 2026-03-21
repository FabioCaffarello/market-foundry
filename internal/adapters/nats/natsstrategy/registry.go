package natsstrategy

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the strategy domain.
type Registry struct {
	MeanReversionEntryResolved    natskit.EventSpec
	MeanReversionEntryLatest      natskit.ControlSpec
	TrendFollowingEntryResolved   natskit.EventSpec
	TrendFollowingEntryLatest     natskit.ControlSpec
	SqueezeBreakoutEntryResolved  natskit.EventSpec
	SqueezeBreakoutEntryLatest    natskit.ControlSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "STRATEGY_EVENTS",
		Subjects: []string{"strategy.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		MeanReversionEntryResolved: natskit.EventSpec{
			Subject: "strategy.events.mean_reversion_entry.resolved",
			Type:    "strategy.events.v1.mean_reversion_entry_resolved",
			Stream:  eventStream,
		},
		MeanReversionEntryLatest: natskit.ControlSpec{
			Subject:     "strategy.query.mean_reversion_entry.latest",
			RequestType: "strategy.query.v1.mean_reversion_entry_latest_request",
			ReplyType:   "strategy.query.v1.mean_reversion_entry_latest_reply",
			QueueGroup:  "strategy.query",
		},
		TrendFollowingEntryResolved: natskit.EventSpec{
			Subject: "strategy.events.trend_following_entry.resolved",
			Type:    "strategy.events.v1.trend_following_entry_resolved",
			Stream:  eventStream,
		},
		TrendFollowingEntryLatest: natskit.ControlSpec{
			Subject:     "strategy.query.trend_following_entry.latest",
			RequestType: "strategy.query.v1.trend_following_entry_latest_request",
			ReplyType:   "strategy.query.v1.trend_following_entry_latest_reply",
			QueueGroup:  "strategy.query",
		},
		SqueezeBreakoutEntryResolved: natskit.EventSpec{
			Subject: "strategy.events.squeeze_breakout_entry.resolved",
			Type:    "strategy.events.v1.squeeze_breakout_entry_resolved",
			Stream:  eventStream,
		},
		SqueezeBreakoutEntryLatest: natskit.ControlSpec{
			Subject:     "strategy.query.squeeze_breakout_entry.latest",
			RequestType: "strategy.query.v1.squeeze_breakout_entry_latest_request",
			ReplyType:   "strategy.query.v1.squeeze_breakout_entry_latest_reply",
			QueueGroup:  "strategy.query",
		},
	}
}

// LatestSpecByType returns the ControlSpec for the strategy type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(strategyType string) (natskit.ControlSpec, bool) {
	switch strategyType {
	case "mean_reversion_entry":
		return r.MeanReversionEntryLatest, true
	case "trend_following_entry":
		return r.TrendFollowingEntryLatest, true
	case "squeeze_breakout_entry":
		return r.SqueezeBreakoutEntryLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// ── Writer Consumer Specs ─────────────────────────────────────────
// Mean reversion entry and trend following entry are codegen-governed (markers below).
// Store consumer specs remain manual:owned.

// codegen:begin consumer_spec family=mean_reversion_entry source=codegen/families/mean_reversion_entry.yaml
// WriterMeanReversionEntryStrategyConsumer defines the durable consumer spec for writer consuming mean reversion entry strategy events.
func WriterMeanReversionEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-strategy-mean-reversion-entry",
		Event: natskit.EventSpec{
			Subject: "strategy.events.mean_reversion_entry.resolved.>",
			Type:    "strategy.events.v1.mean_reversion_entry_resolved",
			Stream: natskit.StreamSpec{
				Name: "STRATEGY_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=mean_reversion_entry

// codegen:begin consumer_spec family=trend_following_entry source=codegen/families/trend_following_entry.yaml
// WriterTrendFollowingEntryStrategyConsumer defines the durable consumer spec for writer consuming
// trend_following_entry strategy events.
func WriterTrendFollowingEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-strategy-trend-following-entry",
		Event: natskit.EventSpec{
			Subject: "strategy.events.trend_following_entry.resolved.>",
			Type:    "strategy.events.v1.trend_following_entry_resolved",
			Stream: natskit.StreamSpec{
				Name: "STRATEGY_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=trend_following_entry

// StoreMeanReversionEntryStrategyConsumer defines the durable consumer spec for store consuming mean reversion entry strategy events.
func StoreMeanReversionEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-strategy-mean-reversion-entry", "strategy.events.mean_reversion_entry.resolved.>", "strategy.events.v1.mean_reversion_entry_resolved", "STRATEGY_EVENTS")
}

// StoreTrendFollowingEntryStrategyConsumer defines the durable consumer spec for store consuming trend following entry strategy events.
func StoreTrendFollowingEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-strategy-trend-following-entry", "strategy.events.trend_following_entry.resolved.>", "strategy.events.v1.trend_following_entry_resolved", "STRATEGY_EVENTS")
}

// WriterSqueezeBreakoutEntryStrategyConsumer defines the durable consumer spec for writer consuming squeeze breakout entry strategy events.
func WriterSqueezeBreakoutEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-strategy-squeeze-breakout-entry",
		Event: natskit.EventSpec{
			Subject: "strategy.events.squeeze_breakout_entry.resolved.>",
			Type:    "strategy.events.v1.squeeze_breakout_entry_resolved",
			Stream: natskit.StreamSpec{
				Name: "STRATEGY_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// StoreSqueezeBreakoutEntryStrategyConsumer defines the durable consumer spec for store consuming squeeze breakout entry strategy events.
func StoreSqueezeBreakoutEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-strategy-squeeze-breakout-entry", "strategy.events.squeeze_breakout_entry.resolved.>", "strategy.events.v1.squeeze_breakout_entry_resolved", "STRATEGY_EVENTS")
}
