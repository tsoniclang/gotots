# gotots

A project-focused Go-to-TypeScript compiler. Its first product is a cleanly
regenerated TSTS compiler translated from a pinned
[typescript-go](https://github.com/microsoft/typescript-go) revision.

Gotots translates **Go language semantics and explicitly selected owned
project source**. Every imported package — including the entire Go standard
library — is external: gotots emits exact typed fail-closed stubs for it and
never implements library behavior. LSP and fourslash are hard exclusions.

The reviewed design packet lives in `.analysis/scope/` (local, untracked).
Durable architecture decisions are recorded in `docs/decisions/`.

## Layout

- `cmd/gotots` — the CLI (`gotots census`, `gotots toolchain-id`).
- `internal/goenv` — the hermetic toolchain environment: read-only module
  policy, no network, no workspace/user configuration, `GOTOOLCHAIN=local`.
- `internal/pinning` — immutable source-pin contract and fail-closed
  verification: revision, cleanliness (before and after loading), module
  identity, go executable sha256, and GOROOT src tree digest.
- `internal/inventory` — census pass 1: a complete module/file/dependency
  enumeration via the go list driver (no syntax), with toolchain evidence
  for the standard-library/module split.
- `internal/profile` — explicit project profile: owned roots, test-only
  roots, categorized hard exclusions, build profiles, strict validation.
- `internal/census` — census pass 2: typed load of owned source only
  (dependencies come from export data, external bodies are never parsed),
  producing identity-bearing records — files, declarations with body
  hashes, directives, rare constructs, contradiction edges — from which all
  aggregates are derived.
- `pins/` — pinned upstream source identities.
- `profiles/` — per-product project profiles (`profiles/tsts`).
- `docs/decisions/` — architecture decision records.

## Census

```sh
go build -o bin/gotots ./cmd/gotots
bin/gotots census \
  --profile profiles/tsts/project.json \
  --source /path/to/pinned/typescript-go \
  --out reports/
```

Outputs `inventory.json` and `census.json` (deterministic, no machine
paths) plus `environment.json` (machine evidence). The census fails closed:
pin or toolchain-digest mismatch, dirty checkout (before or after loading),
any load/type error, an unknown AST node kind, or an owned package present
in pass 1 but missing from pass 2 aborts without output. Two clean runs
produce byte-identical deterministic reports.
