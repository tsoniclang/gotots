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

- `cmd/gotots` — the CLI (`gotots census`).
- `internal/pinning` — immutable source-pin contract and fail-closed
  verification (revision, cleanliness, module, toolchain identity).
- `internal/profile` — explicit project profile: owned roots, test-only
  roots, categorized hard exclusions, build profiles.
- `internal/census` — the typed source census: partition, declaration and
  body inventories, construct/builtin/directive census, external import
  obligations, and profile contradiction edges.
- `pins/` — pinned upstream source identities.
- `profiles/` — per-product project profiles (`profiles/tsts`).
- `docs/decisions/` — architecture decision records.

## Census

```sh
go build -o bin/gotots ./cmd/gotots
bin/gotots census \
  --profile profiles/tsts/project.json \
  --source /path/to/pinned/typescript-go \
  --out census.json
```

The census fails closed: pin mismatch, dirty checkout, toolchain mismatch, or
any load/type error aborts without output. Two clean runs produce
byte-identical reports.
