package main

import (
	"context"
	"testing"

	"internal/application/evidenceclient"
	configctlcontracts "internal/application/configctl/contracts"
	"internal/shared/problem"
	"internal/shared/settings"
)

type readinessConfigctlGatewayStub struct {
	listConfigsReply configctlcontracts.ListConfigsReply
	listConfigsProb  *problem.Problem
}

func (s *readinessConfigctlGatewayStub) CreateDraft(context.Context, configctlcontracts.CreateDraftCommand) (configctlcontracts.CreateDraftReply, *problem.Problem) {
	return configctlcontracts.CreateDraftReply{}, nil
}

func (s *readinessConfigctlGatewayStub) GetConfig(context.Context, configctlcontracts.GetConfigQuery) (configctlcontracts.GetConfigReply, *problem.Problem) {
	return configctlcontracts.GetConfigReply{}, nil
}

func (s *readinessConfigctlGatewayStub) GetActiveConfig(context.Context, configctlcontracts.GetActiveConfigQuery) (configctlcontracts.GetActiveConfigReply, *problem.Problem) {
	return configctlcontracts.GetActiveConfigReply{}, nil
}

func (s *readinessConfigctlGatewayStub) ListActiveRuntimeProjections(context.Context, configctlcontracts.ListActiveRuntimeProjectionsQuery) (configctlcontracts.ListActiveRuntimeProjectionsReply, *problem.Problem) {
	return configctlcontracts.ListActiveRuntimeProjectionsReply{}, nil
}

func (s *readinessConfigctlGatewayStub) ListActiveIngestionBindings(context.Context, configctlcontracts.ListActiveIngestionBindingsQuery) (configctlcontracts.ListActiveIngestionBindingsReply, *problem.Problem) {
	return configctlcontracts.ListActiveIngestionBindingsReply{}, nil
}

func (s *readinessConfigctlGatewayStub) ListConfigs(context.Context, configctlcontracts.ListConfigsQuery) (configctlcontracts.ListConfigsReply, *problem.Problem) {
	return s.listConfigsReply, s.listConfigsProb
}

func (s *readinessConfigctlGatewayStub) ValidateDraft(context.Context, configctlcontracts.ValidateDraftCommand) (configctlcontracts.ValidateDraftReply, *problem.Problem) {
	return configctlcontracts.ValidateDraftReply{}, nil
}

func (s *readinessConfigctlGatewayStub) ValidateConfig(context.Context, configctlcontracts.ValidateConfigCommand) (configctlcontracts.ValidateConfigReply, *problem.Problem) {
	return configctlcontracts.ValidateConfigReply{}, nil
}

func (s *readinessConfigctlGatewayStub) CompileConfig(context.Context, configctlcontracts.CompileConfigCommand) (configctlcontracts.CompileConfigReply, *problem.Problem) {
	return configctlcontracts.CompileConfigReply{}, nil
}

func (s *readinessConfigctlGatewayStub) ActivateConfig(context.Context, configctlcontracts.ActivateConfigCommand) (configctlcontracts.ActivateConfigReply, *problem.Problem) {
	return configctlcontracts.ActivateConfigReply{}, nil
}

type readinessEvidenceGatewayStub struct {
	reply evidenceclient.CandleLatestReply
	prob  *problem.Problem
}

func (s *readinessEvidenceGatewayStub) GetLatestCandle(_ context.Context, _ evidenceclient.CandleLatestQuery) (evidenceclient.CandleLatestReply, *problem.Problem) {
	return s.reply, s.prob
}

func (s *readinessEvidenceGatewayStub) GetCandleHistory(_ context.Context, _ evidenceclient.CandleHistoryQuery) (evidenceclient.CandleHistoryReply, *problem.Problem) {
	return evidenceclient.CandleHistoryReply{}, nil
}

func (s *readinessEvidenceGatewayStub) GetLatestTradeBurst(_ context.Context, _ evidenceclient.TradeBurstLatestQuery) (evidenceclient.TradeBurstLatestReply, *problem.Problem) {
	return evidenceclient.TradeBurstLatestReply{}, nil
}

func (s *readinessEvidenceGatewayStub) GetLatestVolume(_ context.Context, _ evidenceclient.VolumeLatestQuery) (evidenceclient.VolumeLatestReply, *problem.Problem) {
	return evidenceclient.VolumeLatestReply{}, nil
}

func TestGatewayReadinessCheckerPassesWithConfigctl(t *testing.T) {
	t.Parallel()

	checker := newGatewayReadinessChecker(settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true},
	}, &readinessConfigctlGatewayStub{}, nil)

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected readiness to pass, got %v", err)
	}
}

func TestGatewayReadinessCheckerPassesWithEvidenceStore(t *testing.T) {
	t.Parallel()

	checker := newGatewayReadinessChecker(settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true},
	}, &readinessConfigctlGatewayStub{}, &readinessEvidenceGatewayStub{})

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected readiness to pass, got %v", err)
	}
}

func TestGatewayReadinessCheckerPassesWhenEvidenceStoreIsUnavailable(t *testing.T) {
	t.Parallel()

	// Evidence store probe failure should NOT fail readiness — it's non-blocking.
	checker := newGatewayReadinessChecker(settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true},
	}, &readinessConfigctlGatewayStub{}, &readinessEvidenceGatewayStub{
		prob: problem.New(problem.Unavailable, "store is down"),
	})

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expected readiness to pass even when evidence store is unavailable, got %v", err)
	}
}

func TestGatewayReadinessCheckerFailsWhenNATSIsDisabled(t *testing.T) {
	t.Parallel()

	checker := newGatewayReadinessChecker(settings.AppConfig{}, &readinessConfigctlGatewayStub{}, nil)
	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("expected readiness to fail when nats is disabled")
	}
}

func TestGatewayReadinessCheckerFailsWhenConfigctlIsNil(t *testing.T) {
	t.Parallel()

	checker := newGatewayReadinessChecker(settings.AppConfig{
		NATS: settings.NATSConfig{Enabled: true},
	}, nil, nil)
	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("expected readiness to fail when configctl is nil")
	}
}
