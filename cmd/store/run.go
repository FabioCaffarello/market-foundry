package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
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

	pid := engine.Spawn(storeactor.NewStoreSupervisor(config, trackers), "store")

	// Collect all trackers for health server.
	allTrackers := make([]*healthz.Tracker, 0, len(trackers))
	for _, t := range trackers {
		allTrackers = append(allTrackers, t)
	}

	// Start health server for operational visibility.
	checks := buildReadinessChecks(config)
	srv := healthz.NewHealthServer(
		config.HTTP.Addr,
		checks,
		allTrackers,
	)
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("health server failed", "error", err)
		}
	}()

	actorcommon.WaitTillShutdown(engine, pid)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func buildReadinessChecks(config settings.AppConfig) []healthz.ReadinessCheck {
	return []healthz.ReadinessCheck{
		{
			Name: "nats",
			Check: func(ctx context.Context) error {
				if !config.NATS.Enabled {
					return fmt.Errorf("nats is not enabled")
				}
				return dialNATS(config.NATS.URL)
			},
		},
	}
}

// dialNATS performs a TCP dial to the NATS server to verify connectivity.
func dialNATS(natsURL string) error {
	u, err := url.Parse(natsURL)
	if err != nil {
		return fmt.Errorf("parse nats url: %w", err)
	}
	host := u.Host
	if host == "" {
		host = u.Opaque
	}
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return fmt.Errorf("nats dial: %w", err)
	}
	conn.Close()
	return nil
}
