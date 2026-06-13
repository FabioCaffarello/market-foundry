package insights

import "internal/shared/events"

const (
	// EventVolumeProfileSampled is emitted by derive when a volume
	// profile window is sampled (interim or finalized). It is an
	// insights event (INSIGHTS_EVENTS stream) — decision-support,
	// never a directive (ADR-0027).
	EventVolumeProfileSampled events.Name = "volumeprofile.sampled"
)

// VolumeProfileSampledEvent carries a sampled VolumeProfile.
type VolumeProfileSampledEvent struct {
	Metadata      events.Metadata `json:"metadata"`
	VolumeProfile VolumeProfile   `json:"volume_profile"`
}

func (e VolumeProfileSampledEvent) EventName() events.Name         { return EventVolumeProfileSampled }
func (e VolumeProfileSampledEvent) EventMetadata() events.Metadata { return e.Metadata }
