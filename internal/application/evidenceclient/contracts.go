package evidenceclient

import (
	"internal/domain/instrument"

	"internal/domain/evidence"
)

// CandleLatestQuery is the request contract for querying the latest candle.
type CandleLatestQuery struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// CandleLatestReply is the response contract for the latest candle query.
type CandleLatestReply struct {
	Candle *evidence.EvidenceCandle `json:"candle,omitempty"`
}

// CandleHistoryQuery is the request contract for querying candle history.
//
// Range semantics (all optional, combine freely):
//   - Since/Until: unix seconds, inclusive. 0 = unset.
//   - Limit: max candles returned. Default 10, max 100.
//
// When Since and Until are both set, only candles whose OpenTime falls
// within [since, until] are returned. Results are always newest-first.
type CandleHistoryQuery struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
	Limit      int                            `json:"limit"`
	Since      int64                          `json:"since,omitempty"` // unix seconds, inclusive lower bound (0 = unset)
	Until      int64                          `json:"until,omitempty"` // unix seconds, inclusive upper bound (0 = unset)
}

// CandleHistoryReply is the response contract for the candle history query.
type CandleHistoryReply struct {
	Candles []evidence.EvidenceCandle `json:"candles"`
}

// TradeBurstLatestQuery is the request contract for querying the latest trade burst.
type TradeBurstLatestQuery struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// TradeBurstLatestReply is the response contract for the latest trade burst query.
type TradeBurstLatestReply struct {
	TradeBurst *evidence.EvidenceTradeBurst `json:"trade_burst,omitempty"`
}

// VolumeLatestQuery is the request contract for querying the latest volume profile.
type VolumeLatestQuery struct {
	Source     string                         `json:"source"`
	Instrument instrument.CanonicalInstrument `json:"instrument"`
	Timeframe  int                            `json:"timeframe"`
}

// VolumeLatestReply is the response contract for the latest volume query.
type VolumeLatestReply struct {
	Volume *evidence.EvidenceVolume `json:"volume,omitempty"`
}
