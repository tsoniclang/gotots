# ADR 0004: Closed Discriminated Interface Representation

Date: 2026-07-16
Status: accepted
Owner: gotots-maintainers
Implementation revision: `e04d202`
Schema/ABI impact: `abi-v16`
Spec reference: `docs/spec/06-interfaces-generics-functions.md`
Registry entry: `docs/decisions/registry.json#ADR-0004`

## Context

The interface carrier held an erased payload (`GoIfaceBox.v: unknown`)
recovered by casts at every dispatch, assertion, and equality site. The
specification (06-interfaces-generics-functions.md, Dynamic Type
Identity) requires a closed, statically typed payload with no erased
recovery, exact value/pointer method sets, exhaustive statically typed
branches, exact typed nil, and complete external participation. Two
prior mechanisms were removed for cause: the `Record<string, Function>`
method table (name-selected dispatch) and whole-universe token switches
at call sites (direct calls to every implementer added artificial ESM
import edges, executing package initializers absent from Go's
initialization graph and risking import cycles).

## Alternatives

### A. Direct finite per-interface unions with call-site dispatch

`type I$union = undefined | { k: "pkg.A"; v: A } | { k: "*pkg.B"; v: B }`
with each dispatch site switching on `k` and calling `A$M` directly.
Exact typing, but every dispatch site must value-import every
implementer's module: the artificial import-edge and cycle class the
audit rejected (non-Go package initialization, ESM TDZ hazards on
interned rtti constants). Rejected.

### B. Interface-specific generated dispatch modules

Per interface, one generated module holding `I$M(box, args)` dispatch
functions; call sites import only that module. Centralizes but does not
remove the artificial edges: the interface module value-imports every
implementer, so any caller pulls every implementer's initializer — the
same non-Go initialization graph, plus a placement problem for
anonymous and cross-package interfaces. Rejected.

## Decision

### C. Discriminated union with per-type typed vtables (selected)

The box is a member of a closed discriminated union:

```ts
{ k: "pkg.A"; r: typeof A$rtti; v: A; m: typeof A$vtable }
```

- `k` is a string-literal discriminant; switching on it narrows `v` and
  `m` to the member's exact types — no cast, no erased recovery.
- `A$vtable` is one shared `const` in A's OWN package: exactly typed
  adapters (value and pointer flavors separately) that perform receiver
  deref/copy and promotion chains internally. Constructed where the
  concrete type is statically known; no per-box allocation.
- Box construction sites already import the concrete type's module, so
  passing the vtable adds no import edge. Dispatch sites reference only
  string literals and the box — no value imports of implementers; union
  alias spelling uses `import type` (erased, no runtime edge).
- External types join through vtables built over their typed stub
  exports at box sites; their comparability stays fail-closed.
- Typed nil: the nil interface is `undefined`; a typed-nil pointer boxes
  with its pointer token and `v: undefined`.

Property access on a typed record after literal narrowing is ordinary
statically typed TypeScript — not name-selected dispatch (no string
index, no `Function`, no erased table). This is specification candidate
3 (typed box/adapter carrying dynamic identity) with the union-typed
payload the specification's example shape requires.

## Effects

- The ABI's interface carrier becomes the discriminated union member
  shape; the erased `v: unknown` field and all dispatch-site recovery
  casts are removed.
- Every concrete type with a method set gains generated value and
  pointer vtable consts in its own package.
- Box construction, dispatch, assertion, equality, method values,
  promoted methods, panic(error), and external variants migrate to the
  narrowed-union forms.
- Gates 08 and 10 unblock when the emitted output matches this shape.

## Migration

Single-step replacement on the active feature branch: the carrier and
every consumer change together (no compatibility path, no dual
representation), validated by the differential-fixture suite under the
strict per-fixture typecheck, the symbol-resolved staticness verifier,
and the full acceptance gate.

## Evidence

- contracts/necessity-records.json `interface-box` (counterexample,
  alternatives, oracle and mutation tests).
- internal/staticness/astverify.go (erased-callee and computed-member
  rejection; mechanism discovery by resolved symbol).
- The fixture suite's interface oracles (method sets, equality,
  assertions, typed nil, generic instantiation) under per-fixture tsc.

## Costs (measured at acceptance)

- Allocation: one 4-field box per conversion (previously 2-field);
  vtables are shared consts, not per-box.
- Imports: zero new runtime edges; type-only imports for alias spelling.
- Bundle: one vtable const per concrete type, proportional to its
  method-set size.
- Dispatch: literal switch plus direct property call; no member lookup.

The necessity record `interface-box` in contracts/necessity-records.json
carries the counterexample, alternatives, oracle/mutation evidence, and
reopening conditions; gates 08 and 10 stay blocked until the emitted
output matches this decision (no `v: unknown` carrier remains).

## Reconsider when

- TypeScript gains nominal discrimination over object identity tokens
  (removing the need for the literal `k`).
- The measured vtable bundle or allocation cost exceeds the committed
  performance budget for the selected product.
- The selected program's dynamic-type sets become statically closed per
  region (direct-value representation everywhere).
