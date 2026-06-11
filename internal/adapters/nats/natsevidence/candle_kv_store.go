package natsevidence

import (
	"internal/domain/instrument"

	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"internal/adapters/nats/natskit"
	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	CandleLatestBucket  = "CANDLE_LATEST"
	CandleHistoryBucket = "CANDLE_HISTORY"
	candleKVMaxBytes    = 64 * 1024 * 1024
	historyKVMaxBytes   = 256 * 1024 * 1024
	historyKVMaxAge     = 24 * time.Hour
)

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

func (s *CandleKVStore) Put(ctx context.Context, candle evidence.EvidenceCandle) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.kv == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "candle KV store is unavailable")
	}

	key := candleKey(candle.Source, candle.Instrument.SubjectToken(), candle.Timeframe)

	existing, err := s.kv.Get(ctx, key)
	if err == nil {
		var current evidence.EvidenceCandle
		if jsonErr := json.Unmarshal(existing.Value(), &current); jsonErr == nil {
			if current.OpenTime.After(candle.OpenTime) {
				return natskit.PutSkippedStale, nil
			}
			if current.OpenTime.Equal(candle.OpenTime) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(candle)
	if err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Internal, "marshal candle for KV")
	}

	if _, err := s.kv.Put(ctx, key, data); err != nil {
		return natskit.PutWritten, problem.Wrap(err, problem.Unavailable, "put candle to KV")
	}

	return natskit.PutWritten, nil
}

func (s *CandleKVStore) Get(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int) (*evidence.EvidenceCandle, *problem.Problem) {
	if s == nil || s.kv == nil {
		return nil, problem.New(problem.Unavailable, "candle KV store is unavailable")
	}

	key := candleKey(source, inst.SubjectToken(), timeframe)

	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get candle from KV")
	}

	var candle evidence.EvidenceCandle
	if err := json.Unmarshal(entry.Value(), &candle); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal candle from KV")
	}

	return &candle, nil
}

func (s *CandleKVStore) PutHistory(ctx context.Context, candle evidence.EvidenceCandle) *problem.Problem {
	if s == nil || s.history == nil {
		return problem.New(problem.Unavailable, "candle history KV store is unavailable")
	}

	key := candleHistoryKey(candle.Source, candle.Instrument.SubjectToken(), candle.Timeframe, candle.OpenTime)

	data, err := json.Marshal(candle)
	if err != nil {
		return problem.Wrap(err, problem.Internal, "marshal candle for history KV")
	}

	if _, err := s.history.Put(ctx, key, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put candle to history KV")
	}

	return nil
}

func (s *CandleKVStore) GetHistory(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe, limit int, since, until int64) ([]evidence.EvidenceCandle, *problem.Problem) {
	if s == nil || s.history == nil {
		return nil, problem.New(problem.Unavailable, "candle history KV store is unavailable")
	}

	prefix := candleKey(source, inst.SubjectToken(), timeframe)

	keys, err := s.history.Keys(ctx, jetstream.MetaOnly())
	if err != nil {
		if err == jetstream.ErrNoKeysFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "list history keys")
	}

	var matched []string
	prefixDot := prefix + "."
	for _, k := range keys {
		if len(k) <= len(prefixDot) || k[:len(prefixDot)] != prefixDot {
			continue
		}
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

	sort.Sort(sort.Reverse(sort.StringSlice(matched)))

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	candles := make([]evidence.EvidenceCandle, 0, len(matched))
	for _, k := range matched {
		entry, err := s.history.Get(ctx, k)
		if err != nil {
			continue
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
