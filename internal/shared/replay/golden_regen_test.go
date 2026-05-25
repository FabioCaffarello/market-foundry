//go:build goldenregen

// Regeneration helper for the replay-cycle scope golden fixture.
//
// This file is compiled only with `-tags goldenregen` so it does
// not run during normal `go test ./...` invocations. The Makefile
// target `make golden-regen SCOPE=replay-cycle` invokes it via:
//
//	go test -tags goldenregen \
//	  -run TestRegenerateGoldenReplayCycle \
//	  ./internal/shared/replay/...
//
// After regeneration, the developer reviews the diff with:
//
//	git diff internal/shared/replay/testdata/golden/replay-cycle/
//
// and commits intentionally. The PR review surface is where the
// intentionality of a fixture change is recorded.

package replay_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"internal/shared/replay"
)

func TestRegenerateGoldenReplayCycle(t *testing.T) {
	events := generateSyntheticReplayCycleEvents()
	if len(events) != 100 {
		t.Fatalf("generator emitted %d events; expected 100", len(events))
	}

	r := replay.NewRecorder()
	for _, e := range events {
		r.Record(e)
	}

	var buf bytes.Buffer
	if _, err := r.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}

	dir := filepath.Dir(goldenReplayCyclePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(goldenReplayCyclePath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write %s: %v", goldenReplayCyclePath, err)
	}

	t.Logf("regenerated %s: %d events, %d bytes", goldenReplayCyclePath, len(events), buf.Len())
}
