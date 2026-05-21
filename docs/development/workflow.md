# Development Workflow

## Canonical Loop

1. Run `make check` before changing code.
2. Run `make tdd` to choose the narrowest validation surface.
3. Implement the smallest correct change.
4. Run `make verify` after the change.
5. Escalate to `make check-deep` or the relevant `make smoke*` target when the
   change is structurally broader.

## Runtime Bring-Up

Fastest path:

```bash
make bootstrap
make live
make smoke
```

Controlled manual path:

```bash
make up
make seed       # or make seed-multi
make smoke
```

## Troubleshooting First

Use this order:

```bash
make diag
make ps
make logs SERVICE=gateway
```

Escalate to direct scripts, `docker compose`, or raw `go` commands only when
you are debugging below the repository workflow contract.

## References

- Root quick reference: [`../../DEVELOPMENT.md`](../../DEVELOPMENT.md)
- Commands and proofs: [`commands-and-proofs.md`](commands-and-proofs.md)
- Repository map: [`repository-map.md`](repository-map.md)
