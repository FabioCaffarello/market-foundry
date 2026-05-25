package replay_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	envv1 "internal/shared/contracts/envelope/v1"
	"internal/shared/replay"
)

const baseInstantNanos int64 = 1700000000000000000

func sampleEvent(seq int64) envv1.CanonicalEvent {
	exchange := time.Unix(0, baseInstantNanos+seq*int64(time.Millisecond)).UTC()
	ingest := exchange.Add(50 * time.Microsecond)
	return envv1.CanonicalEvent{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     &exchange,
		TsIngest:       ingest,
		Seq:            seq,
		IdempotencyKey: fmt.Sprintf("key-%04d", seq),
		Payload:        []byte(fmt.Sprintf(`{"p":50000,"s":%d}`, seq)),
	}
}

func TestRecorder_CapturesSequence(t *testing.T) {
	r := replay.NewRecorder()
	for i := int64(0); i < 100; i++ {
		r.Record(sampleEvent(i))
	}
	if r.Len() != 100 {
		t.Fatalf("Len = %d, want 100", r.Len())
	}

	var buf bytes.Buffer
	n, err := r.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != int64(buf.Len()) {
		t.Fatalf("WriteTo n=%d but buf=%d", n, buf.Len())
	}
	if got := strings.Count(buf.String(), "\n"); got != 100 {
		t.Fatalf("expected 100 newlines; got %d", got)
	}
}

func TestRoundTrip_RecorderPlayer(t *testing.T) {
	original := make([]envv1.CanonicalEvent, 100)
	r := replay.NewRecorder()
	for i := int64(0); i < 100; i++ {
		e := sampleEvent(i)
		original[i] = e
		r.Record(e)
	}

	var buf bytes.Buffer
	if _, err := r.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	p, err := replay.NewPlayer(&buf)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if p.Len() != 100 {
		t.Fatalf("Player.Len = %d, want 100", p.Len())
	}

	for i, want := range original {
		got, ok := p.Next()
		if !ok {
			t.Fatalf("Next exhausted at i=%d", i)
		}
		if got.Type != want.Type {
			t.Errorf("event %d Type: got %q want %q", i, got.Type, want.Type)
		}
		if got.Version != want.Version {
			t.Errorf("event %d Version: got %d want %d", i, got.Version, want.Version)
		}
		if got.Venue != want.Venue {
			t.Errorf("event %d Venue: got %q want %q", i, got.Venue, want.Venue)
		}
		if got.Instrument != want.Instrument {
			t.Errorf("event %d Instrument: got %q want %q", i, got.Instrument, want.Instrument)
		}
		if got.Seq != want.Seq {
			t.Errorf("event %d Seq: got %d want %d", i, got.Seq, want.Seq)
		}
		if got.IdempotencyKey != want.IdempotencyKey {
			t.Errorf("event %d IdempotencyKey: got %q want %q", i, got.IdempotencyKey, want.IdempotencyKey)
		}
		if !got.TsIngest.Equal(want.TsIngest) {
			t.Errorf("event %d TsIngest: got %s want %s", i, got.TsIngest, want.TsIngest)
		}
		if got.TsIngest.Location() != time.UTC {
			t.Errorf("event %d TsIngest location: got %s want UTC", i, got.TsIngest.Location())
		}
		if got.TsExchange == nil {
			t.Errorf("event %d TsExchange: got nil, want non-nil", i)
		} else if !got.TsExchange.Equal(*want.TsExchange) {
			t.Errorf("event %d TsExchange: got %s want %s", i, got.TsExchange, want.TsExchange)
		} else if got.TsExchange.Location() != time.UTC {
			t.Errorf("event %d TsExchange location: got %s want UTC", i, got.TsExchange.Location())
		}
		if !bytes.Equal(got.Payload, want.Payload) {
			t.Errorf("event %d Payload mismatch: got %q want %q", i, got.Payload, want.Payload)
		}
	}
	if _, ok := p.Next(); ok {
		t.Fatal("expected Player exhausted after 100 reads")
	}
}

func TestPlayer_ReplaysDeterministic(t *testing.T) {
	r := replay.NewRecorder()
	for i := int64(0); i < 50; i++ {
		r.Record(sampleEvent(i))
	}
	var fixture bytes.Buffer
	if _, err := r.WriteTo(&fixture); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	source := fixture.Bytes()

	var prev []byte
	for run := 0; run < 10; run++ {
		p, err := replay.NewPlayer(bytes.NewReader(source))
		if err != nil {
			t.Fatalf("run %d NewPlayer: %v", run, err)
		}
		r2 := replay.NewRecorder()
		for {
			ce, ok := p.Next()
			if !ok {
				break
			}
			r2.Record(ce)
		}
		var buf bytes.Buffer
		if _, err := r2.WriteTo(&buf); err != nil {
			t.Fatalf("run %d WriteTo: %v", run, err)
		}
		if prev == nil {
			prev = append([]byte{}, buf.Bytes()...)
			continue
		}
		if !bytes.Equal(prev, buf.Bytes()) {
			t.Fatalf("run %d output diverged from previous run", run)
		}
	}
}

