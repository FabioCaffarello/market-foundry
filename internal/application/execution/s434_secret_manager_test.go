package execution

import (
	"os"
	"testing"
)

// ── CredentialProvider interface tests ──────────────────────────────────

func TestEnvCredentialProvider_Resolve(t *testing.T) {
	os.Setenv("MF_VENUE_TESTEX_API_KEY", "key123")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_KEY")

	p := EnvCredentialProvider{}
	if got := p.Resolve("testex", "API_KEY"); got != "key123" {
		t.Errorf("expected key123, got %s", got)
	}
	if got := p.Resolve("testex", "MISSING"); got != "" {
		t.Errorf("expected empty for missing key, got %s", got)
	}
}

func TestEnvCredentialProvider_Name(t *testing.T) {
	p := EnvCredentialProvider{}
	if got := p.Name(); got != "env" {
		t.Errorf("expected env, got %s", got)
	}
}

// ── Custom CredentialProvider tests ────────────────────────────────────

type staticProvider struct {
	secrets map[string]string
}

func (s *staticProvider) Resolve(venueType, key string) string {
	return s.secrets[venueType+"/"+key]
}

func (s *staticProvider) Name() string { return "static" }

func TestLoadCredentialsFrom_CustomProvider(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"binance_spot_mainnet/API_KEY":    "customkey1234567890abcdef",
		"binance_spot_mainnet/API_SECRET": "customsecret1234567890ab",
	}}

	creds, prob := LoadCredentialsFrom(p, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}
	if creds.Get("API_KEY") != "customkey1234567890abcdef" {
		t.Errorf("unexpected API_KEY value")
	}
	if creds.ProviderName() != "static" {
		t.Errorf("expected provider name static, got %s", creds.ProviderName())
	}
}

func TestLoadCredentialsFrom_MissingKey_FailsClosed(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"binance_spot_mainnet/API_KEY": "somevalidkey1234567890ab",
	}}

	_, prob := LoadCredentialsFrom(p, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected problem for missing credential")
	}
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", prob.Code)
	}
}

// ── Format validation tests ───────────────────────────────────────────

func TestLoadCredentials_TruncatedValue_Rejected(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"binance_spot_mainnet/API_KEY":    "short",
		"binance_spot_mainnet/API_SECRET": "alsoshort",
	}}

	_, prob := LoadCredentialsFrom(p, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected problem for truncated credential")
	}
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", prob.Code)
	}
}

func TestLoadCredentials_WhitespaceValue_Rejected(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"binance_spot_mainnet/API_KEY":    "validkey1234567890abcdef ",
		"binance_spot_mainnet/API_SECRET": "validsecret1234567890abc",
	}}

	_, prob := LoadCredentialsFrom(p, "binance_spot_mainnet", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected problem for whitespace in credential")
	}
}

func TestLoadCredentials_TestnetShortKey_Accepted(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"binance_spot_testnet/API_KEY":    "test-key",
		"binance_spot_testnet/API_SECRET": "test-secret",
	}}

	creds, prob := LoadCredentialsFrom(p, "binance_spot_testnet", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("testnet should accept short credentials: %s", prob.Message)
	}
	if creds.Get("API_KEY") != "test-key" {
		t.Errorf("unexpected value")
	}
}

func TestLoadCredentials_NonBinanceVenue_NoFormatValidation(t *testing.T) {
	p := &staticProvider{secrets: map[string]string{
		"other_venue/API_KEY": "x",
	}}

	creds, prob := LoadCredentialsFrom(p, "other_venue", []string{"API_KEY"})
	if prob != nil {
		t.Fatalf("non-binance venues should skip format validation: %s", prob.Message)
	}
	if creds.Get("API_KEY") != "x" {
		t.Errorf("unexpected value")
	}
}

// ── SetCredentialProvider tests ───────────────────────────────────────

func TestSetCredentialProvider_NilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil provider")
		}
	}()
	SetCredentialProvider(nil)
}

func TestDefaultCredentialProvider_IsEnv(t *testing.T) {
	p := DefaultCredentialProvider()
	if p.Name() != "env" {
		t.Errorf("expected default provider to be env, got %s", p.Name())
	}
}

// ── CredentialSet.ProviderName tests ──────────────────────────────────

func TestCredentialSet_ProviderName_NilSafe(t *testing.T) {
	var creds *CredentialSet
	if creds.ProviderName() != "" {
		t.Error("expected empty provider name from nil CredentialSet")
	}
}

// ── Backward compatibility: LoadCredentials still works via env ──────

func TestLoadCredentials_BackwardCompatible(t *testing.T) {
	os.Setenv("MF_VENUE_TESTEX_API_KEY", "key1234567890abcdefghij")
	os.Setenv("MF_VENUE_TESTEX_API_SECRET", "secret1234567890abcdefg")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_KEY")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_SECRET")

	creds, prob := LoadCredentials("testex", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected problem: %s", prob.Message)
	}
	if creds.ProviderName() != "env" {
		t.Errorf("expected env provider, got %s", creds.ProviderName())
	}
}
