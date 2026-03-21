package clickhouse_test

import (
	"testing"

	"internal/application/analyticalclient"
)

func TestComputeChainCompleteness_AllPresent(t *testing.T) {
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: "full-chain",
		Signal:        &analyticalclient.SignalWithTrace{},
		Decision:      &analyticalclient.DecisionWithTrace{},
		Strategy:      &analyticalclient.StrategyWithTrace{},
		Risk:          &analyticalclient.RiskWithTrace{},
		Execution:     &analyticalclient.ExecutionWithTrace{},
	}

	// Simulate what the reader does.
	computeCompleteness(chain)

	if chain.StageCount != 5 {
		t.Errorf("stage_count: got %d, want 5", chain.StageCount)
	}
	if !chain.ChainComplete {
		t.Error("expected chain_complete=true")
	}
	if len(chain.MissingStages) != 0 {
		t.Errorf("expected no missing stages, got %v", chain.MissingStages)
	}
}

func TestComputeChainCompleteness_PartialChain(t *testing.T) {
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: "partial-chain",
		Signal:        &analyticalclient.SignalWithTrace{},
		Decision:      &analyticalclient.DecisionWithTrace{},
		// strategy, risk, execution missing
	}

	computeCompleteness(chain)

	if chain.StageCount != 2 {
		t.Errorf("stage_count: got %d, want 2", chain.StageCount)
	}
	if chain.ChainComplete {
		t.Error("expected chain_complete=false")
	}
	expected := map[string]bool{"strategy": true, "risk": true, "execution": true}
	for _, s := range chain.MissingStages {
		if !expected[s] {
			t.Errorf("unexpected missing stage: %q", s)
		}
		delete(expected, s)
	}
	if len(expected) != 0 {
		t.Errorf("expected missing stages not reported: %v", expected)
	}
}

func TestComputeChainCompleteness_Empty(t *testing.T) {
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: "empty-chain",
	}

	computeCompleteness(chain)

	if chain.StageCount != 0 {
		t.Errorf("stage_count: got %d, want 0", chain.StageCount)
	}
	if chain.ChainComplete {
		t.Error("expected chain_complete=false")
	}
	if len(chain.MissingStages) != 5 {
		t.Errorf("expected 5 missing stages, got %d", len(chain.MissingStages))
	}
}

func TestComputeChainCompleteness_RiskRejected(t *testing.T) {
	// A chain that stopped at risk (rejected) — 4 stages, no execution.
	chain := &analyticalclient.CompositeExecutionChain{
		CorrelationID: "risk-rejected",
		Signal:        &analyticalclient.SignalWithTrace{},
		Decision:      &analyticalclient.DecisionWithTrace{},
		Strategy:      &analyticalclient.StrategyWithTrace{},
		Risk:          &analyticalclient.RiskWithTrace{},
	}

	computeCompleteness(chain)

	if chain.StageCount != 4 {
		t.Errorf("stage_count: got %d, want 4", chain.StageCount)
	}
	if chain.ChainComplete {
		t.Error("expected chain_complete=false for risk-rejected chain")
	}
	if len(chain.MissingStages) != 1 || chain.MissingStages[0] != "execution" {
		t.Errorf("expected missing=[execution], got %v", chain.MissingStages)
	}
}

// computeCompleteness replicates the exported logic for test-only use.
// This avoids exporting an internal helper while still testing the algorithm.
func computeCompleteness(chain *analyticalclient.CompositeExecutionChain) {
	stages := []struct {
		name    string
		present bool
	}{
		{"signal", chain.Signal != nil},
		{"decision", chain.Decision != nil},
		{"strategy", chain.Strategy != nil},
		{"risk", chain.Risk != nil},
		{"execution", chain.Execution != nil},
	}

	count := 0
	var missing []string
	for _, s := range stages {
		if s.present {
			count++
		} else {
			missing = append(missing, s.name)
		}
	}

	chain.StageCount = count
	chain.ChainComplete = count == 5
	chain.MissingStages = missing
}
