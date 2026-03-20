package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"rsi", "RSI"},
		{"paper_order", "PaperOrder"},
		{"rsi_oversold", "RSIOversold"},
		{"mean_reversion_entry", "MeanReversionEntry"},
		{"position_exposure", "PositionExposure"},
		{"candle", "Candle"},
		{"signal", "Signal"},
		{"evidence", "Evidence"},
		{"execution", "Execution"},
		{"strategy", "Strategy"},
		{"risk", "Risk"},
		{"decision", "Decision"},
	}
	for _, tt := range tests {
		got := toPascalCase(tt.input)
		if got != tt.want {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDerivedFields_RSI(t *testing.T) {
	spec := &FamilySpec{
		Family: FamilyMeta{Name: "rsi", Layer: "signal", Tier: 1},
		NATS:   NATSSpec{Durable: "writer-signal-rsi"},
		Writer: WriterSpec{Table: "signals", Mapper: "mapSignalRow"},
		Domain: DomainSpec{EventPackage: "signal", EventType: "SignalGeneratedEvent"},
	}
	d := spec.Derived()

	assertField(t, "ConsumerSpecFunc", d.ConsumerSpecFunc, "WriterRSISignalConsumer")
	assertField(t, "ConsumerName", d.ConsumerName, "writer-signal-rsi-consumer")
	assertField(t, "InserterName", d.InserterName, "writer-signal-rsi-inserter")
	assertField(t, "IsEnabledMethod", d.IsEnabledMethod, "IsSignalFamilyEnabled")
	assertField(t, "RegistryField", d.RegistryField, "signal")
	assertField(t, "NewConsumerFunc", d.NewConsumerFunc, "NewConsumer")
	assertField(t, "PascalFamily", d.PascalFamily, "RSI")
	assertField(t, "PascalLayer", d.PascalLayer, "Signal")
	assertField(t, "StarterFunc", d.StarterFunc, "NewSignalStarter")
	assertField(t, "InsertSQL", d.InsertSQL, "INSERT INTO signals")
	assertField(t, "HyphenFamily", d.HyphenFamily, "rsi")
	assertField(t, "PackageAlias", d.PackageAlias, "natssignal")
}

func TestDerivedFields_PaperOrder(t *testing.T) {
	spec := &FamilySpec{
		Family: FamilyMeta{Name: "paper_order", Layer: "execution", Tier: 1},
		NATS:   NATSSpec{Durable: "writer-execution-paper-order"},
		Writer: WriterSpec{Table: "executions", Mapper: "mapExecutionRow"},
		Domain: DomainSpec{EventPackage: "execution", EventType: "PaperOrderSubmittedEvent"},
	}
	d := spec.Derived()

	assertField(t, "ConsumerSpecFunc", d.ConsumerSpecFunc, "WriterPaperOrderExecutionConsumer")
	assertField(t, "ConsumerName", d.ConsumerName, "writer-execution-paper-order-consumer")
	assertField(t, "InserterName", d.InserterName, "writer-execution-paper-order-inserter")
	assertField(t, "IsEnabledMethod", d.IsEnabledMethod, "IsExecutionFamilyEnabled")
	assertField(t, "NewConsumerFunc", d.NewConsumerFunc, "NewConsumer")
	assertField(t, "PascalFamily", d.PascalFamily, "PaperOrder")
	assertField(t, "PascalLayer", d.PascalLayer, "Execution")
	assertField(t, "HyphenFamily", d.HyphenFamily, "paper-order")
	assertField(t, "PackageAlias", d.PackageAlias, "natsexecution")
}

func TestDerivedFields_Evidence(t *testing.T) {
	spec := &FamilySpec{
		Family: FamilyMeta{Name: "candle", Layer: "evidence", Tier: 1},
		Writer: WriterSpec{Table: "evidence_candles"},
	}
	d := spec.Derived()

	assertField(t, "ConsumerSpecFunc", d.ConsumerSpecFunc, "WriterCandleConsumer")
	assertField(t, "ConsumerName", d.ConsumerName, "writer-candle-consumer")
	assertField(t, "InserterName", d.InserterName, "writer-candle-inserter")
	assertField(t, "IsEnabledMethod", d.IsEnabledMethod, "IsFamilyEnabled")
	assertField(t, "NewConsumerFunc", d.NewConsumerFunc, "NewCandleConsumer")
	assertField(t, "StarterFunc", d.StarterFunc, "NewCandleStarter")
	assertField(t, "PackageAlias", d.PackageAlias, "natsevidence")
}

func TestDerivedFields_InsertSQLWithColumns(t *testing.T) {
	spec := &FamilySpec{
		Family: FamilyMeta{Name: "rsi", Layer: "signal", Tier: 1},
		Writer: WriterSpec{Table: "signals", Columns: "event_id, occurred_at, type"},
	}
	d := spec.Derived()
	assertField(t, "InsertSQL", d.InsertSQL, "INSERT INTO signals (event_id, occurred_at, type)")
}

func TestDerivedFields_InsertSQLWithoutColumns(t *testing.T) {
	spec := &FamilySpec{
		Family: FamilyMeta{Name: "rsi", Layer: "signal", Tier: 1},
		Writer: WriterSpec{Table: "signals"},
	}
	d := spec.Derived()
	assertField(t, "InsertSQL", d.InsertSQL, "INSERT INTO signals")
}

func TestLoadSpec(t *testing.T) {
	dir := t.TempDir()
	content := `
family:
  name: test_family
  layer: signal
  tier: 1
nats:
  subject: "signal.events.test.generated.>"
  event_type: "signal.events.v1.test_generated"
  stream: SIGNAL_EVENTS
  durable: writer-signal-test
writer:
  table: signals
  mapper: mapSignalRow
  pipeline_family_key: test_family
  config_array: signal_families
domain:
  event_package: signal
  event_type: TestGeneratedEvent
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	spec, err := LoadSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Family.Name != "test_family" {
		t.Errorf("family.name = %q, want test_family", spec.Family.Name)
	}
	if err := spec.Validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	spec := &FamilySpec{}
	err := spec.Validate()
	if err == nil {
		t.Fatal("expected validation error for empty spec")
	}
}

func TestValidateCrossSpec_NoDuplicates(t *testing.T) {
	specs := []*FamilySpec{
		{
			Family: FamilyMeta{Name: "rsi", Layer: "signal", Tier: 1},
			NATS:   NATSSpec{Subject: "signal.events.rsi.generated.>", Durable: "writer-signal-rsi", EventType: "t1", Stream: "S1"},
			Writer: WriterSpec{Table: "signals", Mapper: "mapSignalRow", PipelineFamilyKey: "rsi", ConfigArray: "signals"},
			Domain: DomainSpec{EventPackage: "signal", EventType: "SignalGeneratedEvent"},
		},
		{
			Family: FamilyMeta{Name: "candle", Layer: "evidence", Tier: 1},
			NATS:   NATSSpec{Subject: "evidence.events.candle.sampled.>", Durable: "writer-candle", EventType: "t2", Stream: "S2"},
			Writer: WriterSpec{Table: "evidence_candles", Mapper: "mapCandleRow", PipelineFamilyKey: "candle", ConfigArray: "evidence"},
			Domain: DomainSpec{EventPackage: "evidence", EventType: "CandleSampledEvent"},
		},
	}
	if err := ValidateCrossSpec(specs); err != nil {
		t.Errorf("unexpected cross-spec error: %v", err)
	}
}

func TestValidateCrossSpec_DuplicateDurable(t *testing.T) {
	specs := []*FamilySpec{
		{Family: FamilyMeta{Name: "a"}, NATS: NATSSpec{Subject: "s1", Durable: "same-durable"}},
		{Family: FamilyMeta{Name: "b"}, NATS: NATSSpec{Subject: "s2", Durable: "same-durable"}},
	}
	err := ValidateCrossSpec(specs)
	if err == nil {
		t.Fatal("expected error for duplicate durable")
	}
	if !strings.Contains(err.Error(), "duplicate nats.durable") {
		t.Errorf("error should mention duplicate durable, got: %v", err)
	}
}

func TestValidateCrossSpec_DuplicateSubject(t *testing.T) {
	specs := []*FamilySpec{
		{Family: FamilyMeta{Name: "a"}, NATS: NATSSpec{Subject: "same.subject", Durable: "d1"}},
		{Family: FamilyMeta{Name: "b"}, NATS: NATSSpec{Subject: "same.subject", Durable: "d2"}},
	}
	err := ValidateCrossSpec(specs)
	if err == nil {
		t.Fatal("expected error for duplicate subject")
	}
	if !strings.Contains(err.Error(), "duplicate nats.subject") {
		t.Errorf("error should mention duplicate subject, got: %v", err)
	}
}

func TestValidateCrossSpec_DuplicateName(t *testing.T) {
	specs := []*FamilySpec{
		{Family: FamilyMeta{Name: "dup"}, NATS: NATSSpec{Subject: "s1", Durable: "d1"}},
		{Family: FamilyMeta{Name: "dup"}, NATS: NATSSpec{Subject: "s2", Durable: "d2"}},
	}
	err := ValidateCrossSpec(specs)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if !strings.Contains(err.Error(), "duplicate family.name") {
		t.Errorf("error should mention duplicate name, got: %v", err)
	}
}

func TestLoadAllSpecs(t *testing.T) {
	// Verify that all specs in families/ load and pass cross-spec validation.
	specs, err := LoadAllSpecs("families")
	if err != nil {
		t.Fatalf("LoadAllSpecs: %v", err)
	}
	if len(specs) < 6 {
		t.Fatalf("expected at least 6 family specs, got %d", len(specs))
	}
	for _, spec := range specs {
		if err := spec.Validate(); err != nil {
			t.Errorf("spec %s invalid: %v", spec.Family.Name, err)
		}
	}
	if err := ValidateCrossSpec(specs); err != nil {
		t.Errorf("cross-spec validation failed: %v", err)
	}
}

func assertField(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}
