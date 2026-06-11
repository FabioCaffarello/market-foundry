//! check-subjects — H-6.e canonical subject-token invariant.
//!
//! Statically enforces the ADR-0009 erratum (2026-06-10, Onda H-6.e):
//! the `{symbol}` token of every published NATS subject is derived
//! exclusively via `CanonicalInstrument.SubjectToken()` — never via
//! the transitory `VenueSymbol()` helper and never hand-formatted.
//!
//! Declarative algorithm: reads `tools/raccoon-cli/policies/subjects.toml`
//! and scans production publisher files (`file_suffix`, excluding
//! `_test.go`) under `scan_root`. A subject-composition block starts
//! at a line containing `subject_marker` and ends when the call's
//! parentheses balance back to zero. Inside a block, any occurrence
//! of `forbidden_call` is an error finding.
//!
//! Sections accreted per sub-onda, all block-scoped rather than
//! whole-file pattern matches: `[subjects]` (H-6.e, subject
//! composition), `[keys]` (H-6.e.2, KV partition keys), `[dedup]`
//! (H-6.f.1, JetStream dedup keys — domain composers + publisher
//! inline assignments). Log labels remain the only message-adjacent
//! surface where `VenueSymbol()` is still legal, until its deletion
//! in H-6.f.2 (pós-TTL).
//!
//! Detection model mirrors check-metrics / check-instruments:
//! line-based scan, not AST — robust enough for the gofmt'd,
//! uniform publisher shape, and cheap to run in the gate.

use std::fs;
use std::path::{Path, PathBuf};

use serde::Deserialize;

use crate::error::Result;
use crate::models::{CheckResult, Finding, Report};

const SUBJECTS_POLICY_PATH: &str = "tools/raccoon-cli/policies/subjects.toml";

#[derive(Debug, Deserialize)]
struct SubjectsPolicy {
    subjects: SubjectsSection,
    /// H-6.e.2: KV partition-key invariant (optional until the
    /// section ships with the e.2 policy).
    keys: Option<KeysSection>,
    /// H-6.f.1: JetStream dedup-key invariant (optional until the
    /// section ships with the f.1 policy).
    dedup: Option<DedupSection>,
}

#[derive(Debug, Deserialize)]
struct DedupSection {
    /// Directory walked for domain composer files (e.g. "internal/domain").
    domain_scan_root: String,
    /// Line marker that opens a DeduplicationKey implementation.
    func_marker: String,
    /// Directory walked for publisher files (e.g. "internal/adapters/nats").
    publisher_scan_root: String,
    /// Production file suffix that builds inline dedup keys.
    publisher_file_suffix: String,
    /// Line marker that opens an inline dedup-key assignment.
    inline_marker: String,
    /// Call that must NOT appear inside either scanned surface.
    forbidden_call: String,
    /// Call that MUST appear in composers listed in `required_receivers`.
    required_call: String,
    /// Receiver type names whose DeduplicationKey embeds an instrument
    /// token; composers with other receivers (session/trade identity)
    /// are scanned for the forbidden call only.
    required_receivers: Vec<String>,
}

#[derive(Debug, Deserialize)]
struct KeysSection {
    /// Directory walked for domain files (e.g. "internal/domain").
    scan_root: String,
    /// Line marker that opens a PartitionKey implementation.
    func_marker: String,
    /// Call that must NOT appear inside the function body.
    forbidden_call: String,
    /// Call that MUST appear inside the function body.
    required_call: String,
}

#[derive(Debug, Deserialize)]
struct SubjectsSection {
    /// Directory walked for publisher files (e.g. "internal/adapters/nats").
    scan_root: String,
    /// Production file suffix that composes subjects (e.g. "publisher.go").
    file_suffix: String,
    /// Line marker that opens a subject-composition block.
    subject_marker: String,
    /// Call that must NOT appear inside a subject-composition block.
    forbidden_call: String,
}