func TestRoundTrip_NilTsExchange(t *testing.T) {
	e := envv1.CanonicalEvent{
		Type:           "observation.synthetic",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     nil,
		TsIngest:       time.Unix(0, baseInstantNanos).UTC(),
		Seq:            0,
		IdempotencyKey: "key-nil-ts",
		Payload:        []byte("payload"),
	}
	r := replay.NewRecorder()
	r.Record(e)
	var buf bytes.Buffer
	if _, err := r.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if !strings.Contains(buf.String(), `"ts_exchange":null`) {
		t.Errorf("expected explicit null for ts_exchange in fixture; got: %s", buf.String())
	}
	p, err := replay.NewPlayer(&buf)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	got, ok := p.Next()
	if !ok {
		t.Fatal("expected one event")
	}
	if got.TsExchange != nil {
		t.Errorf("expected nil TsExchange after round-trip; got %v", got.TsExchange)
	}
}

func TestRoundTrip_EmptyPayload(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"nil", nil},
		{"empty_slice", []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := envv1.CanonicalEvent{
				Type:           "observation.synthetic",
				Version:        1,
				Venue:          "binance",
				Instrument:     "BTC/USDT-spot",
				TsIngest:       time.Unix(0, baseInstantNanos).UTC(),
				Seq:            0,
				IdempotencyKey: "key-empty",
				Payload:        tc.payload,
			}
			r := replay.NewRecorder()
			r.Record(e)
			var buf bytes.Buffer
			if _, err := r.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			if !strings.Contains(buf.String(), `"payload":""`) {
				t.Errorf("expected canonical empty payload \"\" in fixture; got: %s", buf.String())
			}
			if strings.Contains(buf.String(), `"payload":null`) {
				t.Errorf("expected no null payload in fixture; got: %s", buf.String())
			}
			p, err := replay.NewPlayer(&buf)
			if err != nil {
				t.Fatalf("NewPlayer: %v", err)
			}
			got, ok := p.Next()
			if !ok {
				t.Fatal("expected one event")
			}
			if got.Payload == nil {
				t.Error("Payload should be []byte{} after normalize, not nil")
			}
			if len(got.Payload) != 0 {
				t.Errorf("Payload length = %d, want 0", len(got.Payload))
			}
		})
	}
}

func TestPlayer_NextExhaustionAndReset(t *testing.T) {
	r := replay.NewRecorder()
	r.Record(sampleEvent(0))
	r.Record(sampleEvent(1))
	var buf bytes.Buffer
	if _, err := r.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	p, err := replay.NewPlayer(&buf)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if _, ok := p.Next(); !ok {
		t.Fatal("first Next failed")
	}
	if _, ok := p.Next(); !ok {
		t.Fatal("second Next failed")
	}
	if _, ok := p.Next(); ok {
		t.Fatal("third Next should be exhausted")
	}
	p.Reset()
	if _, ok := p.Next(); !ok {
		t.Fatal("Next after Reset failed")
	}
}

func TestRecorder_EmptyProducesEmptyOutput(t *testing.T) {
	r := replay.NewRecorder()
	var buf bytes.Buffer
	n, err := r.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if n != 0 || buf.Len() != 0 {
		t.Fatalf("expected empty output for empty recorder; got n=%d buf.Len=%d", n, buf.Len())
	}
}

func TestNewPlayer_EmptyFixture(t *testing.T) {
	p, err := replay.NewPlayer(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if p.Len() != 0 {
		t.Errorf("Len = %d, want 0", p.Len())
	}
	if _, ok := p.Next(); ok {
		t.Fatal("expected exhausted Player from empty fixture")
	}
}

func TestNewPlayer_MalformedLineReturnsError(t *testing.T) {
	bad := []byte(`{"type":"observation.trade","version":1}` + "\n" + `not-json` + "\n")
	_, err := replay.NewPlayer(bytes.NewReader(bad))
	if err == nil {
		t.Fatal("expected error parsing malformed line")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("expected error to identify line 2; got: %v", err)
	}
}

func TestRecorder_Events_ReturnsCopy(t *testing.T) {
	r := replay.NewRecorder()
	r.Record(sampleEvent(0))
	r.Record(sampleEvent(1))
	got := r.Events()
	if len(got) != 2 {
		t.Fatalf("Events len = %d, want 2", len(got))
	}
	got[0].Type = "mutated"
	again := r.Events()
	if again[0].Type == "mutated" {
		t.Error("Events() returned a slice sharing underlying storage; expected a copy")
	}
}

func TestWriteThenRead_ByteStable(t *testing.T) {
	r := replay.NewRecorder()
	for i := int64(0); i < 20; i++ {
		r.Record(sampleEvent(i))
	}
	var first bytes.Buffer
	if _, err := r.WriteTo(&first); err != nil {
		t.Fatalf("first WriteTo: %v", err)
	}

	p, err := replay.NewPlayer(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	r2 := replay.NewRecorder()
	for {
		ce, ok := p.Next()
		if !ok {
			break
		}
		r2.Record(ce)
	}
	var second bytes.Buffer
	if _, err := r2.WriteTo(&second); err != nil {
		t.Fatalf("second WriteTo: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("round-trip not byte-stable\nfirst:  %s\nsecond: %s", first.String(), second.String())
	}
}
