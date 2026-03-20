package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// FamilySpec is the parsed representation of a codegen/families/*.yaml file.
// Field names and structure match the S193 frozen specification exactly.
type FamilySpec struct {
	Family FamilyMeta `yaml:"family"`
	NATS   NATSSpec   `yaml:"nats"`
	Writer WriterSpec `yaml:"writer"`
	Domain DomainSpec `yaml:"domain"`
}

type FamilyMeta struct {
	Name  string `yaml:"name"`
	Layer string `yaml:"layer"`
	Tier  int    `yaml:"tier"`
}

type NATSSpec struct {
	Subject   string `yaml:"subject"`
	EventType string `yaml:"event_type"`
	Stream    string `yaml:"stream"`
	Durable   string `yaml:"durable"`
}

type WriterSpec struct {
	Table             string `yaml:"table"`
	Columns           string `yaml:"columns"`
	Mapper            string `yaml:"mapper"`
	PipelineFamilyKey string `yaml:"pipeline_family_key"`
	ConfigArray       string `yaml:"config_array"`
}

type DomainSpec struct {
	EventPackage string `yaml:"event_package"`
	EventType    string `yaml:"event_type"`
}

// DerivedFields holds all naming conventions computed from spec fields.
// These are deterministic — same spec always produces same derived fields.
type DerivedFields struct {
	ConsumerSpecFunc string // e.g. WriterRSISignalConsumer
	ConsumerName     string // e.g. writer-signal-rsi-consumer
	InserterName     string // e.g. writer-signal-rsi-inserter
	IsEnabledMethod  string // e.g. IsSignalFamilyEnabled
	RegistryField    string // e.g. signal
	NewConsumerFunc  string // e.g. NewConsumer (or NewCandleConsumer for evidence)
	StarterFunc      string // e.g. NewSignalStarter (or NewCandleStarter for evidence)
	PascalFamily     string // e.g. RSI
	PascalLayer      string // e.g. Signal
	InsertSQL        string // e.g. INSERT INTO signals (col1, col2, ...)
	HyphenFamily     string // e.g. rsi (or paper-order)
	PackageAlias     string // e.g. natssignal, natsevidence
}

// knownAbbreviations maps lowercase tokens to their Go-idiomatic uppercase forms.
var knownAbbreviations = map[string]string{
	"rsi": "RSI",
	"ema": "EMA",
	"id":  "ID",
	"url": "URL",
	"api": "API",
}

