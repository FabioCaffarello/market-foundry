mod change_targets;

use crate::analyzers;
use crate::cli::{
    BriefingArgs, ChangeCommands, CheckCommands, Cli, Commands, ContractUsageMapArgs,
    InspectCommands, LegacyCommands, LspEnrichArgs, QualityGateArgs, RecommendArgs,
    RenameSafetyArgs, RuntimeSmokeArgs,
};
use crate::error::{CliError, Result};
use crate::gate;
use crate::output::{self, OutputFormat};
use crate::smoke;
use std::path::Path;

pub(crate) fn run(cli: Cli) -> i32 {
    let context = AppContext::from(&cli);

    match execute(context, cli.command) {
        Ok(code) => code,
        Err(error) => {
            eprintln!("error: {error}");
            2
        }
    }
}

#[derive(Debug, Clone)]
struct AppContext {
    json: bool,
    verbose: bool,
    project_root: std::path::PathBuf,
    format: OutputFormat,
}

impl From<&Cli> for AppContext {
    fn from(cli: &Cli) -> Self {
        Self {
            json: cli.json,
            verbose: cli.verbose,
            project_root: cli.project_root.clone(),
            format: cli.output_format(),
        }
    }
}

impl From<crate::cli::GateProfile> for gate::Profile {
    fn from(profile: crate::cli::GateProfile) -> Self {
        match profile {
            crate::cli::GateProfile::Fast => gate::Profile::Fast,
            crate::cli::GateProfile::Ci => gate::Profile::Ci,
            crate::cli::GateProfile::Deep => gate::Profile::Deep,
        }
    }
}

fn execute(context: AppContext, command: Commands) -> Result<i32> {
    match command {
        Commands::Check(CheckCommands::Gate(args)) | Commands::QualityGate(args) => {
            run_gate(&context, args)
        }
        Commands::BaselineDrift(args) => run_baseline_drift(&context, args.baseline),
        Commands::Snapshot(args) => run_snapshot(&context, args.output),
        Commands::SnapshotDiff(args) => {
            run_snapshot_diff(&context, args.before, args.after, args.after_live)
        }
        Commands::Change(ChangeCommands::Recommend(args)) | Commands::Recommend(args) => {
            run_recommend(&context, args)
        }
        Commands::Inspect(InspectCommands::Lsp(args)) | Commands::LspEnrich(args) => {
            run_lsp_enrich(&context, args)
        }
        Commands::Inspect(InspectCommands::ContractUsage(args))
        | Commands::ContractUsageMap(args) => run_contract_usage_map(&context, args),
        Commands::Change(ChangeCommands::Rename(args)) | Commands::RenameSafety(args) => {
            run_rename_safety(&context, args)
        }
        Commands::Inspect(InspectCommands::Symbol(args)) | Commands::SymbolTrace(args) => {
            run_symbol_trace(&context, args)
        }
        Commands::Change(ChangeCommands::Briefing(args)) | Commands::Briefing(args) => {
            run_briefing(&context, args)
        }
        Commands::Change(ChangeCommands::Impact(args)) | Commands::ImpactMap(args) => {
            run_impact_map(&context, args)
        }
        Commands::Change(ChangeCommands::Tdd(args)) | Commands::Tdd(args) => {
            run_tdd(&context, args.files)
        }
        Commands::Check(CheckCommands::Repo) | Commands::Doctor => {
            emit_standard_report(&context, analyzers::doctor::analyze(&context.project_root)?)
        }
        Commands::Check(CheckCommands::Topology) | Commands::TopologyDoctor => {
            emit_standard_report(
                &context,
                analyzers::topology::analyze(&context.project_root)?,
            )
        }
        Commands::Check(CheckCommands::Contracts) | Commands::ContractAudit => {
            emit_standard_report(
                &context,
                analyzers::contracts::analyze(&context.project_root)?,
            )
        }
        Commands::Check(CheckCommands::Bindings) | Commands::RuntimeBindings => {
            emit_standard_report(
                &context,
                analyzers::runtime_bindings::analyze(&context.project_root)?,
            )
        }
        Commands::Check(CheckCommands::Drift) | Commands::DriftDetect => emit_standard_report(
            &context,
            analyzers::drift_detect::analyze(&context.project_root)?,
        ),
        Commands::Check(CheckCommands::Arch) | Commands::ArchGuard => emit_standard_report(
            &context,
            analyzers::arch_guard::analyze(&context.project_root)?,
        ),
        Commands::Check(CheckCommands::Proto) => emit_standard_report(
            &context,
            analyzers::check_proto::analyze(&context.project_root)?,
        ),
        Commands::Check(CheckCommands::Determinism) => emit_standard_report(
            &context,
            analyzers::check_determinism::analyze(&context.project_root)?,
        ),
        Commands::Inspect(InspectCommands::Coverage) | Commands::CoverageMap => {
            emit_standard_report(
                &context,
                analyzers::coverage_map::analyze(&context.project_root)?,
            )
        }
        Commands::Legacy(LegacyCommands::RuntimeSmoke(args)) | Commands::RuntimeSmoke(args) => {
            run_runtime_smoke(&context, args)
        }
    }
}

