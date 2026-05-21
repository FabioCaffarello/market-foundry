package execution

import (
	"os"
	"path/filepath"
	"strings"
)

// FileCredentialProvider resolves credentials from files on disk.
// S439: This is the canonical external secret manager integration provider.
//
// File layout convention:
//
//	{basePath}/{venue_type}/{key}
//
// Example:
//
//	/run/secrets/venue/binance_spot_mainnet/API_KEY
//	/run/secrets/venue/binance_spot_mainnet/API_SECRET
//
// This layout is compatible with:
//   - Docker secrets (mount to /run/secrets)
//   - Kubernetes secrets (projected volume)
//   - Vault Agent / CSI driver (file injection)
//   - AWS Secrets Manager via External Secrets Operator (file sync)
//
// Security invariants:
//   - File contents are trimmed of leading/trailing whitespace (newline-safe).
//   - File read errors are treated as missing (fail-closed: empty string).
//   - File contents are never logged, printed, or included in error messages.
//   - The provider is safe for concurrent use (stateless reads).
type FileCredentialProvider struct {
	basePath string
}

// NewFileCredentialProvider creates a provider that reads credentials from
// files under basePath. The basePath must exist and be a directory.
// S439: Validation of basePath is done at preflight, not at construction.
func NewFileCredentialProvider(basePath string) *FileCredentialProvider {
	return &FileCredentialProvider{basePath: basePath}
}

// Resolve reads the credential from {basePath}/{venue_type}/{key}.
// Returns empty string if the file does not exist or cannot be read.
// File contents are trimmed of whitespace (handles trailing newlines from
// echo, heredoc, and secret mount tooling).
func (f *FileCredentialProvider) Resolve(venueType, key string) string {
	filePath := filepath.Join(f.basePath, strings.ToLower(venueType), strings.ToUpper(key))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Name returns "file" for audit logging.
func (f *FileCredentialProvider) Name() string { return "file" }

// BasePath returns the configured base path for diagnostic/audit purposes.
// S439: Used by preflight checks to verify the path exists.
func (f *FileCredentialProvider) BasePath() string { return f.basePath }

// ValidateBasePath checks that the base path exists and is a directory.
// Returns an error describing the issue if not.
// S439: Called by preflight, not at construction — this allows tests to
// construct providers with non-existent paths for unit testing.
func (f *FileCredentialProvider) ValidateBasePath() error {
	info, err := os.Stat(f.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &PathError{Path: f.basePath, Reason: "directory does not exist"}
		}
		return &PathError{Path: f.basePath, Reason: err.Error()}
	}
	if !info.IsDir() {
		return &PathError{Path: f.basePath, Reason: "path is not a directory"}
	}
	return nil
}

// PathError describes a problem with the credential file base path.
type PathError struct {
	Path   string
	Reason string
}

func (e *PathError) Error() string {
	return "credential path " + e.Path + ": " + e.Reason
}