pub fn analyze(project_root: &Path) -> Result<Report> {
    let mut report = Report::new("check-subjects");

    let policy_path = project_root.join(SUBJECTS_POLICY_PATH);
    let policy: SubjectsPolicy = match fs::read_to_string(&policy_path) {
        Ok(raw) => match toml::from_str(&raw) {
            Ok(p) => {
                report.add(CheckResult::pass("policy-present"));
                p
            }
            Err(e) => {
                report.add(CheckResult::from_findings(
                    "policy-present",
                    vec![Finding::error(
                        "policy-parse",
                        format!("{}: {e}", policy_path.display()),
                    )],
                ));
                return Ok(report);
            }
        },
        Err(e) => {
            report.add(CheckResult::from_findings(
                "policy-present",
                vec![Finding::error(
                    "policy-missing",
                    format!("{}: {e}", policy_path.display()),
                )],
            ));
            return Ok(report);
        }
    };

    let scan_root = project_root.join(&policy.subjects.scan_root);
    let subjects_root_ok = scan_root.is_dir();
    if subjects_root_ok {
        report.add(CheckResult::pass("scan-root"));
    } else {
        report.add(CheckResult::skip(
            "scan-root",
            format!("{} not found", scan_root.display()),
        ));
    }

    let mut files: Vec<PathBuf> = Vec::new();
    if subjects_root_ok {
        collect_publisher_files(&scan_root, &policy.subjects.file_suffix, &mut files)?;
        files.sort();
    }

    let mut findings: Vec<Finding> = Vec::new();
    let mut blocks_scanned: usize = 0;

    for file in &files {
        let content = fs::read_to_string(file)?;
        let rel = file
            .strip_prefix(project_root)
            .unwrap_or(file)
            .display()
            .to_string();

        let mut depth: i64 = 0;
        let mut in_block = false;

        for (idx, line) in content.lines().enumerate() {
            let lineno = idx + 1;
            let trimmed = line.trim_start();
            if trimmed.starts_with("//") {
                continue;
            }

            if !in_block && line.contains(&policy.subjects.subject_marker) {
                in_block = true;
                blocks_scanned += 1;
                depth = 0;
            }

            if in_block {
                if line.contains(&policy.subjects.forbidden_call) {
                    findings.push(Finding::error(
                        "subject-composition",
                        format!(
                            "{rel}:{lineno}: subject composed via {} — the {{symbol}} \
                             token must come from Instrument.SubjectToken() \
                             (ADR-0009 erratum 2026-06-10, Onda H-6.e)",
                            policy.subjects.forbidden_call
                        ),
                    ));
                }
                depth += paren_delta(line);
                if depth <= 0 {
                    in_block = false;
                }
            }
        }
    }

    if findings.is_empty() && subjects_root_ok {
        report.add(CheckResult::pass("subject-composition"));
        report.add(CheckResult::pass(format!(
            "subject-blocks-scanned ({} blocks in {} publisher files)",
            blocks_scanned,
            files.len()
        )));
    } else {
        report.add(CheckResult::from_findings("subject-composition", findings));
    }

    if let Some(keys) = &policy.keys {
        check_partition_keys(project_root, keys, &mut report)?;
    }

    if let Some(dedup) = &policy.dedup {
        check_dedup_keys(project_root, dedup, &mut report)?;
    }

    Ok(report)
}