fn run_gate(context: &AppContext, args: QualityGateArgs) -> Result<i32> {
    let gate_config = gate::GateConfig {
        project_root: context.project_root.clone(),
        profile: args.profile.into(),
        base_url: args.base_url,
        fail_fast: args.fail_fast,
    };
    let gate_report = gate::run(&gate_config)?;
    let rendered = gate::render(&gate_report, context.format)?;
    print!("{rendered}");
    Ok(if gate_report.passed { 0 } else { 1 })
}

fn run_baseline_drift(context: &AppContext, baseline: std::path::PathBuf) -> Result<i32> {
    let report =
        analyzers::baseline_drift::analyze(&baseline, &context.project_root).map_err(|error| {
            CliError::Command {
                message: error.to_string(),
            }
        })?;
    emit_structured_report(
        context,
        &report,
        analyzers::baseline_drift::render_json,
        analyzers::baseline_drift::render_human,
        report.verdict == analyzers::baseline_drift::Verdict::Drifted,
    )
}

fn run_snapshot(context: &AppContext, output_path: Option<std::path::PathBuf>) -> Result<i32> {
    let snapshot = analyzers::snapshot::generate(&context.project_root);

    if context.json || output_path.is_some() {
        let json = analyzers::snapshot::render_json(&snapshot)?;
        if let Some(path) = output_path {
            std::fs::write(&path, json)?;
            eprintln!("Snapshot written to {}", path.display());
        } else {
            print!("{json}");
        }
    } else {
        print!(
            "{}",
            analyzers::snapshot::render_human(&snapshot, context.verbose)
        );
    }

    Ok(0)
}

fn run_snapshot_diff(
    context: &AppContext,
    before: std::path::PathBuf,
    after: Option<std::path::PathBuf>,
    after_live: bool,
) -> Result<i32> {
    let before_snapshot =
        analyzers::snapshot_diff::load_snapshot(&before).map_err(|error| CliError::Command {
            message: format!("failed to load 'before' snapshot: {error}"),
        })?;

    let after_snapshot = if after_live {
        analyzers::snapshot::generate(&context.project_root)
    } else {
        let after_path = after.ok_or_else(|| CliError::Command {
            message: "'after' snapshot path required (or use --after-live)".to_string(),
        })?;
        analyzers::snapshot_diff::load_snapshot(&after_path).map_err(|error| CliError::Command {
            message: format!("failed to load 'after' snapshot: {error}"),
        })?
    };

    let diff =
        analyzers::snapshot_diff::diff(&before_snapshot, &after_snapshot).map_err(|error| {
            CliError::Command {
                message: error.to_string(),
            }
        })?;

    emit_structured_report(
        context,
        &diff,
        analyzers::snapshot_diff::render_json,
        analyzers::snapshot_diff::render_human,
        false,
    )
}

