package monitoringclient

import (
	"context"
	"time"

	"internal/application/executionclient"
	"internal/domain/monitoring"
	"internal/shared/problem"
)

// SessionLister lists sessions for operational state monitoring.
type SessionLister interface {
	Execute(context.Context, executionclient.SessionListQuery) (executionclient.SessionListReply, *problem.Problem)
}

// GateReader reads the execution control gate for operational state monitoring.
type GateReader interface {
	Execute(context.Context, executionclient.ExecutionControlQuery) (executionclient.ExecutionControlReply, *problem.Problem)
}

// GetOperationalStateUseCase aggregates session, gate, and surface availability
// into a single consolidated monitoring snapshot.
//
// S486: This use case exists to close the monitoring gap where an operator
// needed 3+ separate calls to understand "what is the system doing right now?"
type GetOperationalStateUseCase struct {
	sessionLister SessionLister
	gateReader    GateReader
	surfaces      monitoring.SurfaceAvailability
}

// NewGetOperationalStateUseCase creates the operational state use case.
// sessionLister and gateReader are optional — nil means that surface is unavailable.
// surfaces is the static availability snapshot captured at gateway composition time.
func NewGetOperationalStateUseCase(
	sessionLister SessionLister,
	gateReader GateReader,
	surfaces monitoring.SurfaceAvailability,
) *GetOperationalStateUseCase {
	return &GetOperationalStateUseCase{
		sessionLister: sessionLister,
		gateReader:    gateReader,
		surfaces:      surfaces,
	}
}

func (uc *GetOperationalStateUseCase) Execute(ctx context.Context, _ OperationalStateQuery) (OperationalStateReply, *problem.Problem) {
	state := monitoring.OperationalState{
		ObservedAt: time.Now().UTC(),
		Surfaces:   uc.surfaces,
	}

	// Fetch latest session (first in newest-first list).
	if uc.sessionLister != nil {
		reply, prob := uc.sessionLister.Execute(ctx, executionclient.SessionListQuery{})
		if prob == nil && len(reply.Sessions) > 0 {
			summary := monitoring.NewSessionSummary(reply.Sessions[0])
			state.Session = &summary
		}
	}

	// Fetch current gate state.
	if uc.gateReader != nil {
		reply, prob := uc.gateReader.Execute(ctx, executionclient.ExecutionControlQuery{})
		if prob == nil {
			gate := reply.Gate
			gs := monitoring.GateSummary{
				Status: string(gate.Status),
				Reason: gate.Reason,
			}
			if !gate.UpdatedAt.IsZero() {
				gs.UpdatedAt = &gate.UpdatedAt
			}
			state.Gate = &gs
		}
	}

	return OperationalStateReply{State: state}, nil
}
