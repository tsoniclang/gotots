# ADR 0008: Interface, Member, And Conversion Artifact Ownership

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `50ea69f` (the commit recording these decisions;
measurement evidence binds to baseline `65810f5`)
Implementation status: planned
Schema/ABI impact: `lowering-plan-v1`
Spec reference: `docs/spec/interfaces-generics-functions.md`,
`docs/spec/representation-output.md`
Registry entry: `docs/decisions/registry.json#ADR-0008`

## Context

Consumer modules each emit their own copies of interface machinery: the
byte-attribution baseline measured 663 alias copies over 72 canonical
identities (42.4% of core output), 3,897 inline box vtables, one 661 KB
`any`-typed union duplicated across 101 modules, and `GoParamRtti<V> =>
unknown` boxing born of the missing canonical type names. Calibration
fixture D3 measures the tail: a 424-byte Go function expands 140× purely
through interface formatting machinery.

## Alternatives

- Keep consumer-local aliases and inline vtables: rejected — the single
  largest measured duplication class.
- One global interface registry module: rejected — a monolith recreating
  the module-size failure and an initialization chokepoint.
- Erase interfaces to `unknown` and refine at use: rejected — recovers
  semantic value from erased types, banned by the static contract.

## Decision

The canonical interface artifact owns the exact discriminated-union TYPE
and the target method contract, once per canonical identity, importing
member payload types type-only. Each concrete member's module owns that
member's RTTI, its target-specific minimal vtables (only the target
interface's methods), its box constructors, and its adapters, once per
(member, target) pair. Source-union-to-target-interface conversions are
canonical conversion-edge artifacts placed by the runtime dependency
graph; function-reference fan-in is acceptable, initialization-time
fan-in is not. The empty interface carries an empty method table;
assertions to narrower interfaces reconstruct the target box through the
member's own constructor, with an explicit nil arm in panic forms. No
semantic value is ever recovered from `any`/`unknown`.

## Effects

Definitions emit once and references repeat — enforced by the
duplicate-definition shape-gate detector. The 661 KB union and the
per-consumer vtable copies become single canonical artifacts. Generic
boxing operations become typed against canonical target names, deleting
the `unknown`-returning RTTI forms.

## Migration

Consumer-local aliases, per-module finalization loops, full concrete
vtables in narrow-interface boxes, and erased-result recovery are
deleted in the interface-ownership wave (numbered-order steps 19-20);
D3 and B8 are re-measured at the shape/size gate afterward.

## Evidence

Byte-attribution baseline `.analysis/rebaseline-65810f5/byte-attribution.md`
(663/72 alias measurement) at the implementation revision; calibration
fixtures D3 (140× interface-formatting tail), B8 (10.2× conversion
site), B3/B4 (assertion/type-switch shapes); detector
`CountDuplicateDefinitions` in `internal/shapegate`.

## Reconsider when

An ESM initialization-order counterexample arises that the conversion-
edge placement rules cannot express without an initialization cycle.
