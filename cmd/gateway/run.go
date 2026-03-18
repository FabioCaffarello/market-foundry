package main

import (
	actorcommon "internal/actors/common"
	actorgateway "internal/actors/scopes/gateway"
	configctlclient "internal/application/configctlclient"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/signalclient"
	"internal/application/strategyclient"
	"internal/interfaces/http/routes"
	"internal/shared/bootstrap"
	"internal/shared/settings"
	"log/slog"
	"os"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log)
	slog.SetDefault(logger)

	logger.Info("gateway starting", "addr", config.HTTP.Addr)
	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	gateway, closeFn, prob := newConfigctlGateway(config)
	if prob != nil {
		logger.Error("create configctl gateway", "error", prob)
		os.Exit(1)
	}
	if closeFn != nil {
		defer func() {
			if err := closeFn(); err != nil {
				logger.Error("close configctl gateway", "error", err)
			}
		}()
	}

	createDraftUseCase := configctlclient.NewCreateDraftUseCase(gateway)
	getConfigUseCase := configctlclient.NewGetConfigUseCase(gateway)
	getActiveUseCase := configctlclient.NewGetActiveConfigUseCase(gateway)
	listActiveRuntimeProjectionsUseCase := configctlclient.NewListActiveRuntimeProjectionsUseCase(gateway)
	listActiveIngestionBindingsUseCase := configctlclient.NewListActiveIngestionBindingsUseCase(gateway)
	listConfigsUseCase := configctlclient.NewListConfigsUseCase(gateway)
	validateDraftUseCase := configctlclient.NewValidateDraftUseCase(gateway)
	validateConfigUseCase := configctlclient.NewValidateConfigUseCase(gateway)
	compileConfigUseCase := configctlclient.NewCompileConfigUseCase(gateway)
	activateConfigUseCase := configctlclient.NewActivateConfigUseCase(gateway)

	// Evidence gateway — queries the store binary for candle read models.
	// Optional: degrades gracefully if the store is not running.
	evGateway, evCloseFn, evProb := newEvidenceGateway(config)
	if evProb != nil {
		logger.Warn("evidence gateway unavailable", "error", evProb)
	}
	if evCloseFn != nil {
		defer func() {
			if err := evCloseFn(); err != nil {
				logger.Error("close evidence gateway", "error", err)
			}
		}()
	}
	var getLatestCandleUseCase *evidenceclient.GetLatestCandleUseCase
	var getCandleHistoryUseCase *evidenceclient.GetCandleHistoryUseCase
	var getLatestTradeBurstUseCase *evidenceclient.GetLatestTradeBurstUseCase
	var getLatestVolumeUseCase *evidenceclient.GetLatestVolumeUseCase
	if evGateway != nil {
		getLatestCandleUseCase = evidenceclient.NewGetLatestCandleUseCase(evGateway)
		getCandleHistoryUseCase = evidenceclient.NewGetCandleHistoryUseCase(evGateway)
		getLatestTradeBurstUseCase = evidenceclient.NewGetLatestTradeBurstUseCase(evGateway)
		getLatestVolumeUseCase = evidenceclient.NewGetLatestVolumeUseCase(evGateway)
	}

	// Signal gateway — queries the store binary for signal read models.
	// Optional: degrades gracefully if the store is not running.
	sigGateway, sigCloseFn, sigProb := newSignalGateway(config)
	if sigProb != nil {
		logger.Warn("signal gateway unavailable", "error", sigProb)
	}
	if sigCloseFn != nil {
		defer func() {
			if err := sigCloseFn(); err != nil {
				logger.Error("close signal gateway", "error", err)
			}
		}()
	}
	var getLatestSignalUseCase *signalclient.GetLatestSignalUseCase
	if sigGateway != nil {
		getLatestSignalUseCase = signalclient.NewGetLatestSignalUseCase(sigGateway)
	}

	// Decision gateway — queries the store binary for decision read models.
	// Optional: degrades gracefully if the store is not running.
	decGateway, decCloseFn, decProb := newDecisionGateway(config)
	if decProb != nil {
		logger.Warn("decision gateway unavailable", "error", decProb)
	}
	if decCloseFn != nil {
		defer func() {
			if err := decCloseFn(); err != nil {
				logger.Error("close decision gateway", "error", err)
			}
		}()
	}
	var getLatestDecisionUseCase *decisionclient.GetLatestDecisionUseCase
	if decGateway != nil {
		getLatestDecisionUseCase = decisionclient.NewGetLatestDecisionUseCase(decGateway)
	}

	// Strategy gateway — queries the store binary for strategy read models.
	// Optional: degrades gracefully if the store is not running.
	stratGateway, stratCloseFn, stratProb := newStrategyGateway(config)
	if stratProb != nil {
		logger.Warn("strategy gateway unavailable", "error", stratProb)
	}
	if stratCloseFn != nil {
		defer func() {
			if err := stratCloseFn(); err != nil {
				logger.Error("close strategy gateway", "error", err)
			}
		}()
	}
	var getLatestStrategyUseCase *strategyclient.GetLatestStrategyUseCase
	if stratGateway != nil {
		getLatestStrategyUseCase = strategyclient.NewGetLatestStrategyUseCase(stratGateway)
	}

	gatewayRoutes := routes.DefaultRoutes(routes.Dependencies{
		Readiness:                    newGatewayReadinessChecker(config, gateway, evGateway),
		CreateDraft:                  createDraftUseCase,
		GetConfig:                    getConfigUseCase,
		GetActive:                    getActiveUseCase,
		ListActiveRuntimeProjections: listActiveRuntimeProjectionsUseCase,
		ListActiveIngestionBindings:  listActiveIngestionBindingsUseCase,
		ListConfigs:                  listConfigsUseCase,
		ValidateDraft:                validateDraftUseCase,
		ValidateConfig:               validateConfigUseCase,
		CompileConfig:                compileConfigUseCase,
		ActivateConfig:               activateConfigUseCase,
		Evidence: routes.EvidenceFamilyDeps{
			GetLatestCandle:     getLatestCandleUseCase,
			GetCandleHistory:    getCandleHistoryUseCase,
			GetLatestTradeBurst: getLatestTradeBurstUseCase,
			GetLatestVolume:     getLatestVolumeUseCase,
		},
		Signal: routes.SignalFamilyDeps{
			GetLatestSignal: getLatestSignalUseCase,
		},
		Decision: routes.DecisionFamilyDeps{
			GetLatestDecision: getLatestDecisionUseCase,
		},
		Strategy: routes.StrategyFamilyDeps{
			GetLatestStrategy: getLatestStrategyUseCase,
		},
	})

	pid := engine.Spawn(actorgateway.NewGateway(config.HTTP, gatewayRoutes), "gateway")
	actorcommon.WaitTillShutdown(engine, pid)
}
