# ADR 0011: Denominator Vocabulary And Byte-Attribution Schema

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `50ea69f` (the commit recording these decisions;
measurement evidence binds to baseline `65810f5`)
Implementation status: planned
Schema/ABI impact: `reconciliation-report-v1`
Spec reference: `docs/spec/input-census-publication.md`,
`docs/spec/representation-output.md`
Registry entry: `docs/decisions/registry.json#ADR-0011`

## Context

Historical reports mixed denominators (9,518 versus 9,513 bodies; 694
versus 695 external sites) and quoted ratios without naming what they
divide; the byte-attribution baseline showed one duplication class
producing 42.4% of core output while a residual bucket was misread as
"the code ratio". Progress claims were not mechanically joinable to
evidence.

## Alternatives

- Keep prose-reconciled counts: rejected — the class of error this wave
  repeatedly caught (drifting denominators, unlabeled ratios).
- One universal denominator: rejected — production bodies, support
  records, and external obligations are genuinely different sets.
- Percentages without categories: rejected — moving bytes between
  unattributed buckets can hide regressions.

## Decision

Every published count carries a NAMED denominator from the
reconciliation report's vocabulary (census-production-bodies,
support-body-records, extern-obligations, and peers), produced by
`internal/reconcile` with identity joins, sorted-set digests for both
sides, and a classified disposition for every delta. Size ratios always
name numerator and denominator; total-product, generated-core
(including interface types), core-residual (excluding measured alias
definitions), and external output are reported separately against the
exact selected Go source denominator frozen at scope reconciliation.
Byte attribution assigns every output byte exactly one category
(ordinary bodies, declarations, imports, specializations, canonical
interface artifacts, ABI, external contracts, exception lowering,
formatting); unattributed bytes must be zero at the calibration gate.
Size thresholds derive only from reviewed hand ports; no pre-frozen
allowance exists.

## Effects

Progress percentages without named denominators are non-evidence; the
shape/size gate consumes this schema; category moves cannot hide bytes;
the reconciliation Markdown is a pure renderer of the JSON facts.

## Migration

Already implemented for counts in `internal/reconcile` (numbered-order
steps 2-4, hardened with mutation fixtures); the byte-category
attribution binds to the typed emitter at the shape/size gate
(numbered-order steps 13 and 22). Historical reports are not rewritten;
they are superseded by rendered reports.

## Evidence

`internal/reconcile` with six mutation fixtures green;
`.analysis/rebaseline-65810f5/count-reconciliation.json` (identity
joins, digests, dispositioned deltas, in-JSON history);
`calibration/measurements.json` (per-fixture named byte columns);
`internal/shapegate` size-verdict skeleton.

## Reconsider when

A new output class appears that no existing category honestly covers;
extension is by adding a reviewed category, never by widening an
existing one.