// LoadAllSpecs reads all family specs from the given directory.
func LoadAllSpecs(familiesDir string) ([]*FamilySpec, error) {
	entries, err := os.ReadDir(familiesDir)
	if err != nil {
		return nil, fmt.Errorf("read families dir: %w", err)
	}
	var specs []*FamilySpec
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		spec, err := LoadSpec(familiesDir + "/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

// ValidateCrossSpec checks cross-spec invariants across all families.
// No two specs may share the same durable consumer name, the same
// NATS subject, or the same (family.name).
func ValidateCrossSpec(specs []*FamilySpec) error {
	durables := make(map[string]string)  // durable → family
	subjects := make(map[string]string)  // subject → family
	names := make(map[string]string)     // family.name → spec file
	var errs []string

	for _, spec := range specs {
		name := spec.Family.Name

		// Unique family name.
		if prev, exists := names[name]; exists {
			errs = append(errs, fmt.Sprintf("duplicate family.name %q (also in %s)", name, prev))
		}
		names[name] = name

		// Unique durable consumer.
		if prev, exists := durables[spec.NATS.Durable]; exists {
			errs = append(errs, fmt.Sprintf("duplicate nats.durable %q: families %q and %q", spec.NATS.Durable, prev, name))
		}
		durables[spec.NATS.Durable] = name

		// Unique NATS subject.
		if prev, exists := subjects[spec.NATS.Subject]; exists {
			errs = append(errs, fmt.Sprintf("duplicate nats.subject %q: families %q and %q", spec.NATS.Subject, prev, name))
		}
		subjects[spec.NATS.Subject] = name
	}

	if len(errs) > 0 {
		return fmt.Errorf("cross-spec validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// LoadSpec reads and parses a family spec YAML file.
func LoadSpec(path string) (*FamilySpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	var spec FamilySpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	return &spec, nil
}

// Validate checks that all required fields are present and structurally valid.
func (s *FamilySpec) Validate() error {
	var errs []string
	if s.Family.Name == "" {
		errs = append(errs, "family.name is required")
	}
	if s.Family.Layer == "" {
		errs = append(errs, "family.layer is required")
	}
	validLayers := map[string]bool{
		"evidence": true, "signal": true, "decision": true,
		"strategy": true, "risk": true, "execution": true,
	}
	if !validLayers[s.Family.Layer] {
		errs = append(errs, fmt.Sprintf("family.layer %q is not a valid layer", s.Family.Layer))
	}
	if s.Family.Tier != 1 && s.Family.Tier != 2 {
		errs = append(errs, fmt.Sprintf("family.tier must be 1 or 2, got %d", s.Family.Tier))
	}
	if s.NATS.Subject == "" {
		errs = append(errs, "nats.subject is required")
	}
	if s.NATS.EventType == "" {
		errs = append(errs, "nats.event_type is required")
	}
	if s.NATS.Stream == "" {
		errs = append(errs, "nats.stream is required")
	}
	if s.NATS.Durable == "" {
		errs = append(errs, "nats.durable is required")
	}
	if s.Writer.Table == "" {
		errs = append(errs, "writer.table is required")
	}
	if s.Writer.Mapper == "" {
		errs = append(errs, "writer.mapper is required")
	}
	if s.Writer.PipelineFamilyKey == "" {
		errs = append(errs, "writer.pipeline_family_key is required")
	}
	if s.Writer.ConfigArray == "" {
		errs = append(errs, "writer.config_array is required")
	}
	if s.Domain.EventPackage == "" {
		errs = append(errs, "domain.event_package is required")
	}
	if s.Domain.EventType == "" {
		errs = append(errs, "domain.event_type is required")
	}
	if len(errs) > 0 {
		return fmt.Errorf("spec validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// Derived computes all naming conventions from the spec fields.
func (s *FamilySpec) Derived() DerivedFields {
	pascalFamily := toPascalCase(s.Family.Name)
	pascalLayer := toPascalCase(s.Family.Layer)
	hyphenFamily := strings.ReplaceAll(s.Family.Name, "_", "-")

	// Consumer/inserter names: writer-{layer}-{family-hyphenated}-{role}
	// Exception: evidence layer omits layer prefix.
	var consumerName, inserterName string
	if s.Family.Layer == "evidence" {
		consumerName = "writer-" + hyphenFamily + "-consumer"
		inserterName = "writer-" + hyphenFamily + "-inserter"
	} else {
		consumerName = "writer-" + s.Family.Layer + "-" + hyphenFamily + "-consumer"
		inserterName = "writer-" + s.Family.Layer + "-" + hyphenFamily + "-inserter"
	}

	// Consumer spec function: Writer{PascalFamily}{PascalLayer}Consumer
	// Exception: evidence layer omits layer from function name.
	var consumerSpecFunc string
	if s.Family.Layer == "evidence" {
		consumerSpecFunc = "Writer" + pascalFamily + "Consumer"
	} else {
		consumerSpecFunc = "Writer" + pascalFamily + pascalLayer + "Consumer"
	}

	// IsEnabled method: Is{PascalLayer}FamilyEnabled
	// Exception: evidence layer uses IsFamilyEnabled.
	var isEnabledMethod string
	if s.Family.Layer == "evidence" {
		isEnabledMethod = "IsFamilyEnabled"
	} else {
		isEnabledMethod = "Is" + pascalLayer + "FamilyEnabled"
	}

	// NewConsumerFunc: evidence layer uses New{PascalFamily}Consumer (e.g. NewCandleConsumer),
	// all other layers use NewConsumer (since the package already encodes the layer).
	var newConsumerFunc string
	if s.Family.Layer == "evidence" {
		newConsumerFunc = "New" + pascalFamily + "Consumer"
	} else {
		newConsumerFunc = "NewConsumer"
	}

	// StarterFunc: evidence layer uses New{PascalFamily}Starter (e.g. NewCandleStarter),
	// all other layers use New{PascalLayer}Starter (e.g. NewSignalStarter).
	var starterFunc string
	if s.Family.Layer == "evidence" {
		starterFunc = "New" + pascalFamily + "Starter"
	} else {
		starterFunc = "New" + pascalLayer + "Starter"
	}

	// InsertSQL includes explicit column list when writer.columns is specified.
	insertSQL := "INSERT INTO " + s.Writer.Table
	if s.Writer.Columns != "" {
		insertSQL += " (" + s.Writer.Columns + ")"
	}

	return DerivedFields{
		ConsumerSpecFunc: consumerSpecFunc,
		ConsumerName:     consumerName,
		InserterName:     inserterName,
		IsEnabledMethod:  isEnabledMethod,
		RegistryField:    s.Family.Layer,
		NewConsumerFunc:  newConsumerFunc,
		StarterFunc:      starterFunc,
		PascalFamily:     pascalFamily,
		PascalLayer:      pascalLayer,
		InsertSQL:        insertSQL,
		HyphenFamily:     hyphenFamily,
		PackageAlias:     "nats" + s.Family.Layer,
	}
}

// toPascalCase converts a snake_case string to PascalCase,
// respecting known Go abbreviations (RSI, EMA, ID, etc.).
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, part := range parts {
		if upper, ok := knownAbbreviations[strings.ToLower(part)]; ok {
			b.WriteString(upper)
		} else {
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}
