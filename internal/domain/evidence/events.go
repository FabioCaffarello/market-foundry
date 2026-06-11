package evidence

import "internal/shared/events"

const (
	EventCandleSampled     events.Name = "candle.sampled"
	EventTradeBurstSampled events.Name = "tradeburst.sampled"
	EventVolumeSampled     events.Name = "volume.sampled"
)

// CandleSampledEvent is emitted by derive when a candle window is sampled (interim or finalized).
type CandleSampledEvent struct {
	Metadata events.Metadata `json:"metadata"`
	Candle   EvidenceCandle  `json:"candle"`
}

func (e CandleSampledEvent) EventName() events.Name         { return EventCandleSampled }
func (e CandleSampledEvent) EventMetadata() events.Metadata { return e.Metadata }

// TradeBurstSampledEvent is emitted by derive when a trade burst window is sampled.
type TradeBurstSampledEvent struct {
	Metadata   events.Metadata    `json:"metadata"`
	TradeBurst EvidenceTradeBurst `json:"trade_burst"`
}

func (e TradeBurstSampledEvent) EventName() events.Name         { return EventTradeBurstSampled }
func (e TradeBurstSampledEvent) EventMetadata() events.Metadata { return e.Metadata }

// VolumeSampledEvent is emitted by derive when a volume profile window is sampled.
type VolumeSampledEvent struct {
	Metadata events.Metadata `json:"metadata"`
	Volume   EvidenceVolume  `json:"volume"`
}

func (e VolumeSampledEvent) EventName() events.Name         { return EventVolumeSampled }
func (e VolumeSampledEvent) EventMetadata() events.Metadata { return e.Metadata }
