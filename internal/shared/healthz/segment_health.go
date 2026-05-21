package healthz

import (
	"sync"
)

// SegmentStatus represents the health state of a single market segment.
type SegmentStatus struct {
	Segment string `json:"segment"`
	Enabled bool   `json:"enabled"`
	Adapter string `json:"adapter,omitempty"`
	Phase   string `json:"phase"` // "disabled", "ready", "active", "idle", "degraded"

	// Counters from the segment's tracker subset.
	Processed int64 `json:"processed"`
	Filled    int64 `json:"filled"`
	Rejected  int64 `json:"rejected"`
	Errors    int64 `json:"errors,omitempty"`
}

// SegmentDescriptor describes a configured segment for health tracking.
type SegmentDescriptor struct {
	Name    string // "spot" or "futures"
	Enabled bool
	Adapter string // adapter type name, e.g. "binance_spot_testnet"
}

// SegmentHealthRegistry tracks per-segment health state.
// It is safe for concurrent use.
type SegmentHealthRegistry struct {
	mu       sync.RWMutex
	segments map[string]*segmentEntry
}

type segmentEntry struct {
	descriptor SegmentDescriptor
	tracker    *Tracker // shared with VenueAdapterActor counters
}

// NewSegmentHealthRegistry creates a registry with no segments.
func NewSegmentHealthRegistry() *SegmentHealthRegistry {
	return &SegmentHealthRegistry{
		segments: make(map[string]*segmentEntry),
	}
}

// Register adds a segment with its tracker to the registry.
// The tracker should use counter names prefixed with the segment name
// (e.g. "spot:processed", "spot:filled").
func (r *SegmentHealthRegistry) Register(desc SegmentDescriptor, tracker *Tracker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.segments[desc.Name] = &segmentEntry{
		descriptor: desc,
		tracker:    tracker,
	}
}

// Status returns the current health status for all registered segments,
// ordered by segment name (futures, spot — alphabetical).
func (r *SegmentHealthRegistry) Status() []SegmentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Canonical order: futures, spot (alphabetical).
	order := []string{"futures", "spot"}
	var result []SegmentStatus

	for _, name := range order {
		entry, ok := r.segments[name]
		if !ok {
			continue
		}
		result = append(result, r.buildStatus(entry))
	}

	// Append any segments not in the canonical order (future-proofing).
	for name, entry := range r.segments {
		if name != "futures" && name != "spot" {
			result = append(result, r.buildStatus(entry))
		}
	}

	return result
}

// SegmentPhase returns the phase for a specific segment, or "unknown" if not registered.
func (r *SegmentHealthRegistry) SegmentPhase(segment string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.segments[segment]
	if !ok {
		return "unknown"
	}
	return r.computeSegmentPhase(entry)
}

func (r *SegmentHealthRegistry) buildStatus(entry *segmentEntry) SegmentStatus {
	ss := SegmentStatus{
		Segment: entry.descriptor.Name,
		Enabled: entry.descriptor.Enabled,
		Adapter: entry.descriptor.Adapter,
		Phase:   r.computeSegmentPhase(entry),
	}

	if entry.tracker == nil {
		return ss
	}

	// Read segment-prefixed counters from the shared tracker.
	prefix := entry.descriptor.Name + ":"
	counters := entry.tracker.Counters()
	ss.Processed = counters[prefix+"processed"]
	ss.Filled = counters[prefix+"filled"]
	ss.Rejected = counters[prefix+"rejected"]
	ss.Errors = counters[prefix+"errors"]

	return ss
}

func (r *SegmentHealthRegistry) computeSegmentPhase(entry *segmentEntry) string {
	if !entry.descriptor.Enabled {
		return "disabled"
	}
	if entry.tracker == nil {
		return "ready"
	}

	prefix := entry.descriptor.Name + ":"
	counters := entry.tracker.Counters()
	processed := counters[prefix+"processed"]
	errors := counters[prefix+"errors"]

	if processed == 0 && errors == 0 {
		return "ready" // enabled, awaiting first event
	}
	if errors > 0 && processed == 0 {
		return "degraded"
	}
	if processed > 0 {
		return "active"
	}
	return "ready"
}

// IsHealthy returns true if all enabled segments are in a non-degraded phase.
func (r *SegmentHealthRegistry) IsHealthy() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entry := range r.segments {
		if !entry.descriptor.Enabled {
			continue
		}
		phase := r.computeSegmentPhase(entry)
		if phase == "degraded" {
			return false
		}
	}
	return true
}

// RegisteredSegments returns the names of all registered segments.
func (r *SegmentHealthRegistry) RegisteredSegments() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.segments))
	for name := range r.segments {
		names = append(names, name)
	}
	return names
}
