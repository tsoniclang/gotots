# ADR 0002: Census Before Translation; Go Frontend

Date: 2026-07-15
Status: accepted
Owner: gotots-maintainers
Implementation revision: `f7f2aea`
Schema/ABI impact: `census-schema-v4`, `profile-schema-v1`
Spec reference: `docs/spec/authority-scope.md`,
`docs/spec/input-census-publication.md`, and
`docs/spec/compiler-semantic-ir.md`

## Context

Lexical scans cannot prove Go package selection, declaration identity,
builtins, method selection, generics, or operation semantics. Translation
coverage needs a typed identity-bearing denominator before body lowering.

## Alternatives

- Regex/text census: rejected because spelling is not semantic identity.
- Infer the denominator from emitted TypeScript: rejected because unsupported
  input can disappear before counting.
- Reimplement Go typing in TypeScript: rejected because it creates a second
  semantic authority and drifts from the pinned frontend.

## Decision

1. The first deliverable is `gotots census`: a typed, reproducible,
   fail-closed census of the pinned source under an explicit project profile.
   No translation work precedes it. The census is the completeness authority
   that replaces every historical lexical estimate in the design packet.

2. The census (and the semantic frontend that will grow out of it) is
   implemented in Go against the pinned toolchain's own `go/parser`,
   `go/types`, `go/constant`, and `golang.org/x/tools/go/packages`. Go
   language semantics are never re-derived by hand: classification uses
   `go/types` object identity, not source spelling (a local function named
   `append` is not the builtin; an imported package named `maps` is not the
   language map operation).

3. The middle-end, representation planner, and emitter are implemented in Go.
   Their input remains typed semantic evidence produced by the Go frontend,
   with one semantic source of truth.

## Effects

The Go frontend and exact selected census become mandatory predecessors of all
translation and support claims. Outside roots contribute no census records;
permitted imported packages contribute typed external obligations only.

## Migration

Lexical estimates and reports without canonical identity are non-authoritative.
They are replaced by regenerated census-schema-v4 records; no adapter preserves
their counts as product evidence.

## Profile model fixed by this ADR

The project profile (`profiles/tsts/project.json`) selects non-overlapping
production, test, and tool roots. LSP, fourslash, and editor-service roots are
completely outside the GoToTS input universe and are filtered before source
census. Their files, declarations, bodies, tests, and external dependencies
receive no census records.

The frontend computes the dependency closure from selected roots. An edge into
an outside root is a profile error. Packages in other modules, including the Go
standard library, are external declaration obligations rather than source
inputs.

## Evidence

Registry entry: `docs/decisions/registry.json#ADR-0002`. Proof artifacts:
`profiles/tsts/project.json`, `internal/census`, `internal/inventory`, and
`internal/profile`.

The published census bundle—not this document—is the authoritative record.
Numbers here orient the accepted selected-source run against pin `c78d39e7`
(`linux-amd64`), byte-identical across two clean runs and with zero load/type
errors:

- owned production: 56 packages, 290 files, 186,642 lines, 9,627 bodies
  (2,332 functions + 7,295 methods), 0 bodyless declarations,
  116,232 body statements (derived from per-declaration records);
- owned test: 37 packages, 113 files, 39,283 lines, 1,038 bodies, including
  35 black-box test files retaining their `_test` package identity;
- selected external obligations derive only from selected production/test
  dependency closure;
- the profile excludes the `internal/execute` driver family by ADR 0003
  because it depends on editor-service packages outside the universe.

## Reconsider when

- A replacement implementation is accepted only through a new ADR proving the
  complete frontend, IR, proof, determinism, and selected-product contract.
