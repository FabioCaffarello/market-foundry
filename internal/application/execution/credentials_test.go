package execution

import (
	"os"
	"testing"
)

func TestLoadCredentials_AllPresent(t *testing.T) {
	os.Setenv("MF_VENUE_TESTEX_API_KEY", "key123")
	os.Setenv("MF_VENUE_TESTEX_API_SECRET", "secret456")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_KEY")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_SECRET")

	creds, prob := LoadCredentials("testex", []string{"API_KEY", "API_SECRET"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob.Message)
	}
	if creds.Get("API_KEY") != "key123" {
		t.Errorf("expected key123, got %s", creds.Get("API_KEY"))
	}
	if creds.Get("API_SECRET") != "secret456" {
		t.Errorf("expected secret456, got %s", creds.Get("API_SECRET"))
	}
	if creds.VenueType() != "testex" {
		t.Errorf("expected testex, got %s", creds.VenueType())
	}
}

func TestLoadCredentials_MissingRequired_FailsFast(t *testing.T) {
	os.Setenv("MF_VENUE_TESTEX_API_KEY", "key123")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_KEY")
	os.Unsetenv("MF_VENUE_TESTEX_API_SECRET")

	_, prob := LoadCredentials("testex", []string{"API_KEY", "API_SECRET"})
	if prob == nil {
		t.Fatal("expected problem for missing credential")
	}
	if prob.Code != "VAL_INVALID_ARGUMENT" {
		t.Errorf("expected VAL_INVALID_ARGUMENT, got %s", prob.Code)
	}
}

func TestLoadCredentials_NoRequiredKeys_Succeeds(t *testing.T) {
	creds, prob := LoadCredentials("paper", nil)
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob.Message)
	}
	if creds.VenueType() != "paper" {
		t.Errorf("expected paper, got %s", creds.VenueType())
	}
}

func TestLoadCredentials_HasKey(t *testing.T) {
	os.Setenv("MF_VENUE_TESTEX_API_KEY", "key123")
	defer os.Unsetenv("MF_VENUE_TESTEX_API_KEY")

	creds, prob := LoadCredentials("testex", []string{"API_KEY"})
	if prob != nil {
		t.Fatalf("unexpected problem: %v", prob.Message)
	}
	if !creds.HasKey("API_KEY") {
		t.Error("expected HasKey(API_KEY) to be true")
	}
	if creds.HasKey("API_SECRET") {
		t.Error("expected HasKey(API_SECRET) to be false")
	}
}

func TestLoadCredentials_NilCredentialSet(t *testing.T) {
	var creds *CredentialSet
	if creds.Get("foo") != "" {
		t.Error("expected empty string from nil CredentialSet")
	}
	if creds.VenueType() != "" {
		t.Error("expected empty venue type from nil CredentialSet")
	}
	if creds.HasKey("foo") {
		t.Error("expected false from nil CredentialSet")
	}
}
