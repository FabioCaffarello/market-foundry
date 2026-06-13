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

// CrossVenueLatestBucket holds the most recent cross-venue snapshot per
// partition. Unlike volume profile / TPO, the key is
// {subject_token}.{timeframe} with NO source — cross-venue fusion spans
// sources by design (the canonical instrument is the join key).
const CrossVenueLatestBucket = "INSIGHTS_CROSS_VENUE_LATEST"

type CrossVenueKVStore struct {
	url    string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewCrossVenueKVStore(url string) *CrossVenueKVStore {
	return &CrossVenueKVStore{url: url}
}

func (s *CrossVenueKVStore) Start() error {
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
		Bucket:   CrossVenueLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 64 * 1024 * 1024,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", CrossVenueLatestBucket, err)
	}
	s.nc = nc
	s.latest = latest
	return nil
}

// Put writes the snapshot as the latest for its partition, skipping
// stale (older window) and duplicate (same window) writes.
func (s *CrossVenueKVStore) Put(ctx context.Context, cv insights.CrossVenueSnapshot) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "cross venue KV store is unavailable")
	}

	key := crossVenueKey(cv.Instrument.SubjectToken(), cv.Timeframe)

	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev insights.CrossVenueSnapshot
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if cv.OpenTime.Before(prev.OpenTime) {
				return natskit.PutSkippedStale, nil
			}
			if cv.OpenTime.Equal(prev.OpenTime) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(cv)
	if err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Internal, "marshal cross venue snapshot")
	}
	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Unavailable, "put cross venue latest")
	}
	return natskit.PutWritten, nil
}

// Get returns the latest cross-venue snapshot for a partition, or nil
// when absent.
func (s *CrossVenueKVStore) Get(ctx context.Context, inst instrument.CanonicalInstrument, timeframe int) (*insights.CrossVenueSnapshot, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "cross venue KV store is unavailable")
	}

	key := crossVenueKey(inst.SubjectToken(), timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get cross venue from KV")
	}

	var cv insights.CrossVenueSnapshot
	if err := json.Unmarshal(entry.Value(), &cv); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal cross venue snapshot")
	}
	return &cv, nil
}

func (s *CrossVenueKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func crossVenueKey(symbol string, timeframe int) string {
	return fmt.Sprintf("%s.%d", symbol, timeframe)
}
