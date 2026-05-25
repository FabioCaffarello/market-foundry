// Golden tests for the replay-cycle scope.
//
// INV-D3 (byte-identical replay) and INV-D4 (N=50 byte-stability)
// per ADR-0019. The "end-to-end" scope for H-4 is the replay layer
// cycle: record → persist → load → re-record → byte-identical.
// Derive evaluator goldens (true observation→evidence end-to-end)
// will land in a future wave that migrates derive to Clock/Source
// ports; that wave can extend this golden suite without reshaping
// the existing tests.
//
// When a logic change in the replay layer intentionally
// invalidates a golden, regenerate with:
//
//	make golden-regen SCOPE=replay-cycle
//
// then `git diff internal/shared/replay/testdata/golden/replay-cycle/`
// before committing.

package replay_test

import (
	"bytes"
	"os"
	"testing"

	"internal/shared/replay"
)

// TestGolden_Synthetic100_ByteIdentical validates the canonical
// record → persist → load → re-record cycle is byte-stable: the
// fixture committed to testdata equals what the recorder produces
// from the player's parsed event sequence.
//
// Failure signal: the fixture was modified manually without a
// matching regen, OR the record/play round-trip is no longer
// byte-stable (e.g., struct field reorder, payload normalization
// drift, timestamp encoding change).
func TestGolden_Synthetic100_ByteIdentical(t *testing.T) {
	fixtureBytes, err := os.ReadFile(goldenReplayCyclePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", goldenReplayCyclePath, err)
	}

	p, err := replay.NewPlayer(bytes.NewReader(fixtureBytes))
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}

	r := replay.NewRecorder()
	for {
		ce, ok := p.Next()
		if !ok {
			break
		}
		r.Record(ce)
	}

	var out bytes.Buffer
	if _, err := r.WriteTo(&out); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	if !bytes.Equal(fixtureBytes, out.Bytes()) {
		t.Fatalf(
			"golden fixture and re-recorded output diverge\n"+
				"  fixture path: %s\n"+
				"  fixture bytes (%d): %s\n"+
				"  re-recorded   (%d): %s\n"+
				"\nIf this divergence is intentional, regenerate with:\n"+
				"  make golden-regen SCOPE=replay-cycle\n"+
				"and review the diff before committing.",
			goldenReplayCyclePath,
			len(fixtureBytes), truncate(fixtureBytes, 256),
			out.Len(), truncate(out.Bytes(), 256),
		)
	}
}

// TestGolden_ByteStability_N50 validates INV-D4: replaying the
// same fixture 50 times in-process produces byte-identical output
// on every iteration. This is the canary for hidden non-
// determinism: map iteration order, scheduler timing, undeclared
// global state, unseeded PRNG, time.Now leak in a non-domain
// layer touched by the replay path.
//
// Cross-process N=50 would additionally exercise linker order and
// init-order side effects, but those are rare in Go and add CI
// time disproportionate to the benefit. If a future wave surfaces
// such a regression, add a TestGolden_ByteStability_N5_CrossProcess
// sibling rather than replacing this one.
func TestGolden_ByteStability_N50(t *testing.T) {
	fixtureBytes, err := os.ReadFile(goldenReplayCyclePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", goldenReplayCyclePath, err)
	}

	const iterations = 50
	var firstOutput []byte
	for i := 0; i < iterations; i++ {
		p, err := replay.NewPlayer(bytes.NewReader(fixtureBytes))
		if err != nil {
			t.Fatalf("iteration %d NewPlayer: %v", i, err)
		}
		r := replay.NewRecorder()
		for {
			ce, ok := p.Next()
			if !ok {
				break
			}
			r.Record(ce)
		}
		var out bytes.Buffer
		if _, err := r.WriteTo(&out); err != nil {
			t.Fatalf("iteration %d WriteTo: %v", i, err)
		}
		if i == 0 {
			firstOutput = append([]byte(nil), out.Bytes()...)
			continue
		}
		if !bytes.Equal(firstOutput, out.Bytes()) {
			t.Fatalf(
				"iteration %d output diverged from iteration 0\n"+
					"  fixture path: %s\n"+
					"  iter 0 (%d): %s\n"+
					"  iter %d (%d): %s\n"+
					"\nThis indicates hidden non-determinism: map iteration order, "+
					"scheduler, unseeded globals, or time.Now leaking through a "+
					"non-domain layer touched by the replay path.",
				i,
				goldenReplayCyclePath,
				len(firstOutput), truncate(firstOutput, 256),
				i, out.Len(), truncate(out.Bytes(), 256),
			)
		}
	}
}

// truncate returns at most max bytes of b, suffixed with an
// ellipsis marker if truncation occurred. Helps keep failure
// output from drowning the test log.
func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + " …(truncated)"
}
