# gotots

A project-focused Go-to-TypeScript compiler, **not** a general-purpose
one. Its mission: mechanically translate the complete selected
[typescript-go](https://github.com/microsoft/typescript-go) source corpus
into high-quality, statically analyzable TypeScript, with exact semantic
preservation, deterministic regeneration, and automatic detection of new
constructs introduced by future TS-Go revisions. Its first product is a
cleanly regenerated TSTS compiler.

The scope is closed-world completeness over the selected compiler corpus with
open-ended detection for future revisions. LSP, fourslash, and editor-service
roots are filtered before census and are completely outside the input universe.
Source, declaration, body, artifact, operation, and test ownership inside the
selected universe must be independently complete and reconciled. Supported behavior
is expressed in Go semantics—never source names. A recognized difficult class
may remain explicitly unimplemented; affected artifacts are withheld rather
than approximated. Every imported package outside the selected owned package
set, including the entire Go standard library, is an external typed contract.

The normative committed specification lives only in `docs/spec/`; durable
architecture decisions live in `docs/decisions/`.

## Specification

Start with `docs/spec/README.md`. It defines the complete reading order across
authority, compiler architecture, semantic IR, fixed-point representation
planning, language domains, external/manual/extension ownership, testing,
performance, and upgrades. `docs/spec/manifest.json` is the enforced inventory.

The output target is a strict static ESM TSTS compiler core, exact external
contracts, and separately owned TSTS product extensions composed through typed
generated seams. Complete Go semantics stay in typed IR and proof records;
runtime output uses the simplest representation proven exact for each region.

## Layout

- `cmd/gotots` — the CLI (`gotots census`, `gotots gate`,
  `gotots translate-probe`, `gotots toolchain-id`).
- `internal/goenv` — the hermetic toolchain environment: read-only module
  policy, no network, no workspace/user configuration, `GOTOOLCHAIN=local`.
- `internal/pinning` — immutable source-pin contract and fail-closed
  verification: the go executable digest is checked before the binary is
  ever executed, the in-process Go frontend must match the pin, the GOROOT
  src tree is digest-verified, and every analyzed file is reconciled
  byte-for-byte against the pinned commit tree (cleanliness is checked
  before and after loading).
- `internal/inventory` — source discovery and attestation. Its completed
  selected-universe contract filters outside LSP/fourslash/editor roots before
  census, while the Go loader supplies build/package semantics and external
  dependency evidence.
- `internal/profile` — explicit project profile: selected roots, completely
  outside roots, selected test/tool roots, complete build-profile inputs, and
  strict dependency-boundary validation.
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
`docs/spec/11-testing-acceptance.md`. Local checkpoints run formatting,
vetting, unit/fixture tests, and diff checks; pushes additionally run the race
suite. Product acceptance adds exact census/disposition proof, declaration and
body verification, Go semantic oracles, strict generated-TypeScript checks,
byte-identical regeneration, no-extension TS-Go differential behavior, TSTS
extension tests, the complete selected compiler corpus, proof projects,
common projects, and measured performance gates.

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
