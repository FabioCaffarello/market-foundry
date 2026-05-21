package main

import (
	"log/slog"

	natsexecution "internal/adapters/nats/natsexecution"
	"internal/application/executionclient"
	"internal/domain/execution"
)

// verificationTrigger holds the background consumer that reacts to session
// lifecycle events by running automated verification and unified report generation.
// S490: Event-driven verification trigger — closes G-OA1.
// S491: Extended to also produce unified operational report — closes G-OA2, G-OA5.
type verificationTrigger struct {
	consumer *natsexecution.SessionLifecycleConsumer
	logger   *slog.Logger
}

// startVerificationTrigger creates and starts the event-driven verification
// trigger. Returns nil if the trigger cannot start (degraded, not fatal).
//
// The trigger is optional: if NATS is unavailable or the consumer fails to
// start, the gateway operates normally — manual verification still works.
//
// S491: reportUC is optional. When non-nil, the trigger generates a unified
// report after verification completes, closing the E2E automation loop.
func startVerificationTrigger(natsURL string, verifyUC *executionclient.VerifySessionUseCase, reportUC *executionclient.GenerateUnifiedReportUseCase, logger *slog.Logger) *verificationTrigger {
	if verifyUC == nil {
		logger.Info("verification trigger skipped: verify use case not wired")
		return nil
	}

	triggerUC := executionclient.NewTriggerVerifySessionUseCase(verifyUC, reportUC, logger)

	registry := natsexecution.DefaultRegistry()
	spec := natsexecution.GatewaySessionLifecycleConsumer()

	consumer := natsexecution.NewSessionLifecycleConsumer(
		natsURL,
		spec,
		registry,
		func(event execution.SessionLifecycleEvent) {
			triggerUC.Handle(event)
		},
		logger.With("component", "verification-trigger"),
	)

	if err := consumer.Start(); err != nil {
		logger.Warn("verification trigger unavailable — event-driven verification degraded",
			"error", err,
		)
		return nil
	}

	logger.Info("verification trigger started",
		"consumer_durable", spec.Durable,
		"consumer_subject", spec.Event.Subject,
	)

	return &verificationTrigger{
		consumer: consumer,
		logger:   logger,
	}
}

func (t *verificationTrigger) Close() {
	if t == nil || t.consumer == nil {
		return
	}
	if err := t.consumer.Close(); err != nil {
		t.logger.Error("close verification trigger consumer", "error", err)
	}
}
