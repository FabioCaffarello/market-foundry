package executionclient

import "internal/domain/execution"

// ExecutionControlQuery is the request contract for querying the execution control gate.
type ExecutionControlQuery struct{}

// ExecutionControlReply is the response contract for the execution control gate query.
type ExecutionControlReply struct {
	Gate execution.ControlGate `json:"gate"`
}

// SetExecutionControlCommand is the request contract for updating the execution control gate.
type SetExecutionControlCommand struct {
	Status    string `json:"status"`
	Reason    string `json:"reason,omitempty"`
	UpdatedBy string `json:"updated_by,omitempty"`
}