/// H-6.f.1 invariant (Decisão #4): JetStream dedup keys carry the
/// canonical `SubjectToken()`, never the transitory `VenueSymbol()`.
/// Two surfaces, both block-scoped:
///
/// 1. Domain `DeduplicationKey()` composers — brace-scoped from the
///    function marker, mirroring the `[keys]` scan. `forbidden_call`
///    flags in every body; `required_call` is enforced only for
///    receivers in `required_receivers` (composers without an
///    instrument token, e.g. SessionLifecycleEvent, compose from
///    other identity and legitimately omit it).
/// 2. Publisher inline `dedupKey :=` assignments — statement-scoped:
///    the assignment continues while parentheses stay open or a line
///    ends with a `+`/`,` continuation. `forbidden_call` flags; no
///    required half (fill/delegating assignments carry no token).
fn check_dedup_keys(
    project_root: &Path,
    dedup: &DedupSection,
    report: &mut Report,
) -> Result<()> {
    let mut findings: Vec<Finding> = Vec::new();

    // ── Surface 1: domain composers ────────────────────────────────
    let domain_root = project_root.join(&dedup.domain_scan_root);
    let mut composers_scanned: usize = 0;
    if !domain_root.is_dir() {
        report.add(CheckResult::skip(
            "dedup-keys-domain-scan-root",
            format!("{} not found", domain_root.display()),
        ));
    } else {
        let mut files: Vec<PathBuf> = Vec::new();
        collect_publisher_files(&domain_root, ".go", &mut files)?;
        files.sort();

        for file in &files {
            let content = fs::read_to_string(file)?;
            let rel = file
                .strip_prefix(project_root)
                .unwrap_or(file)
                .display()
                .to_string();

            let mut depth: i64 = 0;
            let mut in_func = false;
            let mut has_required = false;
            let mut requires_token = false;
            let mut func_line = 0usize;

            for (idx, line) in content.lines().enumerate() {
                let lineno = idx + 1;
                let trimmed = line.trim_start();
                if trimmed.starts_with("//") {
                    continue;
                }

                if !in_func && line.contains(&dedup.func_marker) {
                    in_func = true;
                    composers_scanned += 1;
                    has_required = false;
                    requires_token = receiver_type(line)
                        .map(|r| dedup.required_receivers.iter().any(|t| t == &r))
                        .unwrap_or(false);
                    func_line = lineno;
                    depth = 1;
                    continue;
                }

                if in_func {
                    if line.contains(&dedup.forbidden_call) {
                        findings.push(Finding::error(
                            "dedup-key-composition",
                            format!(
                                "{rel}:{lineno}: DeduplicationKey composed via {} — dedup \
                                 keys use Instrument.SubjectToken() since H-6.f.1 \
                                 (Decisão #4)",
                                dedup.forbidden_call
                            ),
                        ));
                    }
                    if line.contains(&dedup.required_call) {
                        has_required = true;
                    }
                    depth += paren_brace_delta(line);
                    if depth <= 0 {
                        if requires_token && !has_required {
                            findings.push(Finding::error(
                                "dedup-key-composition",
                                format!(
                                    "{rel}:{func_line}: DeduplicationKey does not call {} — \
                                     this composer embeds an instrument token and must \
                                     derive it canonically since H-6.f.1",
                                    dedup.required_call
                                ),
                            ));
                        }
                        in_func = false;
                    }
                }
            }
        }
    }

    // ── Surface 2: publisher inline assignments ────────────────────
    let publisher_root = project_root.join(&dedup.publisher_scan_root);
    let mut blocks_scanned: usize = 0;
    if !publisher_root.is_dir() {
        report.add(CheckResult::skip(
            "dedup-keys-publisher-scan-root",
            format!("{} not found", publisher_root.display()),
        ));
    } else {
        let mut files: Vec<PathBuf> = Vec::new();
        collect_publisher_files(&publisher_root, &dedup.publisher_file_suffix, &mut files)?;
        files.sort();

        for file in &files {
            let content = fs::read_to_string(file)?;
            let rel = file
                .strip_prefix(project_root)
                .unwrap_or(file)
                .display()
                .to_string();

            let mut depth: i64 = 0;
            let mut in_block = false;

            for (idx, line) in content.lines().enumerate() {
                let lineno = idx + 1;
                let trimmed = line.trim_start();
                if trimmed.starts_with("//") {
                    continue;
                }

                if !in_block && line.contains(&dedup.inline_marker) {
                    in_block = true;
                    blocks_scanned += 1;
                    depth = 0;
                }

                if in_block {
                    if line.contains(&dedup.forbidden_call) {
                        findings.push(Finding::error(
                            "dedup-key-composition",
                            format!(
                                "{rel}:{lineno}: inline dedup key composed via {} — dedup \
                                 keys use Instrument.SubjectToken() since H-6.f.1 \
                                 (Decisão #4)",
                                dedup.forbidden_call
                            ),
                        ));
                    }
                    depth += paren_delta(line);
                    let trimmed_end = line.trim_end();
                    let continues =
                        trimmed_end.ends_with('+') || trimmed_end.ends_with(',') || depth > 0;
                    if !continues {
                        in_block = false;
                    }
                }
            }
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("dedup-key-composition"));
        report.add(CheckResult::pass(format!(
            "dedup-keys-scanned ({composers_scanned} composers, {blocks_scanned} inline blocks)"
        )));
    } else {
        report.add(CheckResult::from_findings("dedup-key-composition", findings));
    }
    Ok(())
}

