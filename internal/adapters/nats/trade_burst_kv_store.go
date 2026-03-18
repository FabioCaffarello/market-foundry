package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	TradeBurstLatestBucket  = "TRADE_BURST_LATEST"
	tradeBurstKVMaxBytes    = 64 * 1024 * 1024 // 64 MB
)

// TradeBurstKVStore persists the latest finalized trade burst per source/symbol/timeframe
// using NATS JetStream KeyValue. Data survives process restarts.
type TradeBurstKVStore struct {
	url string
	nc  *nats.Conn
	kv  jetstream.KeyValue
}

func NewTradeBurstKVStore(url string) *TradeBurstKVStore {
	return &TradeBurstKVStore{url: url}
}

func (s *TradeBurstKVStore) Start() error {
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

	kv, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   TradeBurstLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: tradeBurstKVMaxBytes,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("create trade burst KV bucket: %w", err)
	}

	s.nc = nc
	s.kv = kv
	return nil
}

// Put stores the latest trade burst with a monotonicity guard.
// If the existing trade burst has a newer or equal OpenTime, the write is skipped.
// Key format: {source}.{symbol}.{timeframe}
func (s *TradeBurstKVStore) Put(ctx context.Context, burst evidence.EvidenceTradeBurst) (PutResult, *problem.Problem) {
	if s == nil || s.kv == nil {
		return PutWritten, problem.New(problem.Unavailable, "trade burst KV store is unavailable")
	}

	key := tradeBurstKey(burst.Source, burst.Symbol, burst.Timeframe)

	existing, err := s.kv.Get(ctx, key)
	if err == nil {
		var current evidence.EvidenceTradeBurst
		if jsonErr := json.Unmarshal(existing.Value(), &current); jsonErr == nil {
			if current.OpenTime.After(burst.OpenTime) {
				return PutSkippedStale, nil
			}
			if current.OpenTime.Equal(burst.OpenTime) {
				return PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(burst)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal trade burst for KV")
	}

	if _, err := s.kv.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put trade burst to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest trade burst for the given key.
func (s *TradeBurstKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*evidence.EvidenceTradeBurst, *problem.Problem) {
	if s == nil || s.kv == nil {
		return nil, problem.New(problem.Unavailable, "trade burst KV store is unavailable")
	}

	key := tradeBurstKey(source, symbol, timeframe)

	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get trade burst from KV")
	}

	var burst evidence.EvidenceTradeBurst
	if err := json.Unmarshal(entry.Value(), &burst); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal trade burst from KV")
	}

	return &burst, nil
}

func (s *TradeBurstKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func tradeBurstKey(source, symbol string, timeframe int) string {
	return source + "." + symbol + "." + strconv.Itoa(timeframe)
}
