package monitoringclient

import "internal/domain/monitoring"

// OperationalStateQuery is the request contract for the consolidated
// operational state monitoring endpoint. It requires no parameters —
// the endpoint returns the current system snapshot.
type OperationalStateQuery struct{}

// OperationalStateReply is the response contract for the operational state query.
type OperationalStateReply struct {
	State monitoring.OperationalState `json:"state"`
}
