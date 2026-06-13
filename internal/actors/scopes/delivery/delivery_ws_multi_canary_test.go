//go:build integration

package delivery_test

// delivery_ws_multi_canary_test.go — H-11.b canaries: the delivery
// pipeline now carries ALL insights families (volume profile, TPO,
// cross-venue) over the single 'deliver-insights' durable, demuxed by
// subject. Each canary isolates its run with a unique subject so the
// durable's replayed history can't flood or DropNewest-evict the
// asserted frame.
//
// Reuses canaryNATSURL / canaryConn from delivery_ws_canary_test.go.
// Requires a running NATS server at localhost:4222 (or NATS_URL).

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"log/slog"

	"internal/actors/scopes/delivery"
	"internal/adapters/nats/natsinsights"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/events"

	"github.com/anthdm/hollywood/actor"
)

func startDelivery(t *testing.T, url string) *delivery.Runtime {
	t.Helper()
	engine, err := actor.NewEngine(actor.NewEngineConfig())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	rt, err := delivery.Start(engine, url, slog.Default())
	if err != nil {
		t.Fatalf("delivery start: %v", err)
	}
	return rt
}

// readFrame reads one client frame (the {subject,event} wrapper) within
// the deadline, returning the subject.
func readFrame(t *testing.T, conn *canaryConn, d time.Duration) string {
	t.Helper()
	select {
	case raw := <-conn.frames:
		var w struct {
			Subject string `json:"subject"`
		}
		if err := json.Unmarshal(raw, &w); err != nil {
			t.Fatalf("decode client frame: %v (raw=%s)", err, raw)
		}
		return w.Subject
	case <-time.After(d):
		t.Fatal("timed out waiting for a delivery frame")
		return ""
	}
}

func TestDeliveryWS_TPO(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rt := startDelivery(t, url)
	defer func() { _ = rt.Close() }()

	inst, prob := instrument.New("ETH", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}
	source := fmt.Sprintf("h11btpo%d", time.Now().UnixNano())

	conn := newCanaryConn()
	handle := rt.Hub.Admit(conn)
	defer handle.Close()
	handle.Subscribe("insights.events.tpo.sampled." + source + ".>")
	time.Sleep(200 * time.Millisecond)

	pub := natsinsights.NewPublisher(url, source, natsinsights.DefaultRegistry())
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	openTime := time.Now().UTC().Truncate(time.Minute)
	if prob := pub.PublishTPOProfile(ctx, insights.TPOProfileSampledEvent{
		Metadata: events.NewMetadata(),
		TPOProfile: insights.TPOProfile{
			Source: source, Instrument: inst, Timeframe: 60,
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
		},
	}); prob != nil {
		t.Fatalf("publish tpo: %v", prob)
	}

	subject := readFrame(t, conn, 20*time.Second)
	if !strings.HasPrefix(subject, "insights.events.tpo.sampled."+source+".") {
		t.Fatalf("delivered subject = %q, want tpo family for source %q", subject, source)
	}
}

func TestDeliveryWS_CrossVenue(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rt := startDelivery(t, url)
	defer func() { _ = rt.Close() }()

	// Cross-venue has no source slot (subject ...crossvenue.sampled.crossvenue.<token>.<tf>),
	// so isolate via a unique synthetic base asset (uppercase alnum, ≤16).
	base := fmt.Sprintf("CV%d", time.Now().UnixNano()%1_000_000_000_000)
	inst, prob := instrument.New(base, "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New(%q): %v", base, prob)
	}

	conn := newCanaryConn()
	handle := rt.Hub.Admit(conn)
	defer handle.Close()
	handle.Subscribe("insights.events.crossvenue.sampled.crossvenue." + inst.SubjectToken() + ".>")
	time.Sleep(200 * time.Millisecond)

	pub := natsinsights.NewPublisher(url, "h11bcv", natsinsights.DefaultRegistry())
	if err := pub.Start(); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	defer func() { _ = pub.Close() }()

	openTime := time.Now().UTC().Truncate(time.Minute)
	if prob := pub.PublishCrossVenue(ctx, insights.CrossVenueSampledEvent{
		Metadata: events.NewMetadata(),
		CrossVenueSnapshot: insights.CrossVenueSnapshot{
			Instrument: inst, Timeframe: 60,
			OpenTime: openTime, CloseTime: openTime.Add(time.Minute),
		},
	}); prob != nil {
		t.Fatalf("publish cross-venue: %v", prob)
	}

	subject := readFrame(t, conn, 20*time.Second)
	want := "insights.events.crossvenue.sampled.crossvenue." + inst.SubjectToken() + "."
	if !strings.HasPrefix(subject, want) {
		t.Fatalf("delivered subject = %q, want prefix %q", subject, want)
	}
}

