package natssequencer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"internal/adapters/nats/natskit"
	"internal/shared/sequencer"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// SequencerStateBucket is the NATS JetStream KV bucket name for
// per-StreamKey high-water marks (ADR-0020).
const SequencerStateBucket = "SEQUENCER_STATE_LATEST"

// keyPrefix is the static leading token used in every KV key
// (per ADR-0020 format).
const keyPrefix = "seq"

// Store persists Sequencer snapshots to the
// SEQUENCER_STATE_LATEST KV bucket and restores them on boot.
//
// The Store does not implement a batching policy; callers
// invoke SaveSnapshot at whatever cadence is appropriate to
// their event rate. See package doc for rationale.
type Store struct {
	url         string
	ownerBinary string
	nc          *nats.Conn
	bucket      jetstream.KeyValue
}

// NewStore returns a Store for the given owner binary identifier.
// ownerBinary is one of "ingest", "derive", "execute",
// "configctl" — the writer that owns this Sequencer's keys per
// ADR-0008.
//
// The returned Store is not connected; call Start before use.
func NewStore(url, ownerBinary string) *Store {
	return &Store{url: url, ownerBinary: ownerBinary}
}

// Start connects to NATS and creates (or attaches to) the
// SEQUENCER_STATE_LATEST bucket. Safe to call once per Store
// instance; subsequent calls are not supported (close + reopen
// via Close + NewStore + Start instead).
func (s *Store) Start(ctx context.Context) error {
	nc, err := natskit.Connect(s.url)
	if err != nil {
		return err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create jetstream context: %w", err)
	}

	startCtx, cancel := context.WithTimeout(ctx, natskit.DefaultSetupTimeout)
	defer cancel()

	bucket, err := js.CreateOrUpdateKeyValue(startCtx, jetstream.KeyValueConfig{
		Bucket:   SequencerStateBucket,
		Storage:  jetstream.FileStorage,
		MaxBytes: 4 * 1024 * 1024, // 4 MB — single int64 per stream key; thousands fit
	})
	if err != nil {
		nc.Close()
		return fmt.Errorf("ensure %s bucket: %w", SequencerStateBucket, err)
	}

	s.nc = nc
	s.bucket = bucket
	return nil
}

// LoadSnapshot reads every entry in the bucket whose key has
// this Store's ownerBinary prefix and returns a map suitable for
// passing to sequencer.Sequencer.Restore.
//
// Returns an empty map (not nil) if the bucket is empty or
// contains no entries for this owner. Errors are returned only
// for genuine I/O or parse failures; an empty/missing bucket is
// not an error.
func (s *Store) LoadSnapshot(ctx context.Context) (map[sequencer.StreamKey]int64, error) {
	if s == nil || s.bucket == nil {
		return nil, fmt.Errorf("natssequencer: store is not started")
	}
	out := make(map[sequencer.StreamKey]int64)

	lister, err := s.bucket.ListKeys(ctx)
	if err != nil {
		// An empty bucket is signaled by ErrNoKeysFound on some
		// NATS server versions; treat as empty, not error.
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return out, nil
		}
		return nil, fmt.Errorf("natssequencer: list keys: %w", err)
	}
	defer func() { _ = lister.Stop() }()

	for k := range lister.Keys() {
		owner, sk, ok := parseKey(k)
		if !ok {
			continue
		}
		if owner != s.ownerBinary {
			continue
		}
		entry, err := s.bucket.Get(ctx, k)
		if err != nil {
			return nil, fmt.Errorf("natssequencer: get %s: %w", k, err)
		}
		v, err := strconv.ParseInt(string(entry.Value()), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("natssequencer: parse value for %s: %w", k, err)
		}
		out[sk] = v
	}
	return out, nil
}

// SaveSnapshot writes every entry in the given map to the
// bucket. Entries belonging to this Store's ownerBinary are
// keyed per ADR-0020; the caller is expected to pass a snapshot
// produced by its own sequencer.Sequencer.Snapshot — keys not
// owned by this store should not appear.
//
// SaveSnapshot is replace-per-key, not transactional: each Put
// is independent. A partial failure leaves the bucket in a state
// where some keys reflect the new snapshot and others the prior
// one; recovery semantics in ADR-0020 tolerate this via the
// "monotonicity always, density best-effort" contract.
func (s *Store) SaveSnapshot(ctx context.Context, snap map[sequencer.StreamKey]int64) error {
	if s == nil || s.bucket == nil {
		return fmt.Errorf("natssequencer: store is not started")
	}
	for sk, v := range snap {
		k := formatKey(s.ownerBinary, sk)
		data := []byte(strconv.FormatInt(v, 10))
		if _, err := s.bucket.Put(ctx, k, data); err != nil {
			return fmt.Errorf("natssequencer: put %s: %w", k, err)
		}
	}
	return nil
}

// Close releases the underlying NATS connection.
func (s *Store) Close() error {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
	return nil
}

// formatKey encodes a StreamKey as the ADR-0020 KV key.
func formatKey(owner string, sk sequencer.StreamKey) string {
	return keyPrefix + "." + owner + "." + sk.Venue + "." + sk.Instrument + "." + sk.EventType
}

// parseKey inverts formatKey. The first four dot-separated
// tokens are seq/owner/venue/instrument; the remainder rejoins
// as event_type (which may itself contain dots — `observation.trade`,
// `observation.book.snapshot`). Returns ok=false for any key
// that does not have at least five tokens or whose leading
// token is not the expected prefix.
func parseKey(k string) (owner string, sk sequencer.StreamKey, ok bool) {
	parts := strings.SplitN(k, ".", 5)
	if len(parts) < 5 {
		return "", sequencer.StreamKey{}, false
	}
	if parts[0] != keyPrefix {
		return "", sequencer.StreamKey{}, false
	}
	return parts[1], sequencer.StreamKey{
		Venue:      parts[2],
		Instrument: parts[3],
		EventType:  parts[4],
	}, true
}
