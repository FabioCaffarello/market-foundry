package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderConsumerSpec_RSI(t *testing.T) {
	spec := rsiSpec()
	templatesDir := findTemplatesDir(t)

	output, err := RenderArtifact(spec, "consumer_spec", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	mustContain(t, output, "WriterRSISignalConsumer")
	mustContain(t, output, `Durable: "writer-signal-rsi"`)
	mustContain(t, output, `Subject: "signal.events.rsi.generated.>"`)
	mustContain(t, output, `Type:    "signal.events.v1.rsi_generated"`)
	mustContain(t, output, `Name: "SIGNAL_EVENTS"`)
}

func TestRenderPipelineEntry_RSI(t *testing.T) {
	spec := rsiSpec()
	templatesDir := findTemplatesDir(t)

	output, err := RenderArtifact(spec, "pipeline_entry", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	mustContain(t, output, `family:       "rsi"`)
	mustContain(t, output, `consumerName: "writer-signal-rsi-consumer"`)
	mustContain(t, output, `inserterName: "writer-signal-rsi-inserter"`)
	mustContain(t, output, `table:        "signals"`)
	mustContain(t, output, `adapternats.WriterRSISignalConsumer()`)
	mustContain(t, output, `p.IsSignalFamilyEnabled("rsi")`)
	mustContain(t, output, `adapternats.NewSignalConsumer(`)
	mustContain(t, output, `reg.signal`)
	mustContain(t, output, `signal.SignalGeneratedEvent`)
	mustContain(t, output, `mapSignalRow(event)`)
}

func TestRenderConsumerSpec_PaperOrder(t *testing.T) {
	spec := paperOrderSpec()
	templatesDir := findTemplatesDir(t)

	output, err := RenderArtifact(spec, "consumer_spec", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	mustContain(t, output, "WriterPaperOrderExecutionConsumer")
	mustContain(t, output, `Durable: "writer-execution-paper-order"`)
	mustContain(t, output, `Subject: "execution.events.paper_order.submitted.>"`)
	mustContain(t, output, `Name: "EXECUTION_EVENTS"`)
}

func TestRenderPipelineEntry_PaperOrder(t *testing.T) {
	spec := paperOrderSpec()
	templatesDir := findTemplatesDir(t)

	output, err := RenderArtifact(spec, "pipeline_entry", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	mustContain(t, output, `family:       "paper_order"`)
	mustContain(t, output, `consumerName: "writer-execution-paper-order-consumer"`)
	mustContain(t, output, `adapternats.WriterPaperOrderExecutionConsumer()`)
	mustContain(t, output, `p.IsExecutionFamilyEnabled("paper_order")`)
	mustContain(t, output, `adapternats.NewExecutionConsumer(`)
	mustContain(t, output, `reg.execution`)
	mustContain(t, output, `execution.PaperOrderSubmittedEvent`)
	mustContain(t, output, `mapExecutionRow(event)`)
}

func TestGoldenComparison_RSI_ConsumerSpec(t *testing.T) {
	spec := rsiSpec()
	baseDir := findBaseDir(t)
	templatesDir := filepath.Join(baseDir, "templates")

	generated, err := RenderArtifact(spec, "consumer_spec", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", "rsi", "consumer_spec.go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Errorf("golden comparison failed:\n%s", FormatCompareResult(result))
	}
}

func TestGoldenComparison_RSI_PipelineEntry(t *testing.T) {
	spec := rsiSpec()
	baseDir := findBaseDir(t)
	templatesDir := filepath.Join(baseDir, "templates")

	generated, err := RenderArtifact(spec, "pipeline_entry", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", "rsi", "pipeline_entry.go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Errorf("golden comparison failed:\n%s", FormatCompareResult(result))
	}
}

func TestGoldenComparison_PaperOrder_ConsumerSpec(t *testing.T) {
	spec := paperOrderSpec()
	baseDir := findBaseDir(t)
	templatesDir := filepath.Join(baseDir, "templates")

	generated, err := RenderArtifact(spec, "consumer_spec", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", "paper_order", "consumer_spec.go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Errorf("golden comparison failed:\n%s", FormatCompareResult(result))
	}
}

func TestGoldenComparison_PaperOrder_PipelineEntry(t *testing.T) {
	spec := paperOrderSpec()
	baseDir := findBaseDir(t)
	templatesDir := filepath.Join(baseDir, "templates")

	generated, err := RenderArtifact(spec, "pipeline_entry", templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", "paper_order", "pipeline_entry.go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		t.Errorf("golden comparison failed:\n%s", FormatCompareResult(result))
	}
}

// ── Cross-Family Golden Comparisons (S196) ────────────────────

func TestGoldenComparison_Candle_ConsumerSpec(t *testing.T) {
	spec := candleSpec()
	goldenTest(t, spec, "consumer_spec")
}

func TestGoldenComparison_Candle_PipelineEntry(t *testing.T) {
	spec := candleSpec()
	goldenTest(t, spec, "pipeline_entry")
}

func TestGoldenComparison_RSIOversold_ConsumerSpec(t *testing.T) {
	spec := rsiOversoldSpec()
	goldenTest(t, spec, "consumer_spec")
}

func TestGoldenComparison_RSIOversold_PipelineEntry(t *testing.T) {
	spec := rsiOversoldSpec()
	goldenTest(t, spec, "pipeline_entry")
}

func TestGoldenComparison_MeanReversionEntry_ConsumerSpec(t *testing.T) {
	spec := meanReversionEntrySpec()
	goldenTest(t, spec, "consumer_spec")
}

func TestGoldenComparison_MeanReversionEntry_PipelineEntry(t *testing.T) {
	spec := meanReversionEntrySpec()
	goldenTest(t, spec, "pipeline_entry")
}

func TestGoldenComparison_PositionExposure_ConsumerSpec(t *testing.T) {
	spec := positionExposureSpec()
	goldenTest(t, spec, "consumer_spec")
}

func TestGoldenComparison_PositionExposure_PipelineEntry(t *testing.T) {
	spec := positionExposureSpec()
	goldenTest(t, spec, "pipeline_entry")
}

// TestCheckAllFamilies validates that every spec in families/ has matching
// golden snapshots and the engine reproduces them. This is the S196 cross-family
// equivalence gate — if this test fails, the codegen model has drift.
func TestCheckAllFamilies(t *testing.T) {
	baseDir := findBaseDir(t)
	familiesDir := filepath.Join(baseDir, "families")

	entries, err := os.ReadDir(familiesDir)
	if err != nil {
		t.Fatal(err)
	}

	artifacts := SupportedArtifacts()
	var passed, failed int

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		specPath := filepath.Join(familiesDir, entry.Name())
		spec, err := LoadSpec(specPath)
		if err != nil {
			t.Errorf("load %s: %v", specPath, err)
			failed++
			continue
		}
		if err := spec.Validate(); err != nil {
			t.Errorf("validate %s: %v", specPath, err)
			failed++
			continue
		}

		templatesDir := filepath.Join(baseDir, "templates")
		for _, artifact := range artifacts {
			generated, err := RenderArtifact(spec, artifact, templatesDir)
			if err != nil {
				t.Errorf("generate %s/%s: %v", spec.Family.Name, artifact, err)
				failed++
				continue
			}

			goldenPath := filepath.Join(baseDir, "golden-snapshots", spec.Family.Name, artifact+".go.golden")
			result, err := CompareWithGolden(generated, goldenPath)
			if err != nil {
				t.Errorf("compare %s/%s: %v", spec.Family.Name, artifact, err)
				failed++
				continue
			}
			result.Family = spec.Family.Name
			result.Artifact = artifact

			if !result.Pass {
				t.Errorf("golden mismatch:\n%s", FormatCompareResult(result))
				failed++
			} else {
				passed++
			}
		}
	}

	t.Logf("cross-family validation: %d passed, %d failed", passed, failed)
	if failed > 0 {
		t.Fatalf("%d golden comparisons failed", failed)
	}
}

func goldenTest(t *testing.T, spec *FamilySpec, artifact string) {
	t.Helper()
	baseDir := findBaseDir(t)
	templatesDir := filepath.Join(baseDir, "templates")

	generated, err := RenderArtifact(spec, artifact, templatesDir)
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", spec.Family.Name, artifact+".go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pass {
		result.Family = spec.Family.Name
		result.Artifact = artifact
		t.Errorf("golden comparison failed:\n%s", FormatCompareResult(result))
	}
}

// ── Test Fixtures ──────────────────────────────────────────────

func rsiSpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "rsi", Layer: "signal", Tier: 1},
		NATS: NATSSpec{
			Subject:   "signal.events.rsi.generated.>",
			EventType: "signal.events.v1.rsi_generated",
			Stream:    "SIGNAL_EVENTS",
			Durable:   "writer-signal-rsi",
		},
		Writer: WriterSpec{
			Table:             "signals",
			Mapper:            "mapSignalRow",
			PipelineFamilyKey: "rsi",
			ConfigArray:       "signal_families",
		},
		Domain: DomainSpec{
			EventPackage: "signal",
			EventType:    "SignalGeneratedEvent",
		},
	}
}

func paperOrderSpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "paper_order", Layer: "execution", Tier: 1},
		NATS: NATSSpec{
			Subject:   "execution.events.paper_order.submitted.>",
			EventType: "execution.events.v1.paper_order_submitted",
			Stream:    "EXECUTION_EVENTS",
			Durable:   "writer-execution-paper-order",
		},
		Writer: WriterSpec{
			Table:             "executions",
			Mapper:            "mapExecutionRow",
			PipelineFamilyKey: "paper_order",
			ConfigArray:       "execution_families",
		},
		Domain: DomainSpec{
			EventPackage: "execution",
			EventType:    "PaperOrderSubmittedEvent",
		},
	}
}

func candleSpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "candle", Layer: "evidence", Tier: 1},
		NATS: NATSSpec{
			Subject:   "evidence.events.candle.sampled.>",
			EventType: "evidence.events.v1.candle_sampled",
			Stream:    "EVIDENCE_EVENTS",
			Durable:   "writer-candle",
		},
		Writer: WriterSpec{
			Table:             "evidence_candles",
			Mapper:            "mapCandleRow",
			PipelineFamilyKey: "candle",
			ConfigArray:       "families",
		},
		Domain: DomainSpec{
			EventPackage: "evidence",
			EventType:    "CandleSampledEvent",
		},
	}
}

func rsiOversoldSpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "rsi_oversold", Layer: "decision", Tier: 1},
		NATS: NATSSpec{
			Subject:   "decision.events.rsi_oversold.evaluated.>",
			EventType: "decision.events.v1.rsi_oversold_evaluated",
			Stream:    "DECISION_EVENTS",
			Durable:   "writer-decision-rsi-oversold",
		},
		Writer: WriterSpec{
			Table:             "decisions",
			Mapper:            "mapDecisionRow",
			PipelineFamilyKey: "rsi_oversold",
			ConfigArray:       "decision_families",
		},
		Domain: DomainSpec{
			EventPackage: "decision",
			EventType:    "DecisionEvaluatedEvent",
		},
	}
}

func meanReversionEntrySpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "mean_reversion_entry", Layer: "strategy", Tier: 1},
		NATS: NATSSpec{
			Subject:   "strategy.events.mean_reversion_entry.resolved.>",
			EventType: "strategy.events.v1.mean_reversion_entry_resolved",
			Stream:    "STRATEGY_EVENTS",
			Durable:   "writer-strategy-mean-reversion-entry",
		},
		Writer: WriterSpec{
			Table:             "strategies",
			Mapper:            "mapStrategyRow",
			PipelineFamilyKey: "mean_reversion_entry",
			ConfigArray:       "strategy_families",
		},
		Domain: DomainSpec{
			EventPackage: "strategy",
			EventType:    "StrategyResolvedEvent",
		},
	}
}

func positionExposureSpec() *FamilySpec {
	return &FamilySpec{
		Family: FamilyMeta{Name: "position_exposure", Layer: "risk", Tier: 1},
		NATS: NATSSpec{
			Subject:   "risk.events.position_exposure.assessed.>",
			EventType: "risk.events.v1.position_exposure_assessed",
			Stream:    "RISK_EVENTS",
			Durable:   "writer-risk-position-exposure",
		},
		Writer: WriterSpec{
			Table:             "risk_assessments",
			Mapper:            "mapRiskRow",
			PipelineFamilyKey: "position_exposure",
			ConfigArray:       "risk_families",
		},
		Domain: DomainSpec{
			EventPackage: "risk",
			EventType:    "RiskAssessedEvent",
		},
	}
}

func findTemplatesDir(t *testing.T) string {
	t.Helper()
	dir := findBaseDir(t)
	return filepath.Join(dir, "templates")
}

func findBaseDir(t *testing.T) string {
	t.Helper()
	// Walk up from test working directory to find codegen/templates/
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Check if we're already in the codegen directory
	if _, err := os.Stat(filepath.Join(dir, "templates")); err == nil {
		return dir
	}
	// Check if codegen/ is a subdirectory
	codegen := filepath.Join(dir, "codegen")
	if _, err := os.Stat(filepath.Join(codegen, "templates")); err == nil {
		return codegen
	}
	t.Fatal("cannot find codegen templates directory")
	return ""
}

func mustContain(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Errorf("output does not contain %q\noutput:\n%s", want, output)
	}
}
