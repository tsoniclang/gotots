# Verification

## Proof Principle

No single test proves translation. Each construct family is accepted only when
source coverage, target structure, strict typing, behavior, and cost agree at
one exact revision.

Verification must be independent at the compared boundary. A checker using the
same handler or canonicalizer as the producer is a consistency check, not
independent proof.

## Baseline Gates

The repository always enforces:

1. `AGENTS.md` and `CLAUDE.md` are byte-identical;
2. the pinned TS-Go binary, revision, schema, protocol version, encoder
   contract, and generated Go bindings agree;
3. only allowed packages import `go/ast`, `go/types`, and target factories;
4. no production package named or acting as `ir`, `plan`, `lower`, `catalog`,
   `inventory`, `legacy`, `compat`, or `fallback` exists;
5. production emission contains no raw TypeScript fragments, local formatter,
   alternate target tree, or TS-Go `internal` import/fork;
6. dynamic import, `.call`, `.apply`, `.bind`, `any`/`unknown` recovery,
   reflection, spelling dispatch, and source-text semantic scans are absent;
7. maintained non-generated files stay below 600 physical lines; and
8. semantic handlers follow the recursive domain/owner structure, root emitter
   files contain orchestration only, and no maintained directory exceeds the
   organization bound;
9. generated support/runtime imports resolve inside the generated product or
   an explicit Go environment contract, and no unrelated compiler, transpiler,
   target, or product appears as a semantic or verification dependency.

## Construct Coverage

Coverage has two distinct proofs:

- an independent test derives the complete selected-toolchain `go/ast`,
  `go/token`, and `types.Universe` domains at runtime, exact-joins every
  production dispatch case to that universe, fingerprints exact AST field
  shapes/token values/predeclared contracts, and proves each category has one
  typed-unsupported default; and
- each implemented handler's focused tests account for its complete direct
  child contract, roles, order, contextual variants, and parent-consumed
  syntax.

There is no checked-in disposition registry, case manifest, or per-program
inventory. Removing a handler case is caught by its construct artifact tests;
adding or duplicating a case is caught by the independent universe join and Go
typechecking; a newly selected-toolchain AST form reaches the typed default
until its first construct case is proved.

Mutation tests remove a handler, skip a child edge, alter child order, erase a
context role, and route one form to two handlers. Each mutation must fail with
the exact source occurrence or selected-toolchain form.

An AST kind is not the unit of semantic completion. A construct case includes
the form, parent role, type evidence, expected result shape, and evaluation
context. These assignments are separate cases:

```go
a = b
a, b = b, a
value, ok = values[key]
a, b = pair()
values[i], i = next, i+1
*pointer = value
a += b
```

Support for one case does not admit the others. Encountering an unproved
contextual variant returns a typed unsupported diagnostic.

Each handler's child contract has an independent conservation proof: every
direct field is accounted for as owner-consumed, delegated, absent, metadata,
or impossible; every delegated child has the expected role and order; and no
forbidden child or automatic descendant enters emission. Mutations omit and
duplicate a delegated child, mislabel owner-consumed syntax, change a role or
category, reorder siblings, admit an impossible child, and replace owner
traversal with generic recursion.

Invalid Go syntax is a loader/parser failure, not a valid red translation
test. Valid Go that reaches an unsupported contextual case must fail at the
typed unsupported boundary. Mutations admit a parser-recovery `BadExpr`,
`BadStmt`, and `BadDecl`; each must fail before dispatch.

## Test-First Construct Expansion

Every construct case follows this sequence:

1. Add the smallest Go fixture, intended TypeScript artifact, and focused typed
   Go test beside the semantic owner.
2. Run that exact test before production support. It must fail at the owning
   typed unsupported boundary. A parser, loader, target client, crash, or
   unrelated failure is not an acceptable red result.
3. Add or extend the one semantic-owner handler and its closed child contract.
4. Assert the constructed typed TS-Go protocol tree and child roles directly.
5. Encode it, print only through pinned TS-Go, reparse, compare normalized
   target structure, and strict-typecheck.
6. Compile the generated TypeScript to ESM JavaScript, execute the same behavior
   through Go and the generated program, and compare observable results.
7. Add boundary and interaction cases required by the semantic rule.
8. Mutate the production decision or child route and prove the owning gate
   fails.
9. Inspect the generated artifact and applicable source, generated,
   typecheck, generation, and runtime costs.

