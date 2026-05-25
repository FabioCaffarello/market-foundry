// Package contracts hosts the typed proto-derived contracts that
// bridge the foundry's mesh wire format and its domain types.
//
// Per ADR-0018 (protobuf contract layer):
//   - Generated proto types (*.pb.go) live exclusively in this
//     package tree at internal/shared/contracts/<family>/v<n>/.
//   - internal/domain/ MUST NOT import this package or any
//     subpackage (boundary PROTO-G3).
//   - Converters between proto types and foundry domain types
//     live in this package tree as well.
//
// The check proto analyzer (tools/raccoon-cli) enforces these
// boundaries statically. See PROGRAM-0002 (Fase Wire) for the
// delivery roadmap and ADR-0017 for the canonical event envelope
// shape this package serializes.
package contracts
