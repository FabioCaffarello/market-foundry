package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/domain/risk"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const RiskPositionExposureLatestBucket = "RISK_POSITION_EXPOSURE_LATEST"

// RiskKVStore persists the latest finalized risk assessment per source/symbol/timeframe.
// One instance per risk type; the bucket name is injected at construction.
type RiskKVStore struct {
	url    string
	bucket string
	nc     *nats.Conn
	latest jetstream.KeyValue
}

func NewRiskKVStore(url, bucket string) *RiskKVStore {
	return &RiskKVStore{url: url, bucket: bucket}
}

func (s *RiskKVStore) Start() error {
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

// Put stores a risk assessment in the latest bucket with a monotonicity guard on Timestamp.
func (s *RiskKVStore) Put(ctx context.Context, assessment risk.RiskAssessment) (PutResult, *problem.Problem) {
	if s == nil || s.latest == nil {
		return PutWritten, problem.New(problem.Unavailable, "risk KV store is unavailable")
	}

	key := assessment.PartitionKey()

	// Monotonicity guard: read existing, compare Timestamp.
	existing, err := s.latest.Get(ctx, key)
	if err == nil {
		var prev risk.RiskAssessment
		if jsonErr := json.Unmarshal(existing.Value(), &prev); jsonErr == nil {
			if prev.Timestamp.After(assessment.Timestamp) {
				return PutSkippedStale, nil
			}
			if prev.Timestamp.Equal(assessment.Timestamp) {
				return PutSkippedDuplicate, nil
			}
		}
	}
	// ErrKeyNotFound is fine — first write for this key.

	data, err := json.Marshal(assessment)
	if err != nil {
		return PutWritten, problem.Wrap(err, problem.Internal, "marshal risk for KV")
	}

	if _, err := s.latest.Put(ctx, key, data); err != nil {
		return PutWritten, problem.Wrap(err, problem.Unavailable, "put risk to KV")
	}

	return PutWritten, nil
}

// Get retrieves the latest risk assessment for a given source/symbol/timeframe.
func (s *RiskKVStore) Get(ctx context.Context, source, symbol string, timeframe int) (*risk.RiskAssessment, *problem.Problem) {
	if s == nil || s.latest == nil {
		return nil, problem.New(problem.Unavailable, "risk KV store is unavailable")
	}

	key := fmt.Sprintf("%s.%s.%d", source, symbol, timeframe)
	entry, err := s.latest.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get risk from KV")
	}

	var assessment risk.RiskAssessment
	if err := json.Unmarshal(entry.Value(), &assessment); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal risk from KV")
	}
	return &assessment, nil
}

func (s *RiskKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