A case is supported only when this sequence is green at one revision. New
project coverage remains incremental: first reproduce a new unsupported case,
then extend its shared owner. Do not pre-create speculative handlers,
directories, fixtures, or support dispositions.

Foundation capabilities are also test-first. The bootstrap order is: pinned
TS-Go tool/contract test, generated binding/encoder test, native `printNode`
round-trip test, loader coherence test, unsupported-dispatch test, then the
first construct case.

The demand-driven scheduler is verified independently from final target
declarations and source bindings: every selected root and resolved emitted
reference reaches exactly one target declaration or explicit obligation.
Mutations omit an enqueue, duplicate an owner, break a cycle reservation, and
silently drop a function-value/interface/callback target.
Multi-package scheduler proof also includes checkout relocation, root-order
reversal, recursive cycles, unreachable declarations, cross-package/local-name
collisions, direct tuple flow, and nested control flow. Generated module paths,
encoded TS-Go source files, reachable declaration sets, strict typechecking,
and Go-versus-generated-program behavior must remain exact.

## Native TS-Go Target Proof

For every generated file:

1. every target value is constructed through node-specific bindings generated
   from the exact pinned official TS-Go schema;
2. compile-time child types and generated factory tests reject invalid shapes;
3. the generated Go encoder produces the exact pinned protocol version;
4. the pinned `tsgo --api` process accepts the binary value, decodes it through
   its real factory, and prints it through its real printer;
5. TS-Go reparses the printed file;
6. normalized schema-level structure matches the constructed value;
7. source mappings and declaration ownership reconcile; and
8. strict TypeScript resolution/typechecking succeeds.

The Go encoder is compared against the pinned upstream TypeScript encoder over
the same representative target graphs, including optional children, lists,
tokens, literals, declarations, expressions, and source files. Exact encoded
bytes must agree unless the pinned protocol explicitly permits alternatives.

Required-child validation follows the concrete pinned TS-Go variant, not only
the shared schema member. The pinned `CaseOrDefaultClause` schema exposes an
`Expression` member for both discriminants, while the pinned parser constructs
`DefaultClause` with that child absent and the pinned protocol encoder records
the absent-child mask. The generated Go encoder must do the same; it must not
fabricate an expression merely to satisfy the shared member. Every such
discriminant-specific absence requires its own exact-byte differential against
the pinned upstream factory and encoder. It is not a general optional-child
escape hatch.

Mutations change a schema digest, protocol version, node kind, field shape,
child order, optional-child mask, encoded bytes, TS-Go binary revision,
declaration ownership, or formatted text. They also introduce a generic target
node, local formatter, raw fragment, second printer, and post-format edit. The
owning gate must reject each.

## Semantic Proof

Every implemented semantic family includes:

- focused source fixtures for ordinary and context-sensitive forms;
- boundary/property cases;
- Go-versus-generated-ESM differential execution;
- interaction cases for evaluation order, aliasing, copies, nil, panic,
  generics, method selection, and environment boundaries where applicable;
- real-project samples unrelated to the discovering corpus; and
- production-path mutations that prove the gate detects the intended defect.

Examples include:

- one-result versus comma-ok map indexing;
- one RHS producing multiple assignment results;
- direct tuple declaration/return/forwarding, blank-result stores, and one
  multi-valued call supplying a complete argument list;
- side effects in parallel assignment and short-circuit expressions;
- nil pointer receiver methods that do and do not dereference;
- embedded methods whose Go static selection differs from TypeScript virtual
  dispatch;
- interfaces with many implementers, proving constant-size call sites;
- nested function literals requiring branch-local versus file-level placement.

The first named-struct family additionally proves:

- two field-identical named Go structs remain incompatible in strict
  TypeScript through an erased nominal brand;
- zero values allocate fresh nested records;
- assignment, initialization, arguments, results, and value receivers do not
  alias mutable struct storage;
- borrowed values are copied once at each admitted boundary while fresh
  composite/call results and single-result returns transfer ownership without
  a duplicate field walk;
- `$assign` preserves destination storage identity while copying every nested
  value field;
- keyed composite literals retain source evaluation order even when field
  declaration order differs;
- equality is field-wise and recursive rather than target object identity;
- concrete value-receiver calls use the exact `go/types.Selection` and a named
  receiver function, never a class method or virtual dispatch; and
- tags, embedding, pointers, interfaces, method values/expressions, generics,
  and unsupported field representations fail at their typed owner.

