package execute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	actorcommon "internal/actors/common"
	natsexecution "internal/adapters/nats/natsexecution"
	appexec "internal/application/execution"
	"internal/application/ports"
	domainexec "internal/domain/execution"
	"internal/shared/events"
	"internal/shared/healthz"

	"github.com/anthdm/hollywood/actor"
)

// VenueAdapterConfig holds the configuration for the venue adapter actor.
type VenueAdapterConfig struct {
	NATSURL          string
	Source           string
	Registry         natsexecution.Registry
	Venue            ports.VenuePort
	StalenessMaxAge  time.Duration
	SubmitTimeout    time.Duration
	Tracker          *healthz.Tracker
	// VenueQuery is the query port for post-200 reconciliation (S322).
	// When set, the Post200Reconciler is composed around the submit pipeline.
	// When nil, reconciliation is skipped (e.g. paper adapter has no query path).
	VenueQuery       ports.VenueQueryPort
}

// VenueAdapterActor consumes execution intents, checks kill switch + staleness,
// calls VenuePort.SubmitOrder, and publishes fill events.
//
// S328: The submit pipeline is composed at startup via decorator stacking:
//
//	Post200Reconciler → RetrySubmitter(+hooks) → rawAdapter
//
// The composed venue replaces the raw adapter for all onIntent calls.
type VenueAdapterActor struct {
	cfg           VenueAdapterConfig
	logger        *slog.Logger
	controlStore  *natsexecution.ControlKVStore
	fillPublisher *natsexecution.Publisher
	safetyGate    *appexec.SafetyGate
	// venue is the fully composed submit pipeline, assembled in start().
	// Before start(), this is nil; onIntent uses this instead of cfg.Venue.
	venue         ports.VenuePort
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
	staleness := appexec.NewStalenessGuard(a.cfg.StalenessMaxAge)

	// Connect to execution control KV for kill switch.
	var gateChecker appexec.GateChecker
	controlStore := natsexecution.NewControlKVStore(a.cfg.NATSURL)
	if err := controlStore.Start(); err != nil {
		a.logger.Warn("execution control KV store unavailable — gate check disabled", "error", err)
	} else {
		a.controlStore = controlStore
		gateChecker = controlStore
	}

	// Assemble the safety gate with the available components.
	a.safetyGate = appexec.NewSafetyGate(gateChecker, 2*time.Second, staleness)

	// --- S328: Compose decorator pipeline around the raw venue adapter ---
	//
	// Decorator order (innermost → outermost):
	//   1. rawAdapter        — the venue HTTP adapter (e.g. BinanceFuturesTestnetAdapter)
	//   2. RetrySubmitter    — retries retryable failures with backoff, deadline, halt check
	//   3. Post200Reconciler — recovers body-read-failure-after-200 via QueryOrder
	//
	// Rationale: RetrySubmitter wraps the raw adapter so transient failures are
	// retried before surfacing. body-read-failure-after-200 is non-retryable
	// (the venue accepted the order), so it passes through RetrySubmitter unchanged
	// and is caught by Post200Reconciler at the outer layer.
	//
	// Observability hooks (PWT-3): WithHaltChecker, WithLogger, WithTracker are
	// attached to RetrySubmitter so retry events are observable in structured logs
	// and health counters.
	rawVenue := a.cfg.Venue

	// PWT-1: Wrap with RetrySubmitter for retryable failure recovery.
	retrySubmitter := appexec.NewRetrySubmitter(rawVenue, appexec.DefaultRetryPolicy())
	if gateChecker != nil {
		retrySubmitter = retrySubmitter.WithHaltChecker(gateChecker)
	}
	// PWT-3: Attach observability hooks.
	retrySubmitter = retrySubmitter.WithLogger(a.logger.With("component", "retry-submitter"))
	if a.cfg.Tracker != nil {
		retrySubmitter = retrySubmitter.WithTracker(a.cfg.Tracker)
	}

	// PWT-2: Wrap with Post200Reconciler for body-read-failure recovery.
	var composedVenue ports.VenuePort = retrySubmitter
	reconcilerActive := false
	if a.cfg.VenueQuery != nil {
		composedVenue = appexec.NewPost200Reconciler(retrySubmitter, a.cfg.VenueQuery, 0)
		reconcilerActive = true
	}

	a.venue = composedVenue

	// Connect fill publisher for publishing fill events.
	fillPub := natsexecution.NewPublisher(a.cfg.NATSURL, a.cfg.Source, a.cfg.Registry)
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
		"retry_submitter", true,
		"retry_halt_checker", gateChecker != nil,
		"post200_reconciler", reconcilerActive,
	)
}

func (a *VenueAdapterActor) onIntent(msg intentReceivedMessage) {
	tracker := a.cfg.Tracker
	if tracker != nil {
		tracker.Counter("processed").Add(1)
	}
	intent := msg.Event.ExecutionIntent
	if tracker != nil {
		tracker.Counter("processed:" + intent.Symbol).Add(1)
	}

	// Gates 1+2: Kill switch and staleness guard.
	verdict := a.safetyGate.Check(intent.Timestamp, time.Now().UTC())
	if !verdict.Allowed {
		switch verdict.Reason {
		case "kill_switch":
			if tracker != nil {
				tracker.Counter("skipped_halt").Add(1)
			}
			a.logger.Warn("intent blocked by kill switch",
				"source", intent.Source,
				"symbol", intent.Symbol,
				"timeframe", intent.Timeframe,
				"correlation_id", msg.Event.Metadata.CorrelationID,
			)
		case "stale":
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
		default:
			if tracker != nil {
				tracker.RecordError()
			}
			a.logger.Error("safety gate blocked with unknown reason",
				"reason", verdict.Reason,
				"source", intent.Source,
				"symbol", intent.Symbol,
			)
		}
		return
	}

	// Gate 3: Submit to venue adapter with configurable timeout.
	submitTimeout := a.cfg.SubmitTimeout
	if submitTimeout == 0 {
		submitTimeout = 10 * time.Second
	}
	submitCtx, submitCancel := context.WithTimeout(context.Background(), submitTimeout)
	defer submitCancel()

	receipt, prob := a.venue.SubmitOrder(
		submitCtx,
		ports.VenueOrderRequest{Intent: intent},
	)
	if prob != nil {
		if tracker != nil {
			tracker.RecordError()
		}
		logAttrs := []any{
			"error", prob.Message,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		}
		// Surface retry metadata from Problem.Details for structured observability.
		for _, key := range []string{
			"retry_attempts", "retry_exhausted",
			"retry_halted", "retry_deadline_exceeded",
		} {
			if v, ok := prob.Details[key]; ok {
				logAttrs = append(logAttrs, key, v)
			}
		}
		a.logger.Error("venue submit failed", logAttrs...)
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
		tracker.Counter("filled:" + intent.Symbol).Add(1)
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
