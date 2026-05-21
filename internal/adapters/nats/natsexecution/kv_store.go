package natsexecution

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/execution"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const PaperOrderLatestBucket = "EXECUTION_PAPER_ORDER_LATEST"

// KVStore persists the latest finalized execution intent per source/symbol/timeframe.
// One instance per execution type; the bucket name is injected at construction.
type KVStore struct {
	url    string
	bucket string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewKVStore(url, bucket string) *KVStore {
	return &KVStore{url: url, bucket: bucket}
}

func (s *KVStore) Start() error {
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
func (s *KVStore) Put(ctx context.Context, intent execution.ExecutionIntent) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "execution KV store is unavailable")
	}

	key := intent.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev execution.ExecutionIntent
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(intent.Timestamp) {
				return natskit.PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(intent.Timestamp) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(intent)
	if err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Internal, "marshal execution for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Unavailable, "put execution to KV")
	}

	return natskit.PutWritten, nil
}

// Get retrieves the latest execution intent for a given source/symbol/timeframe.
func (s *KVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*execution.ExecutionIntent, *problem.Problem) {
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

// Keys returns all partition keys currently stored in the bucket.
// Returns an empty slice when the bucket is empty or unavailable.
// S413: Enables lifecycle list queries by enumerating tracked partition keys.
func (s *KVStore) Keys(ctx context.Context) ([]string, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "execution KV store is unavailable")
	}

	lister, err := s.latest.ListKeys(ctx)
	if err != nil {
		// NATS returns an error when no keys exist — treat as empty.
		return []string{}, nil
	}

	var keys []string
	for key := range lister.Keys() {
		keys = append(keys, key)
	}

	return keys, nil
}

func (s *KVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
