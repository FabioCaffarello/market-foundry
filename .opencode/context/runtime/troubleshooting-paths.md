# Troubleshooting Paths

First-line order:

1. `make diag`
2. `make ps`
3. `make logs SERVICE=gateway` or the failing service
4. `SERVICE=<name> make restart`
5. `make down` then canonical bring-up again

Typical pivots:

- gateway `readyz` fails -> inspect `gateway`, then `configctl`, then `store`
- no data after seed -> rerun `make seed`, then inspect `ingest` and `derive`
- analytical smoke fails -> inspect `writer`, `clickhouse`, `make migrate-status`
- compose health fail -> run `make compose-config` and inspect `deploy/compose/docker-compose.yaml`
- topology/config drift suspicion -> run `make drift-detect` or direct `raccoon-cli check topology`

Escalate to direct scripts only when you need `../../../scripts/diag-check.sh` in `--local` mode or other harness-level flags.
