package settings

import (
	"fmt"
	"internal/shared/problem"
	"strings"
	"time"
)

type AppConfig struct {
	Log      LogConfig      `json:"log"`
	HTTP     HTTPConfig     `json:"http"`
	NATS     NATSConfig     `json:"nats"`
	Pipeline PipelineConfig `json:"pipeline"`
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
	"rsi": true,
}

var knownDecisionFamilies = map[string]bool{
	"rsi_oversold": true,
}

var knownStrategyFamilies = map[string]bool{
	"mean_reversion_entry": true,
}

// ── Cross-layer dependency rules ────────────────────────────────────
// Each signal family declares which evidence families it requires.
// Each decision family declares which signal families it requires.
// Each strategy family declares which decision families it requires.

var signalDependsOnEvidence = map[string][]string{
	"rsi": {"candle"},
}

var decisionDependsOnSignal = map[string][]string{
	"rsi_oversold": {"rsi"},
}

var strategyDependsOnDecision = map[string][]string{
	"mean_reversion_entry": {"rsi_oversold"},
}

// PipelineConfig holds optional processing parameters used by derive and store.
type PipelineConfig struct {
	Timeframes       []int    `json:"timeframes"`
	Families         []string `json:"families"`
	SignalFamilies   []string `json:"signal_families"`
	DecisionFamilies []string `json:"decision_families"`
	StrategyFamilies []string `json:"strategy_families"`
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

// EnabledFamilies returns the configured families list, or nil if all are enabled.
func (p PipelineConfig) EnabledFamilies() []string {
	if len(p.Families) == 0 {
		return nil
	}
	result := make([]string, len(p.Families))
	copy(result, p.Families)
	return result
}

// ValidatePipeline checks that pipeline family names are known and that
// cross-layer dependency rules are satisfied. It returns nil when valid.
func (p PipelineConfig) ValidatePipeline() *problem.Problem {
	var issues []problem.ValidationIssue

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

	if len(issues) == 0 {
		return nil
	}
	return validationProblem("pipeline config has invalid family dependencies", issues...)
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
	issues = append(issues, extractIssues(c.Pipeline.ValidatePipeline())...)

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
