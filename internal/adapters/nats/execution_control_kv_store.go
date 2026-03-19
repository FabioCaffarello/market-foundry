package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/domain/execution"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const ExecutionControlBucket = "EXECUTION_CONTROL"

// ExecutionControlKey is the KV key for the global execution gate.
const ExecutionControlKey = "global"

// ExecutionControlKVStore reads and writes the execution control gate from a NATS KV bucket.
// Used by:
//   - store query responder (read + write — serves gateway queries)
//   - derive publisher actor (read only — gate check before publishing)
type ExecutionControlKVStore struct {
	url    string
	nc     *nats.Conn
	bucket jetstream.KeyValue
}

func NewExecutionControlKVStore(url string) *ExecutionControlKVStore {
	return &ExecutionControlKVStore{url: url}
}

func (s *ExecutionControlKVStore) Start() error {
	nc, err := connect(s.url)
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultSetupTimeout)
	defer cancel()

	bucket, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   ExecutionControlBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 1 * 1024 * 1024, // 1 MB — control state is tiny
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", ExecutionControlBucket, err)
	}

	s.nc = nc
	s.bucket = bucket
	return nil
}

// Get retrieves the current execution control gate.
// Returns DefaultControlGate (active) if no gate entry exists (fail-open).
func (s *ExecutionControlKVStore) Get(ctx context.Context) (execution.ControlGate, *problem.Problem) {
	if s == nil || s.bucket == nil {
		return execution.DefaultControlGate(), problem.New(problem.Unavailable, "execution control KV store is unavailable")
	}

	entry, err := s.bucket.Get(ctx, ExecutionControlKey)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return execution.DefaultControlGate(), nil
		}
		return execution.DefaultControlGate(), problem.Wrap(err, problem.Unavailable, "get execution control from KV")
	}

	var gate execution.ControlGate
	if err := json.Unmarshal(entry.Value(), &gate); err != nil {
		return execution.DefaultControlGate(), problem.Wrap(err, problem.Internal, "unmarshal execution control from KV")
	}

	return gate, nil
}

// Put stores the execution control gate.
func (s *ExecutionControlKVStore) Put(ctx context.Context, gate execution.ControlGate) *problem.Problem {
	if s == nil || s.bucket == nil {
		return problem.New(problem.Unavailable, "execution control KV store is unavailable")
	}

	if !execution.ValidGateStatus(gate.Status) {
		return problem.New(problem.InvalidArgument, "gate status must be 'active' or 'halted'")
	}

	data, err := json.Marshal(gate)
	if err != nil {
		return problem.Wrap(err, problem.Internal, "marshal execution control for KV")
	}

	if _, err := s.bucket.Put(ctx, ExecutionControlKey, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put execution control to KV")
	}

	return nil
}

// IsHalted reads the gate and returns true if execution is halted.
// Returns false (active) on any error (fail-open).
func (s *ExecutionControlKVStore) IsHalted(ctx context.Context) bool {
	gate, prob := s.Get(ctx)
	if prob != nil {
		return false
	}
	return gate.IsHalted()
}

func (s *ExecutionControlKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
