package bybits

import (
	"encoding/json"
	"strings"
	"time"

	"internal/domain/instrument"
	"internal/domain/observation"
	"internal/shared/events"
	"internal/shared/problem"
)

// PublicTradeFrame represents a raw Bybit v5 publicTrade WebSocket
// frame. Unlike Binance's one-trade-per-message aggTrade stream,
// Bybit batches: Data carries N trades per frame.
// See: https://bybit-exchange.github.io/docs/v5/websocket/public/trade
type PublicTradeFrame struct {
	Topic string        `json:"topic"` // "publicTrade.BTCUSDT"
	Type  string        `json:"type"`  // "snapshot"
	Ts    int64         `json:"ts"`    // frame timestamp (ms)
	Data  []PublicTrade `json:"data"`
}

// PublicTrade is one trade inside a publicTrade frame.
type PublicTrade struct {
	TradeTime int64  `json:"T"` // Trade time (ms)
	Symbol    string `json:"s"` // Symbol (uppercase from Bybit)
	Side      string `json:"S"` // Taker side: "Buy" or "Sell"
	Quantity  string `json:"v"` // Trade size
	Price     string `json:"p"` // Trade price
	TradeID   string `json:"i"` // Trade ID
}

const (
	sourceName       = "bybits"
	publicTradeTopic = "publicTrade."
)

// frameHeader is the discriminating first pass over any v5 socket
// message: operational frames (subscribe acks, pongs) carry `op`;
// data frames carry `topic`. Decoding the header before the full
// frame keeps non-trade topics (orderbook deltas have an OBJECT
// `data`, not an array) from failing the trade-frame unmarshal.
type frameHeader struct {
	Op    string `json:"op"`
	Topic string `json:"topic"`
}

// ParsePublicTrade decodes a raw WebSocket message into a
// PublicTradeFrame. The tri-state return distinguishes:
//
//   - (frame, true, nil) — a publicTrade data frame.
//   - (zero, false, nil) — a control frame (subscribe ack, pong) or
//     a non-trade topic; callers skip silently. Bybit multiplexes
//     these on the data socket, so they are expected traffic, not
//     errors (unlike Binance's URL-per-stream model).
//   - (zero, false, prob) — malformed payload.
func ParsePublicTrade(data []byte) (PublicTradeFrame, bool, *problem.Problem) {
	var head frameHeader
	if err := json.Unmarshal(data, &head); err != nil {
		return PublicTradeFrame{}, false, problem.Wrap(err, problem.InvalidArgument, "failed to parse publicTrade message")
	}
	if head.Op != "" || !strings.HasPrefix(head.Topic, publicTradeTopic) {
		return PublicTradeFrame{}, false, nil
	}

	var frame PublicTradeFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return PublicTradeFrame{}, false, problem.Wrap(err, problem.InvalidArgument, "failed to parse publicTrade message")
	}
	if len(frame.Data) == 0 {
		return PublicTradeFrame{}, false, problem.New(problem.InvalidArgument, "publicTrade frame has no trades")
	}
	return frame, true, nil
}

// Normalize converts a Bybit Spot publicTrade frame into canonical
// ObservationTrade events — one per trade in the frame (Bybit
// batches N trades per frame). The symbol parameter is the
// venue-native form supplied by the connector (e.g., "btcusdt");
// it is parsed into a CanonicalInstrument here, at the adapter /
// domain boundary, per ADR-0021.
//
// Bybit's `S` field is the TAKER side: a "Sell" taker means the
// buyer was the resting maker order, so BuyerMaker = (S == "Sell")
// — the explicit inversion that maps Bybit's encoding onto the
// ObservationTrade contract (which follows Binance's `m` flag).
func Normalize(frame PublicTradeFrame, symbol string) ([]observation.TradeReceivedEvent, *problem.Problem) {
	inst, prob := parseBybitSpotSymbol(symbol)
	if prob != nil {
		return nil, prob
	}

	out := make([]observation.TradeReceivedEvent, 0, len(frame.Data))
	for _, td := range frame.Data {
		trade := observation.ObservationTrade{
			Source:     sourceName,
			Instrument: inst,
			Price:      td.Price,
			Quantity:   td.Quantity,
			TradeID:    td.TradeID,
			BuyerMaker: td.Side == "Sell",
			Timestamp:  time.UnixMilli(td.TradeTime).UTC(),
		}
		if prob := trade.Validate(); prob != nil {
			return nil, prob
		}
		out = append(out, observation.TradeReceivedEvent{
			Metadata: events.NewMetadata().WithOccurredAt(trade.Timestamp),
			Trade:    trade,
		})
	}
	return out, nil
}

// parseBybitSpotSymbol translates a venue-native Bybit Spot symbol
// into a CanonicalInstrument. Like the Binance Spot adapter, only
// USDT-quoted pairs are accepted: a non-USDT symbol on this
// connector signals a misconfiguration that must not become an
// invalid canonical instrument silently.
func parseBybitSpotSymbol(symbol string) (instrument.CanonicalInstrument, *problem.Problem) {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	if s == "" {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"bybits symbol is invalid",
			problem.ValidationIssue{Field: "symbol", Message: "must not be empty"},
		)
	}

	const quote = "USDT"
	if !strings.HasSuffix(s, quote) || len(s) <= len(quote) {
		return instrument.CanonicalInstrument{}, problem.Validation(
			problem.InvalidArgument,
			"bybits symbol is invalid",
			problem.ValidationIssue{
				Field:   "symbol",
				Message: "must end with USDT (bybits accepts USDT-quoted spot pairs only)",
				Value:   symbol,
			},
		)
	}
	base := s[:len(s)-len(quote)]
	return instrument.New(base, quote, instrument.ContractSpot)
}
