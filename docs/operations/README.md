# Operations

This directory holds operational guides for running and maintaining
market-foundry. The 4 documents below cover the four most common
operational concerns:

| Document | Purpose |
|---|---|
| [deployment.md](deployment.md) | Mode selection (paper/testnet/mainnet), compose variants, credentials, promotion strategy |
| [troubleshooting.md](troubleshooting.md) | Common scenarios and diagnostic steps beyond the basic cases covered in DEVELOPMENT.md |
| [backups.md](backups.md) | ClickHouse backup/restore strategy, scripts, and recovery procedures |
| [smoke-tests.md](smoke-tests.md) | Smoke test selection, when to use which, and how to diagnose failures |

For higher-level orientation:

- New to the project? Start with [`../RESUMPTION.md`](../RESUMPTION.md).
- Daily workflow basics? See [`../DEVELOPMENT.md`](../DEVELOPMENT.md).
- Architecture overview? See [`../ARCHITECTURE.md`](../ARCHITECTURE.md).
- Runtime topology and ports? See [`../RUNTIME.md`](../RUNTIME.md).

## Operating posture

market-foundry is currently designed for **local single-operator**
deployment by default. The operations docs reflect this:

- The default deployment is local Docker Compose, paper mode.
- Mainnet and testnet variants require explicit credentials and
  configuration.
- There is no production-ops document for a multi-tenant or
  high-availability deployment — that is not the current shape.

If the project evolves toward a different deployment posture (e.g.,
cloud-hosted, multi-region, multi-user), the operations docs will
need a major revision.
