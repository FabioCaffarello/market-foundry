# Raccoon CLI Command Trustworthiness And Error Semantics

## Purpose

This document defines what it means for `raccoon-cli` to be trustworthy as a
development CLI.

Trustworthiness here means a contributor can run a command and understand:

- what the command inspected;
- whether the result is complete enough to trust;
- whether the command failed operationally or the repository failed validation;
- what to do next.

## Core Trust Rules

### 1. Input provenance must be visible

Commands that infer targets from `git status` must say so.

The command must distinguish:

- explicit targets;
- auto-detected structural targets;
- fallback to all changed paths;
- clean worktree;
- non-git repository root;
- git unavailable/failed.

Silently collapsing these cases into “no targets” is not acceptable.

### 2. Empty scope must be explained

An empty report is only trustworthy when the user can tell why it is empty.

Examples:

- clean worktree;
- not a git repository;
- omitted explicit arguments;
- baseline-only invocation.

### 3. Check failure is not execution failure

If the CLI can inspect the repository and produce a valid report, it should do
so and return exit code `1` when the checks fail.

Exit code `2` is reserved for:

- clap argument failures;
- file IO or JSON decode errors that prevent a trustworthy report;
- other command runtime failures.

### 4. Human and JSON output must describe the same truth

If JSON reports `not_git_repository`, the human output must not imply “clean
worktree”.

If human output says a command auto-detected structural files, JSON should also
carry source metadata that confirms that claim.

### 5. Stderr must stay reserved for real command errors

Routine validation failures belong in stdout report output.

stderr should be used for:

- parse/runtime errors;
- explicit write notifications when the command intentionally uses a file side
  effect.

## Command-Specific Semantics

## `tdd`

- Must explain why no files were available.
- Must expose `input_source`.
- Must filter docs-only noise when auto-detecting structural scope.

## `recommend`

- Must expose `input.detection_mode`.
- Must keep “no input files” distinct from “baseline supplied”.
- Must preserve actionable output even when no scope can be derived.

## `briefing`

- Must expose `input_source`.
- Must not imply LSP enrichment unless it was actually requested and used.
- Must explain empty target resolution instead of silently returning an empty
  briefing.

## `impact-map`

- Must expose `input_source`.
- Must preserve scope disclaimers, especially when no targets resolve.

## `quality-gate`

- Must keep deterministic step ordering.
- Must preserve `0/1/2` semantics.
- Must keep runtime-smoke clearly marked as compatibility and profile-gated.

## Reliability Smells

Any of the following should be treated as trustworthiness regressions:

- “no changes detected” when the root is actually outside git;
- machine output that omits provenance while human output includes it;
- commands that switch between exit code `1` and `2` for the same repository
  condition;
- stderr noise on ordinary failed checks;
- human output that requires repository context not shown in the report.

## Review Checklist

When reviewing CLI changes, ask:

- Does the command reveal how it found its scope?
- Is empty scope explained, not guessed?
- Are human and JSON outputs aligned?
- Is exit code `2` used only for true execution/parsing failures?
- Can this behavior be asserted with one fast integration test?
