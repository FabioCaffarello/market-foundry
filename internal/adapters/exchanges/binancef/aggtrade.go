package binancef

import (
	"encoding/json"
	"time"

	"internal/domain/observation"
	"internal/shared/events"
	"internal/shared/problem"
)

// AggTrade represents the raw Binance Futures aggTrade WebSocket message.
// See: https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-market-streams/Aggregate-Trade-Streams
type AggTrade struct {
	EventType     string `json:"e"` // "aggTrade"
	EventTime     int64  `json:"E"` // Event time (ms)
	Symbol        string `json:"s"` // Symbol (uppercase from Binance)
	AggTradeID    int64  `json:"a"` // Aggregate trade ID
	Price         string `json:"p"` // Price
	Quantity      string `json:"q"` // Quantity
	FirstTradeID  int64  `json:"f"` // First trade ID
	LastTradeID   int64  `json:"l"` // Last trade ID
	TradeTime     int64  `json:"T"` // Trade time (ms)
	IsBuyerMaker  bool   `json:"m"` // Is the buyer the maker?
}

const sourceName = "binancef"

// ParseAggTrade decodes a raw WebSocket message into an AggTrade.
func ParseAggTrade(data []byte) (AggTrade, *problem.Problem) {
	var raw AggTrade
	if err := json.Unmarshal(data, &raw); err != nil {
		return AggTrade{}, problem.Wrap(err, problem.InvalidArgument, "failed to parse aggTrade message")
	}
	if raw.EventType != "aggTrade" {
		return AggTrade{}, problem.New(problem.InvalidArgument, "unexpected event type: "+raw.EventType)
	}
	return raw, nil
}

// Normalize converts a Binance Futures AggTrade into a canonical ObservationTrade.
func Normalize(raw AggTrade, symbol string) (observation.TradeReceivedEvent, *problem.Problem) {
	trade := observation.ObservationTrade{
		Source:     sourceName,
		Symbol:     symbol,
		Price:      raw.Price,
		Quantity:   raw.Quantity,
		TradeID:    formatTradeID(raw.AggTradeID),
		BuyerMaker: raw.IsBuyerMaker,
		Timestamp:  time.UnixMilli(raw.TradeTime).UTC(),
	}

	if prob := trade.Validate(); prob != nil {
		return observation.TradeReceivedEvent{}, prob
	}

	return observation.TradeReceivedEvent{
		Metadata: events.NewMetadata().WithOccurredAt(trade.Timestamp),
		Trade:    trade,
	}, nil
}

func formatTradeID(id int64) string {
	// Fast int-to-string without fmt import.
	if id == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	if id < 0 {
		buf = append(buf, '-')
		id = -id
	}
	start := len(buf)
	for id > 0 {
		buf = append(buf, byte('0'+id%10))
		id /= 10
	}
	// Reverse the digits.
	for i, j := start, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
