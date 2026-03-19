package main

import (
	adapternats "internal/adapters/nats"
	"internal/shared/problem"
	"internal/shared/settings"
)

// newGatewayConn creates a NATS request/reply gateway using the provided builder.
// All gateway connections follow the same pattern: create a request client, wrap it
// with a domain-specific gateway constructor, return (gateway, closer, problem).
// The builder receives the request client and returns the typed gateway.
func newGatewayConn[T any](config settings.AppConfig, label string, build func(*adapternats.NATSRequestClient) T) (T, func() error, *problem.Problem) {
	var zero T
	if !config.NATS.Enabled {
		return zero, nil, nil
	}

	requestClient, err := adapternats.NewNATSRequestClientWithURL(config.NATS.URL, config.NATS.RequestTimeoutDuration())
	if err != nil {
		return zero, nil, problem.Wrap(err, problem.Unavailable, "failed to initialize "+label+" request client")
	}

	return build(requestClient), requestClient.Close, nil
}
