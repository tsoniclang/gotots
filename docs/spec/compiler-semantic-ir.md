# Compiler Architecture And Semantic IR

## Required Pipeline

The compiler has these ordered layers:

1. attested typed Go frontend;
2. selected-universe and declaration graph;
3. semantic body IR;
4. operation and effect summaries;
5. value-flow, alias, escape, address, and call graphs;
6. representation constraints;
7. deterministic fixed-point solution;
8. lowering plan;
9. typed TypeScript AST;
10. static assembly and staged publication; and
11. independent verification.

Later layers consume identities and decisions from earlier layers. Emission
cannot query source text, repeat type selection, infer an operation from a
name, or repair missing semantic evidence.

## Typed Go Frontend

The frontend uses the pinned Go parser, type checker, package loader, constant
model, selection data, instantiation data, and build semantics. It provides:

- exact syntax and source spans;
- package and object identity;
- aliases, defined types, underlying types, and method sets;
- constant values and representability;
- expression types and values;
- selections and implicit address/dereference adjustments;
- generic origin, type parameters, constraints, and instantiations;
- builtin and compiler-intrinsic identity;
- addressability, assignability, convertibility, and comparability; and
- selected source files for the profile.

GoToTS never reconstructs these facts from spelling or TypeScript output.

## Declaration Graph

Every package, file, declaration, receiver, field, method, parameter, result,
type parameter, constant, variable, initializer, and implementation receives a
canonical ID. Edges record semantic ownership and dependencies.

The graph retains:

- declaration order where Go makes it observable;
- embedded field and promoted selection paths;
- generic origin and instantiated identity;
- package initialization dependencies;
- external declaration references;
- manual-body and extension-seam ownership; and
- selected test ownership.

Generated TypeScript names are allocations attached to canonical IDs. They are
never semantic identity.

## Semantic Body IR

The body IR describes Go semantics, not JavaScript syntax. It has typed nodes
for at least:

- constants, variables, storage locations, loads, and stores;
- field, index, slice, map, pointer, and interface operations;
- calls, conversions, builtins, method values, and function values;
- tuple and multiple-result evaluation;
- assignments with explicit evaluation and commit phases;
- branches, loops, switches, type switches, select, labels, and fallthrough;
- range over every selected operand class;
- returns and named results;
- defer, panic, recover, goroutine, send, receive, and close;
- copies, zero construction, equality, hashing, and bounds checks;
- package initialization; and
- compiler-recognized intrinsics.

IR construction either represents the complete selected operation or emits one
categorized unimplemented record. It never substitutes a superficially similar
host operation.

## Evaluation And Storage

The IR separates:

1. evaluation of operands in Go order;
2. capture of values and storage locations;
3. checks and possible panic points;
4. copy or conversion operations;
5. mutation commit; and
6. result production.

This separation covers parallel assignment, compound assignment, calls,
indexing, map operations, deferred calls, and panicking stores. A TypeScript
expression may combine phases only after the lowering plan proves identical
order and effects.

Storage identities include locals, package variables, fields, fixed-array and
slice-backing elements, pointer indirections, closure cells, and external
storage contracts. Address-taking refers to storage, never to a copied value.

## Semantic Type Descriptor

Every value and storage location references an immutable descriptor containing:

- canonical type identity, alias/defined status, and underlying type;
- numeric width, sign, and constant class;
- fields, array length, element/key/value/pointee types;
- nilability and zero behavior;
- value-copy class and addressability;
- comparability, equality, and hashing requirements;
- method set and interface dynamic-type obligations;
- function signature, receiver, variadic state, and results;
- generic origin, arguments, constraints, and permitted operations;
- channel direction; and
- external/manual/extension boundary requirements.

An emitted representation may omit a descriptor property only when the planner
proves selected execution cannot observe it. The descriptor remains available
for regeneration and verification.

## Effect Summaries

Each implementation has a machine-derived summary covering:

- reads and writes;
- allocation and copying;
- address-taking and pointer escape;
- slice/map mutation and retained aliases;
- interface boxing and dynamic operations;
- callback retention and unknown calls;
- panic, recover, defer, blocking, and goroutine creation;
- package-global effects;
- external calls; and
- extension seam interaction.

Generated bodies derive summaries from IR. Manual and external bodies supply
reviewed conservative contracts. Missing effects make the affected edge
unimplemented or conservatively exact; they never authorize optimization.

## Analysis Graphs

GoToTS builds deterministic graphs for:

- calls and possible function targets;
- value flow and representation regions;
- aliases and backing/storage identity;
- escape and lifetime;
- copy boundaries;
- interface dynamic types;
- generic instantiations;
- package initialization;
- blocking/concurrency effects; and
- boundary crossings.

Graph nodes and edges carry canonical IDs and source evidence. Graph
construction is complete before planning.

## Constraint Propagation

Each region begins with the least semantic requirements implied by its type and
construction. Operations add constraints. Constraints propagate through graph
edges until no record changes.

Examples:

- `cap(s)` adds capacity observability to the complete alias/call region of
  `s`;
- an escaping `&value.field` adds stable storage-location identity;
- interface comparison adds dynamic type, comparability, and typed-nil
  requirements;
- a generic equality operation propagates the equality strategy of every
  reachable instantiation;
- an unknown external retention effect prevents local pointer or slice-state
  elimination.

The solver is monotonic and deterministic. A requirement can only be retained
or strengthened. Contradictory constraints block with both provenances.

## Semantic Classes

Pre-planning operation classes use a stable key derived from language/profile
version, operation kind, builtin/intrinsic/declaration semantic identity where
behavior depends on it, instantiated operand/result shape, and effect contract.
They do not include observations discovered by representation planning.

Post-planning representation classes add the fixed-point requirement set and
selected boundary obligations. Two calls with identical TypeScript-looking
signatures but different Go retention, alias, panic, blocking, or callback
effects are different classes unless a proof establishes equivalence.

Both key schemas exclude source path, incidental package spelling, identifier
spelling, and traversal order. Source-upgrade diffs compare operation classes
before planning and representation classes after fixed-point convergence.

Implementation support and custom-mechanism review are class-level decisions.
Every site is assigned mechanically. An unclassified site creates an
unimplemented class rather than an ad hoc lowering.

## Representation Plan

For each closed region, the plan records:

- semantic descriptor and operation classes;
- value-flow membership;
- accumulated constraints;
- selected storage and static TypeScript shape;
- explicit boundary conversions;
- copy, zero, equality, hash, bounds, and panic operations;
- helper or specialization identities;
- proof dependencies and invalidation hashes;
- custom-mechanism necessity record, when applicable; and
- tests that cover the selection.

Emission consumes this plan without semantic rediscovery.

## Lowering And Emission

Lowering produces a typed TypeScript AST. It does not generate TypeScript text
and parse it back. Every assertion, temporary, helper, import, and control-flow
rewrite is represented structurally and tied to its source operation.

The emitter is deterministic and intentionally simple: allocate canonical
names, print the typed AST, generate source maps, and report structural hashes.
It cannot choose semantics.

## Independent Verification

The verifier does not trust planner summary flags. It checks:

- all selected units and operations have records;
- graph and region membership are exhaustive;
- fixed-point constraints agree with source operations;
- representation choices satisfy their predicates;
- custom mechanisms have accepted necessity evidence;
- unimplemented dependency closures emit no product artifact;
- generated AST operations agree with the plan; and
- manifests hash the exact staged files.

Shared generator and verifier code is limited to schema definitions and
canonical encoding primitives; decision computation is independently checked.
