package main

import (
	"fmt"
	"os"
	"strings"
)

// CompareResult holds the outcome of comparing generated output against a golden snapshot.
type CompareResult struct {
	Family   string
	Artifact string
	Pass     bool
	// Diffs contains line-by-line differences when Pass is false.
	Diffs []LineDiff
}

// LineDiff describes a single line difference.
type LineDiff struct {
	Line     int
	Expected string
	Got      string
}

// CompareWithGolden loads the golden snapshot file and compares it against
// the generated output using structural normalization.
func CompareWithGolden(generated string, goldenPath string) (*CompareResult, error) {
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		return nil, fmt.Errorf("read golden snapshot %s: %w", goldenPath, err)
	}

	goldenNorm := normalizeForComparison(string(goldenData))
	generatedNorm := normalizeForComparison(generated)

	goldenLines := strings.Split(goldenNorm, "\n")
	generatedLines := strings.Split(generatedNorm, "\n")

	var diffs []LineDiff

	maxLen := len(goldenLines)
	if len(generatedLines) > maxLen {
		maxLen = len(generatedLines)
	}

	for i := 0; i < maxLen; i++ {
		var gl, genl string
		if i < len(goldenLines) {
			gl = goldenLines[i]
		}
		if i < len(generatedLines) {
			genl = generatedLines[i]
		}
		if gl != genl {
			diffs = append(diffs, LineDiff{
				Line:     i + 1,
				Expected: gl,
				Got:      genl,
			})
		}
	}

	return &CompareResult{
		Pass:  len(diffs) == 0,
		Diffs: diffs,
	}, nil
}

// normalizeForComparison applies structural normalization per S194 rules:
// 1. Strip single-line comments (// ...)
// 2. Trim leading/trailing whitespace per line
// 3. Remove empty lines
// 4. Normalize tab-to-space for indentation comparison
func normalizeForComparison(s string) string {
	lines := strings.Split(s, "\n")
	var normalized []string
	for _, line := range lines {
		// Strip single-line comments.
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		// Normalize tabs to single space.
		line = strings.ReplaceAll(line, "\t", " ")
		// Trim whitespace.
		line = strings.TrimSpace(line)
		// Skip empty lines.
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}
	return strings.Join(normalized, "\n")
}

// FormatCompareResult returns a human-readable report of the comparison.
func FormatCompareResult(r *CompareResult) string {
	if r.Pass {
		return fmt.Sprintf("PASS  %s/%s", r.Family, r.Artifact)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "FAIL  %s/%s  (%d differences)\n", r.Family, r.Artifact, len(r.Diffs))
	for _, d := range r.Diffs {
		fmt.Fprintf(&b, "  line %d:\n", d.Line)
		fmt.Fprintf(&b, "    expected: %s\n", d.Expected)
		fmt.Fprintf(&b, "    got:      %s\n", d.Got)
	}
	return b.String()
}
