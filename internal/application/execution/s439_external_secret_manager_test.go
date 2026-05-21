package execution

import (
	"os"
	"path/filepath"
	"testing"
)

// ── FileCredentialProvider — basic resolution ─────────────────────────

func TestFileCredentialProvider_Resolve_Present(t *testing.T) {
	dir := t.TempDir()
	venueDir := filepath.Join(dir, "binance_spot_mainnet")
	os.MkdirAll(venueDir, 0o700)
	os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte("live-key-1234567890abcdef"), 0o600)
	os.WriteFile(filepath.Join(venueDir, "API_SECRET"), []byte("live-secret-1234567890abc"), 0o600)

	fp := NewFileCredentialProvider(dir)
	if got := fp.Resolve("binance_spot_mainnet", "API_KEY"); got != "live-key-1234567890abcdef" {
		t.Errorf("expected live-key-..., got %q", got)
	}
	if got := fp.Resolve("binance_spot_mainnet", "API_SECRET"); got != "live-secret-1234567890abc" {
		t.Errorf("expected live-secret-..., got %q", got)
	}
}

func TestFileCredentialProvider_Resolve_Missing(t *testing.T) {
	dir := t.TempDir()
	fp := NewFileCredentialProvider(dir)
	if got := fp.Resolve("binance_spot_mainnet", "API_KEY"); got != "" {
		t.Errorf("expected empty for missing file, got %q", got)
	}
}

func TestFileCredentialProvider_Resolve_TrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	venueDir := filepath.Join(dir, "binance_spot_mainnet")
	os.MkdirAll(venueDir, 0o700)
	os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte("  key-with-whitespace-1234  \n"), 0o600)

	fp := NewFileCredentialProvider(dir)
	got := fp.Resolve("binance_spot_mainnet", "API_KEY")
	if got != "key-with-whitespace-1234" {
		t.Errorf("expected trimmed value, got %q", got)
	}
}

func TestFileCredentialProvider_Name(t *testing.T) {
	fp := NewFileCredentialProvider("/tmp")
	if got := fp.Name(); got != "file" {
		t.Errorf("expected file, got %s", got)
	}
}

// ── FileCredentialProvider — path validation ──────────────────────────

func TestFileCredentialProvider_ValidateBasePath_Exists(t *testing.T) {
	dir := t.TempDir()
	fp := NewFileCredentialProvider(dir)
	if err := fp.ValidateBasePath(); err != nil {
		t.Errorf("expected no error for existing dir, got %v", err)
	}
}

func TestFileCredentialProvider_ValidateBasePath_NotExists(t *testing.T) {
	fp := NewFileCredentialProvider("/nonexistent/path/for/test")
	err := fp.ValidateBasePath()
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	pe, ok := err.(*PathError)
	if !ok {
		t.Fatalf("expected PathError, got %T", err)
	}
	if pe.Path != "/nonexistent/path/for/test" {
		t.Errorf("expected path in error, got %s", pe.Path)
	}
}

func TestFileCredentialProvider_ValidateBasePath_NotDir(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	os.WriteFile(f, []byte("x"), 0o600)

	fp := NewFileCredentialProvider(f)
	err := fp.ValidateBasePath()
	if err == nil {
		t.Fatal("expected error for file (not dir)")
	}
	pe, ok := err.(*PathError)
	if !ok {
		t.Fatalf("expected PathError, got %T", err)
	}
	if pe.Reason != "path is not a directory" {
		t.Errorf("unexpected reason: %s", pe.Reason)
	}
}

// ── FileCredentialProvider — integration with LoadCredentialsFrom ─────

func TestFileCredentialProvider_LoadCredentialsFrom_Success(t *testing.T) {
	dir := t.TempDir()
	venueDir := filepath.Join(dir, "binance_spot_mainnet")
	os.MkdirAll(venueDir, 0o700)
	os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte("filekey12345678901234567"), 0o600)
	os.WriteFile(filepath.Join(venueDir, "API_SECRET"), []byte("filesecret12345678901234"), 0o600)

	fp := NewFileCredentialProvider(dir)
	creds, prob := LoadCredentialsFrom(fp, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}
	if creds.ProviderName() != "file" {
		t.Errorf("expected provider name file, got %s", creds.ProviderName())
	}
	if creds.Get("API_KEY") != "filekey12345678901234567" {
		t.Errorf("unexpected API_KEY")
	}
}

func TestFileCredentialProvider_LoadCredentialsFrom_MissingFile_FailsClosed(t *testing.T) {
	dir := t.TempDir()
	venueDir := filepath.Join(dir, "binance_spot_mainnet")
	os.MkdirAll(venueDir, 0o700)
	os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte("filekey12345678901234567"), 0o600)
	// API_SECRET intentionally missing

	fp := NewFileCredentialProvider(dir)
	_, prob := LoadCredentialsFrom(fp, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected problem for missing file")
	}
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", prob.Code)
	}
}

// ── FileCredentialProvider — segment isolation ────────────────────────

func TestFileCredentialProvider_SegmentIsolation(t *testing.T) {
	dir := t.TempDir()
	// Create separate directories for spot and futures
	for _, venue := range []string{"binance_spot_mainnet", "binance_futures_mainnet"} {
		venueDir := filepath.Join(dir, venue)
		os.MkdirAll(venueDir, 0o700)
		os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte(venue+"-key-12345678901234"), 0o600)
		os.WriteFile(filepath.Join(venueDir, "API_SECRET"), []byte(venue+"-secret-12345678901"), 0o600)
	}

	fp := NewFileCredentialProvider(dir)

	spotCreds, prob := LoadCredentialsFrom(fp, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("spot load failed: %s", prob.Message)
	}
	futuresCreds, prob := LoadCredentialsFrom(fp, "binance_futures_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("futures load failed: %s", prob.Message)
	}

	if spotCreds.Get("API_KEY") == futuresCreds.Get("API_KEY") {
		t.Error("spot and futures should have distinct API_KEY values")
	}
}

// ── FileCredentialProvider — lowercase venue path ─────────────────────

func TestFileCredentialProvider_LowercasePath(t *testing.T) {
	dir := t.TempDir()
	venueDir := filepath.Join(dir, "binance_spot_mainnet")
	os.MkdirAll(venueDir, 0o700)
	os.WriteFile(filepath.Join(venueDir, "API_KEY"), []byte("found-it-1234567890abcd"), 0o600)

	fp := NewFileCredentialProvider(dir)
	// Venue type comes in as lowercase from config (canonical form).
	got := fp.Resolve("binance_spot_mainnet", "API_KEY")
	if got != "found-it-1234567890abcd" {
		t.Errorf("expected found-it-..., got %q", got)
	}

	// Mixed-case venue type should also resolve (lowercased by provider).
	got = fp.Resolve("BINANCE_SPOT_MAINNET", "API_KEY")
	if got != "found-it-1234567890abcd" {
		t.Errorf("expected found-it-... for uppercase venue, got %q", got)
	}
}

// ── SetCredentialProvider with FileCredentialProvider ─────────────────

func TestSetCredentialProvider_File(t *testing.T) {
	dir := t.TempDir()
	fp := NewFileCredentialProvider(dir)

	// Save and restore default provider.
	original := DefaultCredentialProvider()
	defer SetCredentialProvider(original)

	SetCredentialProvider(fp)
	if DefaultCredentialProvider().Name() != "file" {
		t.Errorf("expected file provider after set, got %s", DefaultCredentialProvider().Name())
	}
}