fn run_recommend(context: &AppContext, args: RecommendArgs) -> Result<i32> {
    let changed = change_targets::resolve_change_targets(&context.project_root, &args.files);
    let mut report = match args.baseline {
        Some(baseline_path) => analyzers::recommend::analyze_with_baseline(
            &context.project_root,
            &changed.targets,
            &baseline_path,
        ),
        None => analyzers::recommend::analyze(&context.project_root, &changed.targets),
    };
    report.input.detection_mode = changed.source.as_str().into();
    report.scope_note = merge_scope_note(changed.scope_note("files"), &report.scope_note);

    emit_structured_report(
        context,
        &report,
        analyzers::recommend::render_json,
        analyzers::recommend::render_human,
        false,
    )
}

fn run_lsp_enrich(context: &AppContext, args: LspEnrichArgs) -> Result<i32> {
    let enriched = with_lsp_mode(&context.project_root, !args.no_lsp, |bridge| {
        bridge.enrich_symbol(&context.project_root, &args.symbol)
    });

    if context.json {
        print!("{}", serde_json::to_string_pretty(&enriched)?);
    } else {
        print!(
            "{}",
            crate::lsp::render_enriched_human(&enriched, context.verbose)
        );
    }

    Ok(0)
}

fn run_contract_usage_map(context: &AppContext, args: ContractUsageMapArgs) -> Result<i32> {
    let report = with_optional_lsp(
        &context.project_root,
        args.lsp && !args.no_lsp,
        analyzers::contract_usage_map::analyze,
        analyzers::contract_usage_map::analyze_with_lsp,
    );

    emit_structured_report(
        context,
        &report,
        analyzers::contract_usage_map::render_json,
        analyzers::contract_usage_map::render_human,
        false,
    )
}

fn run_rename_safety(context: &AppContext, args: RenameSafetyArgs) -> Result<i32> {
    let next_name = args.new_name.as_deref();
    let report = with_optional_lsp(
        &context.project_root,
        args.lsp && !args.no_lsp,
        |project_root| analyzers::rename_safety::check(project_root, &args.symbol, next_name),
        |project_root, bridge| {
            analyzers::rename_safety::check_with_lsp(project_root, &args.symbol, next_name, bridge)
        },
    );

    emit_structured_report(
        context,
        &report,
        analyzers::rename_safety::render_json,
        analyzers::rename_safety::render_human,
        false,
    )
}

fn run_symbol_trace(context: &AppContext, args: crate::cli::SymbolTraceArgs) -> Result<i32> {
    let report = with_optional_lsp(
        &context.project_root,
        args.lsp && !args.no_lsp,
        |project_root| analyzers::symbol_trace::trace(project_root, &args.symbol),
        |project_root, bridge| {
            analyzers::symbol_trace::trace_with_lsp(project_root, &args.symbol, bridge)
        },
    );

    emit_structured_report(
        context,
        &report,
        analyzers::symbol_trace::render_json,
        analyzers::symbol_trace::render_human,
        false,
    )
}

fn run_briefing(context: &AppContext, args: BriefingArgs) -> Result<i32> {
    let changed = change_targets::resolve_change_targets(&context.project_root, &args.targets);
    let mut report = with_optional_lsp(
        &context.project_root,
        args.lsp && !args.no_lsp,
        |project_root| analyzers::briefing::analyze(project_root, &changed.targets),
        |project_root, bridge| {
            analyzers::briefing::analyze_with_lsp(project_root, &changed.targets, bridge)
        },
    );
    report.input_source = changed.source.as_str().into();
    report.scope_note = merge_scope_note(changed.scope_note("targets"), &report.scope_note);

    emit_structured_report(
        context,
        &report,
        analyzers::briefing::render_json,
        analyzers::briefing::render_human,
        false,
    )
}

