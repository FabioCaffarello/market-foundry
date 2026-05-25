package v1

import (
	"strings"
	"testing"
	"time"
)

// TestRoundTrip_AllFieldsPresent validates that ToProto followed by
// FromProto preserves every canonical field, including the optional
// TsExchange when populated.
func TestRoundTrip_AllFieldsPresent(t *testing.T) {
	tsExchange := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	original := CanonicalEvent{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     &tsExchange,
		TsIngest:       time.Unix(0, 1_700_000_000_500_000_000).UTC(),
		Seq:            42,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:42",
		Payload:        []byte("dummy-payload-bytes"),
	}

	proto, err := ToProto(original)
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}

	recovered, err := FromProto(proto)
	if err != nil {
		t.Fatalf("FromProto: %v", err)
	}

	if recovered.Type != original.Type {
		t.Errorf("Type: got %q, want %q", recovered.Type, original.Type)
	}
	if recovered.Version != original.Version {
		t.Errorf("Version: got %d, want %d", recovered.Version, original.Version)
	}
	if recovered.Venue != original.Venue {
		t.Errorf("Venue: got %q, want %q", recovered.Venue, original.Venue)
	}
	if recovered.Instrument != original.Instrument {
		t.Errorf("Instrument: got %q, want %q", recovered.Instrument, original.Instrument)
	}
	if recovered.TsExchange == nil || !recovered.TsExchange.Equal(*original.TsExchange) {
		t.Errorf("TsExchange: got %v, want %v", recovered.TsExchange, original.TsExchange)
	}
	if !recovered.TsIngest.Equal(original.TsIngest) {
		t.Errorf("TsIngest: got %v, want %v", recovered.TsIngest, original.TsIngest)
	}
	if recovered.Seq != original.Seq {
		t.Errorf("Seq: got %d, want %d", recovered.Seq, original.Seq)
	}
	if recovered.IdempotencyKey != original.IdempotencyKey {
		t.Errorf("IdempotencyKey: got %q, want %q", recovered.IdempotencyKey, original.IdempotencyKey)
	}
	if string(recovered.Payload) != string(original.Payload) {
		t.Errorf("Payload: got %q, want %q", recovered.Payload, original.Payload)
	}
}

// TestRoundTrip_TsExchangeAbsent validates that a nil TsExchange on
// the input preserves as nil through ToProto + FromProto, exercising
// ADR-0017's "absent when unknown" semantic for the optional field.
func TestRoundTrip_TsExchangeAbsent(t *testing.T) {
	original := CanonicalEvent{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsExchange:     nil, // explicit absence
		TsIngest:       time.Unix(0, 1_700_000_000_500_000_000).UTC(),
		Seq:            7,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:7",
		Payload:        []byte("synthetic-event-no-venue-time"),
	}

	proto, err := ToProto(original)
	if err != nil {
		t.Fatalf("ToProto: %v", err)
	}
	if proto.TsExchange != nil {
		t.Fatalf("proto TsExchange: expected nil, got %d", *proto.TsExchange)
	}

	recovered, err := FromProto(proto)
	if err != nil {
		t.Fatalf("FromProto: %v", err)
	}
	if recovered.TsExchange != nil {
		t.Fatalf("CanonicalEvent TsExchange after round-trip: expected nil, got %v", recovered.TsExchange)
	}
}

// validEvent returns a CanonicalEvent with every required field
// populated. Tests start from this and clear individual fields to
// drive the negative validation cases.
func validEvent() CanonicalEvent {
	return CanonicalEvent{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsIngest:       time.Unix(0, 1_700_000_000_500_000_000).UTC(),
		Seq:            1,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:1",
		Payload:        []byte("x"),
	}
}

func TestToProto_RequiredFieldValidation(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*CanonicalEvent)
		wantInErr string
	}{
		{
			name:      "missing type",
			mutate:    func(e *CanonicalEvent) { e.Type = "" },
			wantInErr: "Type is required",
		},
		{
			name:      "missing version",
			mutate:    func(e *CanonicalEvent) { e.Version = 0 },
			wantInErr: "Version is required",
		},
		{
			name:      "missing venue",
			mutate:    func(e *CanonicalEvent) { e.Venue = "" },
			wantInErr: "Venue is required",
		},
		{
			name:      "missing instrument",
			mutate:    func(e *CanonicalEvent) { e.Instrument = "" },
			wantInErr: "Instrument is required",
		},
		{
			name:      "missing ts_ingest",
			mutate:    func(e *CanonicalEvent) { e.TsIngest = time.Time{} },
			wantInErr: "TsIngest is required",
		},
		{
			name:      "missing idempotency_key",
			mutate:    func(e *CanonicalEvent) { e.IdempotencyKey = "" },
			wantInErr: "IdempotencyKey is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev := validEvent()
			tc.mutate(&ev)
			_, err := ToProto(ev)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantInErr)
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("error message %q does not contain %q", err.Error(), tc.wantInErr)
			}
		})
	}
}

// validProto returns a proto Envelope with every required field
// populated. Tests start from this and clear individual fields to
// drive the negative validation cases on the FromProto side.
func validProto() *Envelope {
	return &Envelope{
		Type:           "observation.trade",
		Version:        1,
		Venue:          "binance",
		Instrument:     "BTC/USDT-spot",
		TsIngest:       1_700_000_000_500_000_000,
		Seq:            1,
		IdempotencyKey: "binance:BTC/USDT-spot:observation.trade:1",
		Payload:        []byte("x"),
	}
}

func TestFromProto_RequiredFieldValidation(t *testing.T) {
	cases := []struct {
		name      string
		mutate    func(*Envelope)
		wantInErr string
	}{
		{
			name:      "missing type",
			mutate:    func(e *Envelope) { e.Type = "" },
			wantInErr: "type is required",
		},
		{
			name:      "missing version",
			mutate:    func(e *Envelope) { e.Version = 0 },
			wantInErr: "version is required",
		},
		{
			name:      "missing venue",
			mutate:    func(e *Envelope) { e.Venue = "" },
			wantInErr: "venue is required",
		},
		{
			name:      "missing instrument",
			mutate:    func(e *Envelope) { e.Instrument = "" },
			wantInErr: "instrument is required",
		},
		{
			name:      "missing ts_ingest",
			mutate:    func(e *Envelope) { e.TsIngest = 0 },
			wantInErr: "ts_ingest is required",
		},
		{
			name:      "missing idempotency_key",
			mutate:    func(e *Envelope) { e.IdempotencyKey = "" },
			wantInErr: "idempotency_key is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pe := validProto()
			tc.mutate(pe)
			_, err := FromProto(pe)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantInErr)
			}
			if !strings.Contains(err.Error(), tc.wantInErr) {
				t.Fatalf("error message %q does not contain %q", err.Error(), tc.wantInErr)
			}
		})
	}

	t.Run("nil envelope", func(t *testing.T) {
		_, err := FromProto(nil)
		if err == nil {
			t.Fatalf("expected error for nil envelope, got nil")
		}
		if !strings.Contains(err.Error(), "nil") {
			t.Fatalf("error message %q should mention nil", err.Error())
		}
	})
}
