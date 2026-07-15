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

The normative committed specification lives in `docs/spec/`; durable
architecture decisions live in `docs/decisions/`. The design packet in
`.analysis/scope/` is local review context and candidate analysis, not a
substitute for the committed specification.

## Specification

Read the normative documents in this order:

1. `docs/spec/mission-and-scope.md` — corpus boundary and completion model.
2. `docs/spec/translation-and-output.md` — generated TSTS architecture and
   worked Go-to-TypeScript shapes.
3. `docs/spec/performance-and-representation.md` — representation proof,
   benchmarks, and regression gates.
4. `docs/spec/testing-and-acceptance.md` — ordered test layers, differential
   oracles, corpus proof, self-compilation, and final acceptance.
5. `docs/spec/file-size-and-decomposition.md` — mandatory semantic file
   decomposition and repository-wide enforcement.

The output target is a strict static ESM TSTS compiler core, a generated
Go-language ABI and exact external contracts, and separately owned TSTS
product extensions composed through typed generated seams. Generated output
must remain suitable for Tsonic C# and Rust compilation.

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

## Validation

The complete process is normative in
`docs/spec/testing-and-acceptance.md`. Local checkpoints run formatting,
vetting, unit/fixture tests, and diff checks; pushes additionally run the race
suite. Product acceptance adds exact census/disposition proof, declaration and
body verification, Go semantic oracles, strict generated-TypeScript checks,
byte-identical regeneration, no-extension TS-Go differential behavior, TSTS
extension tests, the complete selected compiler corpus, proof projects,
self-compilation/native-target probes, and measured performance gates.

No fixed remembered test count is authoritative. The machine-discovered,
identity-bearing test ledger is the denominator for each run.

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
