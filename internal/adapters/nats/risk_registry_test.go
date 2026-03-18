package nats

import (
	"testing"
)

func TestDefaultRiskRegistry_StreamName(t *testing.T) {
	reg := DefaultRiskRegistry()
	if reg.PositionExposureAssessed.Stream.Name != "RISK_EVENTS" {
		t.Fatalf("expected RISK_EVENTS, got %s", reg.PositionExposureAssessed.Stream.Name)
	}
}

func TestDefaultRiskRegistry_EventSubject(t *testing.T) {
	reg := DefaultRiskRegistry()
	expected := "risk.events.position_exposure.assessed"
	if reg.PositionExposureAssessed.Subject != expected {
		t.Fatalf("expected %s, got %s", expected, reg.PositionExposureAssessed.Subject)
	}
}

func TestDefaultRiskRegistry_QuerySubject(t *testing.T) {
	reg := DefaultRiskRegistry()
	expected := "risk.query.position_exposure.latest"
	if reg.PositionExposureLatest.Subject != expected {
		t.Fatalf("expected %s, got %s", expected, reg.PositionExposureLatest.Subject)
	}
}

func TestRiskRegistry_LatestSpecByType_Known(t *testing.T) {
	reg := DefaultRiskRegistry()
	spec, ok := reg.LatestSpecByType("position_exposure")
	if !ok {
		t.Fatal("expected position_exposure to be registered")
	}
	if spec.Subject != "risk.query.position_exposure.latest" {
		t.Fatalf("unexpected subject: %s", spec.Subject)
	}
}

func TestRiskRegistry_LatestSpecByType_Unknown(t *testing.T) {
	reg := DefaultRiskRegistry()
	_, ok := reg.LatestSpecByType("unknown_risk")
	if ok {
		t.Fatal("expected unknown risk type to return false")
	}
}

func TestStorePositionExposureRiskConsumer(t *testing.T) {
	spec := StorePositionExposureRiskConsumer()
	if spec.Durable != "store-risk-position-exposure" {
		t.Fatalf("expected durable store-risk-position-exposure, got %s", spec.Durable)
	}
	if spec.Event.Stream.Name != "RISK_EVENTS" {
		t.Fatalf("expected stream RISK_EVENTS, got %s", spec.Event.Stream.Name)
	}
}