// TestDeliveryWS_MultiFamilyOneSession proves a single session with two
// subscriptions receives two different families — the multi-family +
// per-subject filtering capability of H-11.b.
func TestDeliveryWS_MultiFamilyOneSession(t *testing.T) {
	url := canaryNATSURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rt := startDelivery(t, url)
	defer func() { _ = rt.Close() }()

	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("instrument.New: %v", prob)
	}
	vpSource := fmt.Sprintf("h11bmvp%d", time.Now().UnixNano())
	tpoSource := fmt.Sprintf("h11bmtpo%d", time.Now().UnixNano())

	conn := newCanaryConn()
	handle := rt.Hub.Admit(conn)
	defer handle.Close()
	handle.Subscribe("insights.events.volumeprofile.sampled." + vpSource + ".>")
	handle.Subscribe("insights.events.tpo.sampled." + tpoSource + ".>")
	time.Sleep(200 * time.Millisecond)

	openTime := time.Now().UTC().Truncate(time.Minute)

	vpPub := natsinsights.NewPublisher(url, vpSource, natsinsights.DefaultRegistry())
	if err := vpPub.Start(); err != nil {
		t.Fatalf("vp publisher start: %v", err)
	}
	defer func() { _ = vpPub.Close() }()
	if prob := vpPub.PublishVolumeProfile(ctx, insights.VolumeProfileSampledEvent{
		Metadata:      events.NewMetadata(),
		VolumeProfile: insights.VolumeProfile{Source: vpSource, Instrument: inst, Timeframe: 60, BucketSize: "1", OpenTime: openTime, CloseTime: openTime.Add(time.Minute)},
	}); prob != nil {
		t.Fatalf("publish vp: %v", prob)
	}

	tpoPub := natsinsights.NewPublisher(url, tpoSource, natsinsights.DefaultRegistry())
	if err := tpoPub.Start(); err != nil {
		t.Fatalf("tpo publisher start: %v", err)
	}
	defer func() { _ = tpoPub.Close() }()
	if prob := tpoPub.PublishTPOProfile(ctx, insights.TPOProfileSampledEvent{
		Metadata:   events.NewMetadata(),
		TPOProfile: insights.TPOProfile{Source: tpoSource, Instrument: inst, Timeframe: 60, OpenTime: openTime, CloseTime: openTime.Add(time.Minute)},
	}); prob != nil {
		t.Fatalf("publish tpo: %v", prob)
	}

	gotVP, gotTPO := false, false
	deadline := time.After(25 * time.Second)
	for !(gotVP && gotTPO) {
		select {
		case raw := <-conn.frames:
			var w struct {
				Subject string `json:"subject"`
			}
			if err := json.Unmarshal(raw, &w); err != nil {
				t.Fatalf("decode client frame: %v (raw=%s)", err, raw)
			}
			switch {
			case strings.HasPrefix(w.Subject, "insights.events.volumeprofile.sampled."+vpSource+"."):
				gotVP = true
			case strings.HasPrefix(w.Subject, "insights.events.tpo.sampled."+tpoSource+"."):
				gotTPO = true
			default:
				t.Fatalf("unexpected frame subject %q", w.Subject)
			}
		case <-deadline:
			t.Fatalf("timed out; gotVP=%v gotTPO=%v", gotVP, gotTPO)
		}
	}
}
