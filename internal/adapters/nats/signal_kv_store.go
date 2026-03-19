package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/domain/signal"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const SignalRSILatestBucket = "SIGNAL_RSI_LATEST"
const SignalEMACrossoverLatestBucket = "SIGNAL_EMA_CROSSOVER_LATEST"

// SignalKVStore persists the latest finalized signal per source/symbol/timeframe.
// One instance per signal type; the bucket name is injected at construction.
type SignalKVStore struct {
	url    string
	bucket string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewSignalKVStore(url, bucket string) *SignalKVStore {
	return &SignalKVStore{url: url, bucket: bucket}
}

func (s *SignalKVStore) Start() error {
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

// Put stores a signal in the latest bucket with a monotonicity guard on Timestamp.
// If the existing signal has a newer or equal Timestamp, the write is skipped.
// This makes the latest projection safe under replay and reprocessing.
// Key format: {source}.{symbol}.{timeframe}
func (s *SignalKVStore) Put(ctx context.Context, sig signal.Signal) (PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return PutWritten, problem.New(problem.Unavailable, "signal KV store is unavailable")
	}

	key := sig.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev signal.Signal
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(sig.Timestamp) {
				return PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(sig.Timestamp) {
				return PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(sig)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal signal for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put signal to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest signal for a given source/symbol/timeframe.
func (s *SignalKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*signal.Signal, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "signal KV store is unavailable")
	}

	key := fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil // no signal yet, not an error
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get signal from KV")
	}

	var sig signal.Signal
	if err := json.Unmarshal(entry.Value(), &sig); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal signal from KV")
	}
	return &sig, nil
}

func (s *SignalKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
