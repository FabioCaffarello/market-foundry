package execute

import (
	"internal/domain/execution"
	"internal/domain/strategy"
)

// intentReceivedMessage is sent from the venue consumer to the venue adapter actor.
// TRANSITIONAL BRIDGE: Carries PaperOrderSubmittedEvent because the execute binary's
// intake consumer currently subscribes to paper_order subjects. When venue-specific
// intent events are introduced, this message type will carry the venue intent event instead.
type intentReceivedMessage struct {
	Event execution.PaperOrderSubmittedEvent
}

// strategyReceivedMessage is sent from the supervisor's strategy consumer handler
// to the StrategyConsumerActor for evaluation.
// S360: strategy-to-execution wiring — the actor evaluates the event and
// forwards a synthetic intentReceivedMessage to the venue adapter actor.
type strategyReceivedMessage struct {
	Event strategy.StrategyResolvedEvent
}