Mutations replace `$copy` with direct assignment, `$assign` with target
rebinding, `$equal` with `===`, remove the private brand, reorder keyed
initializers, attach a receiver method to the class, or admit an unsupported
field. Each fails its owning structural, strict-type, differential, or
unsupported-boundary gate. A scaling fixture doubles fields and proves
definition size/work grows linearly while copy, assignment, equality, and
method call sites remain constant-size. An ownership mutation adds a callee
prologue copy or copies a fresh composite/call result; artifact and operation
counts must reject the duplicate boundary work.

### Execution Authority

Strict TS-Go resolution and typechecking is mandatory for every generated
artifact. The exact TS-Go-printed artifact and every requested GoToTS-owned
support/runtime module are then compiled to ESM JavaScript and executed
directly. The generated product may not rewrite literals after printing, inject
test-only facts, or substitute a different declaration shape.
The pinned compiler may report diagnostics without a failing process status.
The one compiler-process owner therefore treats either a process error or any
diagnostic output as failure; tests and product gates must not invoke the
executable through a status-only wrapper.

Primitive aliases preserve selected Go names in target source but do not prove
runtime range or arithmetic. For example, `int64 = number` does not establish
64-bit precision, overflow, shifts, conversions, or exact large constants. A
small-value runtime test cannot certify the full class. The capability closes
only when direct standalone behavior is exact for the complete admitted domain
or a GoToTS-owned generated/runtime operation implements and differentially
proves that behavior. Otherwise the construct remains typed unsupported.

The signed-integer gate differentially exercises every admitted `int32`
operator at `math.MinInt32`, `math.MaxInt32`, and overflow-producing operands,
and inspects the exact wrapped TS-Go AST. It separately proves that `int64` and
64-bit `int` arithmetic, comparison, compound update, increment/decrement, and
switch fail at their contextual semantic owner. Mutations remove `| 0`,
replace `Math.imul` with ordinary multiplication, admit a wide operation, or
round a wide constant; each must fail its owning differential or unsupported
boundary.

Each checkpoint records the exact Go toolchain, pinned TS-Go revision/schema,
JavaScript runtime, and GoToTS support/runtime revision used by proof. No
external transpiler, target compiler, plugin, or unrelated product may supply
missing semantics or verification.

## Architecture And Cost Proof

Each checkpoint reports absolute values and parent deltas for:

- maintained source files/lines;
- emitted bytes, tokens, typed protocol nodes, and TS-Go-decoded AST nodes;
- largest twenty emitted declarations, expressions, and files;
- definitions, helpers, imports, placeholders, and ownership collisions;
- strict typecheck wall time and peak RSS;
- generation wall time and peak RSS;
- representative runtime wall time and memory;
- dynamic/staticness violations.

Aggregate improvements cannot hide a worsening tail. A one-line Go interface
call expanding into hundreds of target branches is a failed architecture even
if median output is small.

Scaling fixtures distinguish source-proportional work from growth by unrelated
package, declaration, or implementer count. Raising a threshold, adding an
allowlist, or suppressing a diagnostic is not a fix.

## Per-Capability Closure

Before dependent work begins, a capability must have:

1. a reviewed governing rule and concrete Go/TypeScript examples;
2. a focused test observed failing at its owning unsupported or missing
   foundation boundary;
3. one authoritative handler/owner;
4. positive, negative, interaction, and mutation tests;
5. native TS-Go construction/print/reparse proof;
6. strict typechecking and Go-versus-generated-ESM differential execution;
7. artifact-tail inspection and applicable cost bounds;
8. broad searches proving no alternate route remains;
9. a clean pushed revision carrying all evidence.

Unsupported neighboring constructs may remain explicit failures. A capability
is not accepted if it silently falls back, emits a placeholder while claiming
translation, or defers its own proof to a later milestone.

## Heavy Runs

Heavy tests run one process group at a time with:

- bounded worker concurrency;
- a timeout;
- disk-backed output and logs;
- current-stage and process breadcrumbs;
- an OS-enforced memory ceiling; and
- preserved crash/OOM/timeout evidence.

Up to 4 GiB may be used when justified. WCBUBWHB concerns architectural
cleanliness; memory limits are execution safety, not a substitute for design.

## Publication

Publication requires all selected constructs and reachable environment
obligations to be implemented, every generated module strict-typechecked, all
required direct differential/project suites green, no
reachable placeholder, exact deterministic regeneration, and all
architecture/size/typecheck/runtime gates passing at one clean revision.
