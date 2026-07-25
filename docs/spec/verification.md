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
   organization bound.

## Construct Coverage

Coverage has two distinct proofs:

- a toolchain-derived catalog of Go AST forms/tokens/built-ins is reconciled
  against parent-consumed, handler, metadata, or explicit-unsupported
  dispositions for every relevant parent-field-child role; and
- every dispatchable selected-project occurrence is observed entering exactly
  one handler, while parent-consumed syntax enters none.

The catalog is test authority only. Compilation does not produce or consume a
per-program inventory.

Mutation tests remove a handler, skip a child edge, alter child order, erase a
context role, and route one form to two handlers. Each mutation must fail with
the exact source occurrence or catalog form.

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
6. Execute the same behavior through Go and the authoritative generated-code
   consumer and compare observable results.
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
- Go-versus-authoritative-consumer differential execution;
- interaction cases for evaluation order, aliasing, copies, nil, panic,
  generics, method selection, and environment boundaries where applicable;
- real-project samples unrelated to the discovering corpus; and
- production-path mutations that prove the gate detects the intended defect.

Examples include:

- one-result versus comma-ok map indexing;
- one RHS producing multiple assignment results;
- side effects in parallel assignment and short-circuit expressions;
- nil pointer receiver methods that do and do not dereference;
- embedded methods whose Go static selection differs from TypeScript virtual
  dispatch;
- interfaces with many implementers, proving constant-size call sites;
- nested function literals requiring branch-local versus file-level placement.

### Execution Authority

Strict TS-Go resolution and typechecking is mandatory for every generated
artifact. Behavioral execution then follows the owner of the represented
semantics:

- execute generated TypeScript directly when the selected Tsonic source
  carrier and operation have identical JavaScript runtime behavior, such as
  ordinary booleans and strings;
- compile through the selected Tsonic target and execute the target artifact
  when finalized Tsonic facts own behavior not represented by the JavaScript
  carrier, such as fixed-width integer overflow; and
- run both where both are authoritative for the tested domain.

The target-native path consumes the exact TS-Go-printed artifact. It may not
rewrite literals, inject facts, or substitute a test-only declaration shape.
For example, if the real Tsonic virtual declaration exposes `int64` with a
`number` checker carrier and an `int64` target fact, a test declaring
`int64 = bigint` is a false contract and must fail the contract gate. A Node
check over small integer values is useful corroboration but does not prove
width, overflow, shifts, conversions, or exact large constants.

Each checkpoint records the exact Tsonic, source-semantics provider, target
plugin, and target-toolchain identities used by target-native proof. If the
authoritative consumer is unavailable, the affected semantic capability
remains unclosed rather than being certified by a weaker runtime.

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
6. strict typechecking and authoritative-consumer differential execution;
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
required direct and target-native differential/project suites green, no
reachable placeholder, exact deterministic regeneration, and all
architecture/size/typecheck/runtime gates passing at one clean revision.
