# ADR 0007: paper venue adapter as default safe mode

## Status

Accepted.

## Context

Trading systems carry real-money risk. A misconfigured deployment
that connects to a live exchange and submits unintended orders can
cause material financial loss in seconds.

market-foundry's execute binary supports three modes (paper, testnet,
mainnet) with explicit venue adapters per mode. The question is:
**what is the default mode when no explicit configuration says
otherwise?**

The candidates:

- **Default to nothing** (require explicit mode selection). Safe but
  forces every operator to specify mode even for routine development.
- **Default to paper** (`PaperVenueAdapter` synthesizes fills locally,
  no venue contact). Safe — no money at risk in any default scenario.
- **Default to testnet** (real venue but fake money). Less safe —
  requires Binance Testnet credentials, has its own configuration
  surface, and reveals network reachability of test endpoints.
- **Default to mainnet** (with sane safety bounds). Unacceptable —
  no safety bound is sufficient when the default is real money.

## Decision

**Paper mode is the default execution mode.** The execute binary,
when started without explicit mode-specific configuration, runs
`PaperVenueAdapter` which synthesizes fills locally with no venue
contact.

Promotion to testnet or mainnet requires:
- Explicit configuration in `deploy/configs/execute-*.jsonc` selecting
  the variant.
- Use of the appropriate compose file
  (`docker-compose.mainnet-dry-run.yaml`, `docker-compose.mainnet-live.yaml`,
  etc.).
- Provision of API credentials in the appropriate env file.

There is no way to accidentally end up in live mode by misconfiguration
of the default deployment.

## Consequences

### Positive

- **Safe-by-default**: no operator can accidentally cause financial
  loss by running the default bring-up.
- **Reduces operational anxiety**: contributors can develop and test
  without worrying about which mode they're in.
- **Aligns with test/CI**: smoke tests and CI all run paper mode,
  matching the default.
- **Clear "danger zone" boundary**: moving to live requires deliberate
  config + credential setup, making the risk explicit.

### Negative

- **Paper mode is not real**: PaperVenueAdapter synthesizes fills with
  simple rules (typically immediate fill at quoted price). It does
  not model partial fills, slippage, latency, or rejection that real
  venues would produce. Strategies behaving well in paper mode may
  behave differently against a real venue.
- **Default ≠ what live users want**: a deployer who wants live mode
  must configure more. This is intentional (the cost of safety) but
  adds friction.
- **PaperVenueAdapter maintenance**: it must stay reasonably realistic
  to be a useful default. If it diverges significantly from real
  venue behavior, paper testing loses validation power.

## Alternatives considered

**Require explicit mode selection at startup**: rejected because it
adds friction to development without adding safety beyond what
"default to paper" provides.

**Default to testnet**: rejected for two reasons. First, testnet
requires credentials, which makes the default non-functional for
new contributors who haven't set up Binance test accounts. Second,
testnet reveals network reachability of test endpoints, which is
slightly more disclosure than paper mode.

**Default to mainnet with safety checks**: rejected categorically.
No safety check is reliable enough to make this acceptable.

## References

- `internal/application/execution/paper_venue_adapter.go` — the adapter
- `deploy/configs/execute.jsonc` — default config selecting paper mode
- `deploy/configs/execute-mainnet-*.jsonc` — explicit opt-in variants
- [`../operations/deployment.md`](../operations/deployment.md) → modes
- [`../domain/execution.md`](../domain/execution.md) → modes
- ADR [0006](0006-configctl-lifecycle-authority.md) — lifecycle
  authority enforces no implicit mode switching
