package triage

import "testing"

func TestClassifySessionSeverity(t *testing.T) {
	tests := []struct {
		name     string
		verdict  string
		failed   int
		warnings int
		want     TriageSeverity
	}{
		{"inconsistent verdict", "inconsistent", 0, 0, SeverityCritical},
		{"failed checks", "consistent", 2, 0, SeverityCritical},
		{"degraded verdict", "degraded", 0, 0, SeverityWarning},
		{"warnings only", "consistent", 0, 1, SeverityWarning},
		{"all clean", "consistent", 0, 0, SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySessionSeverity(tt.verdict, tt.failed, tt.warnings)
			if got != tt.want {
				t.Errorf("ClassifySessionSeverity(%q, %d, %d) = %q, want %q", tt.verdict, tt.failed, tt.warnings, got, tt.want)
			}
		})
	}
}

func TestClassifyDecisionSeverity(t *testing.T) {
	tests := []struct {
		name       string
		violations int
		incomplete bool
		want       TriageSeverity
	}{
		{"violations", 2, false, SeverityCritical},
		{"incomplete", 0, true, SeverityWarning},
		{"clean", 0, false, SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyDecisionSeverity(tt.violations, tt.incomplete)
			if got != tt.want {
				t.Errorf("ClassifyDecisionSeverity(%d, %v) = %q, want %q", tt.violations, tt.incomplete, got, tt.want)
			}
		})
	}
}

func TestClassifyRoundTripSeverity(t *testing.T) {
	tests := []struct {
		name        string
		flagCount   int
		pnlReliable bool
		feeReliable bool
		want        TriageSeverity
	}{
		{"many flags", 3, true, true, SeverityCritical},
		{"unreliable pnl with flags", 1, false, true, SeverityCritical},
		{"some flags", 1, true, true, SeverityWarning},
		{"unreliable fee", 0, true, false, SeverityWarning},
		{"clean", 0, true, true, SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyRoundTripSeverity(tt.flagCount, tt.pnlReliable, tt.feeReliable)
			if got != tt.want {
				t.Errorf("ClassifyRoundTripSeverity(%d, %v, %v) = %q, want %q", tt.flagCount, tt.pnlReliable, tt.feeReliable, got, tt.want)
			}
		})
	}
}

func TestSortSessionItems(t *testing.T) {
	items := []SessionTriageItem{
		{SessionID: "s1", Severity: SeverityInfo, AnomalyCount: 0},
		{SessionID: "s2", Severity: SeverityCritical, AnomalyCount: 3},
		{SessionID: "s3", Severity: SeverityWarning, AnomalyCount: 1},
		{SessionID: "s4", Severity: SeverityCritical, AnomalyCount: 5},
	}

	SortSessionItems(items)

	if items[0].SessionID != "s4" {
		t.Errorf("expected s4 first (critical, 5 anomalies), got %s", items[0].SessionID)
	}
	if items[1].SessionID != "s2" {
		t.Errorf("expected s2 second (critical, 3 anomalies), got %s", items[1].SessionID)
	}
	if items[2].SessionID != "s3" {
		t.Errorf("expected s3 third (warning), got %s", items[2].SessionID)
	}
	if items[3].SessionID != "s1" {
		t.Errorf("expected s1 last (info), got %s", items[3].SessionID)
	}
}

func TestComputeDomainSummary(t *testing.T) {
	severities := []TriageSeverity{SeverityCritical, SeverityCritical, SeverityWarning, SeverityInfo}
	s := ComputeDomainSummary(severities, 6)

	if s.Total != 6 {
		t.Errorf("Total = %d, want 6", s.Total)
	}
	if s.Critical != 2 {
		t.Errorf("Critical = %d, want 2", s.Critical)
	}
	if s.Warning != 1 {
		t.Errorf("Warning = %d, want 1", s.Warning)
	}
	if s.Info != 1 {
		t.Errorf("Info = %d, want 1", s.Info)
	}
	if s.Clean != 2 {
		t.Errorf("Clean = %d, want 2", s.Clean)
	}
}

func TestSortDecisionItems(t *testing.T) {
	items := []DecisionTriageItem{
		{CorrelationID: "d1", Severity: SeverityInfo, Violations: 0},
		{CorrelationID: "d2", Severity: SeverityCritical, Violations: 1},
		{CorrelationID: "d3", Severity: SeverityCritical, Violations: 3},
	}

	SortDecisionItems(items)

	if items[0].CorrelationID != "d3" {
		t.Errorf("expected d3 first, got %s", items[0].CorrelationID)
	}
	if items[1].CorrelationID != "d2" {
		t.Errorf("expected d2 second, got %s", items[1].CorrelationID)
	}
}

func TestSortRoundTripItems(t *testing.T) {
	items := []RoundTripTriageItem{
		{CorrelationID: "r1", Severity: SeverityInfo, FlagCount: 0},
		{CorrelationID: "r2", Severity: SeverityWarning, FlagCount: 2},
		{CorrelationID: "r3", Severity: SeverityCritical, FlagCount: 4},
	}

	SortRoundTripItems(items)

	if items[0].CorrelationID != "r3" {
		t.Errorf("expected r3 first, got %s", items[0].CorrelationID)
	}
}
