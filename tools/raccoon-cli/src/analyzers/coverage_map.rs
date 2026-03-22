use crate::command_refs;
use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};
use std::collections::BTreeMap;
use std::path::Path;

/// Quality dimension — an area the CLI can validate.
#[derive(Debug, Clone)]
struct Dimension {
    name: &'static str,
    description: &'static str,
    /// CLI command that validates this dimension.
    command: &'static str,
    /// What kind of infrastructure is required.
    requires_infra: bool,
}

const DIMENSIONS: &[Dimension] = &[
    Dimension {
        name: "project-structure",
        description: "go.work, directories, compose, config files",
        command: "raccoon-cli check repo",
        requires_infra: false,
    },
    Dimension {
        name: "topology",
        description: "config/compose/source wiring consistency across binaries",
        command: command_refs::CHECK_TOPOLOGY,
        requires_infra: false,
    },
    Dimension {
        name: "contracts",
        description: "messaging contracts, envelope, codec invariants (NATS streams)",
        command: command_refs::CHECK_CONTRACTS,
        requires_infra: false,
    },
    Dimension {
        name: "runtime-bindings",
        description: "config -> NATS stream -> actor scope binding chain",
        command: command_refs::CHECK_BINDINGS,
        requires_infra: false,
    },
    Dimension {
        name: "architecture",
        description: "clean architecture layer boundaries and purity rules",
        command: command_refs::CHECK_ARCH,
        requires_infra: false,
    },
    Dimension {
        name: "drift",
        description: "cross-layer declaration/config/source/docs alignment",
        command: command_refs::CHECK_DRIFT,
        requires_infra: false,
    },
    Dimension {
        name: "smoke-e2e",
        description: "live E2E pipeline proof: ingest -> derive -> store -> query",
        command: command_refs::MAKE_SMOKE,
        requires_infra: true,
    },
];

/// Sensitive area — a part of the codebase that demands specific validation coverage.
#[derive(Debug, Clone)]
struct SensitiveArea {
    name: &'static str,
    description: &'static str,
    /// Glob patterns for files in this area.
    patterns: &'static [&'static str],
    /// Which dimensions must cover this area.
    required_dimensions: &'static [&'static str],
}

