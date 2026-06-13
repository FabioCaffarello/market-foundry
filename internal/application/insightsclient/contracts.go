// Package insightsclient holds the read-side query contracts and use
// cases for the insights domain (PROGRAM-0005). Insights are
// decision-support (ADR-0027); these are read-only queries.
package insightsclient

import (
	"internal/domain/insights"
	"internal/domain/instrument"
)

// VolumeProfileLatestQuery requests the latest volume profile for a
// partition (source + canonical instrument + timeframe).
type VolumeProfileLatestQuery struct {
	Source     string
	Instrument instrument.CanonicalInstrument
	Timeframe  int
}

// VolumeProfileLatestReply carries the latest profile, or nil when
// none has been materialized yet.
type VolumeProfileLatestReply struct {
	VolumeProfile *insights.VolumeProfile
}
