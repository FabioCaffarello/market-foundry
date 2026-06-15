package natsdelivery

import (
	"context"
	"encoding/json"
	"testing"

	"internal/application/insightsclient"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/problem"
)

func TestParseInsightsSubject(t *testing.T) {
	vp, ok := parseInsightsSubject("insights.events.volumeprofile.sampled.binancef.btc_usdt_perpetual.60")
	if !ok || vp.family != "volumeprofile" || vp.source != "binancef" || vp.timeframe != 60 {
		t.Fatalf("VP parse: %+v ok=%v", vp, ok)
	}
	cv, ok := parseInsightsSubject("insights.events.crossvenue.sampled.crossvenue.btc_usdt_perpetual.60")
	if !ok || cv.family != "crossvenue" || cv.source != "" {
		t.Fatalf("crossvenue parse: %+v ok=%v", cv, ok)
	}

	bad := []string{
		"insights.events.volumeprofile.sampled.binancef.btc_usdt_perpetual.>", // wildcard
		"insights.events.>", // short wildcard
		"insights.events.volumeprofile.sampled.binancef.btc_usdt_perp",       // 6 parts
		"insights.events.tpo.sampled.binancef.btc_usdt_perpetual.abc",        // bad tf
		"insights.events.bogus.sampled.binancef.btc_usdt_perpetual.60",       // unknown family
		"insights.events.crossvenue.sampled.binancef.btc_usdt_perpetual.60",  // crossvenue wrong slot
		"market.events.volumeprofile.sampled.binancef.btc_usdt_perpetual.60", // wrong root
	}
	for _, s := range bad {
		if _, ok := parseInsightsSubject(s); ok {
			t.Fatalf("expected parse to reject %q", s)
		}
	}
}

type fakeSnapGateway struct {
	vp *insights.VolumeProfile
}

func (f *fakeSnapGateway) GetLatestVolumeProfile(_ context.Context, _ insightsclient.VolumeProfileLatestQuery) (insightsclient.VolumeProfileLatestReply, *problem.Problem) {
	return insightsclient.VolumeProfileLatestReply{VolumeProfile: f.vp}, nil
}

func (f *fakeSnapGateway) GetLatestTPOProfile(_ context.Context, _ insightsclient.TPOProfileLatestQuery) (insightsclient.TPOProfileLatestReply, *problem.Problem) {
	return insightsclient.TPOProfileLatestReply{}, nil
}

func (f *fakeSnapGateway) GetLatestCrossVenue(_ context.Context, _ insightsclient.CrossVenueLatestQuery) (insightsclient.CrossVenueLatestReply, *problem.Problem) {
	return insightsclient.CrossVenueLatestReply{}, nil
}

func TestKVSnapshotProvider_VolumeProfile(t *testing.T) {
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}
	subject := "insights.events.volumeprofile.sampled.binancef." + inst.SubjectToken() + ".60"

	vp := &insights.VolumeProfile{Source: "binancef", Instrument: inst, Timeframe: 60, BucketSize: "1"}
	p := NewKVSnapshotProvider(&fakeSnapGateway{vp: vp})

	frame, ok := p.Snapshot(subject)
	if !ok {
		t.Fatal("expected a snapshot frame")
	}
	var w struct {
		Subject string `json:"subject"`
		Event   struct {
			VolumeProfile struct {
				Source string `json:"source"`
			} `json:"volume_profile"`
		} `json:"event"`
	}
	if err := json.Unmarshal(frame, &w); err != nil {
		t.Fatalf("decode frame: %v (raw=%s)", err, frame)
	}
	if w.Subject != subject {
		t.Fatalf("subject=%q want %q", w.Subject, subject)
	}
	if w.Event.VolumeProfile.Source != "binancef" {
		t.Fatalf("event.volume_profile.source=%q want binancef", w.Event.VolumeProfile.Source)
	}

	// No KV data → no snapshot.
	if _, ok := NewKVSnapshotProvider(&fakeSnapGateway{vp: nil}).Snapshot(subject); ok {
		t.Fatal("expected no snapshot when KV is empty")
	}
	// Wildcard subscription → no snapshot.
	if _, ok := p.Snapshot("insights.events.volumeprofile.sampled.>"); ok {
		t.Fatal("expected no snapshot for a wildcard subject")
	}
}
