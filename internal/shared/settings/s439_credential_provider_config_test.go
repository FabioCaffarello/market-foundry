package settings

import (
	"strings"
	"testing"
)

// ── CredentialProviderName defaults ──────────────────────────────────

func TestCredentialProviderName_DefaultsToEnv(t *testing.T) {
	v := VenueConfig{}
	if got := v.CredentialProviderName(); got != "env" {
		t.Errorf("expected env, got %s", got)
	}
}

func TestCredentialProviderName_ExplicitEnv(t *testing.T) {
	v := VenueConfig{CredentialProvider: "env"}
	if got := v.CredentialProviderName(); got != "env" {
		t.Errorf("expected env, got %s", got)
	}
}

func TestCredentialProviderName_File(t *testing.T) {
	v := VenueConfig{CredentialProvider: "file"}
	if got := v.CredentialProviderName(); got != "file" {
		t.Errorf("expected file, got %s", got)
	}
}

// ── Validation: unknown provider rejected ────────────────────────────

func TestVenueConfig_Validate_UnknownProvider_Rejected(t *testing.T) {
	v := VenueConfig{
		CredentialProvider: "vault",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("expected validation problem for unknown provider")
	}
	if !strings.Contains(prob.Message, "venue config is invalid") {
		t.Errorf("unexpected message: %s", prob.Message)
	}
}

// ── Validation: file requires credential_path ────────────────────────

func TestVenueConfig_Validate_File_RequiresPath(t *testing.T) {
	v := VenueConfig{
		CredentialProvider: "file",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("expected validation problem for missing credential_path")
	}
	issues := extractIssues(prob)
	found := false
	for _, issue := range issues {
		if issue.Field == "venue.credential_path" && strings.Contains(issue.Message, "required") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected credential_path required issue, got %s", prob.Message)
	}
}

// ── Validation: file with path passes ────────────────────────────────

func TestVenueConfig_Validate_File_WithPath_Passes(t *testing.T) {
	dryRun := true
	v := VenueConfig{
		CredentialProvider: "file",
		CredentialPath:     "/run/secrets/venue",
		DryRun:             &dryRun,
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob != nil {
		issues := extractIssues(prob)
		for _, issue := range issues {
			if issue.Field == "venue.credential_provider" || issue.Field == "venue.credential_path" {
				t.Errorf("unexpected credential config issue: %s: %s", issue.Field, issue.Message)
			}
		}
	}
}

// ── Validation: credential_path without file provider ────────────────

func TestVenueConfig_Validate_PathWithoutFileProvider_Rejected(t *testing.T) {
	v := VenueConfig{
		CredentialProvider: "env",
		CredentialPath:     "/run/secrets/venue",
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob == nil {
		t.Fatal("expected validation problem for credential_path with env provider")
	}
	issues := extractIssues(prob)
	found := false
	for _, issue := range issues {
		if issue.Field == "venue.credential_path" && strings.Contains(issue.Message, "only used when") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected credential_path mismatch issue, got %s", prob.Message)
	}
}

// ── Validation: env provider (default) passes without path ───────────

func TestVenueConfig_Validate_EnvDefault_NoCred_Passes(t *testing.T) {
	v := VenueConfig{
		Segments: map[MarketSegment]*SegmentVenueConfig{
			MarketSegmentSpot: {Enabled: true, Adapter: VenueTypeBinanceSpotTestnet},
		},
	}
	prob := v.Validate()
	if prob != nil {
		issues := extractIssues(prob)
		for _, issue := range issues {
			if issue.Field == "venue.credential_provider" || issue.Field == "venue.credential_path" {
				t.Errorf("unexpected credential config issue: %s: %s", issue.Field, issue.Message)
			}
		}
	}
}
