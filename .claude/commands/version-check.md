---
name: version-check
description: Verify version consistency across go.work, rust-toolchain.toml, .tool-versions, CI.
---

Check that tool versions are consistent across canonical project
files.

## Sources of truth

- `go.work` — Go workspace version.
- `tools/raccoon-cli/rust-toolchain.toml` — Rust toolchain.
- `.tool-versions` — asdf/mise manifest.
- `.github/workflows/ci.yml` — CI invocations.
- `Makefile` — local build references.

## Run

```bash
echo "=== Version manifest cross-check ==="
echo ""

GO_WORK=$(grep "^go " go.work | awk '{print $2}')
GO_MOD=$(grep "^go " cmd/ingest/go.mod 2>/dev/null | awk '{print $2}')
GO_TOOL_VERSIONS=$(grep "^golang " .tool-versions 2>/dev/null | awk '{print $2}')
GO_CI=$(grep -E "go-version" .github/workflows/ci.yml 2>/dev/null | head -1 | sed 's/.*: *//;s/"//g')

echo "Go versions:"
echo "  go.work:                $GO_WORK"
echo "  cmd/ingest/go.mod:      $GO_MOD"
echo "  .tool-versions:         $GO_TOOL_VERSIONS"
echo "  .github/workflows/ci:   ${GO_CI:-(not pinned)}"
echo ""

RUST_TOOLCHAIN=$(grep "^channel" tools/raccoon-cli/rust-toolchain.toml 2>/dev/null | awk -F'"' '{print $2}')
RUST_TOOL_VERSIONS=$(grep "^rust " .tool-versions 2>/dev/null | awk '{print $2}')

echo "Rust versions:"
echo "  rust-toolchain.toml:    $RUST_TOOLCHAIN"
echo "  .tool-versions:         $RUST_TOOL_VERSIONS"
echo ""

GOLANGCI_TV=$(grep "^golangci-lint " .tool-versions 2>/dev/null | awk '{print $2}')
GOLANGCI_CI=$(grep -oE "version: v?[0-9]+\.[0-9]+\.[0-9]+" .github/workflows/ci.yml 2>/dev/null | head -1)

echo "golangci-lint versions:"
echo "  .tool-versions:         $GOLANGCI_TV"
echo "  .github/workflows/ci:   ${GOLANGCI_CI:-(not pinned — action default)}"
```

## Interpretation

Common findings:

- **Bump in one file forgotten in others**: e.g., `go.work` bumped to
  1.26 but `.tool-versions` still pins 1.25. Sync them.
- **CI uses action default while `.tool-versions` is pinned**:
  intentional for `golangci-lint-action@v6` (uses latest); flag if
  unintentional elsewhere.
- **`go.work` and `go.mod` should match**: workspace version and
  per-module version are independent in Go workspaces but typically
  align in this repo.

Owner decides whether each inconsistency is acceptable (intentional)
or needs sync.
