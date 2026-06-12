// Package capabilities defines the declarative capabilities
// contract every venue adapter ships, per ADR-0022 R1 (multi-venue
// normalization policy, Onda H-7.a).
//
// The declaration is explicit and static: it is not inferred from
// runtime traffic, and an adapter that publishes events outside its
// declared Capabilities() is an architectural bug (enforced
// statically by `raccoon-cli check venue-parity` per R4, and at the
// producer boundary by the R3 guard + undeclared-event counter).
package capabilities

import (
	"internal/domain/instrument"
	"internal/shared/problem"
)

// Capabilities is the structured declaration of which event types a
// venue adapter supports for which contract types (ADR-0022 R1).
type Capabilities struct {
	// Venue identifies the adapter family (e.g. VenueBinance).
	Venue instrument.Venue `json:"venue"`
	// EventTypes lists the declared event types and the contract
	// types each one is supported for.
	EventTypes []EventTypeSupport `json:"event_types"`
	// Contracts lists the contract families the adapter declares
	// overall. Every contract listed here must appear in at least
	// one EventTypeSupport (R4: an adapter that supports a contract
	// but declares zero event types for it is misconfigured).
	Contracts []instrument.ContractType `json:"contracts"`
	// Notes carries free-form per-venue annotations (operational
	// caveats, gating references such as G10).
	Notes map[string]string `json:"notes,omitempty"`
}

// EventTypeSupport declares one event type and the contract types
// for which the adapter supports it.
type EventTypeSupport struct {
	// Type is the canonical event-type name (e.g.
	// "observation.trade", "observation.markprice").
	Type string `json:"type"`
	// Contracts are the contract types this event type is
	// supported for.
	Contracts []instrument.ContractType `json:"contracts"`
}

// Allows reports whether the declaration covers the given
// (eventType, contract) pair. This is the R3 producer-boundary
// check: an adapter receiving a venue-native event for an
// undeclared pair must not publish it (silently rejected +
// undeclared-event counter increment).
func (c Capabilities) Allows(eventType string, contract instrument.ContractType) bool {
	for _, et := range c.EventTypes {
		if et.Type != eventType {
			continue
		}
		for _, ct := range et.Contracts {
			if ct == contract {
				return true
			}
		}
	}
	return false
}

// Validate checks the declaration's internal coherence (the runtime
// mirror of the `check venue-parity` static rules):
//
//   - Venue is a recognized enum value.
//   - Every event type has a non-empty Type and at least one
//     contract, each of them valid and listed in Contracts.
//   - Every declared contract appears in at least one event type
//     (R4: zero event types for a declared contract is
//     misconfiguration).
//
// An empty declaration (no contracts, no event types) is permitted
// here — the analyzer requires an explicit justifying comment at
// the declaration site (ADR-0022 R4).
func (c Capabilities) Validate() *problem.Problem {
	if prob := c.Venue.Validate(); prob != nil {
		return prob
	}

	declared := make(map[instrument.ContractType]bool, len(c.Contracts))
	covered := make(map[instrument.ContractType]bool, len(c.Contracts))
	for _, ct := range c.Contracts {
		if prob := ct.Validate(); prob != nil {
			return prob
		}
		declared[ct] = true
	}

	for _, et := range c.EventTypes {
		if et.Type == "" {
			return problem.Validation(
				problem.InvalidArgument,
				"capabilities declaration is invalid",
				problem.ValidationIssue{Field: "event_types.type", Message: "must not be empty"},
			)
		}
		if len(et.Contracts) == 0 {
			return problem.Validation(
				problem.InvalidArgument,
				"capabilities declaration is invalid",
				problem.ValidationIssue{
					Field:   "event_types.contracts",
					Message: "must declare at least one contract type",
					Value:   et.Type,
				},
			)
		}
		for _, ct := range et.Contracts {
			if prob := ct.Validate(); prob != nil {
				return prob
			}
			if !declared[ct] {
				return problem.Validation(
					problem.InvalidArgument,
					"capabilities declaration is invalid",
					problem.ValidationIssue{
						Field:   "event_types.contracts",
						Message: "contract not listed in the top-level Contracts declaration",
						Value:   et.Type + "/" + string(ct),
					},
				)
			}
			covered[ct] = true
		}
	}

	for _, ct := range c.Contracts {
		if !covered[ct] {
			return problem.Validation(
				problem.InvalidArgument,
				"capabilities declaration is invalid",
				problem.ValidationIssue{
					Field:   "contracts",
					Message: "declared contract has zero event types (ADR-0022 R4 misconfiguration)",
					Value:   string(ct),
				},
			)
		}
	}

	return nil
}