/// Extracts the receiver type name from a Go method declaration line,
/// e.g. `func (e ExecutionIntent) DeduplicationKey() string {` →
/// `ExecutionIntent`. Pointer receivers are unwrapped.
fn receiver_type(line: &str) -> Option<String> {
    let open = line.find("func (")? + "func (".len();
    let close = line[open..].find(')')? + open;
    let recv = line[open..close].trim();
    let ty = recv.split_whitespace().last()?;
    Some(ty.trim_start_matches('*').to_string())
}

/// H-6.e.2 invariant: every `PartitionKey()` implementation under the
/// declared scan_root composes via the canonical `SubjectToken()` and
/// never via the transitory `VenueSymbol()`. Block-scoped by brace
/// depth from the function marker, mirroring the subjects scan —
/// `VenueSymbol()` elsewhere in domain files (log labels, the helper
/// definitions themselves) is legal until H-6.f.2 and must not flag
/// here (dedup keys gained their own `[dedup]` scan in H-6.f.1).
fn check_partition_keys(
    project_root: &Path,
    keys: &KeysSection,
    report: &mut Report,
) -> Result<()> {
    let scan_root = project_root.join(&keys.scan_root);
    if !scan_root.is_dir() {
        report.add(CheckResult::skip(
            "partition-keys-scan-root",
            format!("{} not found", scan_root.display()),
        ));
        return Ok(());
    }

    let mut files: Vec<PathBuf> = Vec::new();
    collect_publisher_files(&scan_root, ".go", &mut files)?;
    files.sort();

    let mut findings: Vec<Finding> = Vec::new();
    let mut funcs_scanned: usize = 0;

    for file in &files {
        let content = fs::read_to_string(file)?;
        let rel = file
            .strip_prefix(project_root)
            .unwrap_or(file)
            .display()
            .to_string();

        let mut depth: i64 = 0;
        let mut in_func = false;
        let mut has_required = false;
        let mut func_line = 0usize;

        for (idx, line) in content.lines().enumerate() {
            let lineno = idx + 1;
            let trimmed = line.trim_start();
            if trimmed.starts_with("//") {
                continue;
            }

            if !in_func && line.contains(&keys.func_marker) {
                in_func = true;
                funcs_scanned += 1;
                has_required = false;
                func_line = lineno;
                depth = 1;
                continue;
            }

            if in_func {
                if line.contains(&keys.forbidden_call) {
                    findings.push(Finding::error(
                        "partition-key-composition",
                        format!(
                            "{rel}:{lineno}: PartitionKey composed via {} — keys use                              Instrument.SubjectToken() since H-6.e.2 (ADR-0021                              criterion #2 erratum)",
                            keys.forbidden_call
                        ),
                    ));
                }
                if line.contains(&keys.required_call) {
                    has_required = true;
                }
                depth += paren_brace_delta(line);
                if depth <= 0 {
                    if !has_required {
                        findings.push(Finding::error(
                            "partition-key-composition",
                            format!(
                                "{rel}:{func_line}: PartitionKey does not call {} —                                  keys compose {{source}}.{{subject_token}}.{{timeframe}}                                  since H-6.e.2",
                                keys.required_call
                            ),
                        ));
                    }
                    in_func = false;
                }
            }
        }
    }

    if findings.is_empty() {
        report.add(CheckResult::pass("partition-key-composition"));
        report.add(CheckResult::pass(format!(
            "partition-keys-scanned ({funcs_scanned} composers)"
        )));
    } else {
        report.add(CheckResult::from_findings(
            "partition-key-composition",
            findings,
        ));
    }
    Ok(())
}

/// Net brace depth for function-body tracking.
fn paren_brace_delta(line: &str) -> i64 {
    let mut delta: i64 = 0;
    for ch in line.chars() {
        match ch {
            '{' => delta += 1,
            '}' => delta -= 1,
            _ => {}
        }
    }
    delta
}

/// Net parenthesis depth contributed by a line. The subject block
/// opens with `fmt.Sprintf(` (+1 net on the marker line) and closes
/// on the line where the call's parens balance back to zero.
fn paren_delta(line: &str) -> i64 {
    let mut delta: i64 = 0;
    for ch in line.chars() {
        match ch {
            '(' => delta += 1,
            ')' => delta -= 1,
            _ => {}
        }
    }
    delta
}

