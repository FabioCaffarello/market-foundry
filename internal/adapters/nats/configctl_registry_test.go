package nats

import "testing"

func TestConfigctlRegistryKeepsSubjectsAndStreamsSeparated(t *testing.T) {
	t.Parallel()

	registry := DefaultConfigctlRegistry()
	subjects := map[string]struct{}{
		registry.CreateDraft.Subject:                  {},
		registry.GetConfig.Subject:                    {},
		registry.GetActive.Subject:                    {},
		registry.ListActiveRuntimeProjections.Subject: {},
		registry.ListActiveIngestionBindings.Subject:  {},
		registry.ListConfigs.Subject:                  {},
		registry.ValidateDraft.Subject:                {},
		registry.ValidateConfig.Subject:               {},
		registry.CompileConfig.Subject:                {},
		registry.ActivateConfig.Subject:               {},
	}

	if len(subjects) != 10 {
		t.Fatalf("expected unique control subjects, got %d", len(subjects))
	}
	if registry.Activated.Stream.Name == "" {
		t.Fatal("expected event stream name")
	}
	if registry.Activated.Stream.Name == registry.CreateDraft.Subject {
		t.Fatal("expected event stream to stay separate from control plane")
	}
	if registry.IngestionRuntimeChanged.Subject == registry.Activated.Subject {
		t.Fatal("expected ingestion runtime changed event subject to stay separate from config.activated")
	}
}
