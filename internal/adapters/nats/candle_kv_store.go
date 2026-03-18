package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CandleLatestBucket  = "CANDLE_LATEST"
	CandleHistoryBucket = "CANDLE_HISTORY"
	candleKVMaxBytes    = 64 * 1024 * 1024  // 64 MB
	historyKVMaxBytes   = 256 * 1024 * 1024 // 256 MB
	historyKVMaxAge     = 24 * time.Hour
)

// PutResult describes the outcome of a projection write to the latest bucket.
type PutResult int

const (
	// PutWritten means the candle was materialized (new or newer than existing).
	PutWritten PutResult = iota
	// PutSkippedStale means an existing candle has a strictly newer OpenTime.
	// This happens during replay when events arrive out of order.
	PutSkippedStale
	// PutSkippedDuplicate means the existing candle has the same OpenTime.
	// The write is skipped because the projection is already up to date.
	PutSkippedDuplicate
)

func (r PutResult) String() string {
	switch r {
	case PutWritten:
		return "written"
	case PutSkippedStale:
		return "skipped_stale"
	case PutSkippedDuplicate:
		return "skipped_duplicate"
	default:
		return "unknown"
	}
}

// CandleKVStore persists the latest finalized candle per source/symbol/timeframe
// using NATS JetStream KeyValue. Data survives process restarts.
type CandleKVStore struct {
	url     string
	nc      *nats.Conn
	kv      jetstream.KeyValue
	history jetstream.KeyValue
}

func NewCandleKVStore(url string) *CandleKVStore {
	return &CandleKVStore{url: url}
}

func (s *CandleKVStore) Start() error {
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
		Bucket:   CandleLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: candleKVMaxBytes,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("create candle KV bucket: %w", err)
	}

	history, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   CandleHistoryBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: historyKVMaxBytes,
		TTL:      historyKVMaxAge,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("create candle history KV bucket: %w", err)
	}

	s.nc = nc
	s.kv = kv
	s.history = history
	return nil
}

// Put stores the latest candle with a monotonicity guard.
// If the existing candle has a newer or equal OpenTime, the write is skipped.
// This makes the latest projection safe under replay and reprocessing.
// Key format: {source}.{symbol}.{timeframe}
func (s *CandleKVStore) Put(ctx context.Context, candle evidence.EvidenceCandle) (PutResult, *problem.Problem) {
	if s == nil || s.kv == nil {
		return PutWritten, problem.New(problem.Unavailable, "candle KV store is unavailable")
	}

	key := candleKey(candle.Source, candle.Symbol, candle.Timeframe)

	// Monotonicity guard: read existing, compare OpenTime.
	existing, err := s.kv.Get(ctx, key)
	if err == nil {
		var current evidence.EvidenceCandle
		if jsonErr := json.Unmarshal(existing.Value(), &current); jsonErr == nil {
			if current.OpenTime.After(candle.OpenTime) {
				return PutSkippedStale, nil
			}
			if current.OpenTime.Equal(candle.OpenTime) {
				return PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(candle)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal candle for KV")
	}

	if _, err := s.kv.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put candle to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest candle for the given key.
func (s *CandleKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*evidence.EvidenceCandle, *problem.Problem) {
	if s == nil || s.kv == nil {
		return nil, problem.New(problem.Unavailable, "candle KV store is unavailable")
	}

	key := candleKey(source, symbol, timeframe)

	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil // no candle yet, not an error
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get candle from KV")
	}

	var candle evidence.EvidenceCandle
	if err := json.Unmarshal(entry.Value(), &candle); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal candle from KV")
	}

	return &candle, nil
}

// PutHistory stores a candle in the history bucket.
// Idempotent by key design: the key includes OpenTime as a unix timestamp,
// so replaying the same candle overwrites the same key with identical data.
// Key format: {source}.{symbol}.{timeframe}.{open_time_unix}
func (s *CandleKVStore) PutHistory(ctx context.Context, candle evidence.EvidenceCandle) *problem.Problem {
	if s == nil || s.history == nil {
		return problem.New(problem.Unavailable, "candle history KV store is unavailable")
	}

	key := candleHistoryKey(candle.Source, candle.Symbol, candle.Timeframe, candle.OpenTime)

	data, err := json.Marshal(candle)
	if err != nil {
		return problem.Wrap(err, problem.Internal, "marshal candle for history KV")
	}

	if _, err := s.history.Put(ctx, key, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put candle to history KV")
	}

	return nil
}

// GetHistory retrieves candles for a given source/symbol/timeframe, sorted newest first.
// since/until are unix seconds (inclusive); 0 means unset.
func (s *CandleKVStore) GetHistory(ctx context.Context, source, symbol string, timeframe, limit int, since, until int64) ([]evidence.EvidenceCandle, *problem.Problem) {
	if s == nil || s.history == nil {
		return nil, problem.New(problem.Unavailable, "candle history KV store is unavailable")
	}

	prefix := candleKey(source, symbol, timeframe)

	keys, err := s.history.Keys(ctx, jetstream.MetaOnly())
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "list history keys")
	}

	// Filter keys matching the prefix and optional time range.
	var matched []string
	prefixDot := prefix + "."
	for _, k := range keys {
		if len(k) <= len(prefixDot) || k[:len(prefixDot)] != prefixDot {
			continue
		}
		// Extract the unix timestamp suffix for range filtering.
		if since > 0 || until > 0 {
			tsSuffix := k[len(prefixDot):]
			ts, err := strconv.ParseInt(tsSuffix, 10, 64)
			if err != nil {
				continue
			}
			if since > 0 && ts < since {
				continue
			}
			if until > 0 && ts > until {
				continue
			}
		}
		matched = append(matched, k)
	}

	// Sort descending (newest first) — keys end with unix timestamp.
	sort.Sort(sort.Reverse(sort.StringSlice(matched)))

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	candles := make([]evidence.EvidenceCandle, 0, len(matched))
	for _, k := range matched {
		entry, err := s.history.Get(ctx, k)
		if err != nil {
			continue // key may have expired between list and get
		}

		var candle evidence.EvidenceCandle
		if err := json.Unmarshal(entry.Value(), &candle); err != nil {
			continue
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

func (s *CandleKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func candleKey(source, symbol string, timeframe int) string {
	return source + "." + symbol + "." + strconv.Itoa(timeframe)
}

func candleHistoryKey(source, symbol string, timeframe int, openTime time.Time) string {
	return candleKey(source, symbol, timeframe) + "." + strconv.FormatInt(openTime.Unix(), 10)
}
