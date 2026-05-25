package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	envv1 "internal/shared/contracts/envelope/v1"
)

// Recorder captures a sequence of CanonicalEvent values for
// later replay. Recorder is NOT safe for concurrent use; callers
// that record from multiple goroutines must serialize
// externally.
type Recorder struct {
	events []envv1.CanonicalEvent
}

// NewRecorder returns an empty Recorder ready to accept Record
// calls.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// Record appends an event to the recorded sequence. Events are
// preserved in record order; the Recorder does not sort, dedup,
// or otherwise transform them at record time. Normalization
// (e.g., empty-payload canonicalization) is applied at
// serialization time in WriteTo.
func (r *Recorder) Record(ce envv1.CanonicalEvent) {
	r.events = append(r.events, ce)
}

// Len returns the number of recorded events.
func (r *Recorder) Len() int {
	return len(r.events)
}

// Events returns a copy of the recorded sequence. The returned
// slice is a fresh allocation; callers may modify it freely
// without affecting subsequent Record calls or WriteTo output.
func (r *Recorder) Events() []envv1.CanonicalEvent {
	out := make([]envv1.CanonicalEvent, len(r.events))
	copy(out, r.events)
	return out
}

// WriteTo serializes the recorded sequence as JSON-lines to w.
// Each event becomes a single JSON object followed by a single
// newline. The byte output is deterministic given a fixed
// fixtureRecord struct definition (per stdlib encoding/json
// guarantees on struct field order). Returns the total number
// of bytes written.
func (r *Recorder) WriteTo(w io.Writer) (int64, error) {
	bw := bufio.NewWriter(w)
	var total int64
	for i, ce := range r.events {
		fr := toFixture(ce)
		data, err := json.Marshal(fr)
		if err != nil {
			return total, fmt.Errorf("replay: marshal event %d: %w", i, err)
		}
		n, err := bw.Write(data)
		total += int64(n)
		if err != nil {
			return total, fmt.Errorf("replay: write event %d: %w", i, err)
		}
		if err := bw.WriteByte('\n'); err != nil {
			return total, fmt.Errorf("replay: write newline after event %d: %w", i, err)
		}
		total++
	}
	if err := bw.Flush(); err != nil {
		return total, fmt.Errorf("replay: flush: %w", err)
	}
	return total, nil
}
