package main

import (
	"log/slog"
	"os"
	"time"

	actorcommon "internal/actors/common"
	storeactor "internal/actors/scopes/store"
	"internal/shared/bootstrap"
	"internal/shared/healthz"
	"internal/shared/settings"
)

func Run(config settings.AppConfig) {
	logger := bootstrap.BuildLogger(config.Log)
	slog.SetDefault(logger)

	logger.Info("store starting")

	engine, err := actorcommon.NewDefaultEngine()
	if err != nil {
		logger.Error("create actor engine", "error", err)
		os.Exit(1)
	}

	// Each projection pipeline gets its own pair of trackers for independent health visibility.
	// Trackers are built dynamically based on pipeline.families config.
	// When no families are configured, all trackers are created (backward compatible).
	type trackerDef struct {
		projName string
		consName string
	}
	allTrackerDefs := []trackerDef{
		{projName: "candle-projection", consName: "candle-consumer"},
		{projName: "trade-burst-projection", consName: "trade-burst-consumer"},
		{projName: "volume-projection", consName: "volume-consumer"},
	}
	// Map tracker names to their family for filtering.
	trackerFamilies := map[string]string{
		"candle-projection":      "candle",
		"candle-consumer":        "candle",
		"trade-burst-projection": "tradeburst",
		"trade-burst-consumer":   "tradeburst",
		"volume-projection":      "volume",
		"volume-consumer":        "volume",
	}

	trackers := make(map[string]*healthz.Tracker)
	for _, def := range allTrackerDefs {
		family := trackerFamilies[def.projName]
		if config.Pipeline.IsFamilyEnabled(family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}
	if len(trackers) == 0 {
		logger.Error("no projection families enabled — check pipeline.families in config")
		os.Exit(1)
	}

	// Signal pipeline trackers (opt-in via pipeline.signal_families).
	type signalTrackerDef struct {
		projName string
		consName string
		family   string
	}
	allSignalTrackerDefs := []signalTrackerDef{
		{projName: "signal-rsi-projection", consName: "signal-rsi-consumer", family: "rsi"},
	}
	for _, def := range allSignalTrackerDefs {
		if config.Pipeline.IsSignalFamilyEnabled(def.family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}

	// Decision pipeline trackers (opt-in via pipeline.decision_families).
	type decisionTrackerDef struct {
		projName string
		consName string
		family   string
	}
	allDecisionTrackerDefs := []decisionTrackerDef{
		{projName: "decision-rsi-oversold-projection", consName: "decision-rsi-oversold-consumer", family: "rsi_oversold"},
	}
	for _, def := range allDecisionTrackerDefs {
		if config.Pipeline.IsDecisionFamilyEnabled(def.family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}

	// Strategy pipeline trackers (opt-in via pipeline.strategy_families).
	type strategyTrackerDef struct {
		projName string
		consName string
		family   string
	}
	allStrategyTrackerDefs := []strategyTrackerDef{
		{projName: "strategy-mean-reversion-entry-projection", consName: "strategy-mean-reversion-entry-consumer", family: "mean_reversion_entry"},
	}
	for _, def := range allStrategyTrackerDefs {
		if config.Pipeline.IsStrategyFamilyEnabled(def.family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}

	// Risk pipeline trackers (opt-in via pipeline.risk_families).
	type riskTrackerDef struct {
		projName string
		consName string
		family   string
	}
	allRiskTrackerDefs := []riskTrackerDef{
		{projName: "risk-position-exposure-projection", consName: "risk-position-exposure-consumer", family: "position_exposure"},
	}
	for _, def := range allRiskTrackerDefs {
		if config.Pipeline.IsRiskFamilyEnabled(def.family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}

	// Execution pipeline trackers (opt-in via pipeline.execution_families).
	type executionTrackerDef struct {
		projName string
		consName string
		family   string
	}
	allExecutionTrackerDefs := []executionTrackerDef{
		{projName: "execution-paper-order-projection", consName: "execution-paper-order-consumer", family: "paper_order"},
		{projName: "execution-venue-market-order-projection", consName: "execution-venue-market-order-consumer", family: "venue_market_order"},
	}
	for _, def := range allExecutionTrackerDefs {
		if config.Pipeline.IsExecutionFamilyEnabled(def.family) {
			trackers[def.projName] = healthz.NewTracker(def.projName)
			trackers[def.consName] = healthz.NewTracker(def.consName)
		}
	}

	pid := engine.Spawn(storeactor.NewStoreSupervisor(config, trackers), "store")

	// Collect all trackers for health server.
	allTrackers := make([]*healthz.Tracker, 0, len(trackers))
	for _, t := range trackers {
		allTrackers = append(allTrackers, t)
	}

	// Start health server for operational visibility.
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		[]healthz.ReadinessCheck{bootstrap.NATSReadinessCheck(config)},
		allTrackers,
	)
	srv.StartInBackground()

	actorcommon.WaitTillShutdown(engine, pid)

	_ = srv.GracefulShutdown(5 * time.Second)
}
