package binances

import (
	"encoding/json"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/domain/observation"
	"internal/shared/events"
	"internal/shared/problem"
)

// AggTrade represents the raw Binance Spot aggTrade WebSocket message.
// See: https://developers.binance.com/docs/binance-spot-api-docs/web-socket-streams#aggregate-trade-streams
type AggTrade struct {
	EventType    string `json:"e"` // "aggTrade"
	EventTime    int64  `json:"E"` // Event time (ms)
	Symbol       string `json:"s"` // Symbol (uppercase from Binance)
	AggTradeID   int64  `json:"a"` // Aggregate trade ID
	Price        string `json:"p"` // Price
	Quantity     string `json:"q"` // Quantity
	FirstTradeID int64  `json:"f"` // First trade ID
	LastTradeID  int64  `json:"l"` // Last trade ID
	TradeTime    int64  `json:"T"` // Trade time (ms)
	IsBuyerMaker bool   `json:"m"` // Is the buyer the maker?
}

const sourceName = "binances"

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

// Normalize converts a Binance Spot AggTrade into a canonical
// ObservationTrade. The symbol parameter is the venue-native form
// supplied by the connector (e.g., "btcusdt"); it is parsed into a
// CanonicalInstrument here, at the adapter / domain boundary, per
// ADR-0021.
func Normalize(raw AggTrade, symbol string) (observation.TradeReceivedEvent, *problem.Problem) {
	inst, prob := parseSpotSymbol(symbol)
	if prob != nil {
		return observation.TradeReceivedEvent{}, prob
	}

	trade := observation.ObservationTrade{
		Source:     sourceName,
		Instrument: inst,
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

// parseSpotSymbol translates a venue-native Binance Spot symbol
// (e.g., "btcusdt") into a CanonicalInstrument. Binance Spot pairs
// are USDT-quoted in the current routing path; anything else is
// rejected at this boundary so a misconfigured connector cannot
// silently inject a non-canonical quote asset.
func parseSpotSymbol(symbol string) (instrument.CanonicalInstrument, *problem.Problem) {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"binances symbol is invalid",
			problem.ValidationIssue{Field: "symbol", Message: "must not be empty"},
		)
	}
	const quote = "USDT"
	if !strings.HasSuffix(s, quote) || len(s) <= len(quote) {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"binances symbol is invalid",
			problem.ValidationIssue{
				Field:   "symbol",
				Message: "must end with USDT (binances H-6.a supports USDT-quoted spot only)",
				Value:   symbol,
			},
		)
	}
	base := s[:len(s)-len(quote)]
	return instrument.New(base, quote, instrument.ContractSpot)
}

func formatTradeID(id int64) string {
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
	for i, j := start, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
