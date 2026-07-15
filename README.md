# gotots

A project-focused Go-to-TypeScript compiler, **not** a general-purpose
one. Its mission: mechanically translate the complete declared
[typescript-go](https://github.com/microsoft/typescript-go) source corpus
into high-quality, statically analyzable TypeScript, with exact semantic
preservation, deterministic regeneration, and automatic detection of new
constructs introduced by future TS-Go revisions. Its first product is a
cleanly regenerated TSTS compiler.

The scope is closed-world completeness over the declared corpus with
open-ended detection for future revisions: every source-bearing unit
receives exactly one disposition (automatically translated, declared
manual body, external stub obligation, or explicitly excluded), supported
behavior is expressed in Go semantics — never source names — and an
unrecognized construct blocks generation instead of degrading it. Every
imported package — including the entire Go standard library — is external:
gotots emits exact typed fail-closed stubs and never implements library
behavior. LSP and fourslash are hard exclusions, inventoried with durable
dispositions.

The normative committed specification lives in `docs/spec/`
(`mission-and-scope.md` first); durable architecture decisions in
`docs/decisions/`; the reviewed design packet in `.analysis/scope/`
(local, untracked).

## Layout

- `cmd/gotots` — the CLI (`gotots census`, `gotots toolchain-id`).
- `internal/goenv` — the hermetic toolchain environment: read-only module
  policy, no network, no workspace/user configuration, `GOTOOLCHAIN=local`.
- `internal/pinning` — immutable source-pin contract and fail-closed
  verification: the go executable digest is checked before the binary is
  ever executed, the in-process Go frontend must match the pin, the GOROOT
  src tree is digest-verified, and every analyzed file is reconciled
  byte-for-byte against the pinned commit tree (cleanliness is checked
  before and after loading).
- `internal/inventory` — census pass 1: the tracked Git tree defines the
  source universe (every tracked `.go` file gets exactly one disposition,
  including nested tool modules and testdata); the go list driver enriches
  it with build/package semantics and toolchain evidence for externals,
  with dependency reachability attributed per scope.
- `internal/profile` — explicit project profile: owned roots, test-only
  roots, categorized hard exclusions, tooling roots, complete build-profile
  inputs, strict validation.
- `internal/census` — census pass 2: typed load of owned source only
  (dependencies come from export data, external bodies are never parsed),
  producing identity-bearing records — files, declarations with validated
  unique IDs and body hashes, directives, rare constructs, contradiction
  edges — from which all aggregates are derived, with exact pass-1/pass-2
  file-set reconciliation.
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

Publication is transactional: reports are staged, hashed into a
`manifest.json` with generator provenance, and atomically renamed into
place — never a mixture of old and new, and never inside the pinned source
tree. `inventory.json` and `census.json` are deterministic with no machine
paths; `environment.json` holds machine evidence. The census fails closed
on pin, digest, or frontend mismatch, dirty or injected source, load/type
errors, unknown AST kinds or directives without dispositions, missing
external evidence, identity collisions, and pass-reconciliation
mismatches. Two clean runs produce byte-identical deterministic reports.
