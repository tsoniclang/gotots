# ADR 0006: Nil-Receiver Calls Require Observational-Equivalence Proof

Date: 2026-07-20
Status: accepted
Owner: gotots-maintainers
Decision revision: `65810f5` (reviewed evidence baseline)
Implementation status: planned
Schema/ABI impact: `lowering-plan-v1`
Spec reference: `docs/spec/interfaces-generics-functions.md`
Registry entry: `docs/decisions/registry.json#ADR-0006`

## Context

Go panics on a nil pointer receiver at the first dereference INSIDE the
method body; a JavaScript method call on `undefined` throws before entry.
The baseline architecture resolved this by lowering every method to a
receiver-taking free function (`Recv$Type$Method(recv, ...)`), which
infects every call site, defeats method syntax, and is a measured driver
of output expansion (calibration fixture A1: 3.5× on an ordinary method).
Some corpus methods genuinely tolerate nil receivers (calibration fixture
B7) or perform observable work before the first dereference.

## Alternatives

- Blanket free-function lowering (baseline): rejected — pays the cost at
  every ordinary call for a property of rare methods.
- Blanket relaxation ("throw at call instead of at dereference"):
  rejected — changes observable panic timing and ordering; not exact.
- Runtime proxy/receiver-guard wrappers: rejected — dynamic mechanism,
  banned by the static-output contract.

## Decision

Exactness is the default; no blanket timing relaxation exists. A method
emits as an ordinary class method — with a call-site nil check only when
flow analysis cannot prove the receiver non-nil — exactly where static
analysis proves the check observationally equivalent to the first
receiver dereference: no effect, observable call, mutation, or
defer/recover boundary executes before that dereference; the Go panic
category and payload are preserved; argument evaluation order is
unchanged. Methods that tolerate nil receivers, or that perform effects
before dereferencing, take the exceptional free-function lowering,
recorded per method with a typed reason. Flow-proven non-nil receivers
call directly with no check.

## Effects

Ordinary methods are the default emission everywhere; the free-function
form becomes a per-method typed exception, never a family default. The
shape gate rejects free-function facades for methods without a recorded
exception. Call sites read as source-shaped TypeScript.

## Migration

The equivalence conditions are computed by the whole-program
effect/nilability analyses (numbered-order step 17) and consumed by the
lowering plan (step 18). The baseline free-function protocol is deleted
at the source-shaped lowering cutover; no compatibility path remains.

## Evidence

Calibration fixtures A1 (ordinary method, hand port 1.0×, baseline 3.5×),
B7 (nil-tolerant receiver: legitimate exception), and the neutral
controls `calibration/neutral/value_receiver_copy.go` and
`calibration/neutral/typed_nil_interface.go`; shape-gate detector
`DetectFreeFunctionFacade` in `internal/shapegate`.

## Reconsider when

A corpus method exists where the proof conditions are statically
undecidable and neither lowering form preserves exact panic behavior, or
differential execution finds a panic-timing divergence in a proven site.
