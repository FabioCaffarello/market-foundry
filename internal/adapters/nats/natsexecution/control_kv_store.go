package natsexecution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/execution"
	"internal/shared/clock"
	"internal/shared/metrics"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const ControlBucket = "EXECUTION_CONTROL"

// ControlKey is the KV key for the global execution gate.
const ControlKey = "global"

// DimensionsKey is the KV key for process-local activation dimensions
// published by the execute binary at startup.
const DimensionsKey = "dimensions"

// ControlKVStore reads and writes the execution control gate from a NATS KV bucket.
// Used by:
//   - store query responder (read + write — serves gateway queries)
//   - derive publisher actor (read only — gate check before publishing)
type ControlKVStore struct {
	url    string
	nc     *nats.Conn
	bucket jetstream.KeyValue
	// clk is the time port used when materializing default gate
	// values; defaults to clock.SystemClock{} via NewControlKVStore
	// and can be overridden via WithClock for tests / replay. Not
	// consumed in this commit — call sites that read clk land in
	// commit 6b (DefaultControlGate migration to clock.Clock).
	clk clock.Clock
}

func NewControlKVStore(url string) *ControlKVStore {
	return &ControlKVStore{url: url, clk: clock.SystemClock{}}
}

// WithClock overrides the Clock used by this store for time-
// sourced fields. Returns the store to allow chaining, e.g.:
//
//	store := natsexecution.NewControlKVStore(url).WithClock(testClock)
//
// Optional; defaults to clock.SystemClock{}.
func (s *ControlKVStore) WithClock(clk clock.Clock) *ControlKVStore {
	if s != nil && clk != nil {
		s.clk = clk
	}
	return s
}

func (s *ControlKVStore) Start() error {
	nc, err := natskit.Connect(s.url)
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), natskit.DefaultSetupTimeout)
	defer cancel()

	bucket, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   ControlBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 1 * 1024 * 1024, // 1 MB — control state is tiny
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", ControlBucket, err)
	}

	s.nc = nc
	s.bucket = bucket
	return nil
}

// Get retrieves the current execution control gate.
// Returns DefaultControlGate (active) if no gate entry exists (fail-open).
func (s *ControlKVStore) Get(ctx context.Context) (execution.ControlGate, *problem.Problem) {
	if s == nil || s.bucket == nil {
		return execution.DefaultControlGate(), problem.New(problem.Unavailable, "execution control KV store is unavailable")
	}

	entry, err := s.bucket.Get(ctx, ControlKey)
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
func (s *ControlKVStore) Put(ctx context.Context, gate execution.ControlGate) *problem.Problem {
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

	if _, err := s.bucket.Put(ctx, ControlKey, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put execution control to KV")
	}

	return nil
}

// GetDimensions retrieves the process-local activation dimensions from KV.
// Returns nil if no dimensions have been published yet.
func (s *ControlKVStore) GetDimensions(ctx context.Context) (*execution.ActivationDimensions, *problem.Problem) {
	if s == nil || s.bucket == nil {
		return nil, problem.New(problem.Unavailable, "execution control KV store is unavailable")
	}

	entry, err := s.bucket.Get(ctx, DimensionsKey)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get activation dimensions from KV")
	}

	var dims execution.ActivationDimensions
	if err := json.Unmarshal(entry.Value(), &dims); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal activation dimensions from KV")
	}

	return &dims, nil
}

// PutDimensions stores the process-local activation dimensions.
// Called once by the execute binary at startup.
func (s *ControlKVStore) PutDimensions(ctx context.Context, dims execution.ActivationDimensions) *problem.Problem {
	if s == nil || s.bucket == nil {
		return problem.New(problem.Unavailable, "execution control KV store is unavailable")
	}

	data, err := json.Marshal(dims)
	if err != nil {
		return problem.Wrap(err, problem.Internal, "marshal activation dimensions for KV")
	}

	if _, err := s.bucket.Put(ctx, DimensionsKey, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put activation dimensions to KV")
	}

	return nil
}

// IsHalted reads the gate and returns true if execution is halted.
// Returns false (active) on any error (fail-open) and increments the
// gate_read_failures_total counter so the silent failure mode is
// monitorable. See ADR 0012 for the posture rationale.
func (s *ControlKVStore) IsHalted(ctx context.Context) bool {
	if s == nil || s.bucket == nil {
		metrics.IncGateReadFailure(metrics.GateReadFailureNilBucket)
		return false
	}

	entry, err := s.bucket.Get(ctx, ControlKey)
	if err != nil {
		switch {
		case errors.Is(err, jetstream.ErrKeyNotFound):
			metrics.IncGateReadFailure(metrics.GateReadFailureKeyNotFound)
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			metrics.IncGateReadFailure(metrics.GateReadFailureCtxTimeout)
		default:
			metrics.IncGateReadFailure(metrics.GateReadFailureKVError)
		}
		return false
	}

	var gate execution.ControlGate
	if err := json.Unmarshal(entry.Value(), &gate); err != nil {
		metrics.IncGateReadFailure(metrics.GateReadFailureUnmarshal)
		return false
	}
	return gate.IsHalted()
}

func (s *ControlKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
