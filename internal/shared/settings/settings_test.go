package settings

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"internal/shared/problem"
)

func TestDefaultsProduceValidConfig(t *testing.T) {
	cfg := Defaults()

	if prob := cfg.Validate(); prob != nil {
		t.Fatalf("expected defaults to be valid, got %v", prob)
	}
}

func TestLoadSupportsJSONCAndDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.jsonc")

	content := `{
		// Override only the log level.
		"log": {
			"level": "debug"
		},
		/* Keep the remaining defaults. */
		"nats": {
			"enabled": true,
			"url": "nats://localhost:4222"
		}
	}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, prob := Load(path)
	if prob != nil {
		t.Fatalf("expected config to load, got %v", prob)
	}

	if cfg.Log.Level != LogLevelDebug {
		t.Fatalf("expected overridden log level, got %q", cfg.Log.Level)
	}
	if cfg.Log.Format != LogFormatText {
		t.Fatalf("expected missing field to keep default, got %q", cfg.Log.Format)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("expected default http addr, got %q", cfg.HTTP.Addr)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{"unknown": true}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, prob := Load(path)
	if prob == nil {
		t.Fatalf("expected parse error")
	}
	if prob.Code != cfgParseError {
		t.Fatalf("expected parse error code, got %q", prob.Code)
	}
}

func TestTimeframeDurationsDefaultsToSixtySeconds(t *testing.T) {
	p := PipelineConfig{}
	durations := p.TimeframeDurations()
	if len(durations) != 1 || durations[0] != 60*time.Second {
		t.Fatalf("expected [60s] fallback, got %v", durations)
	}
}

func TestTimeframeDurationsReturnsConfigured(t *testing.T) {
	p := PipelineConfig{Timeframes: []int{60, 300}}
	durations := p.TimeframeDurations()
	if len(durations) != 2 {
		t.Fatalf("expected 2 durations, got %d", len(durations))
	}
	if durations[0] != 60*time.Second {
		t.Fatalf("expected 60s, got %v", durations[0])
	}
	if durations[1] != 300*time.Second {
		t.Fatalf("expected 300s, got %v", durations[1])
	}
}

func TestTimeframeDurationsSkipsInvalid(t *testing.T) {
	p := PipelineConfig{Timeframes: []int{-1, 0, 60}}
	durations := p.TimeframeDurations()
	if len(durations) != 1 || durations[0] != 60*time.Second {
		t.Fatalf("expected [60s] after filtering invalid, got %v", durations)
	}
}

func TestIsFamilyEnabledDefaultsToAll(t *testing.T) {
	p := PipelineConfig{}
	if !p.IsFamilyEnabled("candle") {
		t.Fatal("expected all families enabled when list is empty")
	}
	if !p.IsFamilyEnabled("signal") {
		t.Fatal("expected all families enabled when list is empty")
	}
}

func TestIsFamilyEnabledFiltersByList(t *testing.T) {
	p := PipelineConfig{Families: []string{"candle", "volume"}}
	if !p.IsFamilyEnabled("candle") {
		t.Fatal("expected candle enabled")
	}
	if !p.IsFamilyEnabled("volume") {
		t.Fatal("expected volume enabled")
	}
	if p.IsFamilyEnabled("tradeburst") {
		t.Fatal("expected tradeburst disabled when not in list")
	}
}

func TestEnabledFamiliesReturnsNilWhenEmpty(t *testing.T) {
	p := PipelineConfig{}
	if p.EnabledFamilies() != nil {
		t.Fatal("expected nil when no families configured")
	}
}

func TestEnabledFamiliesReturnsCopy(t *testing.T) {
	p := PipelineConfig{Families: []string{"candle", "tradeburst"}}
	result := p.EnabledFamilies()
	if len(result) != 2 {
		t.Fatalf("expected 2 families, got %d", len(result))
	}
	// Mutating the result should not affect the original.
	result[0] = "modified"
	if p.Families[0] != "candle" {
		t.Fatal("EnabledFamilies should return a copy")
	}
}

// ── Pipeline family validation ──────────────────────────────────────

func TestValidatePipelineAcceptsKnownFamilies(t *testing.T) {
	p := PipelineConfig{
		Families:         []string{"candle", "tradeburst", "volume"},
		SignalFamilies:   []string{"rsi"},
		DecisionFamilies: []string{"rsi_oversold"},
	}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected valid pipeline, got %v", prob)
	}
}

func TestValidatePipelineRejectsUnknownEvidenceFamily(t *testing.T) {
	p := PipelineConfig{Families: []string{"candle", "cnadle"}}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error for unknown evidence family")
	}
	issues := extractIssues(prob)
	if len(issues) != 1 || issues[0].Value != "cnadle" {
		t.Fatalf("expected one issue for 'cnadle', got %v", issues)
	}
}

func TestValidatePipelineRejectsUnknownSignalFamily(t *testing.T) {
	p := PipelineConfig{SignalFamilies: []string{"macd"}}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error for unknown signal family")
	}
}

func TestValidatePipelineRejectsUnknownDecisionFamily(t *testing.T) {
	p := PipelineConfig{DecisionFamilies: []string{"moonshot"}}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error for unknown decision family")
	}
}

func TestValidatePipelineRejectsSignalWithoutEvidence(t *testing.T) {
	// rsi depends on candle; disable candle by listing only volume
	p := PipelineConfig{
		Families:       []string{"volume"},
		SignalFamilies: []string{"rsi"},
	}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error: rsi needs candle")
	}
	issues := extractIssues(prob)
	found := false
	for _, iss := range issues {
		if iss.Field == "pipeline.signal_families" && iss.Value == "rsi" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependency issue for rsi, got %v", issues)
	}
}

func TestValidatePipelineAcceptsSignalWhenEvidenceDefaultsToAll(t *testing.T) {
	// No explicit families = all evidence enabled (backward compatible)
	p := PipelineConfig{
		SignalFamilies: []string{"rsi"},
	}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected valid: empty families means all enabled, got %v", prob)
	}
}

func TestValidatePipelineRejectsDecisionWithoutSignal(t *testing.T) {
	p := PipelineConfig{
		DecisionFamilies: []string{"rsi_oversold"},
		// no signal_families → rsi not enabled
	}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error: rsi_oversold needs rsi")
	}
}

func TestValidatePipelineAcceptsDecisionWithSignal(t *testing.T) {
	p := PipelineConfig{
		SignalFamilies:   []string{"rsi"},
		DecisionFamilies: []string{"rsi_oversold"},
	}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected valid pipeline, got %v", prob)
	}
}

func TestValidatePipelineRejectsUnknownStrategyFamily(t *testing.T) {
	p := PipelineConfig{StrategyFamilies: []string{"yolo_entry"}}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error for unknown strategy family")
	}
}

func TestValidatePipelineRejectsStrategyWithoutDecision(t *testing.T) {
	p := PipelineConfig{
		SignalFamilies:   []string{"rsi"},
		StrategyFamilies: []string{"mean_reversion_entry"},
		// no decision_families → rsi_oversold not enabled
	}
	prob := p.ValidatePipeline()
	if prob == nil {
		t.Fatal("expected validation error: mean_reversion_entry needs rsi_oversold")
	}
	issues := extractIssues(prob)
	found := false
	for _, iss := range issues {
		if iss.Field == "pipeline.strategy_families" && iss.Value == "mean_reversion_entry" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected dependency issue for mean_reversion_entry, got %v", issues)
	}
}

func TestValidatePipelineAcceptsStrategyWithDecision(t *testing.T) {
	p := PipelineConfig{
		SignalFamilies:   []string{"rsi"},
		DecisionFamilies: []string{"rsi_oversold"},
		StrategyFamilies: []string{"mean_reversion_entry"},
	}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected valid pipeline, got %v", prob)
	}
}

func TestValidatePipelineAcceptsFullChain(t *testing.T) {
	p := PipelineConfig{
		Families:         []string{"candle", "tradeburst", "volume"},
		SignalFamilies:   []string{"rsi"},
		DecisionFamilies: []string{"rsi_oversold"},
		StrategyFamilies: []string{"mean_reversion_entry"},
	}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected full chain to be valid, got %v", prob)
	}
}

func TestIsStrategyFamilyEnabledOptIn(t *testing.T) {
	p := PipelineConfig{}
	if p.IsStrategyFamilyEnabled("mean_reversion_entry") {
		t.Fatal("expected no strategy families enabled when list is empty")
	}

	p = PipelineConfig{StrategyFamilies: []string{"mean_reversion_entry"}}
	if !p.IsStrategyFamilyEnabled("mean_reversion_entry") {
		t.Fatal("expected mean_reversion_entry enabled")
	}
}

func TestEnabledStrategyFamiliesReturnsNilWhenEmpty(t *testing.T) {
	p := PipelineConfig{}
	if p.EnabledStrategyFamilies() != nil {
		t.Fatal("expected nil when no strategy families configured")
	}
}

func TestEnabledStrategyFamiliesReturnsCopy(t *testing.T) {
	p := PipelineConfig{StrategyFamilies: []string{"mean_reversion_entry"}}
	result := p.EnabledStrategyFamilies()
	if len(result) != 1 {
		t.Fatalf("expected 1 strategy family, got %d", len(result))
	}
	result[0] = "modified"
	if p.StrategyFamilies[0] != "mean_reversion_entry" {
		t.Fatal("EnabledStrategyFamilies should return a copy")
	}
}

func TestValidatePipelineEmptyIsValid(t *testing.T) {
	p := PipelineConfig{}
	if prob := p.ValidatePipeline(); prob != nil {
		t.Fatalf("expected empty pipeline to be valid, got %v", prob)
	}
}

func TestValidateAggregatesIssues(t *testing.T) {
	cfg := Defaults()
	cfg.Log.Level = "verbose"
	cfg.HTTP.ReadTimeout = "nope"
	cfg.NATS.Enabled = true
	cfg.NATS.URL = ""

	prob := cfg.Validate()
	if prob == nil {
		t.Fatalf("expected config validation to fail")
	}
	if prob.Code != cfgInvalid {
		t.Fatalf("expected config invalid code, got %q", prob.Code)
	}

	rawIssues := prob.Details[problem.DetailIssues]
	issues, ok := rawIssues.([]problem.ValidationIssue)
	if !ok {
		t.Fatalf("expected typed validation issues, got %#v", rawIssues)
	}

	if len(issues) != 3 {
		t.Fatalf("expected aggregated issues, got %d", len(issues))
	}
}
