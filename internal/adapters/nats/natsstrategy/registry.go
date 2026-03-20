package natsstrategy

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry defines the NATS subject and stream contracts for the strategy domain.
type Registry struct {
	MeanReversionEntryResolved natskit.EventSpec
	MeanReversionEntryLatest   natskit.ControlSpec
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "STRATEGY_EVENTS",
		Subjects: []string{"strategy.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 2 * 1024 * 1024 * 1024,
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
	}
}

// LatestSpecByType returns the ControlSpec for the strategy type's latest query.
// Returns false if the type is not registered.
func (r Registry) LatestSpecByType(strategyType string) (natskit.ControlSpec, bool) {
	switch strategyType {
	case "mean_reversion_entry":
		return r.MeanReversionEntryLatest, true
	default:
		return natskit.ControlSpec{}, false
	}
}

// WriterMeanReversionEntryStrategyConsumer defines the durable consumer spec for writer consuming mean reversion entry strategy events.
func WriterMeanReversionEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("writer-strategy-mean-reversion-entry", "strategy.events.mean_reversion_entry.resolved.>", "strategy.events.v1.mean_reversion_entry_resolved", "STRATEGY_EVENTS")
}

// StoreMeanReversionEntryStrategyConsumer defines the durable consumer spec for store consuming mean reversion entry strategy events.
func StoreMeanReversionEntryStrategyConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec("store-strategy-mean-reversion-entry", "strategy.events.mean_reversion_entry.resolved.>", "strategy.events.v1.mean_reversion_entry_resolved", "STRATEGY_EVENTS")
}
