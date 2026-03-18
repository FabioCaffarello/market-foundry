package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/domain/evidence"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const VolumeLatestBucket = "VOLUME_LATEST"

// VolumeKVStore persists the latest finalized volume profile per source/symbol/timeframe.
type VolumeKVStore struct {
	url    string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewVolumeKVStore(url string) *VolumeKVStore {
	return &VolumeKVStore{url: url}
}

func (s *VolumeKVStore) Start() error {
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
		Bucket:   VolumeLatestBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 64 * 1024 * 1024, // 64 MB
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", VolumeLatestBucket, err)
	}

	s.nc = nc
	s.latest = latest
	return nil
}

func (s *VolumeKVStore) Put(ctx context.Context, vol evidence.EvidenceVolume) (PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return PutWritten, problem.New(problem.Unavailable, "volume KV store is unavailable")
	}

	key := volumeKey(vol.Source, vol.Symbol, vol.Timeframe)

	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev evidence.EvidenceVolume
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if vol.OpenTime.Before(prev.OpenTime) {
				return PutSkippedStale, nil
			}
			if vol.OpenTime.Equal(prev.OpenTime) {
				return PutSkippedDuplicate, nil
			}
		}
	}

	data, err := json.Marshal(vol)
	if err != nil {
		return PutSkippedStale, problem.Wrap(err, problem.Internal, "marshal volume")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return PutSkippedStale, problem.Wrap(err, problem.Unavailable, "put volume latest")
	}

	return PutWritten, nil
}

func (s *VolumeKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*evidence.EvidenceVolume, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "volume KV store is unavailable")
	}

	key := volumeKey(source, symbol, timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil // no volume yet, not an error
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get volume from KV")
	}

	var vol evidence.EvidenceVolume
	if err := json.Unmarshal(entry.Value(), &vol); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal volume")
	}
	return &vol, nil
}

func (s *VolumeKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

func volumeKey(source, symbol string, timeframe int) string {
	return fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
}
