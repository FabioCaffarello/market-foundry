package ingest

import (
	"internal/application/ingest"
	"internal/domain/observation"
)

// publishTradeMessage is sent from the WebSocket adapter actor to the publisher actor.
type publishTradeMessage struct {
	Event observation.TradeReceivedEvent
}

// activateBindingMessage is sent from the binding watcher to the supervisor
// when a new ingestion binding becomes active.
type activateBindingMessage struct {
	Target ingest.BindingTarget
}

// clearBindingMessage is sent from the binding watcher to the supervisor
// when an ingestion binding is deactivated.
type clearBindingMessage struct {
	Target ingest.BindingTarget
}
