package execute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	appexec "internal/application/execution"
	"internal/application/ports"
	adapternats "internal/adapters/nats"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// VenueAdapterConfig holds the configuration for the venue adapter actor.
type VenueAdapterConfig struct {
	NATSURL          string
	Source           string
	Registry         adapternats.ExecutionRegistry
	Venue            ports.VenuePort
	StalenessMaxAge  time.Duration
	SubmitTimeout    time.Duration
	Tracker          *healthz.Tracker
}

// VenueAdapterActor consumes execution intents, checks kill switch + staleness,
// calls VenuePort.SubmitOrder, and publishes fill events.
type VenueAdapterActor struct {
	cfg           VenueAdapterConfig
	logger        *slog.Logger
	controlStore  *adapternats.ExecutionControlKVStore
	fillPublisher *adapternats.ExecutionPublisher
	staleness     *appexec.StalenessGuard
}

func NewVenueAdapterActor(cfg VenueAdapterConfig) actor.Producer {
	return func() actor.Receiver {
		return &VenueAdapterActor{cfg: cfg}
	}
}

func (a *VenueAdapterActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "venue-adapter")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		a.start(c)

	case actor.Stopped:
		a.logStats()
		if a.controlStore != nil {
			_ = a.controlStore.Close()
		}
		if a.fillPublisher != nil {
			_ = a.fillPublisher.Close()
		}

	case intentReceivedMessage:
		a.onIntent(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

func (a *VenueAdapterActor) start(c *actor.Context) {
	a.staleness = appexec.NewStalenessGuard(a.cfg.StalenessMaxAge)

	// Connect to execution control KV for kill switch.
	controlStore := adapternats.NewExecutionControlKVStore(a.cfg.NATSURL)
	if err := controlStore.Start(); err != nil {
		a.logger.Warn("execution control KV store unavailable — gate check disabled", "error", err)
	} else {
		a.controlStore = controlStore
	}

	// Connect fill publisher for publishing fill events.
	fillPub := adapternats.NewExecutionPublisher(a.cfg.NATSURL, a.cfg.Source, a.cfg.Registry)
	if err := fillPub.Start(); err != nil {
		a.logger.Error("start fill publisher", "error", err)
		c.Engine().Poison(c.PID())
		return
	}
	a.fillPublisher = fillPub

	a.logger.Info("venue adapter started",
		"staleness_max_age", a.cfg.StalenessMaxAge.String(),
		"submit_timeout", a.cfg.SubmitTimeout.String(),
		"control_gate", a.controlStore != nil,
	)
}

func (a *VenueAdapterActor) onIntent(msg intentReceivedMessage) {
	tracker := a.cfg.Tracker
	if tracker != nil {
		tracker.Counter("processed").Add(1)
	}
	intent := msg.Event.ExecutionIntent

	// Gate 1: Kill switch check.
	if a.controlStore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		halted := a.controlStore.IsHalted(ctx)
		cancel()
		if halted {
			if tracker != nil {
				tracker.Counter("skipped_halt").Add(1)
			}
			a.logger.Warn("intent blocked by kill switch",
				"source", intent.Source,
				"symbol", intent.Symbol,
				"timeframe", intent.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
			return
		}
	}

	// Gate 2: Staleness guard.
	if a.staleness.IsStale(intent.Timestamp, time.Now().UTC()) {
		if tracker != nil {
			tracker.Counter("skipped_stale").Add(1)
		}
		a.logger.Warn("intent stale — skipped",
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"age", time.Since(intent.Timestamp).String(),
			"max_age", a.cfg.StalenessMaxAge.String(),
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	// Gate 3: Submit to venue adapter with configurable timeout.
	submitTimeout := a.cfg.SubmitTimeout
	if submitTimeout == 0 {
		submitTimeout = 10 * time.Second
	}
	submitCtx, submitCancel := context.WithTimeout(context.Background(), submitTimeout)
	defer submitCancel()

	receipt, prob := a.cfg.Venue.SubmitOrder(
		submitCtx,
		ports.VenueOrderRequest{Intent: intent},
	)
	if prob != nil {
		if tracker != nil {
			tracker.RecordError()
		}
		a.logger.Error("venue submit failed",
			"error", prob.Message,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	// Publish fill event.
	fillEvent := domainexec.VenueOrderFilledEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(msg.Event.Metadata.CorrelationID).
			WithCausationID(msg.Event.Metadata.ID),
		ExecutionIntent: receipt.Intent,
		VenueOrderID:    receipt.VenueOrderID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if pubProb := a.fillPublisher.PublishFill(ctx, fillEvent); pubProb != nil {
		if tracker != nil {
			tracker.RecordError()
		}
		a.logger.Error("publish fill event failed",
			"error", pubProb.Message,
			"venue_order_id", receipt.VenueOrderID,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	if tracker != nil {
		tracker.RecordEvent()
		tracker.Counter("filled").Add(1)
	}

	a.logger.Info("venue order filled",
		"venue_order_id", receipt.VenueOrderID,
		"status", string(receipt.Status),
		"source", intent.Source,
		"symbol", intent.Symbol,
		"timeframe", intent.Timeframe,
		"side", string(intent.Side),
		"quantity", intent.Quantity,
		"filled_quantity", receipt.Intent.FilledQuantity,
		"correlation_id", msg.Event.Metadata.CorrelationID,
	)
}

func (a *VenueAdapterActor) logStats() {
	tracker := a.cfg.Tracker
	if tracker == nil {
		return
	}
	a.logger.Info("venue adapter stats",
		"processed", tracker.Counter("processed").Load(),
		"filled", tracker.Counter("filled").Load(),
		"skipped_stale", tracker.Counter("skipped_stale").Load(),
		"skipped_halt", tracker.Counter("skipped_halt").Load(),
		"errors", tracker.ErrorCount(),
	)
}
