# ADR 0010: Respect httprouter trie constraints

## Status

Accepted.

## Context

The gateway uses [`julienschmidt/httprouter`](https://github.com/julienschmidt/httprouter)
as its HTTP request router. httprouter has a particular constraint:
**static paths and wildcard paths cannot coexist within the same
prefix segment**.

Concretely:

```
/execution/list           ← static last segment
/execution/:type          ← wildcard last segment
```

These two cannot coexist. httprouter panics at registration time:
> panic: ':type' in new path '/execution/:type' conflicts with existing
> handle for path '/execution/list'

This was discovered the hard way during P0 (Phase 0). The system had
been growing with route additions that gradually accumulated
3 simultaneous conflicts (`execution/lifecycle/list` vs `/execution/:type`,
`/session/list` vs `/session/:id`, `/session/batch-audit` vs `/session/:id`).
At some point a deployment surfaced all three at once. Gateway entered
CrashLoopBackoff.

P0.6 resolved by renaming the static paths to use hyphens, removing
the prefix segment conflict:

- `/execution/list` → `/execution-list` (kept for verification)
- `/execution/source-explain` → `/execution-source-explain`
- `/session/list` → `/session-list`
- `/session/batch-audit` → `/session-batch-audit`

A `cmd/gateway/boot_test.go` was added that exercises all routes in a
fresh httprouter at test time, catching future conflicts in CI.

## Decision

**Static paths and wildcard paths must not coexist within the same
prefix segment.** When the two patterns would otherwise conflict:

1. Prefer renaming the static path to use hyphens
   (`/session/list` → `/session-list`).
2. The wildcard path keeps its form
   (`/session/:id` survives).

Any new HTTP route must:
1. Be added to the routes registration in `internal/interfaces/http/routes/`.
2. **Also be added to `cmd/gateway/boot_test.go`'s `routes` slice.**

The boot test is a hard CI gate. A PR adding a route without updating
the test will fail CI.

## Consequences

### Positive

- **No more boot panics from route conflicts**: the boot test catches
  conflicts at CI time, before they reach a container.
- **Explicit route registry**: `routes` slice in boot_test.go is the
  canonical list of all registered routes.
- **Hyphenated paths are a visible signal**: anyone reading the URL
  surface sees that the path was deliberately renamed. The hyphen
  is debt visible to anyone interacting with the system, ensuring
  it's not forgotten.

### Negative

- **Hyphenated paths are aesthetically inconsistent**: most paths use
  slashes; the hyphenated ones stand out. This is documented as D1
  surface debt in [`../RESUMPTION.md`](../RESUMPTION.md).
- **Coupled change requirement**: adding a route is a 2-file change
  (registration + boot_test). Contributors must remember this.
  Mitigated by [`../CONTRIBUTING.md`](../CONTRIBUTING.md) explicitly
  listing the requirement, and by the boot test failing loudly when
  forgotten.
- **httprouter dependency lock-in**: switching to a different router
  (chi, gin) would be a substantial refactor. We are not planning to
  switch, but this is a known coupling.

## Alternatives considered

**Switch to a router that handles this** (chi, gorilla/mux): rejected
because it's a large refactor for a constraint we can work around. The
hyphenated paths are not pretty but they work; the boot test catches
regressions; the cost of the switch isn't justified.

**Allow conflicts and live with route ambiguity**: rejected because
httprouter doesn't allow it — it panics at registration.

**Use deeper paths** (e.g., `/execution/static/list` and `/execution/by-id/:type`):
considered but adds awkward path depth and doesn't generalize well.
The hyphenated approach is more direct.

## References

- `cmd/gateway/boot_test.go` — the test that catches conflicts
- `internal/interfaces/http/routes/` — route registrations
- D1 in [`../RESUMPTION.md`](../RESUMPTION.md) — hyphenated paths
  documented as surface debt
- [`../HTTP-API.md`](../HTTP-API.md) → notes on hyphenated paths
  (`/session-list`, `/session-batch-audit`, `/execution-source-explain`)
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — PR rule about adding
  routes
- ADR [0005](0005-layer-sovereignty.md) — layer enforcement (gateway
  is the only HTTP layer)
