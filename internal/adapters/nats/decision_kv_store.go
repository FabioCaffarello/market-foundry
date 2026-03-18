package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/domain/decision"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const DecisionRSIOversoldLatestBucket = "DECISION_RSI_OVERSOLD_LATEST"

// DecisionKVStore persists the latest finalized decision per source/symbol/timeframe.
// One instance per decision type; the bucket name is injected at construction.
type DecisionKVStore struct {
	url    string
	bucket string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewDecisionKVStore(url, bucket string) *DecisionKVStore {
	return &DecisionKVStore{url: url, bucket: bucket}
}

func (s *DecisionKVStore) Start() error {
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

// Put stores a decision in the latest bucket with a monotonicity guard on Timestamp.
func (s *DecisionKVStore) Put(ctx context.Context, dec decision.Decision) (PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return PutWritten, problem.New(problem.Unavailable, "decision KV store is unavailable")
	}

	key := dec.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev decision.Decision
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(dec.Timestamp) {
				return PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(dec.Timestamp) {
				return PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(dec)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal decision for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put decision to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest decision for a given source/symbol/timeframe.
func (s *DecisionKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*decision.Decision, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "decision KV store is unavailable")
	}

	key := fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get decision from KV")
	}

	var dec decision.Decision
	if err := json.Unmarshal(entry.Value(), &dec); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal decision from KV")
	}
	return &dec, nil
}

func (s *DecisionKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
