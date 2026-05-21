package lineage

import (
	"testing"
)

func TestStageIndex(t *testing.T) {
	tests := []struct {
		stage Stage
		want  int
	}{
		{StageSignal, 0},
		{StageDecision, 1},
		{StageStrategy, 2},
		{StageRisk, 3},
		{StageExecution, 4},
		{Stage("unknown"), -1},
	}
	for _, tt := range tests {
		if got := StageIndex(tt.stage); got != tt.want {
			t.Errorf("StageIndex(%q) = %d, want %d", tt.stage, got, tt.want)
		}
	}
}

func TestValidateChain_Complete(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1", CausationID: ""},
			{Stage: StageDecision, EventID: "dec-001", CorrelationID: "corr-1", CausationID: "sig-001"},
			{Stage: StageStrategy, EventID: "str-001", CorrelationID: "corr-1", CausationID: "dec-001"},
			{Stage: StageRisk, EventID: "rsk-001", CorrelationID: "corr-1", CausationID: "str-001"},
			{Stage: StageExecution, EventID: "exe-001", CorrelationID: "corr-1", CausationID: "rsk-001"},
		},
	}

	if prob := ValidateChain(chain); prob != nil {
		t.Fatalf("valid chain should pass: %s", prob.Message)
	}
	if !IsComplete(chain) {
		t.Error("5-stage chain should be complete")
	}
	if missing := MissingStages(chain); len(missing) != 0 {
		t.Errorf("expected no missing stages, got %v", missing)
	}
}

func TestValidateChain_Partial(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1", CausationID: ""},
			{Stage: StageDecision, EventID: "dec-001", CorrelationID: "corr-1", CausationID: "sig-001"},
		},
	}

	if prob := ValidateChain(chain); prob != nil {
		t.Fatalf("partial chain should pass: %s", prob.Message)
	}
	if IsComplete(chain) {
		t.Error("2-stage chain should not be complete")
	}
	missing := MissingStages(chain)
	if len(missing) != 3 {
		t.Errorf("expected 3 missing stages, got %d: %v", len(missing), missing)
	}
}

func TestValidateChain_Empty(t *testing.T) {
	chain := Chain{Links: nil}
	if prob := ValidateChain(chain); prob == nil {
		t.Error("empty chain should fail validation")
	}
}

func TestValidateChain_BrokenCausation(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1", CausationID: ""},
			{Stage: StageDecision, EventID: "dec-001", CorrelationID: "corr-1", CausationID: "WRONG-ID"},
		},
	}
	prob := ValidateChain(chain)
	if prob == nil {
		t.Fatal("broken causation chain should fail validation")
	}
}

func TestValidateChain_MismatchedCorrelation(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1", CausationID: ""},
			{Stage: StageDecision, EventID: "dec-001", CorrelationID: "corr-DIFFERENT", CausationID: "sig-001"},
		},
	}
	prob := ValidateChain(chain)
	if prob == nil {
		t.Fatal("mismatched correlation should fail validation")
	}
}

func TestValidateChain_OutOfOrder(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageDecision, EventID: "dec-001", CorrelationID: "corr-1", CausationID: ""},
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1", CausationID: "dec-001"},
		},
	}
	prob := ValidateChain(chain)
	if prob == nil {
		t.Fatal("out-of-order stages should fail validation")
	}
}

func TestValidateChain_EmptyEventID(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "", CorrelationID: "corr-1", CausationID: ""},
		},
	}
	prob := ValidateChain(chain)
	if prob == nil {
		t.Fatal("empty event_id should fail validation")
	}
}

func TestMissingStages(t *testing.T) {
	chain := Chain{
		Links: []ChainLink{
			{Stage: StageSignal, EventID: "sig-001", CorrelationID: "corr-1"},
			{Stage: StageRisk, EventID: "rsk-001", CorrelationID: "corr-1", CausationID: "sig-001"},
		},
	}
	missing := MissingStages(chain)
	want := map[Stage]bool{StageDecision: true, StageStrategy: true, StageExecution: true}
	if len(missing) != len(want) {
		t.Fatalf("expected %d missing, got %d: %v", len(want), len(missing), missing)
	}
	for _, s := range missing {
		if !want[s] {
			t.Errorf("unexpected missing stage: %s", s)
		}
	}
}
