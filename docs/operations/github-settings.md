# GitHub Repository Settings

Canonical reference for the GitHub repository settings of `market-foundry`.
Adopted in Phase 3 environment hardening (P3.3, 2026-05-22).

Remote settings have no git history. This file is the only durable
record of the current configuration; treat it as the source of truth
and update it whenever a setting changes via the GitHub UI or API.

## Visibility

- **Public** — anyone can clone, read, view history.
- **Forks**: technically still allowed (`allow_forking: true`). GitHub
  does **not** support disabling forks on personal-owned public repos
  (the API rejects the PATCH with HTTP 422 — "Allow forks setting can
  only be changed on org-owned private repositories").
- **External PRs**: effectively blocked. `pull_request_creation_policy`
  is `collaborators_only`, so only collaborators can open PRs against
  this repo even if an outsider forks it.
- **External issues**: enabled (4 templates present); maintainer
  reviews case-by-case.

If formal fork lockdown becomes necessary later, the repo can be
transferred to a GitHub organization (where `allow_forking` is
settable). The current `collaborators_only` PR policy is the next
strongest signal.

## Branch protection (`main`)

| Setting | Value | Rationale |
|---|---|---|
| `required_status_checks.strict` | true | branch must be up to date with main before merge |
| `required_status_checks.contexts` | Unit Tests, Repository Consistency & Quality Gate, Go Lint (golangci-lint) | fast-feedback essentials |
| `required_pull_request_reviews` | null | solo dev — no PR review requirement |
| `required_linear_history` | true | no merge commits — clean history |
| `allow_force_pushes` | false | prevent main rewriting |
| `allow_deletions` | false | prevent accidental branch delete |
| `enforce_admins` | false | maintainer can bypass in an emergency |
| `lock_branch` | false | direct pushes allowed (gated by checks) |
| `allow_fork_syncing` | false | not applicable (no formal forks pipeline) |

Required status check names must match the `name:` field on the
corresponding jobs in `.github/workflows/ci.yml`. If those names
change, update both this file and the protection rule via:

```bash
gh api -X PUT 'repos/FabioCaffarello/market-foundry/branches/main/protection' \
  --input /tmp/p3.3-branch-protection.json
```

## Security & Analysis

| Toggle | State | Rationale |
|---|---|---|
| `secret_scanning` | enabled | catch committed secrets |
| `secret_scanning_push_protection` | enabled | block push of detected secrets |
| `dependabot_security_updates` | enabled | auto-PR for security CVEs in deps |
| `private_vulnerability_reporting` | enabled | external reporters can submit private security reports |

Two further toggles remain `disabled` by intentional default:
- `secret_scanning_non_provider_patterns` — generic-string heuristics;
  high false-positive rate, can be enabled if a real leak triggers a
  policy review.
- `secret_scanning_validity_checks` — calls the provider to check if a
  detected secret is currently active; off because it sends potential
  secrets to third parties.

Dependabot dependency updates (non-security, version-pin bumps) are
configured separately in `.github/dependabot.yml` (adopted in P2.4).

## Actions permissions

| Setting | Value | Rationale |
|---|---|---|
| `enabled` | true | CI active |
| `allowed_actions` | all | broad ecosystem usage (Dependabot, marketplace) |
| `sha_pinning_required` | true | mitigate tag-rewriting supply chain risk |

**SHA pinning consequence**: tag-pinned actions (e.g.
`actions/checkout@v4`) will be rejected at workflow runtime. The
workflow at `.github/workflows/ci.yml` currently uses tag pins for:

- `actions/checkout`
- `actions/setup-go`
- `actions/cache`
- `golangci/golangci-lint-action`
- `dtolnay/rust-toolchain`

If the next CI run fails because of this, P3.3.1 will migrate the
workflow to SHA pins (e.g.,
`actions/checkout@b4ffde65f46336ab88eb53be808477a3936bae11 # v4`).
Dependabot's `github-actions` ecosystem can automate the bump PRs.

## Repository features

- **Issues**: enabled (4 issue templates: bug_report, feature_request,
  documentation, config).
- **Discussions**: disabled.
- **Wiki**: disabled.
- **Projects**: enabled (lightweight task tracking).

## Webhooks

Currently none.

## Secrets / Variables

Currently none at repository level.

## Modification history

- **2026-05-22** (P3.3): Initial lockdown.
  - Branch protection added on `main` (3 required status checks,
    linear history, no force-push, no deletions).
  - Security & Analysis: `secret_scanning`,
    `secret_scanning_push_protection`,
    `dependabot_security_updates`,
    `private_vulnerability_reporting` enabled.
  - Actions `sha_pinning_required` enabled.
  - `allow_forking: false` attempted and rejected by GitHub (platform
    limitation on personal-owned public repos); manual fallback
    options documented above.
