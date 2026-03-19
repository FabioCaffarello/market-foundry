package execution_test

import (
	"testing"
	"time"

	appexec "internal/application/execution"
	domainexec "internal/domain/execution"
)

func TestPaperFillSimulator_BuyOrder_ProducesFilled(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()

	result, ok := sim.SimulateFill(intent)
	if !ok {
		t.Fatal("expected simulation to succeed")
	}
	if result.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled, got %q", result.Status)
	}
	if result.FilledQuantity != "0.02" {
		t.Fatalf("expected filled_quantity 0.02, got %q", result.FilledQuantity)
	}
	if len(result.Fills) != 1 {
		t.Fatalf("expected 1 fill record, got %d", len(result.Fills))
	}
	if !result.Fills[0].Simulated {
		t.Fatal("expected fill to be simulated")
	}
	if result.Fills[0].Quantity != "0.02" {
		t.Fatalf("expected fill quantity 0.02, got %q", result.Fills[0].Quantity)
	}
	if result.Fills[0].Price != "0" {
		t.Fatalf("expected fill price 0 (paper), got %q", result.Fills[0].Price)
	}
}

func TestPaperFillSimulator_SellOrder_ProducesFilled(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()
	intent.Side = domainexec.SideSell

	result, ok := sim.SimulateFill(intent)
	if !ok {
		t.Fatal("expected simulation to succeed")
	}
	if result.Status != domainexec.StatusFilled {
		t.Fatalf("expected StatusFilled, got %q", result.Status)
	}
}

func TestPaperFillSimulator_NoAction_StaysSubmitted(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()
	intent.Side = domainexec.SideNone
	intent.Quantity = "0"

	result, ok := sim.SimulateFill(intent)
	if !ok {
		t.Fatal("expected simulation to succeed")
	}
	if result.Status != domainexec.StatusSubmitted {
		t.Fatalf("expected StatusSubmitted for no-action, got %q", result.Status)
	}
	if len(result.Fills) != 0 {
		t.Fatalf("expected 0 fill records for no-action, got %d", len(result.Fills))
	}
	if result.FilledQuantity != "" {
		t.Fatalf("expected empty filled_quantity for no-action, got %q", result.FilledQuantity)
	}
}

func TestPaperFillSimulator_NonSubmittedStatus_ReturnsFalse(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()
	intent.Status = domainexec.StatusFilled

	_, ok := sim.SimulateFill(intent)
	if ok {
		t.Fatal("expected simulation to fail for non-submitted status")
	}
}

func TestPaperFillSimulator_PreservesOriginalFields(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()

	result, ok := sim.SimulateFill(intent)
	if !ok {
		t.Fatal("expected simulation to succeed")
	}
	if result.Type != intent.Type {
		t.Fatalf("type changed: %q → %q", intent.Type, result.Type)
	}
	if result.Source != intent.Source {
		t.Fatalf("source changed: %q → %q", intent.Source, result.Source)
	}
	if result.Symbol != intent.Symbol {
		t.Fatalf("symbol changed: %q → %q", intent.Symbol, result.Symbol)
	}
	if result.Side != intent.Side {
		t.Fatalf("side changed: %q → %q", intent.Side, result.Side)
	}
	if result.Quantity != intent.Quantity {
		t.Fatalf("quantity changed: %q → %q", intent.Quantity, result.Quantity)
	}
	if result.Risk.Type != intent.Risk.Type {
		t.Fatalf("risk.type changed: %q → %q", intent.Risk.Type, result.Risk.Type)
	}
}

func TestPaperFillSimulator_FilledIntentPassesValidation(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	intent := submittedBuyIntent()

	result, _ := sim.SimulateFill(intent)
	if prob := result.Validate(); prob != nil {
		t.Fatalf("filled intent should be valid: %s", prob.Message)
	}
}

func TestPaperFillSimulator_MultiSymbol_IndependentFills(t *testing.T) {
	sim := &appexec.PaperFillSimulator{}
	symbols := []string{"btcusdt", "ethusdt", "solusdt"}

	for _, sym := range symbols {
		intent := submittedBuyIntent()
		intent.Symbol = sym

		result, ok := sim.SimulateFill(intent)
		if !ok {
			t.Fatalf("simulation failed for %s", sym)
		}
		if result.Symbol != sym {
			t.Fatalf("symbol bleed: expected %q, got %q", sym, result.Symbol)
		}
		if result.Status != domainexec.StatusFilled {
			t.Fatalf("%s: expected StatusFilled, got %q", sym, result.Status)
		}
		if result.PartitionKey() == "" {
			t.Fatalf("%s: empty partition key", sym)
		}
	}
}

func submittedBuyIntent() domainexec.ExecutionIntent {
	return domainexec.ExecutionIntent{
		Type:      "paper_order",
		Source:    "binancef",
		Symbol:    "btcusdt",
		Timeframe: 60,
		Side:      domainexec.SideBuy,
		Quantity:  "0.02",
		Status:    domainexec.StatusSubmitted,
		Risk: domainexec.RiskInput{
			Type:        "position_exposure",
			Disposition: "approved",
			Confidence:  "0.85",
			Timeframe:   60,
		},
		Parameters: map[string]string{"max_position_pct": "0.02"},
		Final:      true,
		Timestamp:  time.Now().UTC(),
	}
}
