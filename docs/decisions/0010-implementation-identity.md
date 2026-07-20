# ADR 0010: Implementation Identity And Emitted Naming

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `65810f5` (reviewed evidence baseline)
Implementation status: planned
Schema/ABI impact: `implementation-identity-v1`
Spec reference: `docs/spec/representation-output.md`,
`docs/spec/machine-contracts-diagnostics.md`
Registry entry: `docs/decisions/registry.json#ADR-0010`

## Context

Generic family specializations currently share their source declaration
ID: 71 `$ek`/`$pc` family implementations overwrite each other's
analysis artifacts and hashes at the implementation revision — active
evidence loss, confirmed by the reconciliation identity join (the six
"placeholder re-emissions" are family variants of five SyncMap bodies
plus OrderedMap.UnmarshalJSONFrom). The artifact path sanitizer is
non-injective with no collision rejection.

## Alternatives

- Keep source-declaration identity and last-writer-wins artifacts:
  rejected — measured evidence loss today.
- Content-hash identity: rejected — unstable across formatting changes,
  cannot express "same implementation, new body".
- Per-run sequence numbers: rejected — non-deterministic across runs.

## Decision

Source identity and implementation identity are distinct:

    ImplementationID = SourceDeclarationID "/" specializationKey
    specializationKey = "default" | binding-family key | exception key

The grammar covers the current family variants (map-key-encoded,
pointer-cell) AND the plan-derived binding families of ADR 0007, so
identities never migrate twice. Every implementation owns exactly one
emitted symbol, formatted body, post-format sha256, source map segment,
dependency set, and ownership state, all keyed by ImplementationID.
Emitted names derive deterministically from the ID; ledgers store the
full collision-resistant identity; duplicate IDs, duplicate artifact
owners, duplicate hash writes, and shortened-name collisions fail
generation rather than overwriting.

## Effects

The source-declaration proof becomes one-to-many over implementations;
per-implementation hashes become trustworthy evidence; manual ownership
(ADR 0005) binds to implementations rather than source declarations.

## Migration

First semantic code change of the rewrite (numbered-order step 15): the
ledger and artifact writers key by ImplementationID, the 71 overwriting
variants receive distinct identities, and the reconciliation report
gains the implementation-level join.

## Evidence

Reconciliation identity join at the implementation revision
(`internal/reconcile`, variantReemissions record); the 45-occurrence /
39-identity placeholder split proven in the rebaseline count
reconciliation; non-injective sanitizer collision observed during
calibration artifact joining (`internal/calibration`).

## Reconsider when

A source upgrade produces binding families the grammar cannot name, or
the deterministic shortened-name derivation collides at a rate the
rejection path cannot practically resolve.
