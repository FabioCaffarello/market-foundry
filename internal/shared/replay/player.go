package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	envv1 "internal/shared/contracts/envelope/v1"
)

// Player replays a previously recorded sequence of
// CanonicalEvent values parsed from a JSON-lines fixture.
// Player is NOT safe for concurrent use.
type Player struct {
	events []envv1.CanonicalEvent
	cursor int
}

// maxFixtureLineBytes is the per-line buffer ceiling for the
// scanner. Default bufio.Scanner is 64 KiB; bumped to 1 MiB to
// accommodate fixtures with large encoded payloads (e.g., book
// snapshots). Lines exceeding this trigger an error.
const maxFixtureLineBytes = 1 << 20

// NewPlayer parses a JSON-lines fixture from r and returns a
// Player positioned at the first event. Returns an error if any
// non-empty line fails to parse. Empty lines (after newline
// splitting) are silently skipped — they are not a valid fixture
// shape but tolerating them simplifies hand-edits during fixture
// regeneration.
func NewPlayer(r io.Reader) (*Player, error) {
	p := &Player{}
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1<<16)
	scanner.Buffer(buf, maxFixtureLineBytes)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var fr fixtureRecord
		if err := json.Unmarshal(line, &fr); err != nil {
			return nil, fmt.Errorf("replay: parse line %d: %w", lineNum, err)
		}
		p.events = append(p.events, fromFixture(fr))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("replay: read fixture: %w", err)
	}
	return p, nil
}

// Len returns the total number of events in the parsed fixture.
func (p *Player) Len() int {
	return len(p.events)
}

// Next returns the next event and advances the cursor. The
// second return value is false when the sequence is exhausted.
func (p *Player) Next() (envv1.CanonicalEvent, bool) {
	if p.cursor >= len(p.events) {
		return envv1.CanonicalEvent{}, false
	}
	ce := p.events[p.cursor]
	p.cursor++
	return ce, true
}

// Reset rewinds the cursor to the start of the sequence.
// Subsequent Next calls return events from the beginning.
func (p *Player) Reset() {
	p.cursor = 0
}

// Events returns a copy of the parsed sequence regardless of
// cursor position. The returned slice is a fresh allocation.
func (p *Player) Events() []envv1.CanonicalEvent {
	out := make([]envv1.CanonicalEvent, len(p.events))
	copy(out, p.events)
	return out
}