const SENSITIVE_AREAS: &[SensitiveArea] = &[
    SensitiveArea {
        name: "config-files",
        description: "deploy/configs/*.jsonc — service configuration",
        patterns: &["deploy/configs/"],
        required_dimensions: &["project-structure", "topology", "drift"],
    },
    SensitiveArea {
        name: "compose",
        description: "docker-compose.yaml — service orchestration",
        patterns: &["deploy/compose/"],
        required_dimensions: &["project-structure", "topology", "drift"],
    },
    SensitiveArea {
        name: "nats-adapters",
        description: "NATS/JetStream adapter layer — stream transport wiring",
        patterns: &["internal/adapters/nats/"],
        required_dimensions: &["contracts", "runtime-bindings", "architecture"],
    },
    SensitiveArea {
        name: "exchange-adapters",
        description: "exchange adapter layer — ingest-specific WebSocket connections",
        patterns: &["internal/adapters/exchanges/"],
        required_dimensions: &["topology", "runtime-bindings", "architecture"],
    },
    SensitiveArea {
        name: "domain-configctl",
        description: "configctl domain — configuration lifecycle rules",
        patterns: &["internal/domain/configctl/"],
        required_dimensions: &["architecture", "contracts"],
    },
    SensitiveArea {
        name: "domain-observation",
        description: "observation domain — raw market data (trades, ticks)",
        patterns: &["internal/domain/observation/"],
        required_dimensions: &["architecture", "contracts"],
    },
    SensitiveArea {
        name: "domain-evidence",
        description: "evidence domain — derived market data (candles, samples)",
        patterns: &["internal/domain/evidence/"],
        required_dimensions: &["architecture", "contracts"],
    },
    SensitiveArea {
        name: "domain-signal",
        description: "signal domain — derived trading signals (RSI, MACD)",
        patterns: &["internal/domain/signal/"],
        required_dimensions: &["architecture", "contracts", "drift"],
    },
    SensitiveArea {
        name: "domain-decision",
        description: "decision domain — derived trading decisions (RSI oversold)",
        patterns: &["internal/domain/decision/"],
        required_dimensions: &["architecture", "contracts", "drift"],
    },
    SensitiveArea {
        name: "domain-strategy",
        description: "strategy domain — resolved trading strategies (mean reversion entry)",
        patterns: &["internal/domain/strategy/"],
        required_dimensions: &["architecture", "contracts", "drift"],
    },
    SensitiveArea {
        name: "domain-risk",
        description: "risk domain — risk assessments (position exposure) — governed from S63, implementation in S64",
        patterns: &["internal/domain/risk/"],
        required_dimensions: &["architecture", "contracts", "drift"],
    },
    SensitiveArea {
        name: "application",
        description: "application layer — use cases, ports, and clients",
        patterns: &["internal/application/"],
        required_dimensions: &["architecture", "contracts"],
    },
    SensitiveArea {
        name: "http-interface",
        description: "HTTP interface layer — gateway API endpoints",
        patterns: &["internal/interfaces/http/"],
        required_dimensions: &["architecture"],
    },
    SensitiveArea {
        name: "actors-gateway",
        description: "gateway actor scope — HTTP server and query routing",
        patterns: &["internal/actors/scopes/gateway/"],
        required_dimensions: &["architecture", "runtime-bindings"],
    },
    SensitiveArea {
        name: "actors-ingest",
        description: "ingest actor scope — exchange WebSocket to OBSERVATION_EVENTS",
        patterns: &["internal/actors/scopes/ingest/"],
        required_dimensions: &["architecture", "runtime-bindings", "smoke-e2e"],
    },
    SensitiveArea {
        name: "actors-derive",
        description: "derive actor scope — OBSERVATION_EVENTS to EVIDENCE_EVENTS, SIGNAL_EVENTS, DECISION_EVENTS, STRATEGY_EVENTS, and RISK_EVENTS (S64)",
        patterns: &["internal/actors/scopes/derive/"],
        required_dimensions: &["architecture", "runtime-bindings", "smoke-e2e"],
    },
    SensitiveArea {
        name: "actors-store",
        description: "store actor scope — EVIDENCE_EVENTS, SIGNAL_EVENTS, DECISION_EVENTS, STRATEGY_EVENTS, and RISK_EVENTS (S64) to read-model projections",
        patterns: &["internal/actors/scopes/store/"],
        required_dimensions: &["architecture", "runtime-bindings", "smoke-e2e"],
    },
    SensitiveArea {
        name: "config-lifecycle",
        description: "configctl scope — config draft/validate/compile/activate",
        patterns: &[
            "internal/actors/scopes/configctl/",
            "internal/application/configctl/",
        ],
        required_dimensions: &["contracts", "drift", "smoke-e2e"],
    },
];

