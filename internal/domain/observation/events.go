package observation

import "internal/shared/events"

const (
	EventTradeReceived events.Name = "market.trade_received"
)

// TradeReceivedEvent is emitted by ingest when a trade is captured and normalized from an external source.
type TradeReceivedEvent struct {
	Metadata events.Metadata  `json:"metadata"`
	Trade    ObservationTrade `json:"trade"`
}

func (e TradeReceivedEvent) EventName() events.Name         { return EventTradeReceived }
func (e TradeReceivedEvent) EventMetadata() events.Metadata { return e.Metadata }
