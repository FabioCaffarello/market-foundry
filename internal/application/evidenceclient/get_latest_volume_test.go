package evidenceclient

import (
	"context"
	"testing"
	"time"

	"internal/domain/evidence"
	"internal/domain/instrument"
	"internal/shared/problem"
)

func btcUSDTPerpInternal(t *testing.T) instrument.CanonicalInstrument {
	t.Helper()
	inst, prob := instrument.New("BTC", "USDT", instrument.ContractPerpetual)
	if prob != nil {
		t.Fatalf("setup: %v", prob)
	}
	return inst
}

type volumeGatewayStub2 struct {
	vol  *evidence.EvidenceVolume
	prob *problem.Problem
}

func (s volumeGatewayStub2) GetLatestVolume(_ context.Context, _ VolumeLatestQuery) (VolumeLatestReply, *problem.Problem) {
	return VolumeLatestReply{Volume: s.vol}, s.prob
}

func TestGetLatestVolumeUseCase_Validation(t *testing.T) {
	t.Parallel()

	uc := NewGetLatestVolumeUseCase(volumeGatewayStub2{})

	tests := []struct {
		name  string
		query VolumeLatestQuery
	}{
		{"empty source", VolumeLatestQuery{Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60}},
		{"empty symbol", VolumeLatestQuery{Source: "binancef", Timeframe: 60}},
		{"zero timeframe", VolumeLatestQuery{Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 0}},
		{"negative timeframe", VolumeLatestQuery{Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, prob := uc.Execute(context.Background(), tt.query)
			if prob == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGetLatestVolumeUseCase_ReturnsVolume(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(60 * time.Second)
	vol := &evidence.EvidenceVolume{
		Source: "binancef", Instrument: btcUSDTPerpInternal(t), Timeframe: 60,
		BuyVolume: "100000.00", SellVolume: "50000.00",
		TotalVolume: "150000.00", VWAP: "50000.00",
		TradeCount: 42,
		OpenTime:   now, CloseTime: now.Add(60 * time.Second),
		Final: true,
	}

	uc := NewGetLatestVolumeUseCase(volumeGatewayStub2{vol: vol})
	reply, prob := uc.Execute(context.Background(), VolumeLatestQuery{
		Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob != nil {
		t.Fatalf("unexpected error: %s", prob.Message)
	}
	if reply.Volume == nil {
		t.Fatal("expected volume in reply")
	}
	if reply.Volume.VWAP != "50000.00" {
		t.Fatalf("expected VWAP 50000.00, got %s", reply.Volume.VWAP)
	}
}

func TestGetLatestVolumeUseCase_NilGateway(t *testing.T) {
	t.Parallel()

	uc := NewGetLatestVolumeUseCase(nil)
	_, prob := uc.Execute(context.Background(), VolumeLatestQuery{
		Source: "binancef", Instrument: instrument.CanonicalInstrument{Base: "BTC", Quote: "USDT", Contract: instrument.ContractPerpetual}, Timeframe: 60,
	})
	if prob == nil {
		t.Fatal("expected error for nil gateway")
	}
}
