package main

import (
	adapternats "internal/adapters/nats"
	"internal/application/ports"
	"internal/shared/problem"
	"internal/shared/settings"
)

func newConfigctlGateway(config settings.AppConfig) (ports.ConfigctlGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize configctl request client")
	}

	gateway := adapternats.NewConfigctlGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}

// newEvidenceGateway creates a NATS request/reply client for querying the store binary.
// Optional: degrades gracefully if the store is not running.
func newEvidenceGateway(config settings.AppConfig) (ports.EvidenceGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize evidence request client")
	}

	gateway := adapternats.NewEvidenceGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}

// newSignalGateway creates a NATS request/reply client for querying signal projections.
// Optional: degrades gracefully if the store is not running.
func newSignalGateway(config settings.AppConfig) (ports.SignalGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize signal request client")
	}

	gateway := adapternats.NewSignalGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}

// newDecisionGateway creates a NATS request/reply client for querying decision projections.
// Optional: degrades gracefully if the store is not running.
func newDecisionGateway(config settings.AppConfig) (ports.DecisionGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize decision request client")
	}

	gateway := adapternats.NewDecisionGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}

// newStrategyGateway creates a NATS request/reply client for querying strategy projections.
// Optional: degrades gracefully if the store is not running.
func newStrategyGateway(config settings.AppConfig) (ports.StrategyGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize strategy request client")
	}

	gateway := adapternats.NewStrategyGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}

// newRiskGateway creates a NATS request/reply client for querying risk projections.
// Optional: degrades gracefully if the store is not running.
func newRiskGateway(config settings.AppConfig) (ports.RiskGateway, func() error, *problem.Problem) {
	if !config.NATS.Enabled {
		return nil, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return nil, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize risk request client")
	}

	gateway := adapternats.NewRiskGateway(requestClient, "gateway.http")
	return gateway, requestClient.Close, nil
}
