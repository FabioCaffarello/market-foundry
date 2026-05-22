package settings

import (
	"fmt"
	"internal/shared/problem"
	"strings"
	"time"
)

type AppConfig struct {
	Log        LogConfig        `json:"log"`
	HTTP       HTTPConfig       `json:"http"`
	NATS       NATSConfig       `json:"nats"`
	Venue      VenueConfig      `json:"venue"`
	Pipeline   PipelineConfig   `json:"pipeline"`
	ClickHouse ClickHouseConfig `json:"clickhouse"`
}

// ClickHouseConfig holds ClickHouse connection and batching parameters.
// Optional — only used by the writer binary. Other services ignore this section.
type ClickHouseConfig struct {
	Addr           string `json:"addr"`
	Database       string `json:"database"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	BatchSize      int    `json:"batch_size"`
	FlushInterval  string `json:"flush_interval"`
	MaxPending     int    `json:"max_pending"`
	MaxRetries     int    `json:"max_retries"`
	InitialBackoff string `json:"initial_backoff"`
}

// Validate checks that ClickHouse config is structurally valid when configured.
// Returns nil if addr is empty (not configured).
func (c ClickHouseConfig) Validate() *problem.Problem {
	if c.Addr == "" {
		return nil // optional — not every binary uses ClickHouse
	}
	return c.validateFields()
}

// ValidateForWriter checks that ClickHouse config is complete and structurally
// valid for the writer binary, where ClickHouse is a hard requirement.
// Unlike Validate(), this method rejects an empty addr as a configuration error.
func (c ClickHouseConfig) ValidateForWriter() *problem.Problem {
	var issues []problem.ValidationIssue
	if c.Addr == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.addr",
			Message: "must not be empty — writer requires a ClickHouse connection",
		})
	}
	if c.Database == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.database",
			Message: "must not be empty when clickhouse is configured",
		})
	}
	if c.Username == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.username",
			Message: "must not be empty — ClickHouse requires authentication credentials",
		})
	}
	issues = append(issues, c.validateBatchingFields()...)
	if len(issues) == 0 {
		return nil
	}
	return validationProblem("clickhouse config is invalid for writer", issues...)
}

// validateFields checks structural validity of all ClickHouse config fields.
func (c ClickHouseConfig) validateFields() *problem.Problem {
	var issues []problem.ValidationIssue
	if c.Database == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.database",
			Message: "must not be empty when clickhouse is configured",
		})
	}
	issues = append(issues, c.validateBatchingFields()...)
	if len(issues) == 0 {
		return nil
	}
	return validationProblem("clickhouse config is invalid", issues...)
}

// validateBatchingFields checks batching-related fields (shared between Validate and ValidateForWriter).
func (c ClickHouseConfig) validateBatchingFields() []problem.ValidationIssue {
	var issues []problem.ValidationIssue
	if c.BatchSize < 0 {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.batch_size",
			Message: "must not be negative",
			Value:   c.BatchSize,
		})
	}
	if c.MaxPending < 0 {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.max_pending",
			Message: "must not be negative",
			Value:   c.MaxPending,
		})
	}
	if c.MaxRetries < 0 {
		issues = append(issues, problem.ValidationIssue{
			Field:   "clickhouse.max_retries",
			Message: "must not be negative",
			Value:   c.MaxRetries,
		})
	}
	issues = append(issues, durationIssue("clickhouse.flush_interval", c.FlushInterval)...)
	issues = append(issues, durationIssue("clickhouse.initial_backoff", c.InitialBackoff)...)
	return issues
}

// BatchSizeOrDefault returns the configured batch size or 1000.
func (c ClickHouseConfig) BatchSizeOrDefault() int {
	if c.BatchSize <= 0 {
		return 1000
	}
	return c.BatchSize
}

// FlushIntervalOrDefault returns the configured flush interval or 5s.
func (c ClickHouseConfig) FlushIntervalOrDefault() time.Duration {
	return parseDurationOrDefault(c.FlushInterval, 5*time.Second)
}

// MaxPendingOrDefault returns the configured max pending or 10000.
func (c ClickHouseConfig) MaxPendingOrDefault() int {
	if c.MaxPending <= 0 {
		return 10000
	}
	return c.MaxPending
}

// MaxRetriesOrDefault returns the configured max retries or 5.
func (c ClickHouseConfig) MaxRetriesOrDefault() int {
	if c.MaxRetries <= 0 {
		return 5
	}
	return c.MaxRetries
}

// InitialBackoffOrDefault returns the configured initial backoff or 1s.
func (c ClickHouseConfig) InitialBackoffOrDefault() time.Duration {
	return parseDurationOrDefault(c.InitialBackoff, 1*time.Second)
}

// ── Known family registries ─────────────────────────────────────────
// These are the canonical family names recognized by the system.
// Any name not in these sets is rejected at config validation time.

var knownEvidenceFamilies = map[string]bool{
	"candle":     true,
	"tradeburst": true,
	"volume":     true,
}

var knownSignalFamilies = map[string]bool{
	"rsi":           true,
	"ema":           true,
	"ema_crossover": true,
	"bollinger":     true,
	"macd":          true,
	"vwap":          true,
	"atr":           true,
}

var knownDecisionFamilies = map[string]bool{
	"rsi_oversold":     true,
	"bollinger_squeeze": true,
}

var knownStrategyFamilies = map[string]bool{
	"mean_reversion_entry":    true,
	"trend_following_entry":   true,
	"squeeze_breakout_entry":  true,
}

var knownRiskFamilies = map[string]bool{
	"position_exposure": true,
	"drawdown_limit":    true,
}

// knownExecutionFamilies lists the two execution families with distinct ownership:
//
//   paper_order:        Paper family — derive-owned intent events (simulated evaluation).
//                       Stream: EXECUTION_EVENTS. KV: EXECUTION_PAPER_ORDER_LATEST.
//
//   venue_market_order: Venue family — execute-owned fill events (venue submission results).
//                       Stream: EXECUTION_FILL_EVENTS. KV: EXECUTION_VENUE_MARKET_ORDER_LATEST.
//
// Both families can be enabled simultaneously. Enabling venue_market_order does NOT
// disable paper_order — they coexist with independent streams, consumers, and projections.
var knownExecutionFamilies = map[string]bool{
	"paper_order":        true,
	"venue_market_order": true,
}

// ── Cross-layer dependency rules ────────────────────────────────────
// Each signal family declares which evidence families it requires.
// Each decision family declares which signal families it requires.
// Each strategy family declares which decision families it requires.

var signalDependsOnEvidence = map[string][]string{
	"rsi":           {"candle"},
	"ema":           {"candle"},
	"ema_crossover": {"candle"},
	"bollinger":     {"candle"},
	"vwap":          {"candle"},
	"atr":           {"candle"},
}

var decisionDependsOnSignal = map[string][]string{
	"rsi_oversold":      {"rsi"},
	"bollinger_squeeze": {"bollinger"},
}

var strategyDependsOnDecision = map[string][]string{
	"mean_reversion_entry":   {"rsi_oversold"},
	"trend_following_entry":  {"ema_crossover"},
	"squeeze_breakout_entry": {"bollinger_squeeze"},
}

var riskDependsOnStrategy = map[string][]string{
	"position_exposure": {"mean_reversion_entry", "trend_following_entry", "squeeze_breakout_entry"},
	"drawdown_limit":    {"mean_reversion_entry", "trend_following_entry", "squeeze_breakout_entry"},
}

var executionDependsOnRisk = map[string][]string{
	"paper_order":        {"position_exposure", "drawdown_limit"},
	"venue_market_order": {"position_exposure", "drawdown_limit"},
}

// ── Venue adapter types ───────────────────────────────────────────
// These are the valid venue adapter types. Only paper_simulator is approved
// by default. Adding a new type requires an activation gate ceremony.

type VenueType string

const (
	VenueTypePaperSimulator         VenueType = "paper_simulator"
	VenueTypeBinanceFuturesTestnet  VenueType = "binance_futures_testnet"
	VenueTypeBinanceSpotTestnet     VenueType = "binance_spot_testnet"
	VenueTypeBinanceFuturesMainnet  VenueType = "binance_futures_mainnet"
	VenueTypeBinanceSpotMainnet     VenueType = "binance_spot_mainnet"
)

var knownVenueTypes = map[VenueType]bool{
	VenueTypePaperSimulator:        true,
	VenueTypeBinanceFuturesTestnet: true,
	VenueTypeBinanceSpotTestnet:    true,
	VenueTypeBinanceFuturesMainnet: true,
	VenueTypeBinanceSpotMainnet:    true,
}

// ── Market segment types ────────────────────────────────────────────

type MarketSegment string

const (
	MarketSegmentSpot    MarketSegment = "spot"
	MarketSegmentFutures MarketSegment = "futures"
)

// Segment returns the market segment implied by a VenueType.
// Returns empty string for paper_simulator and unknown types.
func (v VenueType) Segment() MarketSegment {
	switch v {
	case VenueTypeBinanceFuturesTestnet, VenueTypeBinanceFuturesMainnet:
		return MarketSegmentFutures
	case VenueTypeBinanceSpotTestnet, VenueTypeBinanceSpotMainnet:
		return MarketSegmentSpot
	default:
		return ""
	}
}

// Environment returns "testnet", "mainnet", or empty for non-exchange types.
func (v VenueType) Environment() string {
	switch v {
	case VenueTypeBinanceFuturesTestnet, VenueTypeBinanceSpotTestnet:
		return "testnet"
	case VenueTypeBinanceFuturesMainnet, VenueTypeBinanceSpotMainnet:
		return "mainnet"
	default:
		return ""
	}
}

// IsMainnet reports whether this adapter targets mainnet endpoints.
func (v VenueType) IsMainnet() bool {
	return v.Environment() == "mainnet"
}

// RequiresSegmentConfig reports whether the venue type requires explicit
// segment enablement in configuration. Paper simulator does not.
func (v VenueType) RequiresSegmentConfig() bool {
	return v.Segment() != ""
}

// sourceForSegment maps each market segment to the canonical source prefix
// used in ingest binding topics and execution intent Source fields.
// S400: Required for multi-segment routing — the execute binary uses this
// to dispatch intents to the correct segment adapter.
var sourceForSegment = map[MarketSegment]string{
	MarketSegmentFutures: "binancef",
	MarketSegmentSpot:    "binances",
}

// SourceForSegment returns the canonical source prefix for a market segment
// (e.g., "binancef" for futures, "binances" for spot).
// Returns empty string for unknown segments.
func SourceForSegment(seg MarketSegment) string {
	return sourceForSegment[seg]
}

// SegmentForSource returns the market segment implied by a source prefix.
// Returns empty string for unknown sources.
func SegmentForSource(source string) MarketSegment {
	for seg, src := range sourceForSegment {
		if src == source {
			return seg
		}
	}
	return ""
}

// ── Segment venue config ────────────────────────────────────────────
// S399: Per-segment adapter configuration. Each market segment carries its
// own adapter type and enabled flag, allowing Spot, Futures, or both to be
// governed by a single unified config file.
//
// Fail-closed: absent or nil entry means NOT enabled. A segment must have
// enabled=true AND a valid adapter to be active.

type SegmentVenueConfig struct {
	Enabled bool      `json:"enabled"`
	Adapter VenueType `json:"adapter"`
}

// knownMarketSegments is the canonical set of recognized segment keys.
var knownMarketSegments = map[MarketSegment]bool{
	MarketSegmentSpot:    true,
	MarketSegmentFutures: true,
}

// adapterSegmentCompatibility maps each segment-requiring adapter to its
// implied segment. Used by validation to reject adapter/segment mismatches.
var adapterSegmentCompatibility = map[VenueType]MarketSegment{
	VenueTypeBinanceFuturesTestnet: MarketSegmentFutures,
	VenueTypeBinanceSpotTestnet:    MarketSegmentSpot,
	VenueTypeBinanceFuturesMainnet: MarketSegmentFutures,
	VenueTypeBinanceSpotMainnet:    MarketSegmentSpot,
}

// VenueConfig controls venue adapter selection for the execute binary.
// Optional for binaries that don't submit orders (derive, store, gateway).
//
// S399: Two modes of adapter selection:
//   - Standalone: venue.type selects a single adapter. Valid for
//     paper_simulator or when no segments are defined.
//   - Segments-based: venue.segments maps each market segment to its
//     adapter config. Supports Spot, Futures, or both in one config.
//     When segments are present, Type must be empty or paper_simulator.
type VenueConfig struct {
	Type            VenueType      `json:"type,omitempty"`
	StalenessMaxAge string         `json:"staleness_max_age,omitempty"`
	SubmitTimeout   string         `json:"submit_timeout,omitempty"`
	// DryRun governs whether the execution pipeline may submit real orders.
	// When true (the default), a DryRunSubmitter intercepts all venue calls
	// and produces auditable dry-run receipts instead of reaching the venue.
	// Fail-closed: omitted or null is treated as true.
	// S379: Setting this to false requires venue.type != paper_simulator.
	// S399: Applies uniformly to all enabled segments.
	DryRun          *bool          `json:"dry_run,omitempty"`
	// Segments maps market segment names to their adapter configuration.
	// S399: When present and at least one segment is enabled, segments govern
	// adapter selection. Each segment carries its own adapter type and enabled flag.
	// Fail-closed: absent map means no segments active — only paper_simulator allowed.
	Segments        map[MarketSegment]*SegmentVenueConfig `json:"segments,omitempty"`
	// CredentialProvider selects the backend for credential resolution.
	// S439: Allowed values: "env" (default), "file".
	// "env"  — reads MF_VENUE_{TYPE}_{KEY} from environment variables.
	// "file" — reads from files at {credential_path}/{venue_type}/{KEY}.
	// Fail-closed: unrecognized values are rejected at config validation.
	CredentialProvider string `json:"credential_provider,omitempty"`
	// CredentialPath is the base directory for the "file" credential provider.
	// S439: Required when credential_provider is "file". Ignored otherwise.
	// Expected layout: {credential_path}/{venue_type}/{KEY}
	// Compatible with Docker secrets, K8s secrets, Vault Agent, AWS ESO.
	CredentialPath string `json:"credential_path,omitempty"`
}

// HasUnifiedSegments reports whether the config uses the segments map
// for adapter selection (S399 unified model).
func (v VenueConfig) HasUnifiedSegments() bool {
	return len(v.Segments) > 0
}

// EnabledSegments returns the list of enabled market segments in canonical
// order (spot before futures). Returns nil when no segments are enabled.
func (v VenueConfig) EnabledSegments() []MarketSegment {
	var segs []MarketSegment
	for _, seg := range []MarketSegment{MarketSegmentSpot, MarketSegmentFutures} {
		if cfg, ok := v.Segments[seg]; ok && cfg != nil && cfg.Enabled {
			segs = append(segs, seg)
		}
	}
	return segs
}

// IsSegmentEnabled reports whether the given market segment is enabled.
// Fail-closed: absent map or absent entry means not enabled.
func (v VenueConfig) IsSegmentEnabled(seg MarketSegment) bool {
	cfg, ok := v.Segments[seg]
	return ok && cfg != nil && cfg.Enabled
}

// EnabledSegmentSources returns the canonical source prefixes for all enabled
// segments (e.g., ["binances", "binancef"]). Returns nil when no segments are
// enabled or when the standalone Type-based config is used.
// S401: Used by the execute binary to build segment-scoped consumer filters.
func (v VenueConfig) EnabledSegmentSources() []string {
	segs := v.EnabledSegments()
	if len(segs) == 0 {
		return nil
	}
	sources := make([]string, 0, len(segs))
	for _, seg := range segs {
		if src := SourceForSegment(seg); src != "" {
			sources = append(sources, src)
		}
	}
	return sources
}

// AdapterForSegment returns the venue adapter type for the given segment.
// Returns empty string if the segment is not configured.
func (v VenueConfig) AdapterForSegment(seg MarketSegment) VenueType {
	cfg, ok := v.Segments[seg]
	if !ok || cfg == nil {
		return ""
	}
	return cfg.Adapter
}

// Validate checks that the venue config is structurally valid.
// S399: Supports both standalone Type-based and segments-based configs.
func (v VenueConfig) Validate() *problem.Problem {
	if v.Type == "" && !v.HasUnifiedSegments() {
		return nil // optional — not every binary uses venue
	}
	var issues []problem.ValidationIssue

	// Validate Type if set.
	if v.Type != "" && !knownVenueTypes[v.Type] {
		allowed := make([]string, 0, len(knownVenueTypes))
		for vt := range knownVenueTypes {
			allowed = append(allowed, string(vt))
		}
		issues = append(issues, problem.ValidationIssue{
			Field:   "venue.type",
			Message: fmt.Sprintf("unknown venue type %q; allowed: %s", v.Type, strings.Join(allowed, ", ")),
			Value:   string(v.Type),
		})
	}

	// Validate duration fields.
	if v.StalenessMaxAge != "" {
		d, err := time.ParseDuration(v.StalenessMaxAge)
		if err != nil {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.staleness_max_age",
				Message: "must be a valid duration",
				Value:   v.StalenessMaxAge,
			})
		} else if d < 30*time.Second || d > 600*time.Second {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.staleness_max_age",
				Message: "must be between 30s and 600s",
				Value:   v.StalenessMaxAge,
			})
		}
	}
	if v.SubmitTimeout != "" {
		d, err := time.ParseDuration(v.SubmitTimeout)
		if err != nil {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.submit_timeout",
				Message: "must be a valid duration",
				Value:   v.SubmitTimeout,
			})
		} else if d < 1*time.Second || d > 60*time.Second {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.submit_timeout",
				Message: "must be between 1s and 60s",
				Value:   v.SubmitTimeout,
			})
		}
	}

	// S379: dry_run=false with paper_simulator is contradictory.
	if v.DryRun != nil && !*v.DryRun {
		if v.Type == VenueTypePaperSimulator || (v.Type == "" && !v.HasUnifiedSegments()) {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.dry_run",
				Message: "dry_run=false requires a venue adapter (paper_simulator is inherently dry-run)",
			})
		}
	}

	// S445: dry_run=false is now authorized for mainnet adapters.
	// Authorization: S443 evidence gate, condition C-6.
	// Scope: Binance Spot, BTCUSDT, market order, minimum quantity, supervised ceremony.
	// Fail-closed behavior is preserved by IsDryRun() defaulting to true when omitted.
	// DryRunSubmitter, SafetyGate, and kill-switch remain fully intact for all profiles
	// where dry_run is not explicitly set to false.
	// Reversal: restore the validation block from S433 (git revert).

	// S439: Credential provider validation.
	issues = append(issues, v.validateCredentialProvider()...)

	// S399: Segment enablement validation — fail-closed.
	issues = append(issues, v.validateSegmentEnablement()...)

	if len(issues) == 0 {
		return nil
	}
	return validationProblem("venue config is invalid", issues...)
}

// knownCredentialProviders lists the allowed credential provider backends.
// S439: "env" is the default; "file" reads from mounted secret files.
var knownCredentialProviders = map[string]bool{
	"":     true, // empty = default to "env"
	"env":  true,
	"file": true,
}

// CredentialProviderName returns the effective credential provider name.
// S439: Defaults to "env" when omitted.
func (v VenueConfig) CredentialProviderName() string {
	if v.CredentialProvider == "" {
		return "env"
	}
	return v.CredentialProvider
}

// validateCredentialProvider checks that the credential provider config is valid.
// S439: Fail-closed — unknown providers are rejected; "file" requires credential_path.
func (v VenueConfig) validateCredentialProvider() []problem.ValidationIssue {
	var issues []problem.ValidationIssue
	if !knownCredentialProviders[v.CredentialProvider] {
		issues = append(issues, problem.ValidationIssue{
			Field:   "venue.credential_provider",
			Message: fmt.Sprintf("unknown credential provider %q; allowed: env, file", v.CredentialProvider),
			Value:   v.CredentialProvider,
		})
		return issues
	}
	if v.CredentialProviderName() == "file" && v.CredentialPath == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "venue.credential_path",
			Message: "credential_path is required when credential_provider is \"file\"",
		})
	}
	if v.CredentialProvider != "file" && v.CredentialPath != "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "venue.credential_path",
			Message: "credential_path is only used when credential_provider is \"file\"; either set credential_provider to \"file\" or remove credential_path",
		})
	}
	return issues
}

// validateSegmentEnablement validates segment configuration.
// S399: Two modes — standalone (Type-based) and segments-based (Segments map).
func (v VenueConfig) validateSegmentEnablement() []problem.ValidationIssue {
	if !v.HasUnifiedSegments() {
		// No segments map → standalone Type-based mode.
		// If Type requires a segment, reject — must use segments map.
		seg := v.Type.Segment()
		if seg != "" {
			return []problem.ValidationIssue{{
				Field:   "venue.segments",
				Message: fmt.Sprintf("venue type %q requires segments config; add segments map with %s segment enabled", v.Type, seg),
			}}
		}
		return nil
	}

	// Segments map is present — validate unified model.
	var issues []problem.ValidationIssue

	// S399: When segments map is used, Type must be empty or paper_simulator.
	// A segment-requiring Type (e.g., binance_futures_testnet) creates ambiguity.
	if v.Type != "" && v.Type != VenueTypePaperSimulator {
		if v.Type.Segment() != "" {
			issues = append(issues, problem.ValidationIssue{
				Field:   "venue.type",
				Message: fmt.Sprintf("venue.type %q selects a segment adapter; use segments map instead (set type to empty or paper_simulator)", v.Type),
			})
		}
	}

	// Validate each segment entry.
	enabledCount := 0
	for seg, cfg := range v.Segments {
		// Unknown segment key.
		if !knownMarketSegments[seg] {
			issues = append(issues, problem.ValidationIssue{
				Field:   fmt.Sprintf("venue.segments.%s", seg),
				Message: fmt.Sprintf("unknown market segment %q; allowed: spot, futures", seg),
			})
			continue
		}
		if cfg == nil {
			continue
		}
		if !cfg.Enabled {
			continue
		}
		enabledCount++

		// Enabled segment must have an adapter.
		if cfg.Adapter == "" {
			issues = append(issues, problem.ValidationIssue{
				Field:   fmt.Sprintf("venue.segments.%s.adapter", seg),
				Message: fmt.Sprintf("enabled segment %q must have an adapter configured", seg),
			})
			continue
		}

		// Adapter must be a known venue type.
		if !knownVenueTypes[cfg.Adapter] {
			issues = append(issues, problem.ValidationIssue{
				Field:   fmt.Sprintf("venue.segments.%s.adapter", seg),
				Message: fmt.Sprintf("unknown adapter %q for segment %q", cfg.Adapter, seg),
			})
			continue
		}

		// Paper simulator cannot be used as a segment adapter.
		if cfg.Adapter == VenueTypePaperSimulator {
			issues = append(issues, problem.ValidationIssue{
				Field:   fmt.Sprintf("venue.segments.%s.adapter", seg),
				Message: fmt.Sprintf("paper_simulator cannot be used as a segment adapter for %q", seg),
			})
			continue
		}

		// Adapter must be compatible with the segment.
		expectedSeg, ok := adapterSegmentCompatibility[cfg.Adapter]
		if ok && expectedSeg != seg {
			issues = append(issues, problem.ValidationIssue{
				Field:   fmt.Sprintf("venue.segments.%s.adapter", seg),
				Message: fmt.Sprintf("adapter %q is for segment %q, not %q", cfg.Adapter, expectedSeg, seg),
			})
		}
	}

	// At least one segment must be enabled when segments map is present.
	if enabledCount == 0 {
		issues = append(issues, problem.ValidationIssue{
			Field:   "venue.segments",
			Message: "segments map is present but no segments are enabled; enable at least one or remove segments",
		})
	}

	return issues
}

// IsDryRun reports whether dry-run mode is active.
// Fail-closed: returns true when DryRun is nil (omitted) or explicitly true.
// S379: the only way to disable dry-run is to set dry_run=false explicitly.
func (v VenueConfig) IsDryRun() bool {
	return v.DryRun == nil || *v.DryRun
}

// StalenessMaxAgeDuration returns the configured staleness max age or the default (120s).
func (v VenueConfig) StalenessMaxAgeDuration() time.Duration {
	return parseDurationOrDefault(v.StalenessMaxAge, 120*time.Second)
}

// SubmitTimeoutDuration returns the configured venue submit timeout or the default (10s).
func (v VenueConfig) SubmitTimeoutDuration() time.Duration {
	return parseDurationOrDefault(v.SubmitTimeout, 10*time.Second)
}

// PipelineConfig holds optional processing parameters used by derive and store.
type PipelineConfig struct {
	Timeframes       []int    `json:"timeframes"`
	Families         []string `json:"families"`
	SignalFamilies   []string `json:"signal_families"`
	DecisionFamilies []string `json:"decision_families"`
	StrategyFamilies []string `json:"strategy_families"`
	RiskFamilies      []string `json:"risk_families"`
	ExecutionFamilies []string `json:"execution_families"`
}

// TimeframeDurations returns the configured timeframes as durations.
// Falls back to [60s] if the list is empty or contains only invalid values.
func (p PipelineConfig) TimeframeDurations() []time.Duration {
	var durations []time.Duration
	for _, secs := range p.Timeframes {
		if secs > 0 {
			durations = append(durations, time.Duration(secs)*time.Second)
		}
	}
	if len(durations) == 0 {
		return []time.Duration{60 * time.Second}
	}
	return durations
}

// IsFamilyEnabled returns true if the given family name is in the configured families list.
// If no families are configured, all families are considered enabled (backward compatible).
func (p PipelineConfig) IsFamilyEnabled(family string) bool {
	if len(p.Families) == 0 {
		return true
	}
	for _, f := range p.Families {
		if f == family {
			return true
		}
	}
	return false
}

// IsSignalFamilyEnabled returns true if the given signal family name is in the configured
// signal_families list. Unlike evidence families, absent signal_families means NO signal
// activation (opt-in, not backward-compatible default).
func (p PipelineConfig) IsSignalFamilyEnabled(family string) bool {
	for _, f := range p.SignalFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EnabledSignalFamilies returns the configured signal families list.
func (p PipelineConfig) EnabledSignalFamilies() []string {
	if len(p.SignalFamilies) == 0 {
		return nil
	}
	result := make([]string, len(p.SignalFamilies))
	copy(result, p.SignalFamilies)
	return result
}

// IsDecisionFamilyEnabled returns true if the given decision family name is in the configured
// decision_families list. Like signal families, absent decision_families means NO decision
// activation (opt-in, not backward-compatible default).
func (p PipelineConfig) IsDecisionFamilyEnabled(family string) bool {
	for _, f := range p.DecisionFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EnabledDecisionFamilies returns the configured decision families list.
func (p PipelineConfig) EnabledDecisionFamilies() []string {
	if len(p.DecisionFamilies) == 0 {
		return nil
	}
	result := make([]string, len(p.DecisionFamilies))
	copy(result, p.DecisionFamilies)
	return result
}

// IsStrategyFamilyEnabled returns true if the given strategy family name is in the configured
// strategy_families list. Like decision families, absent strategy_families means NO strategy
// activation (opt-in, not backward-compatible default).
func (p PipelineConfig) IsStrategyFamilyEnabled(family string) bool {
	for _, f := range p.StrategyFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EnabledStrategyFamilies returns the configured strategy families list.
func (p PipelineConfig) EnabledStrategyFamilies() []string {
	if len(p.StrategyFamilies) == 0 {
		return nil
	}
	result := make([]string, len(p.StrategyFamilies))
	copy(result, p.StrategyFamilies)
	return result
}

// IsRiskFamilyEnabled returns true if the given risk family name is in the configured
// risk_families list. Like strategy families, absent risk_families means NO risk
// activation (opt-in, not backward-compatible default).
func (p PipelineConfig) IsRiskFamilyEnabled(family string) bool {
	for _, f := range p.RiskFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EnabledRiskFamilies returns the configured risk families list.
func (p PipelineConfig) EnabledRiskFamilies() []string {
	if len(p.RiskFamilies) == 0 {
		return nil
	}
	result := make([]string, len(p.RiskFamilies))
	copy(result, p.RiskFamilies)
	return result
}

// IsExecutionFamilyEnabled returns true if the given execution family name is in the configured
// execution_families list. Like risk families, absent execution_families means NO execution
// activation (opt-in, not backward-compatible default).
func (p PipelineConfig) IsExecutionFamilyEnabled(family string) bool {
	for _, f := range p.ExecutionFamilies {
		if f == family {
			return true
		}
	}
	return false
}

// EnabledExecutionFamilies returns the configured execution families list.
func (p PipelineConfig) EnabledExecutionFamilies() []string {
	if len(p.ExecutionFamilies) == 0 {
		return nil
	}
	result := make([]string, len(p.ExecutionFamilies))
	copy(result, p.ExecutionFamilies)
	return result
}

// EnabledFamilies returns the configured families list, or nil if all are enabled.
func (p PipelineConfig) EnabledFamilies() []string {
	if len(p.Families) == 0 {
		return nil
	}
	result := make([]string, len(p.Families))
	copy(result, p.Families)
	return result
}

// ValidateTimeframes checks that configured timeframes are within the
// supported range [10, 86400] seconds and contain no duplicates.
func (p PipelineConfig) ValidateTimeframes() []problem.ValidationIssue {
	var issues []problem.ValidationIssue

	seen := make(map[int]bool, len(p.Timeframes))
	for _, tf := range p.Timeframes {
		if tf < 10 {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.timeframes",
				Message: fmt.Sprintf("timeframe %d is below minimum (10s)", tf),
				Value:   tf,
			})
		} else if tf > 86400 {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.timeframes",
				Message: fmt.Sprintf("timeframe %d exceeds maximum (86400s)", tf),
				Value:   tf,
			})
		}
		if seen[tf] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.timeframes",
				Message: fmt.Sprintf("duplicate timeframe %d", tf),
				Value:   tf,
			})
		}
		seen[tf] = true
	}
	return issues
}

// ValidatePipeline checks that pipeline family names are known, unique, and that
// cross-layer dependency rules are satisfied. It returns nil when valid.
func (p PipelineConfig) ValidatePipeline() *problem.Problem {
	var issues []problem.ValidationIssue

	// 0a. Reject invalid or duplicate timeframes.
	issues = append(issues, p.ValidateTimeframes()...)

	// 0b. Reject duplicate family names in every list.
	issues = append(issues, rejectDuplicates("pipeline.families", p.Families)...)
	issues = append(issues, rejectDuplicates("pipeline.signal_families", p.SignalFamilies)...)
	issues = append(issues, rejectDuplicates("pipeline.decision_families", p.DecisionFamilies)...)
	issues = append(issues, rejectDuplicates("pipeline.strategy_families", p.StrategyFamilies)...)
	issues = append(issues, rejectDuplicates("pipeline.risk_families", p.RiskFamilies)...)
	issues = append(issues, rejectDuplicates("pipeline.execution_families", p.ExecutionFamilies)...)

	// 1. Reject unknown evidence family names (only when explicitly configured).
	for _, f := range p.Families {
		if !knownEvidenceFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.families",
				Message: fmt.Sprintf("unknown evidence family %q", f),
				Value:   f,
			})
		}
	}

	// 2. Reject unknown signal family names.
	for _, f := range p.SignalFamilies {
		if !knownSignalFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.signal_families",
				Message: fmt.Sprintf("unknown signal family %q", f),
				Value:   f,
			})
		}
	}

	// 3. Reject unknown decision family names.
	for _, f := range p.DecisionFamilies {
		if !knownDecisionFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.decision_families",
				Message: fmt.Sprintf("unknown decision family %q", f),
				Value:   f,
			})
		}
	}

	// 4. Signal → evidence dependency: each enabled signal must have its
	//    required evidence families enabled.
	for _, sig := range p.SignalFamilies {
		deps, ok := signalDependsOnEvidence[sig]
		if !ok {
			continue
		}
		for _, ev := range deps {
			if !p.IsFamilyEnabled(ev) {
				issues = append(issues, problem.ValidationIssue{
					Field:   "pipeline.signal_families",
					Message: fmt.Sprintf("signal family %q requires evidence family %q to be enabled", sig, ev),
					Value:   sig,
				})
			}
		}
	}

	// 5. Decision → signal dependency: each enabled decision must have its
	//    required signal families enabled.
	for _, dec := range p.DecisionFamilies {
		deps, ok := decisionDependsOnSignal[dec]
		if !ok {
			continue
		}
		for _, sig := range deps {
			if !p.IsSignalFamilyEnabled(sig) {
				issues = append(issues, problem.ValidationIssue{
					Field:   "pipeline.decision_families",
					Message: fmt.Sprintf("decision family %q requires signal family %q to be enabled", dec, sig),
					Value:   dec,
				})
			}
		}
	}

	// 6. Reject unknown strategy family names.
	for _, f := range p.StrategyFamilies {
		if !knownStrategyFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.strategy_families",
				Message: fmt.Sprintf("unknown strategy family %q", f),
				Value:   f,
			})
		}
	}

	// 7. Strategy → decision dependency: each enabled strategy must have its
	//    required decision families enabled.
	for _, strat := range p.StrategyFamilies {
		deps, ok := strategyDependsOnDecision[strat]
		if !ok {
			continue
		}
		for _, dec := range deps {
			if !p.IsDecisionFamilyEnabled(dec) {
				issues = append(issues, problem.ValidationIssue{
					Field:   "pipeline.strategy_families",
					Message: fmt.Sprintf("strategy family %q requires decision family %q to be enabled", strat, dec),
					Value:   strat,
				})
			}
		}
	}

	// 8. Reject unknown risk family names.
	for _, f := range p.RiskFamilies {
		if !knownRiskFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.risk_families",
				Message: fmt.Sprintf("unknown risk family %q", f),
				Value:   f,
			})
		}
	}

	// 9. Risk → strategy dependency: each enabled risk must have at least
	//    one of its compatible strategy families enabled. Risk evaluators
	//    accept any strategy type (via default scaling factors), so the
	//    constraint is "at least one feeds input", not "all required".
	for _, rsk := range p.RiskFamilies {
		deps, ok := riskDependsOnStrategy[rsk]
		if !ok {
			continue
		}
		hasAny := false
		for _, strat := range deps {
			if p.IsStrategyFamilyEnabled(strat) {
				hasAny = true
				break
			}
		}
		if !hasAny {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.risk_families",
				Message: fmt.Sprintf("risk family %q requires at least one strategy family to be enabled (%s)", rsk, strings.Join(deps, ", ")),
				Value:   rsk,
			})
		}
	}

	// 10. Reject unknown execution family names.
	for _, f := range p.ExecutionFamilies {
		if !knownExecutionFamilies[f] {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.execution_families",
				Message: fmt.Sprintf("unknown execution family %q", f),
				Value:   f,
			})
		}
	}

	// 11. Execution → risk dependency: each enabled execution must have at
	//     least one of its compatible risk families enabled. Execution
	//     evaluators process any risk assessment, so the constraint is
	//     "at least one feeds input", not "all required".
	for _, exec := range p.ExecutionFamilies {
		deps, ok := executionDependsOnRisk[exec]
		if !ok {
			continue
		}
		hasAny := false
		for _, rsk := range deps {
			if p.IsRiskFamilyEnabled(rsk) {
				hasAny = true
				break
			}
		}
		if !hasAny {
			issues = append(issues, problem.ValidationIssue{
				Field:   "pipeline.execution_families",
				Message: fmt.Sprintf("execution family %q requires at least one risk family to be enabled (%s)", exec, strings.Join(deps, ", ")),
				Value:   exec,
			})
		}
	}

	if len(issues) == 0 {
		return nil
	}
	return validationProblem("pipeline config has invalid family dependencies", issues...)
}

// ValidateForWriter checks that pipeline config is valid for the writer binary.
// In addition to the standard pipeline validation, it verifies that at least one
// writer-compatible family is explicitly enabled, preventing a startup that would
// immediately report zero pipelines.
func (p PipelineConfig) ValidateForWriter() *problem.Problem {
	// Run standard pipeline validation first.
	if prob := p.ValidatePipeline(); prob != nil {
		return prob
	}

	// The writer requires at least one family to be enabled across any layer.
	hasAny := len(p.Families) > 0 ||
		len(p.SignalFamilies) > 0 ||
		len(p.DecisionFamilies) > 0 ||
		len(p.StrategyFamilies) > 0 ||
		len(p.RiskFamilies) > 0 ||
		len(p.ExecutionFamilies) > 0

	if !hasAny {
		return validationProblem("writer pipeline config requires at least one family enabled",
			problem.ValidationIssue{
				Field:   "pipeline",
				Message: "no families configured — writer has no pipelines to run",
			},
		)
	}
	return nil
}

// Defaults returns the baseline shared application config.
func Defaults() AppConfig {
	return AppConfig{
		Log: LogConfig{
			Level:  LogLevelInfo,
			Format: LogFormatText,
		},
		HTTP: HTTPConfig{
			Addr:            ":8080",
			ReadTimeout:     "10s",
			WriteTimeout:    "15s",
			IdleTimeout:     "60s",
			ShutdownTimeout: "10s",
		},
		NATS: NATSConfig{
			RequestTimeout: "2s",
		},
	}
}

// ApplyDefaults fills empty fields with the package defaults.
func (c *AppConfig) ApplyDefaults() {
	if c == nil {
		return
	}

	defaults := Defaults()
	c.Log.applyDefaults(defaults.Log)
	c.HTTP.applyDefaults(defaults.HTTP)
	c.NATS.applyDefaults(defaults.NATS)
}

// Validate checks whether the config is structurally valid.
func (c AppConfig) Validate() *problem.Problem {
	var issues []problem.ValidationIssue
	issues = append(issues, extractIssues(c.Log.Validate())...)
	issues = append(issues, extractIssues(c.HTTP.Validate())...)
	issues = append(issues, extractIssues(c.NATS.Validate())...)
	issues = append(issues, extractIssues(c.Venue.Validate())...)
	issues = append(issues, extractIssues(c.Pipeline.ValidatePipeline())...)
	issues = append(issues, extractIssues(c.ClickHouse.Validate())...)

	if len(issues) == 0 {
		return nil
	}

	return validationProblem("application config is invalid", issues...)
}

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// LogConfig controls structured logging output.
type LogConfig struct {
	Level  LogLevel  `json:"level"`
	Format LogFormat `json:"format"`
}

func (l *LogConfig) applyDefaults(defaults LogConfig) {
	if l.Level == "" {
		l.Level = defaults.Level
	}
	if l.Format == "" {
		l.Format = defaults.Format
	}
}

func (l LogConfig) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	switch l.Level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		issues = append(issues, problem.ValidationIssue{
			Field:   "log.level",
			Message: "must be one of debug, info, warn or error",
			Value:   l.Level,
		})
	}

	switch l.Format {
	case LogFormatJSON, LogFormatText:
	default:
		issues = append(issues, problem.ValidationIssue{
			Field:   "log.format",
			Message: "must be one of json or text",
			Value:   l.Format,
		})
	}

	if len(issues) == 0 {
		return nil
	}

	return validationProblem("log config is invalid", issues...)
}

// HTTPConfig controls HTTP server defaults shared by services.
type HTTPConfig struct {
	Addr            string `json:"addr"`
	ReadTimeout     string `json:"read_timeout"`
	WriteTimeout    string `json:"write_timeout"`
	IdleTimeout     string `json:"idle_timeout"`
	ShutdownTimeout string `json:"shutdown_timeout"`
}

func (h *HTTPConfig) applyDefaults(defaults HTTPConfig) {
	if strings.TrimSpace(h.Addr) == "" {
		h.Addr = defaults.Addr
	}
	if strings.TrimSpace(h.ReadTimeout) == "" {
		h.ReadTimeout = defaults.ReadTimeout
	}
	if strings.TrimSpace(h.WriteTimeout) == "" {
		h.WriteTimeout = defaults.WriteTimeout
	}
	if strings.TrimSpace(h.IdleTimeout) == "" {
		h.IdleTimeout = defaults.IdleTimeout
	}
	if strings.TrimSpace(h.ShutdownTimeout) == "" {
		h.ShutdownTimeout = defaults.ShutdownTimeout
	}
}

func (h HTTPConfig) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if strings.TrimSpace(h.Addr) == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "http.addr",
			Message: "must not be empty",
		})
	}

	issues = append(issues, durationIssue("http.read_timeout", h.ReadTimeout)...)
	issues = append(issues, durationIssue("http.write_timeout", h.WriteTimeout)...)
	issues = append(issues, durationIssue("http.idle_timeout", h.IdleTimeout)...)
	issues = append(issues, durationIssue("http.shutdown_timeout", h.ShutdownTimeout)...)

	if len(issues) == 0 {
		return nil
	}

	return validationProblem("http config is invalid", issues...)
}

func (h HTTPConfig) ReadTimeoutDuration() time.Duration {
	return parseDurationOrDefault(h.ReadTimeout, 5*time.Second)
}

func (h HTTPConfig) WriteTimeoutDuration() time.Duration {
	return parseDurationOrDefault(h.WriteTimeout, 10*time.Second)
}

func (h HTTPConfig) IdleTimeoutDuration() time.Duration {
	return parseDurationOrDefault(h.IdleTimeout, time.Minute)
}

func (h HTTPConfig) ShutdownTimeoutDuration() time.Duration {
	return parseDurationOrDefault(h.ShutdownTimeout, 10*time.Second)
}

func parseDurationOrDefault(raw string, fallback time.Duration) time.Duration {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

// NATSConfig keeps transport-neutral connection metadata required by NATS-based services.
type NATSConfig struct {
	Enabled        bool   `json:"enabled"`
	URL            string `json:"url"`
	RequestTimeout string `json:"request_timeout"`
}

// JetStreamConfig preserves the previous type name while the shared package converges on transport-agnostic naming.
type JetStreamConfig = NATSConfig

func (c *NATSConfig) applyDefaults(defaults NATSConfig) {
	if strings.TrimSpace(c.RequestTimeout) == "" {
		c.RequestTimeout = defaults.RequestTimeout
	}
}

func (c NATSConfig) Validate() *problem.Problem {
	var issues []problem.ValidationIssue

	if c.Enabled && strings.TrimSpace(c.URL) == "" {
		issues = append(issues, problem.ValidationIssue{
			Field:   "nats.url",
			Message: "must not be empty when nats is enabled",
		})
	}

	issues = append(issues, durationIssue("nats.request_timeout", c.RequestTimeout)...)

	if len(issues) == 0 {
		return nil
	}

	return validationProblem("nats config is invalid", issues...)
}

func (c NATSConfig) RequestTimeoutDuration() time.Duration {
	return parseDurationOrDefault(c.RequestTimeout, 2*time.Second)
}

func durationIssue(field, raw string) []problem.ValidationIssue {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return []problem.ValidationIssue{{
			Field:   field,
			Message: "must be a valid duration",
			Value:   raw,
		}}
	}

	if duration < 0 {
		return []problem.ValidationIssue{{
			Field:   field,
			Message: "must not be negative",
			Value:   raw,
		}}
	}

	return nil
}

func unexpectedJSONTokenError() error {
	return fmt.Errorf("config file contains more than one JSON document")
}

// rejectDuplicates returns a validation issue for each duplicate entry in a list.
func rejectDuplicates(field string, values []string) []problem.ValidationIssue {
	if len(values) <= 1 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	var issues []problem.ValidationIssue
	for _, v := range values {
		if seen[v] {
			issues = append(issues, problem.ValidationIssue{
				Field:   field,
				Message: fmt.Sprintf("duplicate family %q", v),
				Value:   v,
			})
		}
		seen[v] = true
	}
	return issues
}

// ── Canonical family catalog (exported for tooling and coherence tests) ──

// PipelineDomain identifies a bounded-context domain in the pipeline.
type PipelineDomain string

const (
	DomainEvidence  PipelineDomain = "evidence"
	DomainSignal    PipelineDomain = "signal"
	DomainDecision  PipelineDomain = "decision"
	DomainStrategy  PipelineDomain = "strategy"
	DomainRisk      PipelineDomain = "risk"
	DomainExecution PipelineDomain = "execution"
)

// FamilyDependency describes one family and the upstream families it requires.
type FamilyDependency struct {
	Domain       PipelineDomain
	Family       string
	DependsOn    []string          // upstream families (in the immediately preceding domain)
	DependsDomain PipelineDomain   // domain of the dependencies
}

// KnownFamilies returns the canonical set of recognized family names for the given domain.
// Returns nil for unknown domains.
func KnownFamilies(domain PipelineDomain) []string {
	var registry map[string]bool
	switch domain {
	case DomainEvidence:
		registry = knownEvidenceFamilies
	case DomainSignal:
		registry = knownSignalFamilies
	case DomainDecision:
		registry = knownDecisionFamilies
	case DomainStrategy:
		registry = knownStrategyFamilies
	case DomainRisk:
		registry = knownRiskFamilies
	case DomainExecution:
		registry = knownExecutionFamilies
	default:
		return nil
	}
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// IsKnownFamily reports whether the given family is registered in the canonical catalog for the domain.
func IsKnownFamily(domain PipelineDomain, family string) bool {
	switch domain {
	case DomainEvidence:
		return knownEvidenceFamilies[family]
	case DomainSignal:
		return knownSignalFamilies[family]
	case DomainDecision:
		return knownDecisionFamilies[family]
	case DomainStrategy:
		return knownStrategyFamilies[family]
	case DomainRisk:
		return knownRiskFamilies[family]
	case DomainExecution:
		return knownExecutionFamilies[family]
	default:
		return false
	}
}

// DependencyGraph returns the full cross-layer dependency map.
// Each entry describes one family and the upstream families it requires.
// Evidence families have no upstream dependencies and are not included.
func DependencyGraph() []FamilyDependency {
	var graph []FamilyDependency
	for family, deps := range signalDependsOnEvidence {
		graph = append(graph, FamilyDependency{Domain: DomainSignal, Family: family, DependsOn: deps, DependsDomain: DomainEvidence})
	}
	for family, deps := range decisionDependsOnSignal {
		graph = append(graph, FamilyDependency{Domain: DomainDecision, Family: family, DependsOn: deps, DependsDomain: DomainSignal})
	}
	for family, deps := range strategyDependsOnDecision {
		graph = append(graph, FamilyDependency{Domain: DomainStrategy, Family: family, DependsOn: deps, DependsDomain: DomainDecision})
	}
	for family, deps := range riskDependsOnStrategy {
		graph = append(graph, FamilyDependency{Domain: DomainRisk, Family: family, DependsOn: deps, DependsDomain: DomainStrategy})
	}
	for family, deps := range executionDependsOnRisk {
		graph = append(graph, FamilyDependency{Domain: DomainExecution, Family: family, DependsOn: deps, DependsDomain: DomainRisk})
	}
	return graph
}
