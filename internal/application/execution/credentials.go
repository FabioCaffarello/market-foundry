package execution

import (
	"fmt"
	"os"
	"strings"

	"internal/shared/problem"
)

// CredentialSet holds venue API credentials loaded from environment variables.
// Convention: MF_VENUE_{VENUE_TYPE}_{CREDENTIAL_NAME} (e.g. MF_VENUE_BINANCE_API_KEY).
// Security invariants:
//   - Values are never logged, printed, or included in error messages.
//   - Values are never stored in config files.
//   - Load fails fast on missing required credentials.
type CredentialSet struct {
	venueType   string
	credentials map[string]string
}

// LoadCredentials loads all credentials for a venue type from environment variables.
// requiredKeys lists the credential names that must be present (e.g. "API_KEY", "API_SECRET").
// Returns a Problem if any required credential is missing.
// The prefix used is MF_VENUE_{VENUE_TYPE}_{KEY} where VENUE_TYPE is uppercased.
func LoadCredentials(venueType string, requiredKeys []string) (*CredentialSet, *problem.Problem) {
	prefix := "MF_VENUE_" + strings.ToUpper(venueType) + "_"
	creds := make(map[string]string, len(requiredKeys))

	var missing []problem.ValidationIssue
	for _, key := range requiredKeys {
		envVar := prefix + strings.ToUpper(key)
		val := os.Getenv(envVar)
		if val == "" {
			missing = append(missing, problem.ValidationIssue{
				Field:   envVar,
				Message: "required venue credential is not set",
			})
			continue
		}
		creds[key] = val
	}

	if len(missing) > 0 {
		return nil, problem.Validation(problem.InvalidArgument,
			fmt.Sprintf("missing venue credentials for %s", venueType), missing...)
	}

	return &CredentialSet{
		venueType:   venueType,
		credentials: creds,
	}, nil
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
