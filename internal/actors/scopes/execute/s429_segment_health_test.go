package execute

import (
	"internal/shared/settings"
	"testing"
)

// TestSegmentPrefix verifies source-to-segment counter prefix mapping.
func TestSegmentPrefix(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{"binances", "spot:"},
		{"binancef", "futures:"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := segmentPrefix(tt.source)
		if got != tt.want {
			t.Errorf("segmentPrefix(%q) = %q, want %q", tt.source, got, tt.want)
		}
	}
}

// TestSegmentPrefixConsistencyWithSettings verifies that segmentPrefix is
// consistent with settings.SegmentForSource for all known segments.
func TestSegmentPrefixConsistencyWithSettings(t *testing.T) {
	for _, seg := range []settings.MarketSegment{settings.MarketSegmentSpot, settings.MarketSegmentFutures} {
		source := settings.SourceForSegment(seg)
		prefix := segmentPrefix(source)
		expected := string(seg) + ":"
		if prefix != expected {
			t.Errorf("segmentPrefix(%q) = %q, want %q (segment=%q)", source, prefix, expected, seg)
		}
	}
}
