package natsinsights

import (
	"time"

	"internal/adapters/nats/natskit"

	"github.com/nats-io/nats.go/jetstream"
)

// Registry declares the INSIGHTS_EVENTS subjects/streams. Insights
// are decision-support (ADR-0027): the stream carries descriptive
// analytics (volume profile, later TPO / cross-venue), never
// directives. Single-writer per ADR-0008: derive is the only
// publisher to INSIGHTS_EVENTS.
type Registry struct {
	VolumeProfileSampled natskit.EventSpec
	VolumeProfileLatest  natskit.ControlSpec
	TPOProfileSampled    natskit.EventSpec
	TPOProfileLatest     natskit.ControlSpec
	CrossVenueSampled    natskit.EventSpec
	CrossVenueLatest     natskit.ControlSpec
}

// StoreCrossVenueConsumer is the store binding that projects cross-venue
// snapshots into the KV latest bucket (PROGRAM-0005 / H-8.c).
func StoreCrossVenueConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec(
		"store-cross-venue",
		"insights.events.crossvenue.sampled.>",
		"insights.events.v1.cross_venue_sampled",
		"INSIGHTS_EVENTS",
	)
}

// StoreVolumeProfileConsumer is the store binding that projects
// volume profiles into the KV latest bucket.
func StoreVolumeProfileConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec(
		"store-volume-profile",
		"insights.events.volumeprofile.sampled.>",
		"insights.events.v1.volume_profile_sampled",
		"INSIGHTS_EVENTS",
	)
}

// StoreTPOConsumer is the store binding that projects TPO profiles into
// the KV latest bucket (PROGRAM-0005 / H-8.b).
func StoreTPOConsumer() natskit.ConsumerSpec {
	return natskit.NewConsumerSpec(
		"store-tpo",
		"insights.events.tpo.sampled.>",
		"insights.events.v1.tpo_sampled",
		"INSIGHTS_EVENTS",
	)
}

func DefaultRegistry() Registry {
	eventStream := natskit.StreamSpec{
		Name:     "INSIGHTS_EVENTS",
		Subjects: []string{"insights.events.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   72 * time.Hour,
		MaxBytes: 256 * 1024 * 1024, // 256 MB — sized for local/CI event retention
	}

	return Registry{
		VolumeProfileSampled: natskit.EventSpec{
			Subject: "insights.events.volumeprofile.sampled",
			Type:    "insights.events.v1.volume_profile_sampled",
			Stream:  eventStream,
		},
		VolumeProfileLatest: natskit.ControlSpec{
			Subject:     "insights.query.volumeprofile.latest",
			RequestType: "insights.query.v1.volume_profile_latest_request",
			ReplyType:   "insights.query.v1.volume_profile_latest_reply",
			QueueGroup:  "insights.query",
		},
		TPOProfileSampled: natskit.EventSpec{
			Subject: "insights.events.tpo.sampled",
			Type:    "insights.events.v1.tpo_sampled",
			Stream:  eventStream,
		},
		TPOProfileLatest: natskit.ControlSpec{
			Subject:     "insights.query.tpo.latest",
			RequestType: "insights.query.v1.tpo_latest_request",
			ReplyType:   "insights.query.v1.tpo_latest_reply",
			QueueGroup:  "insights.query",
		},
		CrossVenueSampled: natskit.EventSpec{
			Subject: "insights.events.crossvenue.sampled",
			Type:    "insights.events.v1.cross_venue_sampled",
			Stream:  eventStream,
		},
		CrossVenueLatest: natskit.ControlSpec{
			Subject:     "insights.query.crossvenue.latest",
			RequestType: "insights.query.v1.cross_venue_latest_request",
			ReplyType:   "insights.query.v1.cross_venue_latest_reply",
			QueueGroup:  "insights.query",
		},
	}
}

// codegen:begin consumer_spec family=volume_profile source=codegen/families/volume_profile.yaml
// WriterVolumeProfileConsumer defines the durable consumer spec for writer consuming
// volume_profile insights events.
func WriterVolumeProfileConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-volume-profile",
		Event: natskit.EventSpec{
			Subject: "insights.events.volumeprofile.sampled.>",
			Type:    "insights.events.v1.volume_profile_sampled",
			Stream: natskit.StreamSpec{
				Name: "INSIGHTS_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=volume_profile

// codegen:begin consumer_spec family=tpo source=codegen/families/tpo.yaml
// WriterTPOConsumer defines the durable consumer spec for writer consuming
// tpo insights events.
func WriterTPOConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-tpo",
		Event: natskit.EventSpec{
			Subject: "insights.events.tpo.sampled.>",
			Type:    "insights.events.v1.tpo_sampled",
			Stream: natskit.StreamSpec{
				Name: "INSIGHTS_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=tpo

// codegen:begin consumer_spec family=cross_venue source=codegen/families/cross_venue.yaml
// WriterCrossVenueConsumer defines the durable consumer spec for writer consuming
// cross_venue insights events.
func WriterCrossVenueConsumer() natskit.ConsumerSpec {
	return natskit.ConsumerSpec{
		Durable: "writer-cross-venue",
		Event: natskit.EventSpec{
			Subject: "insights.events.crossvenue.sampled.>",
			Type:    "insights.events.v1.cross_venue_sampled",
			Stream: natskit.StreamSpec{
				Name: "INSIGHTS_EVENTS",
			},
		},
		AckWait:    30 * time.Second,
		MaxDeliver: 5,
	}
}

// codegen:end consumer_spec family=cross_venue
