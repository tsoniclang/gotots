# ADR 0007: Generic Specialization And Factoring Policy

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `50ea69f` (the commit recording these decisions;
measurement evidence binds to baseline `65810f5`)
Implementation status: planned
Schema/ABI impact: `lowering-plan-v1`
Spec reference: `docs/spec/interfaces-generics-functions.md`
Registry entry: `docs/decisions/registry.json#ADR-0007`

## Context

The baseline serialized every possible generic capability at every call
site as hidden arguments (`zero$T`/`eq$T`/`clone$T`/`set$T`/`key`/`rt`
factories), measured at 3.5-10× expansion on calibration fixtures and
visible in every ordinary call. Full naive monomorphization risks
template bloat in the other direction. Fixture C2 (OrderedMap family)
shows real binding-family variance; fixtures A3/A4 show generics that
need no operations at all.

## Alternatives

- Keep hidden-argument capability passing: rejected — the measured
  expansion driver; not source-shaped; infects non-generic callers.
- Always fully monomorphize: rejected — duplicates operation-free
  algorithm bodies without evidence of need.
- Always factor shared cores with thin specialized wrappers: rejected —
  reconstructs the hidden-argument protocol behind a rename.

## Decision

For each generic implementation the planner selects the FIRST exact form
in this order, and machine evidence is required to move past each step:
(1) ordinary parametric TypeScript; (2) an operation owned by the
selected data representation; (3) a direct concrete operation at the
leaf call site; (4) one static specialization per actual operation
binding family, emitted once, called directly, each with its own
ImplementationID; (5) a typed recorded exception. A `pass-specialized`
calibration verdict must record why forms 1-3 were insufficient.
Factoring an operation-free fragment out of a specialization family is
permitted only with measured evidence that full specialization is worse,
and the fragment must be newly typed, receive only source values, and
own its algorithm — never a wrapper over a hidden-argument
implementation. A module-level operations record is a recorded
exception; per-call-site allocation and computed lookup of operation
records are prohibited.

## Effects

Ordinary calls carry zero hidden operation arguments — a shape-gate
detector, not prose. Specializations are enumerable, named, and
individually attributable in the byte-attribution schema.

## Migration

The parallel capability vectors (KeyedParams/HardKeyed/ErasedParams/
PtrParams/RttiParams/RttiArgs) and both factory-argument protocols are
deleted at the lowering-plan cutover (numbered-order steps 18-20); no
call site may retain a hidden argument after that wave.

## Evidence

Calibration fixtures A3/A4 (parametric, hand ports 1.0×), B6
(representation-owned operation), C2 (binding-family specialization,
hand port authored), C1/C3 (planning decisions pending call-site
evidence joins); `calibration/measurements.json` ordinary median 1.02×;
shape-gate detector `CountHiddenOperationArgs` in `internal/shapegate`.

## Reconsider when

A binding family's specialization set is measured to exceed the byte or
performance budget that a factored core would meet, per family, with
both variants built and measured.
