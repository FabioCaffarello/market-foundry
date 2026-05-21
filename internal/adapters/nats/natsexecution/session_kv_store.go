package natsexecution

import (
	"context"
	"encoding/json"
	"fmt"

	"internal/adapters/nats/natskit"
	"internal/domain/execution"
	"internal/shared/problem"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SessionBucket is the KV bucket for operational session metadata.
const SessionBucket = "EXECUTION_SESSION"

// SessionKVStore persists operational session records in a NATS KV bucket.
// Key: session_id. One entry per session.
type SessionKVStore struct {
	url    string
	nc     *nats.Conn
	bucket jetstream.KeyValue
}

func NewSessionKVStore(url string) *SessionKVStore {
	return &SessionKVStore{url: url}
}

func (s *SessionKVStore) Start() error {
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

	bucket, err := js.CreateOrUpdateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket:   SessionBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 16 * 1024 * 1024, // 16 MB — sessions are small, bounded count
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", SessionBucket, err)
	}

	s.nc = nc
	s.bucket = bucket
	return nil
}

// Put stores a session record. Overwrites any existing record for the same session_id.
func (s *SessionKVStore) Put(ctx context.Context, session execution.Session) *problem.Problem {
	if s == nil || s.bucket == nil {
		return problem.New(problem.Unavailable, "session KV store is unavailable")
	}

	if prob := session.Validate(); prob != nil {
		return prob
	}

	data, err := json.Marshal(session)
	if err != nil {
		return problem.Wrap(err, problem.Internal, "marshal session for KV")
	}

	if _, err := s.bucket.Put(ctx, session.SessionID, data); err != nil {
		return problem.Wrap(err, problem.Unavailable, "put session to KV")
	}

	return nil
}

// Get retrieves a session by ID. Returns nil if not found.
func (s *SessionKVStore) Get(ctx context.Context, sessionID string) (*execution.Session, *problem.Problem) {
	if s == nil || s.bucket == nil {
		return nil, problem.New(problem.Unavailable, "session KV store is unavailable")
	}

	entry, err := s.bucket.Get(ctx, sessionID)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return nil, nil
		}
		return nil, problem.Wrap(err, problem.Unavailable, "get session from KV")
	}

	var session execution.Session
	if err := json.Unmarshal(entry.Value(), &session); err != nil {
		return nil, problem.Wrap(err, problem.Internal, "unmarshal session from KV")
	}

	return &session, nil
}

// List returns all session records. Returns newest-first by started_at.
func (s *SessionKVStore) List(ctx context.Context) ([]execution.Session, *problem.Problem) {
	if s == nil || s.bucket == nil {
		return nil, problem.New(problem.Unavailable, "session KV store is unavailable")
	}

	lister, err := s.bucket.ListKeys(ctx)
	if err != nil {
		return []execution.Session{}, nil
	}

	var sessions []execution.Session
	for key := range lister.Keys() {
		entry, err := s.bucket.Get(ctx, key)
		if err != nil {
			continue
		}
		var session execution.Session
		if err := json.Unmarshal(entry.Value(), &session); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	// Sort newest-first by started_at.
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].StartedAt.After(sessions[i].StartedAt) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	return sessions, nil
}

func (s *SessionKVStore) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}
