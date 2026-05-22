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
	"internal/shared/metrics"
	"internal/shared/problem"
	"internal/shared/settings"

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
	// S339: Activation surface dimensions for canonical state reporting.
	AdapterState     domainexec.AdapterState
	CredentialState  domainexec.CredentialState
	// S401: AllowedSources restricts which source prefixes this actor accepts.
	// When non-empty, intents with sources not in this set are rejected before
	// reaching the SegmentRouter — defense-in-depth against cross-segment leakage.
	// When empty, all sources are accepted (backwards-compatible with standalone configs).
	AllowedSources   map[string]bool
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

	// S339: Log canonical activation surface with resolved gate state.
	gateState := domainexec.DefaultControlGate()
	if a.controlStore != nil {
		ctx, gateCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if g, prob := a.controlStore.Get(ctx); prob == nil {
			gateState = g
		}
		gateCancel()

		// S344: Publish process-local activation dimensions to KV for gateway queryability.
		dims := domainexec.ActivationDimensions{
			Adapter:     a.cfg.AdapterState,
			Credentials: a.cfg.CredentialState,
			ReportedAt:  time.Now().UTC(),
			ReportedBy:  "execute",
		}
		dimCtx, dimCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if prob := a.controlStore.PutDimensions(dimCtx, dims); prob != nil {
			a.logger.Warn("failed to publish activation dimensions", "error", prob.Message)
		}
		dimCancel()
	}
	surface := domainexec.NewActivationSurface(a.cfg.AdapterState, gateState, a.cfg.CredentialState)
	a.logger.Info("activation surface resolved",
		"adapter", string(surface.Adapter),
		"gate_status", string(surface.Gate.Status),
		"credentials", string(surface.Credentials),
		"effective", string(surface.Effective),
		"is_live", surface.IsLive(),
	)

	a.logger.Info("venue adapter started",
		"staleness_max_age", a.cfg.StalenessMaxAge.String(),
		"submit_timeout", a.cfg.SubmitTimeout.String(),
		"control_gate", a.controlStore != nil,
		"retry_submitter", true,
		"retry_halt_checker", gateChecker != nil,
		"post200_reconciler", reconcilerActive,
	)
}

// segmentPrefix returns the segment name prefix for tracker counters.
// Returns empty string for unknown sources (counters are not incremented).
func segmentPrefix(source string) string {
	seg := settings.SegmentForSource(source)
	if seg == "" {
		return ""
	}
	return string(seg) + ":"
}

func (a *VenueAdapterActor) onIntent(msg intentReceivedMessage) {
	tracker := a.cfg.Tracker
	if tracker != nil {
		tracker.Counter("processed").Add(1)
	}
	intent := msg.Event.ExecutionIntent
	segPfx := segmentPrefix(intent.Source)
	if tracker != nil {
		tracker.Counter("processed:" + intent.Symbol).Add(1)
		if segPfx != "" {
			tracker.Counter(segPfx + "processed").Add(1)
		}
	}

	// S401: Gate 0 — Segment source guard (defense-in-depth).
	// Rejects intents from sources not in the allowed set before any other processing.
	// This is redundant with the SegmentRouter's own source validation but provides
	// an additional isolation barrier at the actor boundary.
	if len(a.cfg.AllowedSources) > 0 && !a.cfg.AllowedSources[intent.Source] {
		if tracker != nil {
			tracker.Counter("rejected_source").Add(1)
		}
		a.logger.Warn("intent rejected — source not in allowed set",
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	// Gates 1+2: Kill switch and staleness guard.
	verdict := a.safetyGate.Check(intent.Timestamp, time.Now().UTC())
	if !verdict.Allowed {
		// S361: Record gate verdict in Prometheus.
		metrics.IncGateCheck(verdict.Reason, "blocked")
		switch verdict.Reason {
		case "kill_switch":
			metrics.SetGateActive(false)
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
	// S361: Record allowed gate check.
	metrics.IncGateCheck("all", "allowed")
	metrics.SetGateActive(true)

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
			if segPfx != "" {
				tracker.Counter(segPfx + "errors").Add(1)
			}
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

		// S386: Publish rejection event for non-retryable failures (true venue rejections).
		// Retryable failures (transient network/rate-limit) are NOT rejections — they were
		// already retried by RetrySubmitter and exhausted. Both exhausted-retryable and
		// non-retryable problems produce a rejection event because the intent is terminal.
		a.publishRejection(msg, intent, prob)
		return
	}

	// Publish fill event.
	//
	// Note: Counter("filled") is incremented AFTER PublishFill below. This
	// creates a sub-microsecond observability window: a subscriber receiving
	// the fill via NATS may read Counter("filled") as 0 if it reads before
	// the actor's mailbox processes the increment.
	//
	// Current consumers tolerate this:
	//   - In-actor logStats() reads (same mailbox; race-free by construction).
	//   - HTTP /statusz reads (multi-ms HTTP timing dominates the race window).
	//   - Prometheus /metrics uses a separate counter set; not affected.
	//
	// Tests synchronize via the eventuallyAtLeast helper (P4.1.8).
	//
	// Future production consumers with sub-millisecond timing requirements
	// would need: (a) dual-semantic counter (submit_attempted vs
	// submit_succeeded), OR (b) actor reorder with compensating rollback
	// on publish failure. See docs/RESUMPTION.md M7 (design-meta).
	//
	// Decision context: P4.1.8.c investigation; Option (C) accepted per
	// current production patterns.
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
		if segPfx != "" {
			tracker.Counter(segPfx + "filled").Add(1)
		}
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

// publishRejection emits a VenueOrderRejectedEvent for audit trail and observability.
// S386: This closes the gap where rejections existed only as Problem returns with no
// downstream event. The intent is marked as rejected+final before publication.
func (a *VenueAdapterActor) publishRejection(msg intentReceivedMessage, intent domainexec.ExecutionIntent, prob *problem.Problem) {
	rejected := intent
	rejected.Status = domainexec.StatusRejected
	rejected.Final = true

	rejectionEvent := domainexec.VenueOrderRejectedEvent{
		Metadata: events.NewMetadata().
			WithCorrelationID(msg.Event.Metadata.CorrelationID).
			WithCausationID(msg.Event.Metadata.ID),
		ExecutionIntent: rejected,
		RejectionCode:   string(prob.Code),
		RejectionReason: prob.Message,
		VenueDetails:    prob.Details,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if pubProb := a.fillPublisher.PublishRejection(ctx, rejectionEvent); pubProb != nil {
		a.logger.Error("publish rejection event failed",
			"error", pubProb.Message,
			"source", intent.Source,
			"symbol", intent.Symbol,
			"timeframe", intent.Timeframe,
			"correlation_id", msg.Event.Metadata.CorrelationID,
		)
		return
	}

	tracker := a.cfg.Tracker
	if tracker != nil {
		tracker.Counter("rejected").Add(1)
		tracker.Counter("rejected:" + intent.Symbol).Add(1)
		segPfx := segmentPrefix(intent.Source)
		if segPfx != "" {
			tracker.Counter(segPfx + "rejected").Add(1)
		}
	}

	a.logger.Info("venue order rejected",
		"rejection_code", string(prob.Code),
		"source", intent.Source,
		"symbol", intent.Symbol,
		"timeframe", intent.Timeframe,
		"side", string(intent.Side),
		"quantity", intent.Quantity,
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
		"rejected", tracker.Counter("rejected").Load(),
		"skipped_stale", tracker.Counter("skipped_stale").Load(),
		"skipped_halt", tracker.Counter("skipped_halt").Load(),
		"errors", tracker.ErrorCount(),
	)
}
