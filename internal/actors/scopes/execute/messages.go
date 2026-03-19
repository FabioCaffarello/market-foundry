package execute

import "internal/domain/execution"

// intentReceivedMessage is sent from the venue consumer to the venue adapter actor.
// TRANSITIONAL BRIDGE: Carries PaperOrderSubmittedEvent because the execute binary's
// intake consumer currently subscribes to paper_order subjects. When venue-specific
// intent events are introduced, this message type will carry the venue intent event instead.
type intentReceivedMessage struct {
	Event execution.PaperOrderSubmittedEvent
}
