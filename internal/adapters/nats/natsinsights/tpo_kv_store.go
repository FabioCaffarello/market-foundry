package natsinsights

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/insights"
	"internal/domain/instrument"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// TPOLatestBucket holds the most recent TPO profile per partition
// ({source}.{subject_token}.{timeframe}). The full profile (periods +
// levels) is serialized as the value — KV-latest is the read surface
// for H-8.b; ClickHouse history is H-8.b.1.
const TPOLatestBucket = "INSIGHTS_TPO_LATEST"

type TPOKVStore struct {
	url    string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewTPOKVStore(url string) *TPOKVStore {
	return &TPOKVStore{url: url}
}

func (s *TPOKVStore) Start() error {
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
		Bucket:   TPOLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 64 * 1024 * 1024,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", TPOLatestBucket, err)
	}
	s.nc = nc
	s.latest = latest
	return nil
}

// Put writes the TPO profile as the latest for its partition, skipping
// stale (older window) and duplicate (same window) writes — same
// monotonic-by-OpenTime policy as the volume profile KV store.
func (s *TPOKVStore) Put(ctx context.Context, tp insights.TPOProfile) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "tpo KV store is unavailable")
	}

	key := tpoKey(tp.Source, tp.Instrument.SubjectToken(), tp.Timeframe)

	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev insights.TPOProfile
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if tp.OpenTime.Before(prev.OpenTime) {
				return natskit.PutSkippedStale, nil
			}
			if tp.OpenTime.Equal(prev.OpenTime) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(tp)
	if err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Internal, "marshal tpo profile")
	}
	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Unavailable, "put tpo latest")
	}
	return natskit.PutWritten, nil
}

// Get returns the latest TPO profile for a partition, or nil when absent.
func (s *TPOKVStore) Get(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int) (*insights.TPOProfile, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "tpo KV store is unavailable")
	}

	key := tpoKey(source, inst.SubjectToken(), timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get tpo from KV")
	}

	var tp insights.TPOProfile
	if err := json.Unmarshal(entry.Value(), &tp); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal tpo profile")
	}
	return &tp, nil
}

func (s *TPOKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func tpoKey(source, symbol string, timeframe int) string {
	return fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
}
