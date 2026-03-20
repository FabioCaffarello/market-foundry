package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Resolve base directory: codegen tool expects to run from the codegen/ directory
	// or with CODEGEN_ROOT set to the codegen/ directory path.
	baseDir := os.Getenv("CODEGEN_ROOT")
	if baseDir == "" {
		exe, err := os.Executable()
		if err == nil {
			baseDir = filepath.Dir(exe)
		}
		if baseDir == "" {
			baseDir = "."
		}
	}

	switch os.Args[1] {
	case "validate":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: codegen validate <spec.yaml>")
			os.Exit(1)
		}
		cmdValidate(os.Args[2])

	case "generate":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: codegen generate <spec.yaml> <artifact>")
			os.Exit(1)
		}
		cmdGenerate(os.Args[2], os.Args[3], baseDir)

	case "compare":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "usage: codegen compare <spec.yaml> <artifact>")
			os.Exit(1)
		}
		cmdCompare(os.Args[2], os.Args[3], baseDir)

	case "check-all":
		cmdCheckAll(baseDir)

	case "validate-all":
		cmdValidateAll(baseDir)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `usage: codegen <command> [args]

commands:
  validate <spec.yaml>              validate a family spec
  generate <spec.yaml> <artifact>   render an artifact from spec
  compare  <spec.yaml> <artifact>   compare generated vs golden snapshot
  check-all                         compare all families × all artifacts
  validate-all                      validate all specs (per-spec + cross-spec uniqueness)`)
}

func cmdValidate(specPath string) {
	spec, err := LoadSpec(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("VALID  %s (family=%s, layer=%s, tier=%d)\n",
		specPath, spec.Family.Name, spec.Family.Layer, spec.Family.Tier)
}

func cmdGenerate(specPath, artifact, baseDir string) {
	spec, err := LoadSpec(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "INVALID spec: %v\n", err)
		os.Exit(1)
	}

	templatesDir := filepath.Join(baseDir, "templates")
	output, err := RenderArtifact(spec, artifact, templatesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(output)
}

func cmdCompare(specPath, artifact, baseDir string) {
	spec, err := LoadSpec(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	templatesDir := filepath.Join(baseDir, "templates")
	generated, err := RenderArtifact(spec, artifact, templatesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	goldenPath := filepath.Join(baseDir, "golden-snapshots", spec.Family.Name, artifact+".go.golden")
	result, err := CompareWithGolden(generated, goldenPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	result.Family = spec.Family.Name
	result.Artifact = artifact

	fmt.Print(FormatCompareResult(result))
	if !result.Pass {
		os.Exit(1)
	}
}

func cmdValidateAll(baseDir string) {
	familiesDir := filepath.Join(baseDir, "families")
	specs, err := LoadAllSpecs(familiesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Per-spec validation.
	var failed int
	for _, spec := range specs {
		if err := spec.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "INVALID  %s: %v\n", spec.Family.Name, err)
			failed++
		} else {
			fmt.Printf("VALID    %s (layer=%s, tier=%d)\n", spec.Family.Name, spec.Family.Layer, spec.Family.Tier)
		}
	}

	// Cross-spec validation.
	if err := ValidateCrossSpec(specs); err != nil {
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		failed++
	} else {
		fmt.Printf("\nCross-spec uniqueness: OK (%d families, no collisions)\n", len(specs))
	}

	if failed > 0 {
		os.Exit(1)
	}
}

func cmdCheckAll(baseDir string) {
	familiesDir := filepath.Join(baseDir, "families")
	entries, err := os.ReadDir(familiesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR reading families dir: %v\n", err)
		os.Exit(1)
	}

	artifacts := SupportedArtifacts()
	var failed []string
	var passed int

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		specPath := filepath.Join(familiesDir, entry.Name())
		spec, err := LoadSpec(specPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR loading %s: %v\n", specPath, err)
			failed = append(failed, specPath)
			continue
		}

		templatesDir := filepath.Join(baseDir, "templates")
		for _, artifact := range artifacts {
			generated, err := RenderArtifact(spec, artifact, templatesDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR generating %s/%s: %v\n", spec.Family.Name, artifact, err)
				failed = append(failed, spec.Family.Name+"/"+artifact)
				continue
			}

			goldenPath := filepath.Join(baseDir, "golden-snapshots", spec.Family.Name, artifact+".go.golden")
			result, err := CompareWithGolden(generated, goldenPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR comparing %s/%s: %v\n", spec.Family.Name, artifact, err)
				failed = append(failed, spec.Family.Name+"/"+artifact)
				continue
			}
			result.Family = spec.Family.Name
			result.Artifact = artifact

			fmt.Println(FormatCompareResult(result))
			if result.Pass {
				passed++
			} else {
				failed = append(failed, spec.Family.Name+"/"+artifact)
			}
		}
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, len(failed))
	if len(failed) > 0 {
		os.Exit(1)
	}
}
