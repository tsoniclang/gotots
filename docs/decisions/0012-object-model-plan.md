# ADR 0012: Recover The Object Model — Native Classes, Not Go's Workaround

Date: 2026-07-21
Status: accepted
Owner: gotots-maintainers
Decision revision: `acbd833`
Implementation status: planned
Registry entry: `docs/decisions/registry.json#ADR-0012`

## Context

Go has no inheritance, so TS-Go encodes an inheritance hierarchy with
embedded base structs (`NodeDefault{Node}`, `NodeBase{NodeDefault}`,
… `Identifier{PrimaryExpressionBase; FlowNodeBase; Text}`), a
self-reference field (`Node.data nodeData`) pointing at the outer
concrete node, a virtual-method contract interface (`nodeData`), and a
constructor (`newNode`) that ties the base to the concrete object.

Gotots currently reproduces the WORKAROUND mechanically rather than
recovering its MEANING:

- 265 AST classes, ZERO use `extends`;
- embedded bases become nested object fields with promoted forwarding
  chains (`this.PrimaryExpressionBase.MemberExpressionBase.….Node.
  Arguments(...)`);
- `nodeData` becomes a boxed union with a per-concrete vtable and, at
  every `n.data.Method()` call, an exhaustive switch over all 213
  implementers (`ast.Node.Name` is 21,813 bytes / 277×);
- `Identifier` alone is 81,360 bytes with 270 forwarding methods, plus
  an 83,785-byte `Identifier$vtablePtr`.

The measured interface-dispatch tail (p95 6.26×, max 277×) is entirely
this class. A "uniform vtable" only shrinks the switch while leaving the
wrong composition hierarchy, the giant delegates, and the giant vtables
intact. ADR-0004's inline-token-switch dispatch is superseded here for
types that form a native object model.

## Alternatives

- Inline exhaustive-switch dispatch (ADR-0004, current): static and
  erasure-free but O(implementers) per call — the tail.
- Uniform vtable with erased payload: O(1) but recovers the payload
  from `unknown` (guardrail violation) and keeps the wrong hierarchy.
- Per-box bound closures: O(1) but per-conversion allocation.
- **Recover the object model (this ADR)**: prove the inheritance spine
  the Go composition emulates and emit native TypeScript classes with
  `extends` and virtual methods.

## Decision

An atomic whole-program `ObjectModelPlan` proves, per named struct type:

1. the PRIMARY base — the unique embedded field whose type transitively
   reaches the family root that owns the virtual contract; the spine is
   single-inheritance and emits as `extends`;
2. SECONDARY embedded components — every other embedded field; kept as
   owned subobject fields (preserving Go's embedded-subobject identity)
   with forwarding accessors, or flattened only when identity is proven
   unobservable;
3. INHERITED vs OVERRIDDEN methods — a method the concrete type does not
   redefine is inherited; a redefinition is `override`;
4. construction and base-object identity — the self-reference
   (`data`) collapses because data and behavior are one object;
5. the type-assertion / type-switch strategy over the class hierarchy
   (`instanceof` / discriminant), erasure-free;
6. receiver nil/value semantics (ADR-0006 unchanged).

`n.data.Method()` then emits as native virtual dispatch `n.Method()` —
one line, O(1), independent of implementer count. No box, no
per-concrete vtable, no promoted forwarding chains, no `unknown`, no
reflection or name-based dispatch.

Types that do NOT form an object-model family keep their current
representation; the plan is proven per family, never assumed.

## Effects

The interface-dispatch tail collapses (277× → ~1×); `Identifier` and
its vtable shrink from ~165 KB to a small class; promoted forwarding
methods for spine bases disappear. A hard gate requires that an
ordinary virtual/interface call emits O(1) code independent of
implementer count.

## Migration

The ObjectModelPlan is a new sealed whole-program analysis
(numbered-order step 17) consumed by the class/method/dispatch
emitters (steps 19-20). Struct emission gains `extends`; the box/vtable
inline-switch path is deleted for object-model families in the same
checkpoint. ADR-0004 remains for non-family interface unions.

## Evidence

`internal/ast/ast.go` embedding spine; measured AST output at
`dump-acbd8331…` (265 classes, 0 extends, Identifier 81,360 B / 270
methods, Node.Name 21,813 B / 213 arms); `internal/objectmodel`
analysis with its differential and O(1)-dispatch gate.

## Reconsider when

A family's embedding graph has no single primary spine (genuine
multiple inheritance with observable diamonds) that the plan cannot
express; such a family keeps union dispatch with a recorded reason.