fn collect_publisher_files(
    dir: &Path,
    suffix: &str,
    out: &mut Vec<PathBuf>,
) -> std::io::Result<()> {
    for entry in fs::read_dir(dir)? {
        let entry = entry?;
        let path = entry.path();
        if path.is_dir() {
            collect_publisher_files(&path, suffix, out)?;
            continue;
        }
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n,
            None => continue,
        };
        if !name.ends_with(suffix) || name.ends_with("_test.go") {
            continue;
        }
        out.push(path);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    fn write_policy(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("subjects.toml"),
            r#"[subjects]
scan_root = "internal/adapters/nats"
file_suffix = "publisher.go"
subject_marker = "subject := fmt.Sprintf("
forbidden_call = "VenueSymbol()"
"#,
        )
        .unwrap();
    }

    fn write_publisher(root: &Path, pkg: &str, body: &str) {
        let dir = root.join("internal/adapters/nats").join(pkg);
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("publisher.go"), body).unwrap();
    }

    fn report_passed(report: &Report) -> bool {
        report.passed
    }

    #[test]
    fn clean_publisher_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natssignal",
            "package natssignal\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s.%s.%d\",\n\t\tspec.Subject,\n\t\tevent.Signal.Source,\n\t\tevent.Signal.Instrument.SubjectToken(),\n\t\tevent.Signal.Timeframe,\n\t)\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn venue_symbol_inside_subject_block_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natssignal",
            "package natssignal\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s.%s.%d\",\n\t\tspec.Subject,\n\t\tevent.Signal.Source,\n\t\tevent.Signal.VenueSymbol(),\n\t\tevent.Signal.Timeframe,\n\t)\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn venue_symbol_outside_subject_block_is_tolerated() {
        // Dedup keys and log labels legitimately use VenueSymbol()
        // until H-6.e.2 — the analyzer must be block-scoped.
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsevidence",
            "package natsevidence\n\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s.%s\",\n\t\tspec.Subject,\n\t\tevent.Candle.Instrument.SubjectToken(),\n\t)\n\tdedupKey := event.Candle.VenueSymbol() + \":\" + suffix\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn test_files_are_skipped() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        let dir = tmp.path().join("internal/adapters/nats/natssignal");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("publisher.go"),
            "package natssignal\nfunc f() { subject := fmt.Sprintf(\"%s\", a.Instrument.SubjectToken()) }\n",
        )
        .unwrap();
        fs::write(
            dir.join("old_publisher_test.go"),
            "package natssignal\nfunc g() { subject := fmt.Sprintf(\"%s\", a.VenueSymbol()) }\n",
        )
        .unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn comment_lines_are_skipped() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsrisk",
            "package natsrisk\n\n// subject := fmt.Sprintf( historically used VenueSymbol()\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s\", a.Instrument.SubjectToken())\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn missing_policy_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn missing_scan_root_is_skip_after_policy() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        let report = analyze(tmp.path()).unwrap();
        // policy-present passes; scan-root is a skip, not a failure.
        assert!(report_passed(&report), "expected pass/skip: {report:?}");
    }

    #[test]
    fn single_line_block_closes_same_line() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy(tmp.path());
        write_publisher(
            tmp.path(),
            "natsdecision",
            "package natsdecision\nfunc f() {\n\tsubject := fmt.Sprintf(\"%s\", tok)\n\tx := event.D.VenueSymbol()\n\t_ = x\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    fn write_policy_with_keys(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("subjects.toml"),
            r#"[subjects]
scan_root = "internal/adapters/nats"
file_suffix = "publisher.go"
subject_marker = "subject := fmt.Sprintf("
forbidden_call = "VenueSymbol()"

[keys]
scan_root = "internal/domain"
func_marker = ") PartitionKey() string {"
forbidden_call = "VenueSymbol()"
required_call = "SubjectToken()"
"#,
        )
        .unwrap();
    }

    fn write_domain(root: &Path, pkg: &str, body: &str) {
        let dir = root.join("internal/domain").join(pkg);
        fs::create_dir_all(&dir).unwrap();
        fs::write(dir.join("type.go"), body).unwrap();
    }

    #[test]
    fn partition_key_with_subject_token_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_keys(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) PartitionKey() string {
	return fmt.Sprintf(\"%s.%s.%d\", s.Source, s.Instrument.SubjectToken(), s.Timeframe)
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn partition_key_with_venue_symbol_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_keys(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) PartitionKey() string {
	return fmt.Sprintf(\"%s.%s.%d\", s.Source, s.VenueSymbol(), s.Timeframe)
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn partition_key_missing_subject_token_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_keys(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) PartitionKey() string {
	return s.Source
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn venue_symbol_outside_partition_key_is_tolerated() {
        // Log labels and the transitory helper itself stay legal
        // until H-6.f.2. (Dedup keys are covered by the [dedup]
        // section — absent from this policy fixture by design.)
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_keys(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) PartitionKey() string {
	return s.Instrument.SubjectToken()
}

func (s Signal) DeduplicationKey() string {
	return s.VenueSymbol()
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    // ── [dedup] section (H-6.f.1 Decisão #4) ─────────────────────

    fn write_policy_with_dedup(root: &Path) {
        let dir = root.join("tools/raccoon-cli/policies");
        fs::create_dir_all(&dir).unwrap();
        fs::write(
            dir.join("subjects.toml"),
            r#"[subjects]
scan_root = "internal/adapters/nats"
file_suffix = "publisher.go"
subject_marker = "subject := fmt.Sprintf("
forbidden_call = "VenueSymbol()"

[dedup]
domain_scan_root = "internal/domain"
func_marker = ") DeduplicationKey() string {"
publisher_scan_root = "internal/adapters/nats"
publisher_file_suffix = "publisher.go"
inline_marker = "dedupKey :="
forbidden_call = "VenueSymbol()"
required_call = "SubjectToken()"
required_receivers = ["Signal", "ExecutionIntent"]
"#,
        )
        .unwrap();
    }

    #[test]
    fn dedup_composer_with_subject_token_passes() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) DeduplicationKey() string {
	return fmt.Sprintf(\"sig:%s:%s:%s:%d:%d\", s.Type, s.Source, s.Instrument.SubjectToken(), s.Timeframe, s.Timestamp.UnixNano())
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn dedup_composer_with_venue_symbol_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) DeduplicationKey() string {
	return fmt.Sprintf(\"sig:%s:%s\", s.VenueSymbol(), s.Type)
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn dedup_required_receiver_missing_token_is_error() {
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_domain(
            tmp.path(),
            "signal",
            "package signal

func (s Signal) DeduplicationKey() string {
	return s.Source + s.Type
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn dedup_non_required_receiver_without_token_passes() {
        // SessionLifecycleEvent / ObservationTrade shape: no
        // instrument token in the key, composed from other identity.
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_domain(
            tmp.path(),
            "execution",
            "package execution

func (e SessionLifecycleEvent) DeduplicationKey() string {
	return \"session-lifecycle:\" + e.SessionID + \":\" + string(e.Status)
}
",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }

    #[test]
    fn dedup_inline_multiline_concatenation_with_venue_symbol_is_error() {
        // natsevidence shape: assignment continues across lines via
        // trailing '+' with no parentheses to balance.
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_publisher(
            tmp.path(),
            "natsevidence",
            "package natsevidence\n\nfunc f() {\n\tdedupKey := event.Candle.Source + \":\" +\n\t\tevent.Candle.VenueSymbol() + \":\" +\n\t\tstrconv.Itoa(event.Candle.Timeframe)\n\n\tpublish(dedupKey)\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(!report_passed(&report), "expected failure: {report:?}");
    }

    #[test]
    fn dedup_inline_canonical_and_tokenless_blocks_pass() {
        // Canonical concatenation + the fill-style key (no instrument
        // token) + a VenueSymbol() log label AFTER the assignment ends
        // — only the assignment block is scanned.
        let tmp = tempfile::tempdir().unwrap();
        write_policy_with_dedup(tmp.path());
        write_publisher(
            tmp.path(),
            "natsexecution",
            "package natsexecution\n\nfunc f() {\n\tdedupKey := event.Candle.Source + \":\" +\n\t\tevent.Candle.Instrument.SubjectToken() + \":\" +\n\t\tstrconv.Itoa(event.Candle.Timeframe)\n\t_ = dedupKey\n}\n\nfunc g() {\n\tdedupKey := fmt.Sprintf(\"fill:%s:%d\", event.VenueOrderID, ts.Unix())\n\tlogLabel := event.VenueSymbol()\n\t_, _ = dedupKey, logLabel\n}\n",
        );
        let report = analyze(tmp.path()).unwrap();
        assert!(report_passed(&report), "expected pass: {report:?}");
    }
}
