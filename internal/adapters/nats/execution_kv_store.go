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

const ExecutionPaperOrderLatestBucket = "EXECUTION_PAPER_ORDER_LATEST"

// ExecutionKVStore persists the latest finalized execution intent per source/symbol/timeframe.
// One instance per execution type; the bucket name is injected at construction.
type ExecutionKVStore struct {
	url    string
	bucket string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewExecutionKVStore(url, bucket string) *ExecutionKVStore {
	return &ExecutionKVStore{url: url, bucket: bucket}
}

func (s *ExecutionKVStore) Start() error {
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

	latest, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   s.bucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 64 * 1024 * 1024, // 64 MB
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", s.bucket, err)
	}

	s.nc = nc
	s.latest = latest
	return nil
}

// Put stores an execution intent in the latest bucket with a monotonicity guard on Timestamp.
func (s *ExecutionKVStore) Put(ctx context.Context, intent execution.ExecutionIntent) (PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return PutWritten, problem.New(problem.Unavailable, "execution KV store is unavailable")
	}

	key := intent.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev execution.ExecutionIntent
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(intent.Timestamp) {
				return PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(intent.Timestamp) {
				return PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(intent)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal execution for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put execution to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest execution intent for a given source/symbol/timeframe.
func (s *ExecutionKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*execution.ExecutionIntent, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "execution KV store is unavailable")
	}

	key := fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get execution from KV")
	}

	var intent execution.ExecutionIntent
	if err := json.Unmarshal(entry.Value(), &intent); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal execution from KV")
	}

	// Post-read validation: detect corrupted or incomplete data in KV.
	if prob := intent.Validate(); prob != nil {
		return nil, problem.New(problem.Internal, "execution KV entry failed validation: "+prob.Message)
	}

	return &intent, nil
}

func (s *ExecutionKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
