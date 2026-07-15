# Representation Planning And Output

## Governing Rule

Complete Go semantic requirements are computed before storage selection.
Planning chooses the simplest ordinary TypeScript representation satisfying
every requirement in the closed value-flow region.

The compiler does not instantiate a universal exact carrier and then optimize
it away. Exact carriers are emitted only when selected execution requires
first-class state that direct storage or local lowering cannot preserve.

## Region Formation

Regions are connected through:

- assignment and initialization;
- parameter, result, and receiver flow;
- closure capture;
- fields and container elements where representation must agree;
- aliases and backing/storage identity;
- package variables;
- function values and callbacks;
- interfaces and generic instantiations;
- external, manual, and extension boundaries; and
- concurrency sharing.

Conversions split regions only when they are explicit, exact, and represented
in the plan.

## Requirement Product And Candidate Selection

The monotonic lattice is the product of independent semantic requirements, not
a forced ordering of emitted forms. Examples include slice nilability,
mutability, shared-view state, capacity observation, and escape; pointer
root/slot identity and lifetime; interface dynamic identity, typed nil,
dispatch, equality, and hash; and string byte validity and complexity.

Each emitted candidate declares:

- the requirement sets it satisfies;
- preconditions and invalidation edges;
- legal typed conversions;
- boundary ABI;
- runtime/allocation/code-size cost; and
- deterministic tie-break key.

A partial order is recorded only where one candidate genuinely subsumes
another. Incomparable satisfying candidates are chosen by the committed static
cost order. If none satisfies the fixed-point requirement set, the class is
unimplemented. Manual ownership is not a representation candidate.

## Fixed-Point Algorithm

For each region:

1. seed requirements from semantic type and construction;
2. add requirements from every operation;
3. propagate through graph edges;
4. apply conservative boundary effects;
5. repeat in canonical order until stable;
6. select the deterministic least-cost candidate satisfying the stable set;
7. verify selection independently; and
8. lower after all connected regions are stable.

The implementation uses no LLM or per-call-site human classification. Sites are
assigned mechanically to typed operation classes.

## Unknown And Contradictory Evidence

An unknown effect may select a conservative existing representation only when
that representation is already proven exact for every behavior the unknown
edge could perform. It cannot compensate for an unsupported language
operation.

Unsupported semantic behavior becomes unimplemented. Contradictory evidence
blocks and reports both sources. Neither case emits a guessed artifact.

## Custom-Mechanism Necessity

Mandatory semantic IR and operation records are not custom output mechanisms.
Neither is a direct generated declaration, a stateless local expression, or a
helper that merely factors an already accepted operation without adding state,
indirection, dispatch, or ABI.

A custom mechanism is runtime-visible generated state or dispatch beyond
ordinary direct TypeScript: a carrier, storage-location object, specialized
hash table, interface box/dispatch table, scheduler, stateful helper runtime,
or representation-changing boundary adapter. These require necessity proof.

Before adding a custom mechanism, one class-level record must prove:

1. a reachable selected operation observes the missing distinction;
2. ordinary TypeScript fails on a concrete Go example;
3. local rewriting, scalar expansion, direct native storage, specialization,
   and static conversion are insufficient;
4. accepted complete manual ownership is less appropriate for the class;
5. the proposed mechanism is the smallest exact form;
6. Go differential and mutation tests cover the distinction;
7. allocation, runtime, memory, code-size, and startup costs are measured; and
8. invalidation dependencies are complete.

The site list is generated from semantic class membership. Humans review the
class and mechanism once.

## Boundary Planning

Export, package, external, manual, extension, interface, generic, and
concurrency edges do not automatically require a carrier. Their typed effect
contracts contribute constraints like any other edge.

A direct representation may cross a boundary when the contract proves it
cannot observe omitted semantics. An unknown open edge is conservative or
unimplemented.

Boundary conversion cannot silently clone, erase nil, change dynamic type,
alter storage identity, or normalize bytes. Its cost and semantics are explicit.

## Direct Output Examples

Simple slice construction:

```go
values := []int{1, 2}
values = append(values, 3)
```

may emit:

```ts
let values: int[] = [1, 2];
values.push(3);
```

An aliased reslice may instead emit one backing array and local offset/length
state. It needs an escaping view only if that state flows as a first-class
value.

An ordinary struct emits direct fields. Copies are inserted at Go copy
boundaries rather than wrapping every value in a copy-capable carrier.

## Static TypeScript Contract

Generated source is strict ESM and contains:

- explicit `.js` import specifiers;
- statically selected declarations and calls;
- typed parameters, results, fields, container elements, and boundary
  contracts;
- deterministic names and imports; and
- evidence-backed assertions only where TypeScript cannot express a relation
  already proven by Go.

Generated core contains no:

- CommonJS;
- triple-slash references;
- unchecked `any` or `unknown` recovery;
- reflection or source-name dispatch;
- dynamic property names for statically selected fields/methods;
- `eval`, generated functions, or proxies;
- host-shape or emitted-representation inference; or
- fallback import or implementation selection.

Runtime checks of explicitly planned Go semantic state—such as a canonical
interface dynamic-type token, channel close state, or Go panic brand—are
permitted. They are statically selected language operations, not runtime
inference of which generated representation happens to be present.

## Evidence-Backed Assertions

An assertion may encode a relationship already established by the typed Go
frontend and verified lowering plan, such as introducing a compile-time brand
at an explicit defined-type conversion. Every assertion class has a canonical
code, source evidence, and independent checker.

Assertions cannot recover missing declaration identity, bypass a failed
constraint, or widen an unimplemented operation.

## Generated Naming

Names preserve source spelling when it is a valid collision-free TypeScript
identifier. Otherwise deterministic allocation derives from canonical identity
and scope. Generated helper names describe semantic purpose.

Case conversion is not applied as a target-friendly API policy. Import/export
identity and operation selection never depend on generated spelling.

## Module And File Layout

Generated layout is deterministic from package/declaration identity and
dependency SCCs. Files split at semantic declaration boundaries. Imports are
derived structurally from generated AST dependencies.

One generated declaration is not semantically distorted solely to meet a
review-size target. Hand-maintained source and normative documentation obey the
repository limit.

## Source Maps And Provenance

Every generated declaration and operation maps to canonical Go identity and
source span. Synthetic temporaries and helpers map to the semantic operation
that required them.

Per-body records include source semantic hash, IR hash, plan hash, canonical
generated AST hash, emitted file, and source-map identity. Formatting does not
change canonical AST identity.

## Deterministic Emission

The emitter prints a typed AST using canonical formatting. Import ordering,
declaration ordering, helper allocation, string escaping, numeric literals,
source maps, and manifest serialization are deterministic.

Emission cannot read an accepted output tree. Inputs are the pin, profile,
generated semantic artifacts, accepted manual bodies, external contracts, and
extension seam data.

## Implementation-Permitted Behavior

Go permits variation in areas such as map iteration, scheduler choices, and
allocation growth. GoToTS either:

- accepts the complete permitted outcome set in tests; or
- records and tests one deliberate deterministic policy that remains within
  that set.

It does not accidentally inherit host behavior without classifying it.

## Representation Report

Each run reports counts and identities for:

- direct and escalated forms by semantic family;
- proof-driven copy/bounds/pointer/interface eliminations;
- specializations and shared generic bodies;
- custom mechanisms and their site membership;
- manual and unimplemented units;
- boundary conversions; and
- optimization invalidations.

The report is evidence, not semantic input.