fn run_impact_map(context: &AppContext, args: crate::cli::ImpactMapArgs) -> Result<i32> {
    let changed = change_targets::resolve_change_targets(&context.project_root, &args.targets);
    let mut report = with_optional_lsp(
        &context.project_root,
        args.lsp && !args.no_lsp,
        |project_root| analyzers::impact_map::analyze(project_root, &changed.targets),
        |project_root, bridge| {
            analyzers::impact_map::analyze_with_lsp(project_root, &changed.targets, bridge)
        },
    );
    report.input_source = changed.source.as_str().into();
    report.scope_note = merge_scope_note(changed.scope_note("targets"), &report.scope_note);

    emit_structured_report(
        context,
        &report,
        analyzers::impact_map::render_json,
        analyzers::impact_map::render_human,
        false,
    )
}

fn run_tdd(context: &AppContext, files: Vec<String>) -> Result<i32> {
    let changed = change_targets::resolve_change_targets(&context.project_root, &files);
    let mut report = analyzers::tdd::analyze(&context.project_root, &changed.targets);
    report.input_source = changed.source.as_str().into();
    report.scope_note = merge_scope_note(changed.scope_note("files"), &report.scope_note);

    emit_structured_report(
        context,
        &report,
        analyzers::tdd::render_json,
        analyzers::tdd::render_human,
        false,
    )
}

fn run_runtime_smoke(context: &AppContext, args: RuntimeSmokeArgs) -> Result<i32> {
    let config = smoke::SmokeConfig::new(&context.project_root, Some(&args.base_url));
    emit_standard_report(context, smoke::run(&config)?)
}

fn emit_standard_report(context: &AppContext, report: crate::models::Report) -> Result<i32> {
    let rendered = output::render(&report, context.format)?;
    print!("{rendered}");
    Ok(if report.passed() { 0 } else { 1 })
}

fn emit_structured_report<T, E>(
    context: &AppContext,
    report: &T,
    render_json: impl FnOnce(&T) -> std::result::Result<String, E>,
    render_human: impl FnOnce(&T, bool) -> String,
    failed: bool,
) -> Result<i32>
where
    E: Into<CliError>,
{
    if context.json {
        print!("{}", render_json(report).map_err(Into::into)?);
    } else {
        print!("{}", render_human(report, context.verbose));
    }

    Ok(if failed { 1 } else { 0 })
}

fn with_optional_lsp<T>(
    project_root: &Path,
    use_lsp: bool,
    analyze: impl FnOnce(&Path) -> T,
    analyze_with_lsp: impl FnOnce(&Path, &mut crate::lsp::GoplsBridge) -> T,
) -> T {
    if use_lsp {
        with_lsp_mode(project_root, true, |bridge| {
            analyze_with_lsp(project_root, bridge)
        })
    } else {
        analyze(project_root)
    }
}

fn with_lsp_mode<T>(
    project_root: &Path,
    enabled: bool,
    run: impl FnOnce(&mut crate::lsp::GoplsBridge) -> T,
) -> T {
    let mut bridge = if enabled {
        crate::lsp::GoplsBridge::new(project_root)
    } else {
        crate::lsp::GoplsBridge::unavailable("--no-lsp flag: LSP enrichment disabled by user")
    };
    let result = run(&mut bridge);
    bridge.shutdown();
    result
}

fn merge_scope_note(input_scope_note: String, analyzer_scope_note: &str) -> String {
    if analyzer_scope_note.trim().is_empty() {
        input_scope_note
    } else {
        format!("{input_scope_note} {analyzer_scope_note}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gate_profile_from_cli_profile_converts_correctly() {
        assert_eq!(
            gate::Profile::from(crate::cli::GateProfile::Fast),
            gate::Profile::Fast
        );
        assert_eq!(
            gate::Profile::from(crate::cli::GateProfile::Ci),
            gate::Profile::Ci
        );
        assert_eq!(
            gate::Profile::from(crate::cli::GateProfile::Deep),
            gate::Profile::Deep
        );
    }

    #[test]
    fn merge_scope_note_preserves_both_contexts() {
        let merged = merge_scope_note("Input source: explicit.".into(), "AST-only scope.");
        assert!(merged.contains("Input source: explicit."));
        assert!(merged.contains("AST-only scope."));
    }
}
