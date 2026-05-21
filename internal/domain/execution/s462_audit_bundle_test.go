package execution

import (
	"testing"
	"time"
)

func TestNewAuditOrderActivityFromCounters(t *testing.T) {
	counters := []SessionSegmentCounters{
		{Segment: "spot", Processed: 10, Filled: 5, Rejected: 2, Errors: 1},
		{Segment: "futures", Processed: 8, Filled: 3, Rejected: 1, Errors: 0},
	}

	activity := NewAuditOrderActivityFromCounters(counters)

	if !activity.FromSessionCounters {
		t.Error("expected FromSessionCounters=true")
	}
	if activity.TotalIntents != 18 {
		t.Errorf("expected TotalIntents=18, got %d", activity.TotalIntents)
	}
	if activity.TotalFills != 8 {
		t.Errorf("expected TotalFills=8, got %d", activity.TotalFills)
	}
	if activity.TotalRejections != 3 {
		t.Errorf("expected TotalRejections=3, got %d", activity.TotalRejections)
	}
	if activity.TotalErrors != 1 {
		t.Errorf("expected TotalErrors=1, got %d", activity.TotalErrors)
	}
}

func TestNewAuditOrderActivityFromCounters_Empty(t *testing.T) {
	activity := NewAuditOrderActivityFromCounters(nil)
	if !activity.FromSessionCounters {
		t.Error("expected FromSessionCounters=true")
	}
	if activity.TotalIntents != 0 {
		t.Errorf("expected TotalIntents=0, got %d", activity.TotalIntents)
	}
}

func TestNewAuditFeeSummary(t *testing.T) {
	now := time.Now()
	fills := []FillRecord{
		{Price: "100", Quantity: "1", Fee: "0.1", FeeAsset: "BNB", Timestamp: now},
		{Price: "200", Quantity: "2", Fee: "0.2", FeeAsset: "USDT", Timestamp: now},
		{Price: "300", Quantity: "3", Fee: "0", FeeAsset: "", Simulated: true, Timestamp: now},
	}

	summary := NewAuditFeeSummary(fills)

	if summary.TotalFillRecords != 3 {
		t.Errorf("expected TotalFillRecords=3, got %d", summary.TotalFillRecords)
	}
	if summary.FillsWithFee != 2 {
		t.Errorf("expected FillsWithFee=2, got %d", summary.FillsWithFee)
	}
	if summary.FillsWithoutFee != 1 {
		t.Errorf("expected FillsWithoutFee=1, got %d", summary.FillsWithoutFee)
	}
	if summary.SimulatedFills != 1 {
		t.Errorf("expected SimulatedFills=1, got %d", summary.SimulatedFills)
	}
	if summary.FeeCoverageRatio != "2/3" {
		t.Errorf("expected FeeCoverageRatio=2/3, got %s", summary.FeeCoverageRatio)
	}
	if len(summary.FeeAssets) != 2 {
		t.Errorf("expected 2 fee assets, got %d", len(summary.FeeAssets))
	}
}

func TestNewAuditFeeSummary_Empty(t *testing.T) {
	summary := NewAuditFeeSummary(nil)
	if summary.TotalFillRecords != 0 {
		t.Errorf("expected TotalFillRecords=0, got %d", summary.TotalFillRecords)
	}
	if summary.FeeCoverageRatio != "0/0" {
		t.Errorf("expected FeeCoverageRatio=0/0, got %s", summary.FeeCoverageRatio)
	}
}

func TestNewAuditFeeSummary_AllSimulated(t *testing.T) {
	fills := []FillRecord{
		{Price: "100", Quantity: "1", Fee: "0", Simulated: true, Timestamp: time.Now()},
		{Price: "200", Quantity: "2", Fee: "", Simulated: true, Timestamp: time.Now()},
	}

	summary := NewAuditFeeSummary(fills)
	if summary.SimulatedFills != 2 {
		t.Errorf("expected SimulatedFills=2, got %d", summary.SimulatedFills)
	}
	if summary.FillsWithFee != 0 {
		t.Errorf("expected FillsWithFee=0, got %d", summary.FillsWithFee)
	}
	if summary.FeeCoverageRatio != "0/2" {
		t.Errorf("expected FeeCoverageRatio=0/2, got %s", summary.FeeCoverageRatio)
	}
}
