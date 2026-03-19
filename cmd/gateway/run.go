package main

import (
	actorcommon "internal/actors/common"
	actorgateway "internal/actors/scopes/gateway"
	configctlclient "internal/application/configctlclient"
	"internal/application/decisionclient"
	"internal/application/evidenceclient"
	"internal/application/executionclient"
	"internal/application/riskclient"
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

	// Risk gateway — queries the store binary for risk read models.
	// Optional: degrades gracefully if the store is not running.
	riskGateway, riskCloseFn, riskProb := newRiskGateway(config)
	if riskProb != nil {
		logger.Warn("risk gateway unavailable", "error", riskProb)
	}
	if riskCloseFn != nil {
		defer func() {
			if err := riskCloseFn(); err != nil {
				logger.Error("close risk gateway", "error", err)
			}
		}()
	}
	var getLatestRiskUseCase *riskclient.GetLatestRiskUseCase
	if riskGateway != nil {
		getLatestRiskUseCase = riskclient.NewGetLatestRiskUseCase(riskGateway)
	}

	// Execution gateway — queries the store binary for execution read models.
	// Optional: degrades gracefully if the store is not running.
	execGateway, execCloseFn, execProb := newExecutionGateway(config)
	if execProb != nil {
		logger.Warn("execution gateway unavailable", "error", execProb)
	}
	if execCloseFn != nil {
		defer func() {
			if err := execCloseFn(); err != nil {
				logger.Error("close execution gateway", "error", err)
			}
		}()
	}
	var getLatestExecutionUseCase *executionclient.GetLatestExecutionUseCase
	var getExecutionStatusUseCase *executionclient.GetExecutionStatusUseCase
	if execGateway != nil {
		getLatestExecutionUseCase = executionclient.NewGetLatestExecutionUseCase(execGateway)
		getExecutionStatusUseCase = executionclient.NewGetExecutionStatusUseCase(execGateway)
	}

	// Execution control gateway — get/set the execution control gate.
	// Optional: degrades gracefully if the store is not running.
	execControlGateway, execControlCloseFn, execControlProb := newExecutionControlGateway(config)
	if execControlProb != nil {
		logger.Warn("execution control gateway unavailable", "error", execControlProb)
	}
	if execControlCloseFn != nil {
		defer func() {
			if err := execControlCloseFn(); err != nil {
				logger.Error("close execution control gateway", "error", err)
			}
		}()
	}
	var getExecutionControlUseCase *executionclient.GetExecutionControlUseCase
	var setExecutionControlUseCase *executionclient.SetExecutionControlUseCase
	if execControlGateway != nil {
		getExecutionControlUseCase = executionclient.NewGetExecutionControlUseCase(execControlGateway)
		setExecutionControlUseCase = executionclient.NewSetExecutionControlUseCase(execControlGateway)
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
		Risk: routes.RiskFamilyDeps{
			GetLatestRisk: getLatestRiskUseCase,
		},
		Execution: routes.ExecutionFamilyDeps{
			GetLatestExecution:  getLatestExecutionUseCase,
			GetExecutionStatus:  getExecutionStatusUseCase,
			GetExecutionControl: getExecutionControlUseCase,
			SetExecutionControl: setExecutionControlUseCase,
		},
	})

	pid := engine.Spawn(actorgateway.NewGateway(config.HTTP, gatewayRoutes), "gateway")
	actorcommon.WaitTillShutdown(engine, pid)
}
