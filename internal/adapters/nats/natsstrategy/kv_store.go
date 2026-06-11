package natsstrategy

import (
	"internal/domain/instrument"

	"context"
	"encoding/json"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/strategy"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const MeanReversionEntryLatestBucket = "STRATEGY_MEAN_REVERSION_ENTRY_LATEST"
const TrendFollowingEntryLatestBucket = "STRATEGY_TREND_FOLLOWING_ENTRY_LATEST"

// KVStore persists the latest finalized strategy per source/symbol/timeframe.
// One instance per strategy type; the bucket name is injected at construction.
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

// Put stores a strategy in the latest bucket with a monotonicity guard on Timestamp.
func (s *KVStore) Put(ctx context.Context, strat strategy.Strategy) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "strategy KV store is unavailable")
	}

	key := strat.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev strategy.Strategy
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(strat.Timestamp) {
				return natskit.PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(strat.Timestamp) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine -- first write for this key.

	data, err := json.Marshal(strat)
	if err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Internal, "marshal strategy for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Unavailable, "put strategy to KV")
	}

	return natskit.PutWritten, nil
}

// Get retrieves the latest strategy for a given source/symbol/timeframe.
func (s *KVStore) Get(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int) (*strategy.Strategy, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "strategy KV store is unavailable")
	}

	key := fmt.Sprintf("%s.%s.%d", source, inst.SubjectToken(), timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get strategy from KV")
	}

	var strat strategy.Strategy
	if err := json.Unmarshal(entry.Value(), &strat); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal strategy from KV")
	}
	return &strat, nil
}

func (s *KVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
