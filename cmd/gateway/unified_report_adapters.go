package main

import (
	"context"

	"internal/application/monitoringclient"
	"internal/application/triageclient"
	"internal/shared/problem"
)

// monitoringReportAdapter bridges GetOperationalStateUseCase to the
// UnifiedReportMonitoringReader interface expected by the unified report.
// S491: Enables the unified report to capture gate and surface state.
type monitoringReportAdapter struct {
	uc *monitoringclient.GetOperationalStateUseCase
}

func (a *monitoringReportAdapter) GetOperationalState(ctx context.Context) (string, string, []string, *problem.Problem) {
	reply, prob := a.uc.Execute(ctx, monitoringclient.OperationalStateQuery{})
	if prob != nil {
		return "", "", nil, prob
	}

	state := reply.State
	gate := ""
	gateReason := ""
	if state.Gate != nil {
		gate = state.Gate.Status
		gateReason = state.Gate.Reason
	}

	var surfaces []string
	sa := state.Surfaces
	add := func(name string, avail bool) {
		if avail {
			surfaces = append(surfaces, name)
		}
	}
	add("evidence", sa.Evidence)
	add("signal", sa.Signal)
	add("decision", sa.Decision)
	add("strategy", sa.Strategy)
	add("risk", sa.Risk)
	add("execution", sa.Execution)
	add("session", sa.Session)
	add("analytical", sa.Analytical)
	add("activation", sa.Activation)

	return gate, gateReason, surfaces, nil
}

// triageReportAdapter bridges GetTriageOverviewUseCase to the
// UnifiedReportTriageReader interface expected by the unified report.
// S491: Enables the unified report to capture cross-domain anomaly counts.
type triageReportAdapter struct {
	uc *triageclient.GetTriageOverviewUseCase
}

func (a *triageReportAdapter) GetTriageSummary(ctx context.Context) (int, int, int, int, int, int, int, []string, *problem.Problem) {
	reply, prob := a.uc.Execute(ctx, triageclient.TriageOverviewQuery{})
	if prob != nil {
		return 0, 0, 0, 0, 0, 0, 0, nil, prob
	}

	ov := reply.Overview

	var findings []string
	for _, f := range ov.TopFindings {
		findings = append(findings, f.Detail)
	}

	return ov.TotalAnomalies,
		ov.SessionSummary.Critical, ov.SessionSummary.Warning,
		ov.DecisionSummary.Critical, ov.DecisionSummary.Warning,
		ov.RoundTripSummary.Critical, ov.RoundTripSummary.Warning,
		findings, nil
}