/// Analyze coverage: which dimensions exist, which sensitive areas have full coverage.
pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("coverage-map");

    // Check 1: Dimension inventory
    let mut inventory_findings = Vec::new();
    let static_count = DIMENSIONS.iter().filter(|d| !d.requires_infra).count();
    let runtime_count = DIMENSIONS.iter().filter(|d| d.requires_infra).count();
    inventory_findings.push(Finding::info(
        "dimension-inventory",
        format!(
            "{} quality dimensions: {} static (no infra), {} runtime (requires cluster)",
            DIMENSIONS.len(),
            static_count,
            runtime_count
        ),
    ));
    for dim in DIMENSIONS {
        let infra_tag = if dim.requires_infra {
            " [requires infra]"
        } else {
            ""
        };
        inventory_findings.push(Finding::info(
            "dimension-inventory",
            format!(
                "  {}: {} — `{}`{}",
                dim.name, dim.description, dim.command, infra_tag
            ),
        ));
    }
    report.add(CheckResult::from_findings(
        "dimension-inventory",
        inventory_findings,
    ));

    // Check 2: Sensitive area coverage
    let mut coverage_ok = true;
    for area in SENSITIVE_AREAS {
        let mut area_findings = Vec::new();
        let area_exists = area.patterns.iter().any(|p| project_root.join(p).exists());

        if !area_exists {
            area_findings.push(Finding::info(
                &format!("coverage:{}", area.name),
                format!(
                    "{} — area not found in project (patterns: {})",
                    area.description,
                    area.patterns.join(", ")
                ),
            ));
            report.add(CheckResult::from_findings(
                &format!("coverage:{}", area.name),
                area_findings,
            ));
            continue;
        }

        let covered: Vec<&str> = area.required_dimensions.iter().copied().collect();
        let dim_names: Vec<&str> = DIMENSIONS.iter().map(|d| d.name).collect();
        let missing: Vec<&str> = covered
            .iter()
            .filter(|d| !dim_names.contains(d))
            .copied()
            .collect();

        if missing.is_empty() {
            area_findings.push(Finding::info(
                &format!("coverage:{}", area.name),
                format!(
                    "{} — covered by {} dimensions: {}",
                    area.description,
                    covered.len(),
                    covered.join(", ")
                ),
            ));
        } else {
            coverage_ok = false;
            area_findings.push(
                Finding::error(
                    &format!("coverage:{}", area.name),
                    format!(
                        "{} — missing coverage dimensions: {}",
                        area.description,
                        missing.join(", ")
                    ),
                )
                .with_why("sensitive areas without full quality coverage allow unsafe changes")
                .with_help("implement the missing dimension or add a scenario covering this area"),
            );
        }

        // Show which commands validate this area
        for dim_name in &covered {
            if let Some(dim) = DIMENSIONS.iter().find(|d| d.name == *dim_name) {
                area_findings.push(Finding::info(
                    &format!("coverage:{}", area.name),
                    format!("  validate with: `{}`", dim.command),
                ));
            }
        }

        report.add(CheckResult::from_findings(
            &format!("coverage:{}", area.name),
            area_findings,
        ));
    }

    // Check 3: Go test coverage scan
    let go_test_areas = scan_go_tests(project_root);
    let mut go_findings = Vec::new();
    if go_test_areas.is_empty() {
        go_findings.push(
            Finding::warning("go-test-coverage", "no Go test files found")
                .with_why("Go unit tests are the first line of defense for business logic")
                .with_help("add _test.go files alongside your Go source"),
        );
    } else {
        go_findings.push(Finding::info(
            "go-test-coverage",
            format!("{} Go packages with tests detected", go_test_areas.len()),
        ));
        for (pkg, count) in &go_test_areas {
            go_findings.push(Finding::info(
                "go-test-coverage",
                format!("  {} — {} test file(s)", pkg, count),
            ));
        }
    }
    report.add(CheckResult::from_findings("go-test-coverage", go_findings));

    // Check 4: Coverage summary
    let total_areas = SENSITIVE_AREAS.len();
    let existing_areas = SENSITIVE_AREAS
        .iter()
        .filter(|a| a.patterns.iter().any(|p| project_root.join(p).exists()))
        .count();
    let mut summary_findings = Vec::new();
    summary_findings.push(Finding::info(
        "coverage-summary",
        format!(
            "{}/{} sensitive areas present in project, {} quality dimensions available",
            existing_areas,
            total_areas,
            DIMENSIONS.len()
        ),
    ));
    if coverage_ok {
        summary_findings.push(Finding::info(
            "coverage-summary",
            "all present sensitive areas have full dimension coverage",
        ));
    }
    report.add(CheckResult::from_findings(
        "coverage-summary",
        summary_findings,
    ));

    Ok(report)
}

