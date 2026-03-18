package main

import (
	"context"
	"log/slog"

	"internal/application/evidenceclient"
	configctlcontracts "internal/application/configctl/contracts"
	"internal/application/ports"
	"internal/interfaces/http/handlers"
	"internal/shared/problem"
	"internal/shared/settings"
)

func newGatewayReadinessChecker(config settings.AppConfig, configctl ports.ConfigctlGateway, evidence ports.EvidenceGateway) handlers.ReadinessChecker {
	return handlers.ReadinessCheckerFunc(func(ctx context.Context) error {
		if !config.NATS.Enabled {
			return problem.New(problem.Unavailable, "gateway readiness requires nats to be enabled")
		}
		if configctl == nil {
			return problem.New(problem.Unavailable, "configctl gateway is unavailable")
		}

		if _, prob := configctl.ListConfigs(ctx, configctlcontracts.ListConfigsQuery{}); prob != nil {
			return prob
		}

		// Probe the evidence store — non-blocking for readiness.
		// If store is unavailable, the /evidence/candles/latest endpoint returns 503
		// but the gateway itself remains ready to serve configctl routes.
		if evidence != nil {
			if _, prob := evidence.GetLatestCandle(ctx, evidenceclient.CandleLatestQuery{
				Source:    "readiness-probe",
				Symbol:    "readiness-probe",
				Timeframe: 1,
			}); prob != nil {
				slog.Warn("evidence store probe failed during readiness check",
					"error", prob.Message,
				)
			}
		}

		return nil
	})
}
