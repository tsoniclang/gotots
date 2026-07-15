# ADR 0002: Census Before Translation; Go Frontend

Date: 2026-07-15
Status: accepted
Packet reference: `.analysis/scope/24-open-decisions.md` D2 (partially),
`.analysis/scope/22-source-census-and-planning-numbers.md`,
`.analysis/scope/26-language-and-boundary-coverage-matrix.md`

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

3. The implementation language of the middle-end (semantic IR, representation
   planner, emitter) remains **open** (D2). This ADR only fixes the frontend
   boundary: whatever the middle-end is, its input is typed semantic evidence
   produced by the Go frontend, with one semantic source of truth.

## Profile model fixed by this ADR

The project profile (`profiles/tsts/project.json`) partitions the module by
canonical package roots into non-overlapping classes, matched in this
precedence order:

1. `hardExcludedRoots` (categorized: lsp, fourslash, editor-service) — never
   analyzed, never counted, carve-outs allowed inside owned/test-only roots;
2. `testOnlyRoots` — owned test-support source, analyzed under the owned-test
   scope, importable from test scope only;
3. `ownedRoots` — owned production source;
4. anything else in the module — `unselected`;
5. any other module (including the entire Go standard library) — external.

Every import from owned scope into hard-excluded, unselected, or (from
production) test-only packages is a recorded contradiction edge. The census
reports contradictions; generation will refuse to run while any exist.

## Evidence

The published census bundle — not this document — is the authoritative
record; numbers here are orientation from the accepted run. Census against
pin `c78d39e7` (linux-amd64), byte-identical across two clean runs, zero
load/type errors, every tracked source unit classified:

- owned production: 56 packages, 290 files, 186,642 lines, 9,627 bodies
  (2,332 functions + 7,295 methods), 0 bodyless declarations,
  116,232 body statements (derived from per-declaration records);
- owned test: 37 packages, 113 files, 39,283 lines, 1,038 bodies, including
  35 black-box test files retaining their `_test` package identity;
- tracked-source universe: 4,917 tracked `.go` files = 4,899 in inventoried
  packages + 18 tooling (`_tools` nested module); 1 submodule gitlink;
- externals: 152 total with toolchain evidence, 139 reachable from
  owned production/test scope, 13 reachable only through hard-excluded or
  unselected source (excluded from product obligations);
- 2 contradiction edges, both `internal/execute/tsc.go` importing
  hard-excluded editor-service packages (`internal/format`,
  `internal/ls/lsutil`) — a known product finding recorded in the profile
  notes and awaiting a reviewed disposition.

## Reconsider when

- The middle-end language decision (D2) is made after the vertical
  architecture slice; nothing in the census output format may then depend on
  Go-process-internal state that another process could not consume.
