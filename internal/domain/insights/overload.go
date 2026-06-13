package insights

import "internal/shared/problem"

// OverloadLevel is the bucket-cap pressure of a volume profile window
// (ADR-0027 / PROGRAM-0005 Decisão #5). It is the foundry's port of
// the raccoon VPVR overload levels: a per-window bounded-buckets
// signal that lets the sampler degrade gracefully instead of growing
// an unbounded price map under a pathological tick stream.
//
// Levels are advisory at the domain layer (pure classification); the
// sampler acts on them (coalesce/drop) in the application layer.
type OverloadLevel int

const (
	// OverloadL0 — normal: bucket count well under cap.
	OverloadL0 OverloadLevel = 0
	// OverloadL1 — elevated: ≥50% of cap; observe.
	OverloadL1 OverloadLevel = 1
	// OverloadL2 — high: ≥80% of cap; sampler should coalesce
	// adjacent buckets on the next admission.
	OverloadL2 OverloadLevel = 2
	// OverloadL3 — critical: at/over cap; sampler drops admissions
	// for new price levels (existing buckets still accumulate).
	OverloadL3 OverloadLevel = 3
)

// DefaultMaxBucketsPerWindow caps the number of distinct price levels
// a single profile window may hold, bounding memory under adversarial
// or mis-binned input. Mirrors the raccoon VPVRCapBucketsPerWindow
// intent; the value is a foundry default, tunable per sampler config.
const DefaultMaxBucketsPerWindow = 512

// ClassifyOverload maps a current bucket count against a cap to an
// OverloadLevel. Deterministic and pure. A non-positive cap is
// treated as "no cap" and always returns L0.
func ClassifyOverload(bucketCount, cap int) OverloadLevel {
	if cap <= 0 || bucketCount < 0 {
		return OverloadL0
	}
	switch {
	case bucketCount >= cap:
		return OverloadL3
	case bucketCount*100 >= cap*80:
		return OverloadL2
	case bucketCount*100 >= cap*50:
		return OverloadL1
	default:
		return OverloadL0
	}
}

// Validate reports whether the level is a recognized enum value.
func (l OverloadLevel) Validate() *problem.Problem {
	switch l {
	case OverloadL0, OverloadL1, OverloadL2, OverloadL3:
		return nil
	default:
		return problem.New(problem.InvalidArgument, "overload level is invalid")
	}
}

// AdmitsNewLevel reports whether a window at this overload level still
// admits a brand-new price level. At L3 the sampler stops creating
// new buckets (existing ones keep accumulating).
func (l OverloadLevel) AdmitsNewLevel() bool {
	return l < OverloadL3
}

// Label returns the canonical "L0".."L3" string for the level, used as
// the stored value in the analytical overload column (a readable
// LowCardinality(String)). Unknown values fall back to "L?" rather than
// fabricating a level. This is deliberately NOT a Stringer: fmt/log
// output of OverloadLevel stays numeric to avoid disturbing existing
// formatting expectations.
func (l OverloadLevel) Label() string {
	switch l {
	case OverloadL0:
		return "L0"
	case OverloadL1:
		return "L1"
	case OverloadL2:
		return "L2"
	case OverloadL3:
		return "L3"
	default:
		return "L?"
	}
}
