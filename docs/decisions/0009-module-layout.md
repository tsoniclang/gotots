# ADR 0009: Generated Module Layout Mirrors Go Source Files

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `65810f5` (reviewed evidence baseline)
Implementation status: planned
Schema/ABI impact: `module-layout-v2`
Spec reference: `docs/spec/representation-output.md`,
`docs/spec/packages-concurrency.md`
Registry entry: `docs/decisions/registry.json#ADR-0009`

## Context

One module per Go package produced a 23 MB ast module and a 74k-line
checker module, forcing a 12 GB typecheck heap, breaking editor and
review tooling, and hiding duplication inside single files. The
alternative extreme, one module per declaration (~13k modules), hurts
checker throughput and import volume without reducing bytes.

## Alternatives

- Module per Go package (baseline): rejected — measured 23 MB single
  files and 12 GB typecheck heap.
- Module per declaration: rejected — import explosion, no byte savings.
- Hand-curated module map: rejected — non-derivable, drifts from source.

## Decision

Generated layout mirrors Go source files (`internal/ast/node.go` emits
`ast/node.ts`) by default. Canonical interface, conversion, and
specialization artifacts live with their semantic owners. Merging
source-file modules is permitted ONLY on machine evidence from the
runtime dependency graph, whose edges are classified as: type-only
(erased), function references not executed at initialization, class
heritage, package-variable initialization, and top-level
execution/registration. Only the last three can force co-location or
ordering, and every merge names its forcing edge in the layout report.
No ESM cycle is assumed safe or unsafe by spelling; the classified graph
decides. Type-only imports remain erased. Facade modules exist only
where the product API semantically owns a facade.

## Effects

Module sizes become bounded and deterministic; source maps, manual-body
diffs, and reviews operate on file-sized units; implementation IDs are
independent of layout so re-partitioning never changes identity.

## Migration

Layout cutover happens with source-shaped lowering (numbered-order
steps 18-21); the single-module-per-package writer is deleted; the
stage-16 gate re-measures typecheck heap and wall time against the
recorded baseline.

## Evidence

Byte-attribution and module-census at the implementation revision
(23 MB `ast` module, 74k-line `checker` module, 12 GB tsc heap recorded
in `.analysis/rebaseline-65810f5/`); runtime-SCC edge classes settled in
the plan's `root-contract.md` and `message-to-implementer.md`.

## Reconsider when

Stage-16 checker-performance evidence shows the mirrored layout exceeds
the committed typecheck budget and a measured merge policy meets it.
