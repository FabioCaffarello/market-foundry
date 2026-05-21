package execution

import (
	"fmt"
	"os"
	"strings"

	"internal/shared/problem"
)

// CredentialProvider abstracts the lookup of venue credentials.
// S434: This interface decouples credential resolution from a specific backend.
// The default implementation (EnvCredentialProvider) reads from environment variables.
// Future implementations can read from a secret manager (Vault, AWS Secrets Manager, etc.)
// without changing adapter bootstrap code.
//
// Security contract:
//   - Implementations must never log, print, or include credential values in errors.
//   - Implementations must return empty string for missing credentials (never a default).
//   - Implementations should be safe for concurrent use.
type CredentialProvider interface {
	// Resolve returns the credential value for the given venue type and key.
	// Returns empty string if the credential is not found.
	// The provider name is included in audit/error messages (never the value).
	Resolve(venueType, key string) string

	// Name returns a human-readable provider name for audit logging.
	// Example: "env", "vault", "aws-secrets-manager".
	Name() string
}

// EnvCredentialProvider resolves credentials from environment variables.
// Convention: MF_VENUE_{VENUE_TYPE}_{KEY} where VENUE_TYPE and KEY are uppercased.
// S434: This is the canonical default provider for all environments.
type EnvCredentialProvider struct{}

// Resolve reads MF_VENUE_{VENUE_TYPE}_{KEY} from the process environment.
func (EnvCredentialProvider) Resolve(venueType, key string) string {
	envVar := "MF_VENUE_" + strings.ToUpper(venueType) + "_" + strings.ToUpper(key)
	return os.Getenv(envVar)
}

// Name returns "env" for audit logging.
func (EnvCredentialProvider) Name() string { return "env" }

// defaultProvider is the package-level credential provider used by LoadCredentials.
// S434: Tests and future config can override this via SetCredentialProvider.
var defaultProvider CredentialProvider = EnvCredentialProvider{}

// SetCredentialProvider replaces the default credential provider.
// This must be called before any adapter bootstrap (i.e., before Run).
// Not safe for concurrent use — call once during init.
func SetCredentialProvider(p CredentialProvider) {
	if p == nil {
		panic("SetCredentialProvider: provider must not be nil")
	}
	defaultProvider = p
}

// DefaultCredentialProvider returns the current default provider.
func DefaultCredentialProvider() CredentialProvider {
	return defaultProvider
}

// CredentialSet holds venue API credentials loaded from a CredentialProvider.
// Convention: MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME} (e.g. MF_VENUE_BINANCE_API_KEY).
// Security invariants:
//   - Values are never logged, printed, or included in error messages.
//   - Values are never stored in config files.
//   - Load fails fast on missing required credentials.
type CredentialSet struct {
	venueType    string
	providerName string
	credentials  map[string]string
}

// LoadCredentials loads all credentials for a venue type using the default provider.
// requiredKeys lists the credential names that must be present (e.g. "API_KEY", "API_SECRET").
// Returns a Problem if any required credential is missing or fails validation.
// S434: Delegates to LoadCredentialsFrom with the default provider.
func LoadCredentials(venueType string, requiredKeys []string) (*CredentialSet, *problem.Problem) {
	return LoadCredentialsFrom(defaultProvider, venueType, requiredKeys)
}

// LoadCredentialsFrom loads credentials using the given provider.
// S434: This is the canonical entry point for credential resolution.
// It validates presence of all required keys, then runs format validation
// for known credential types (e.g. Binance API key length).
func LoadCredentialsFrom(provider CredentialProvider, venueType string, requiredKeys []string) (*CredentialSet, *problem.Problem) {
	creds := make(map[string]string, len(requiredKeys))

	var issues []problem.ValidationIssue
	for _, key := range requiredKeys {
		val := provider.Resolve(venueType, key)
		if val == "" {
			envHint := "MF_VENUE_" + strings.ToUpper(venueType) + "_" + strings.ToUpper(key)
			issues = append(issues, problem.ValidationIssue{
				Field:   envHint,
				Message: fmt.Sprintf("required venue credential is not set (provider: %s)", provider.Name()),
			})
			continue
		}
		creds[key] = val
	}

	if len(issues) > 0 {
		return nil, problem.Validation(problem.InvalidArgument,
			fmt.Sprintf("missing venue credentials for %s", venueType), issues...)
	}

	// S434: Format validation for known credential shapes.
	formatIssues := validateCredentialFormats(venueType, creds)
	if len(formatIssues) > 0 {
		return nil, problem.Validation(problem.InvalidArgument,
			fmt.Sprintf("venue credentials for %s failed format validation", venueType), formatIssues...)
	}

	return &CredentialSet{
		venueType:    venueType,
		providerName: provider.Name(),
		credentials:  creds,
	}, nil
}

// validateCredentialFormats checks structural validity of credential values.
// S434: Only applied to mainnet Binance venues — testnet credentials have no
// format constraints (test environments routinely use short placeholder values).
// This catches obviously wrong mainnet values (truncated paste, placeholder text)
// without leaking the actual credential into error messages.
func validateCredentialFormats(venueType string, creds map[string]string) []problem.ValidationIssue {
	if !isMainnetBinanceVenue(venueType) {
		return nil
	}
	var issues []problem.ValidationIssue
	for key, val := range creds {
		if len(val) < 16 {
			issues = append(issues, problem.ValidationIssue{
				Field:   key,
				Message: fmt.Sprintf("credential value is too short (%d chars) — likely truncated or placeholder", len(val)),
			})
		}
		if strings.ContainsAny(val, " \t\n\r") {
			issues = append(issues, problem.ValidationIssue{
				Field:   key,
				Message: "credential value contains whitespace — likely copy-paste error",
			})
		}
	}
	return issues
}

// isMainnetBinanceVenue reports whether a venue type string is a mainnet Binance adapter.
// S434: Format validation only applies to mainnet — testnet uses placeholder credentials in tests.
func isMainnetBinanceVenue(venueType string) bool {
	lower := strings.ToLower(venueType)
	return strings.HasPrefix(lower, "binance") && strings.Contains(lower, "mainnet")
}

// Get returns a credential value by key. Returns empty string if not found.
// Callers must never log the returned value.
func (c *CredentialSet) Get(key string) string {
	if c == nil {
		return ""
	}
	return c.credentials[key]
}

// VenueType returns the venue type this credential set was loaded for.
func (c *CredentialSet) VenueType() string {
	if c == nil {
		return ""
	}
	return c.venueType
}

// HasKey reports whether the credential set contains the given key.
func (c *CredentialSet) HasKey(key string) bool {
	if c == nil {
		return false
	}
	_, ok := c.credentials[key]
	return ok
}

// ProviderName returns the name of the provider that resolved these credentials.
// S434: Used for audit logging at bootstrap time.
func (c *CredentialSet) ProviderName() string {
	if c == nil {
		return ""
	}
	return c.providerName
}
