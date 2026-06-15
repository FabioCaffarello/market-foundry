package natsdelivery

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"internal/application/insightsclient"
	"internal/application/ports"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"
	"internal/shared/problem"
)

const snapshotReadTimeout = 2 * time.Second

// snapshotGateway is the read surface the KV-backed snapshot provider
// needs — satisfied by *natsinsights.Gateway (the gateway's KV-direct
// insights reader). A narrow interface keeps the provider unit-testable.
type snapshotGateway interface {
	GetLatestVolumeProfile(context.Context, insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem)
	GetLatestTPOProfile(context.Context, insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem)
	GetLatestCrossVenue(context.Context, insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem)
}

// KVSnapshotProvider implements ports.SnapshotProvider over the insights
// KV-latest stores: it parses a fully-specified insights subject, reads
// the current value, and renders it as a {subject,event} client frame
// (same shape as a live delivery frame). Satisfies ports.SnapshotProvider.
type KVSnapshotProvider struct {
	gw snapshotGateway
}

var _ ports.SnapshotProvider = (*KVSnapshotProvider)(nil)

// NewKVSnapshotProvider builds a snapshot provider over an insights read
// gateway (e.g. *natsinsights.Gateway).
func NewKVSnapshotProvider(gw snapshotGateway) *KVSnapshotProvider {
	return &KVSnapshotProvider{gw: gw}
}

// insightsKey is a fully-specified insights subject decomposed.
type insightsKey struct {
	family    string // "volumeprofile" | "tpo" | "crossvenue"
	source    string // empty for crossvenue
	inst      instrument.CanonicalInstrument
	timeframe int
}

// parseInsightsSubject decomposes a FULLY-SPECIFIED insights event subject
//
//	insights.events.{family}.sampled.{slot}.{token}.{tf}
//
// into its key, or returns ok=false for a wildcard, wrong arity, unknown
// family, or unparseable token/timeframe. For volumeprofile/tpo the slot
// is the source; for crossvenue the slot is the literal "crossvenue" (no
// source — cross-venue spans sources).
func parseInsightsSubject(subject string) (insightsKey, bool) {
	parts := strings.Split(subject, ".")
	if len(parts) != 7 {
		return insightsKey{}, false
	}
	for _, p := range parts {
		if p == "" || p == "*" || p == ">" { // not fully specified
			return insightsKey{}, false
		}
	}
	if parts[0] != "insights" || parts[1] != "events" || parts[3] != "sampled" {
		return insightsKey{}, false
	}
	family, slot, token, tfStr := parts[2], parts[4], parts[5], parts[6]

	tf, err := strconv.Atoi(tfStr)
	if err != nil {
		return insightsKey{}, false
	}
	inst, prob := instrument.FromSubjectToken(token)
	if prob != nil {
		return insightsKey{}, false
	}

	key := insightsKey{family: family, inst: inst, timeframe: tf}
	switch family {
	case "volumeprofile", "tpo":
		key.source = slot
	case "crossvenue":
		if slot != "crossvenue" {
			return insightsKey{}, false
		}
	default:
		return insightsKey{}, false
	}
	return key, true
}

// Snapshot renders the current KV-latest for a fully-specified subject as
// a {subject,event} client frame, or (nil, false) if not applicable.
func (p *KVSnapshotProvider) Snapshot(subject string) ([]byte, bool) {
	if p == nil || p.gw == nil {
		return nil, false
	}
	key, ok := parseInsightsSubject(subject)
	if !ok {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), snapshotReadTimeout)
	defer cancel()

	// Build the same SampledEvent the live path publishes (synthetic
	// metadata + the KV-latest value), so the snapshot frame is shape-
	// identical to a delta frame.
	var event any
	switch key.family {
	case "volumeprofile":
		reply, prob := p.gw.GetLatestVolumeProfile(ctx, insightsclient.VolumeProfileLatestQuery{
			Source: key.source, Instrument: key.inst, Timeframe: key.timeframe,
		})
		if prob != nil || reply.VolumeProfile == nil {
			return nil, false
		}
		event = insights.VolumeProfileSampledEvent{Metadata: events.NewMetadata(), VolumeProfile: *reply.VolumeProfile}
	case "tpo":
		reply, prob := p.gw.GetLatestTPOProfile(ctx, insightsclient.TPOProfileLatestQuery{
			Source: key.source, Instrument: key.inst, Timeframe: key.timeframe,
		})
		if prob != nil || reply.TPOProfile == nil {
			return nil, false
		}
		event = insights.TPOProfileSampledEvent{Metadata: events.NewMetadata(), TPOProfile: *reply.TPOProfile}
	case "crossvenue":
		reply, prob := p.gw.GetLatestCrossVenue(ctx, insightsclient.CrossVenueLatestQuery{
			Instrument: key.inst, Timeframe: key.timeframe,
		})
		if prob != nil || reply.CrossVenueSnapshot == nil {
			return nil, false
		}
		event = insights.CrossVenueSampledEvent{Metadata: events.NewMetadata(), CrossVenueSnapshot: *reply.CrossVenueSnapshot}
	default:
		return nil, false
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, false
	}
	frame, err := json.Marshal(clientFrame{Subject: subject, Event: eventJSON})
	if err != nil {
		return nil, false
	}
	return frame, true
}
