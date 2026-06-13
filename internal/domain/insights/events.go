package insights

import "internal/shared/events"

const (
	// EventVolumeProfileSampled is emitted by derive when a volume
	// profile window is sampled (interim or finalized). It is an
	// insights event (INSIGHTS_EVENTS stream) — decision-support,
	// never a directive (ADR-0027).
	EventVolumeProfileSampled events.Name = "volumeprofile.sampled"

	// EventTPOProfileSampled is emitted by derive when a TPO (Time-Price
	// Opportunity) window is sampled (interim or finalized). Same stream
	// and decision-support posture as the volume profile event.
	EventTPOProfileSampled events.Name = "tpo.sampled"
)

// VolumeProfileSampledEvent carries a sampled VolumeProfile.
type VolumeProfileSampledEvent struct {
	Metadata      events.Metadata `json:"metadata"`
	VolumeProfile VolumeProfile   `json:"volume_profile"`
}

func (e VolumeProfileSampledEvent) EventName() events.Name         { return EventVolumeProfileSampled }
func (e VolumeProfileSampledEvent) EventMetadata() events.Metadata { return e.Metadata }

// TPOProfileSampledEvent carries a sampled TPOProfile.
type TPOProfileSampledEvent struct {
	Metadata   events.Metadata `json:"metadata"`
	TPOProfile TPOProfile      `json:"tpo_profile"`
}

func (e TPOProfileSampledEvent) EventName() events.Name         { return EventTPOProfileSampled }
func (e TPOProfileSampledEvent) EventMetadata() events.Metadata { return e.Metadata }
