package derive

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	actorcommon "internal/actors/common"
	natsexecution "internal/adapters/nats/natsexecution"
	"internal/shared/healthz"
	"internal/shared/problem"

	"github.com/anthdm/hollywood/actor"
)

// ExecutionPublisherConfig holds the configuration for the execution publisher actor.
type ExecutionPublisherConfig struct {
	URL      string
	Source   string
	Registry natsexecution.Registry
	Tracker  *healthz.Tracker
}

// ExecutionPublisherActor owns the NATS JetStream connection for publishing execution events.
// It checks the execution control gate before every publish — halted gate blocks publishing.
type ExecutionPublisherActor struct {
	cfg          ExecutionPublisherConfig
	logger       *slog.Logger
	publisher    *natsexecution.Publisher
	controlStore *natsexecution.ControlKVStore
	published    atomic.Int64
	errors       atomic.Int64
	halted       atomic.Int64
}

func NewExecutionPublisherActor(cfg ExecutionPublisherConfig) actor.Producer {
	return func() actor.Receiver {
		return &ExecutionPublisherActor{cfg: cfg}
	}
}

func (a *ExecutionPublisherActor) Receive(c *actor.Context) {
	if a.logger == nil {
		a.logger = slog.Default().With("actor", "execution-publisher")
	}

	switch msg := c.Message().(type) {
	case actor.Started:
		pub := natsexecution.NewPublisher(a.cfg.URL, a.cfg.Source, a.cfg.Registry)
		if err := pub.Start(); err != nil {
			a.logger.Error("start execution publisher", "error", err)
			c.Engine().Poison(c.PID())
			return
		}
		a.publisher = pub

		controlStore := natsexecution.NewControlKVStore(a.cfg.URL)
		if err := controlStore.Start(); err != nil {
			a.logger.Warn("execution control KV store unavailable — gate check disabled", "error", err)
		} else {
			a.controlStore = controlStore
		}

		a.logger.Info("execution publisher started",
			"stream", a.cfg.Registry.PaperOrderSubmitted.Stream.Name,
			"control_gate", a.controlStore != nil,
		)

	case actor.Stopped:
		a.logger.Info("execution publisher stopping",
			"published", a.published.Load(),
			"errors", a.errors.Load(),
			"halted", a.halted.Load(),
		)
		if a.controlStore != nil {
			if err := a.controlStore.Close(); err != nil {
				a.logger.Error("close execution control KV store", "error", err)
			}
		}
		if a.publisher != nil {
			if err := a.publisher.Close(); err != nil {
				a.logger.Error("close execution publisher", "error", err)
			}
		}

	case publishExecutionMessage:
		a.publishWithRetry(msg)

	default:
		if actorcommon.ShouldIgnoreLifecycleMessage(msg) {
			return
		}
		a.logger.Warn("unknown message", "type", fmt.Sprintf("%T", msg))
	}
}

// publishWithRetry attempts to publish an execution event with a single retry
// for transient (Unavailable) failures. Non-retryable errors fail immediately.
func (a *ExecutionPublisherActor) publishWithRetry(msg publishExecutionMessage) {
	const maxAttempts = 2
	const retryDelay = 500 * time.Millisecond

	// Gate check: block publishing if execution is halted.
	if a.controlStore != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		halted := a.controlStore.IsHalted(ctx)
		cancel()
		if halted {
			a.halted.Add(1)
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.Counter("execution:gate_halted").Add(1)
			}
			a.logger.Warn("execution publish blocked by control gate",
				"gate_status", "halted",
				"type", msg.Event.ExecutionIntent.Type,
				"source", msg.Event.ExecutionIntent.Source,
				"symbol", msg.Event.ExecutionIntent.Symbol,
				"timeframe", msg.Event.ExecutionIntent.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
			return
		}
	}

	intent := msg.Event.ExecutionIntent
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		prob := a.publisher.PublishExecution(ctx, msg.Event)
		cancel()

		if prob == nil {
			a.published.Add(1)
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordEvent()
				a.cfg.Tracker.Counter("execution:" + intent.Type + ":" + string(intent.Side)).Add(1)
				a.cfg.Tracker.Counter("execution:" + intent.Type + ":" + string(intent.Status)).Add(1)
			}
			return
		}

		// Non-retryable errors fail immediately.
		if prob.Code != problem.Unavailable {
			a.errors.Add(1)
			if a.cfg.Tracker != nil {
				a.cfg.Tracker.RecordError()
			}
			a.logger.Error("publish execution failed (non-retryable)",
				"error", prob.Message,
				"code", prob.Code,
				"type", intent.Type,
				"source", intent.Source,
				"symbol", intent.Symbol,
				"timeframe", intent.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
			return
		}

		// Retryable: log and retry once after backoff.
		if attempt < maxAttempts {
			a.logger.Warn("publish execution transient failure, retrying",
				"error", prob.Message,
				"attempt", attempt,
				"type", intent.Type,
				"source", intent.Source,
				"symbol", intent.Symbol,
				"timeframe", intent.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
			time.Sleep(retryDelay)
			continue
		}

		// Final attempt exhausted.
		a.errors.Add(1)
		if a.cfg.Tracker != nil {
			a.cfg.Tracker.RecordError()
		}
		a.logger.Error("publish execution failed after retry",
			"error", prob.Message,
			"code", prob.Code,
			"attempts", maxAttempts,
			"type", intent.Type,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
	}
}
