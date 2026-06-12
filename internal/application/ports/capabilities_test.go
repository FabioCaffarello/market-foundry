package ports_test

import (
	"testing"

	"internal/application/ports"
	"internal/domain/instrument"
)

func validDecl() ports.Capabilities {
	return ports.Capabilities{
		Venue: instrument.VenueBinanceFutures,
		EventTypes: []ports.EventTypeSupport{
			{Type: "observation.trade", Contracts: []instrument.ContractType{
				instrument.ContractPerpetual,
				instrument.ContractUSDTFutures,
			}},
		},
		Contracts: []instrument.ContractType{
			instrument.ContractPerpetual,
			instrument.ContractUSDTFutures,
		},
	}
}

func TestAllows_DeclaredPair(t *testing.T) {
	c := validDecl()
	if !c.Allows("observation.trade", instrument.ContractPerpetual) {
		t.Error("declared pair (observation.trade, perpetual) must be allowed")
	}
	if !c.Allows("observation.trade", instrument.ContractUSDTFutures) {
		t.Error("declared pair (observation.trade, usdtfutures) must be allowed")
	}
}

func TestAllows_UndeclaredPairsRejected(t *testing.T) {
	c := validDecl()
	cases := []struct {
		name      string
		eventType string
		contract  instrument.ContractType
	}{
		{"undeclared_event_type", "observation.markprice", instrument.ContractPerpetual},
		{"undeclared_contract", "observation.trade", instrument.ContractSpot},
		{"both_undeclared", "observation.liquidation", instrument.ContractCoinFutures},
		{"empty_event_type", "", instrument.ContractPerpetual},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if c.Allows(tc.eventType, tc.contract) {
				t.Errorf("undeclared pair (%q, %q) must be rejected (ADR-0022 R3)", tc.eventType, tc.contract)
			}
		})
	}
}

func TestValidate_CoherentDeclarationPasses(t *testing.T) {
	if prob := validDecl().Validate(); prob != nil {
		t.Fatalf("coherent declaration failed validation: %v", prob)
	}
}

func TestValidate_Incoherencies(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ports.Capabilities)
	}{
		{"invalid_venue", func(c *ports.Capabilities) { c.Venue = "nyse" }},
		{"empty_event_type_name", func(c *ports.Capabilities) { c.EventTypes[0].Type = "" }},
		{"event_type_without_contracts", func(c *ports.Capabilities) { c.EventTypes[0].Contracts = nil }},
		{"event_contract_not_in_top_level", func(c *ports.Capabilities) {
			c.EventTypes[0].Contracts = append(c.EventTypes[0].Contracts, instrument.ContractSpot)
		}},
		{"declared_contract_with_zero_event_types", func(c *ports.Capabilities) {
			c.Contracts = append(c.Contracts, instrument.ContractSpot)
		}},
		{"invalid_contract", func(c *ports.Capabilities) {
			c.Contracts[0] = "swap"
			c.EventTypes[0].Contracts[0] = "swap"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validDecl()
			tc.mutate(&c)
			if prob := c.Validate(); prob == nil {
				t.Errorf("incoherent declaration (%s) passed validation", tc.name)
			}
		})
	}
}

// An empty declaration is tolerated at runtime (the analyzer
// requires a justifying comment at the declaration site instead —
// ADR-0022 R4).
func TestValidate_EmptyDeclarationPermitted(t *testing.T) {
	c := ports.Capabilities{Venue: instrument.VenueBinance}
	if prob := c.Validate(); prob != nil {
		t.Fatalf("empty declaration must be permitted at runtime: %v", prob)
	}
}
