package ingest

import (
	"strings"

	"internal/shared/problem"
)

// BindingTarget represents a parsed ingestion binding topic.
// Convention: topic = "{source}.{symbol}" (e.g., "binancef.btcusdt").
type BindingTarget struct {
	Source string
	Symbol string
}

// Key returns the unique identifier for this binding target.
func (b BindingTarget) Key() string {
	return b.Source + "." + b.Symbol
}

// ParseBindingTopic extracts source and symbol from a binding topic.
func ParseBindingTopic(topic string) (BindingTarget, *problem.Problem) {
	parts := strings.SplitN(topic, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return BindingTarget{}, problem.New(problem.InvalidArgument,
			"binding topic must follow the format 'source.symbol', got: "+topic,
		)
	}

	return BindingTarget{
		Source: strings.ToLower(parts[0]),
		Symbol: strings.ToLower(parts[1]),
	}, nil
}