/// Scan project for Go test files and return package -> test count mapping.
fn scan_go_tests(project_root: &Path) -> BTreeMap<String, usize> {
    let mut results = BTreeMap::new();
    let internal = project_root.join("internal");
    if !internal.is_dir() {
        return results;
    }
    scan_go_tests_recursive(&internal, project_root, &mut results);
    results
}

fn scan_go_tests_recursive(dir: &Path, project_root: &Path, results: &mut BTreeMap<String, usize>) {
    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return,
    };
    let mut test_count = 0usize;
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            scan_go_tests_recursive(&path, project_root, results);
        } else if let Some(name) = path.file_name().and_then(|n| n.to_str()) {
            if name.ends_with("_test.go") {
                test_count += 1;
            }
        }
    }
    if test_count > 0 {
        let rel = dir.strip_prefix(project_root).unwrap_or(dir);
        results.insert(rel.display().to_string(), test_count);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dimensions_are_non_empty() {
        assert!(!DIMENSIONS.is_empty());
    }

    #[test]
    fn sensitive_areas_are_non_empty() {
        assert!(!SENSITIVE_AREAS.is_empty());
    }

    #[test]
    fn all_required_dimensions_exist() {
        let dim_names: Vec<&str> = DIMENSIONS.iter().map(|d| d.name).collect();
        for area in SENSITIVE_AREAS {
            for required in area.required_dimensions {
                assert!(
                    dim_names.contains(required),
                    "area '{}' requires dimension '{}' which doesn't exist",
                    area.name,
                    required
                );
            }
        }
    }

    #[test]
    fn dimension_names_are_unique() {
        let names: Vec<&str> = DIMENSIONS.iter().map(|d| d.name).collect();
        for (i, name) in names.iter().enumerate() {
            assert!(
                !names[i + 1..].contains(name),
                "duplicate dimension name: {name}"
            );
        }
    }

    #[test]
    fn sensitive_area_names_are_unique() {
        let names: Vec<&str> = SENSITIVE_AREAS.iter().map(|a| a.name).collect();
        for (i, name) in names.iter().enumerate() {
            assert!(
                !names[i + 1..].contains(name),
                "duplicate area name: {name}"
            );
        }
    }

    #[test]
    fn analyze_on_nonexistent_project() {
        let report = analyze(Path::new("/nonexistent")).unwrap();
        // Should still produce a valid report (areas won't exist)
        assert!(report.passed());
        assert!(!report.checks.is_empty());
    }

    #[test]
    fn scan_go_tests_on_nonexistent() {
        let results = scan_go_tests(Path::new("/nonexistent"));
        assert!(results.is_empty());
    }

    #[test]
    fn all_dimensions_have_commands() {
        for dim in DIMENSIONS {
            assert!(
                !dim.command.is_empty(),
                "dimension '{}' has no command",
                dim.name
            );
            assert!(
                dim.command.starts_with("raccoon-cli ") || dim.command.starts_with("make "),
                "dimension '{}' command should reference a canonical support entrypoint",
                dim.name
            );
        }
    }

    #[test]
    fn all_sensitive_areas_have_patterns() {
        for area in SENSITIVE_AREAS {
            assert!(
                !area.patterns.is_empty(),
                "area '{}' has no patterns",
                area.name
            );
        }
    }

    #[test]
    fn all_sensitive_areas_have_required_dimensions() {
        for area in SENSITIVE_AREAS {
            assert!(
                !area.required_dimensions.is_empty(),
                "area '{}' has no required dimensions",
                area.name
            );
        }
    }

    #[test]
    fn no_kafka_references() {
        // market-foundry has no kafka — ensure coverage map is clean
        for dim in DIMENSIONS {
            assert!(
                !dim.description.to_lowercase().contains("kafka"),
                "dimension '{}' still references kafka",
                dim.name
            );
        }
        for area in SENSITIVE_AREAS {
            assert!(
                !area.name.contains("kafka"),
                "sensitive area '{}' still references kafka",
                area.name
            );
        }
    }
}
