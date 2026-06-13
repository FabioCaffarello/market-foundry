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

// VolumeProfileLatestBucket holds the most recent volume profile per
// partition ({source}.{subject_token}.{timeframe}). The full profile
// (all price buckets) is serialized as the value — KV-latest is the
// read surface for H-8.a; ClickHouse history is a later sub-onda
// (the buckets[] shape doesn't map to the 1-row codegen path — see
// PROGRAM-0005 / G-successor).
const VolumeProfileLatestBucket = "INSIGHTS_VOLUME_PROFILE_LATEST"

type VolumeProfileKVStore struct {
	url    string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewVolumeProfileKVStore(url string) *VolumeProfileKVStore {
	return &VolumeProfileKVStore{url: url}
}

func (s *VolumeProfileKVStore) Start() error {
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
		Bucket:   VolumeProfileLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 64 * 1024 * 1024,
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", VolumeProfileLatestBucket, err)
	}
	s.nc = nc
	s.latest = latest
	return nil
}

// Put writes the volume profile as the latest for its partition,
// skipping stale (older window) and duplicate (same window) writes —
// same monotonic-by-OpenTime policy as the evidence KV stores.
func (s *VolumeProfileKVStore) Put(ctx context.Context, vp insights.VolumeProfile) (natskit.PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return natskit.PutWritten, problem.New(problem.Unavailable, "volume profile KV store is unavailable")
	}

	key := volumeProfileKey(vp.Source, vp.Instrument.SubjectToken(), vp.Timeframe)

	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev insights.VolumeProfile
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if vp.OpenTime.Before(prev.OpenTime) {
				return natskit.PutSkippedStale, nil
			}
			if vp.OpenTime.Equal(prev.OpenTime) {
				return natskit.PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(vp)
	if err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Internal, "marshal volume profile")
	}
	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return natskit.PutSkippedStale, problem.Wrap(err, problem.Unavailable, "put volume profile latest")
	}
	return natskit.PutWritten, nil
}

// Get returns the latest volume profile for a partition, or nil when
// absent.
func (s *VolumeProfileKVStore) Get(ctx context.Context, source string, inst instrument.CanonicalInstrument, timeframe int) (*insights.VolumeProfile, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "volume profile KV store is unavailable")
	}

	key := volumeProfileKey(source, inst.SubjectToken(), timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get volume profile from KV")
	}

	var vp insights.VolumeProfile
	if err := json.Unmarshal(entry.Value(), &vp); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal volume profile")
	}
	return &vp, nil
}

func (s *VolumeProfileKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func volumeProfileKey(source, symbol string, timeframe int) string {
	return fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
}
