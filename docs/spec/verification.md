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
   target non-null assertions, reflection, spelling dispatch, and source-text
   semantic scans are absent;
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
collisions, colliding portable encodings of distinct Unicode/ASCII package
names, direct tuple flow, and nested control flow. Generated module paths,
encoded TS-Go source files, reachable declaration sets, strict typechecking,
and Go-versus-generated-program behavior must remain exact.
An imported declaration used as both a type and a value must produce one value
import regardless of request order. Mutations that retain both imports, let the
type-only request dominate, or assign different local names must fail at the
placement or strict-type gate.

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

## Declaration Assembly Proof

Every use-dependent declaration requirement proves the complete open-to-sealed
lifecycle:

1. the initial declaration owns one typed TS-Go assembly keyed by its exact
   `types.Object`;
2. a later reached use requests a closed requirement without receiving the
   declaration node;
3. the root deduplicates the typed requirement and routes it to that
   declaration's semantic owner;
4. the owner reconstructs and replaces the complete in-memory declaration
   assembly through TS-Go factories;
5. requirements produced by that reconstruction re-enter the same fixed-point
   queue;
6. root order and duplicate request order produce byte-identical sealed files;
7. exactly one final definition exists and every requested addition is
   adjacent to or incorporated into its declared owner as specified; and
8. sealing rejects any later request.

The first non-vacuous proof uses named-struct static zero/copy/equality
operations:
the class is constructed before a reached assignment, argument, return, zero,
or equality site discovers the operation; the subsequent requirement rebuilds
the owner's assembly and the final artifact contains exactly the requested
static members in closed operation order. Calls use the exact statically
selected class, never an instance, and no top-level operation helper remains.
A nested struct operation must request its nested owner through the same queue.

Mutations route a requirement by spelling, apply it in the caller's file,
append a duplicate final definition, let a non-owner mutate the class, process
requirements in discovery order, seal with pending work, accept a request after
sealing, patch printed text, or retain a mutable on-disk AST. Each fails at the
identity, ownership, determinism, lifecycle, schema, or broad-search gate.
Scaling reports initial declarations, applied requirements, owner
reconstructions, final definitions, AST nodes, and bytes separately; moving
growth into repeated reconstruction does not hide it.

## Observable Contract Propagation Proof

The generic pre-seal artifact graph is accepted only with all of these
independent proofs:

1. canonical facet projections are built from typed TS-Go AST and encode
   identically for semantically identical signatures despite different bodies
   or explicitly typed initializers, while inference-dependent visible
   declarations fail closed;
2. changing one facet dirties exactly the consumers subscribed to that facet;
3. publishing an identical contract performs zero consumer reconstruction;
4. one-hop and transitive changes reach every affected consumer once in
   deterministic object order;
5. duplicate references and duplicate requests do not duplicate edges,
   requirements, imports, declarations, or reconstructions;
6. reconstruction atomically replaces AST roots, root requests, dependencies,
   and the contract, so dropped imports and dependencies do not survive;
7. a convergent dependency cycle seals, while a contract oscillation that
   repeats a previous non-current structural contract fails with the exact
   artifact and changed facets;
8. real named-struct static-operation discovery reconstructs the provider and
   any signature-relevant consumer, while a consumer whose own callable
   signature remains unchanged does not notify its callers; and
9. a body-only consumer with an explicit empty contract is reconstructed but
   cannot propagate further, while an absent contract fails closed; and
10. sealing rejects pending dirty artifacts or dependencies from an enclosing
   target owner that is not reconstructible.

Request-transport gates additionally prove that nested composition retains
exact atomic-leaf order, rejects an invalid leaf at any depth, exposes no
mutable child backing, and keeps the top-level carrier count bounded while the
leaf count doubles. Restoring eager transitive flattening must fail this
scaling mutation. Existing generated artifacts remain byte-identical, and
large-corpus profiles report request-composition allocation and peak-RSS
deltas separately.

The atomic request handle has a frozen pointer-scale size gate. Moving its
immutable payload fields back into every copied transport value must fail that
gate even when semantic tests remain green.

Mutations remove a dependency edge, widen it to all facets, compare only a
hash, retain full historical snapshots, notify on equal contracts, ignore a
changed contract, retain old dependencies, restore full-set dirty scans,
process dirty work nondeterministically, mutate a provider node through a
consumer, or accept an oscillating cycle. Each must fail at the contract,
graph, lifecycle, determinism, ownership, scaling, or broad-search gate. A
forced fingerprint-collision foil must remain unequal after exact historical
reconstruction. A growing-contract fixture must prove retained history tracks
exact changed regions rather than the sum of all prior full contracts, and a
fan-out fixture must bound dirty-owner comparisons by `O(n log n)`.
An adversarial fan-in fixture reverses Go-object order relative to provider
edges and proves one provider-before-consumer wave; restoring global object
order must reconstruct the consumer before a still-dirty provider and fail.
Another fixture discovers a requirement during an early reconstruction,
finishes the current dependency wave, then proves the requirement is applied
and its exact owner reconstructed in the next wave. Rebuilding the wave after
every discovered request must fail the wave-construction work bound.
Measure declarations, graph vertices/edges, contract bytes, revisions,
reconstructions, final AST bytes, generation time, typecheck time/RSS, and
runtime. Current graph state must remain O(artifacts + consumed facet edges +
current contract bytes + losslessly encoded exact historical changed regions);
convergence evidence retains compressed reverse deltas and no entry for an
unchanged rebuild.
Use-site count may add deduplicated edges but must not duplicate provider
contracts.

## Semantic Proof

Every implemented semantic family includes:

- focused source fixtures for ordinary and context-sensitive forms;
- boundary/property cases;
- Go-versus-generated-ESM differential execution;
- interaction cases for evaluation order, aliasing, copies, nil, panic,
  generics, method selection, and environment boundaries where applicable;
- real-project samples unrelated to the discovering corpus; and
- production-path mutations that prove the gate detects the intended defect.

Assignment-boundary proof covers every value-transfer consumer: direct and
captured arguments, single and multiple results, local and package
declarations, parallel assignment, struct/array/slice/map elements, channel
send/receive, and generated generic operation calls. The matrix includes each
defined reference or aggregate family in both legal directions between a
defined and an unnamed identical-underlying value, defined basics through
their legal untyped-constant assignment contexts, and two distinct defined
types that `go/types` rejects.

The canonical mutation replaces the source projection in:

```go
type Value []byte
func consume([]byte)
func use(value Value) { consume(value) }
```

with the wrapper expression itself. Strict typechecking must fail at the call
boundary. A second mutation omits destination construction for the inverse
assignment and must fail at that boundary. Differential execution proves
slice aliasing/capacity, map identity, pointer identity, channel identity,
callable invocation, primitive value, and array-copy behavior rather than
merely accepting the target shape. Broad search must find no target-only
`Copy` API and no handler-owned defined-family transfer branch.

Structured-loop certification additionally covers prerequisite-bearing
conditions, multi-result posts, blocking conditions/posts under the cooperative
profile, normal and labeled continue, break, return, and panic. Mutations
restore a synthetic condition/post callback, run post on the first iteration,
skip post after continue, or run post after break/return/panic; each fails the
strict-type, target-shape, or differential owner.

The constant-context family additionally proves:

- `return Scale`, `Scale + Scale`, assignment, argument, case, conversion,
  package selector, and dot-import contexts select the exact parent-owned or
  enclosing-expression checker fact;
- a named untyped float constant used at `float32` and `float64` proves that
  contextual occurrence rounding is not mistaken for declaration-value drift;
- sibling and nested local `const Width` declarations stay in their lexical
  blocks and produce no duplicate target binding;
- two packages exporting the same constant spelling import distinct
  `(constant identity, representation)` projections;
- whole-file and exported-Go-API roots account for an unused untyped constant
  without runtime output, a generic representation root rejects it as
  ambiguous, and an explicit concrete-projection root materializes exactly the
  requested representation or fails;
- removing parent expected-type propagation, restoring `types.Default`,
  flattening a local projection into a function prologue, keying an import by
  spelling, accepting an ambiguous constant root, dropping an explicit
  projection, or restoring a generic empty declaration path fails an owning
  gate; and
- 1x/2x/4x named uses retain one value payload per projection and constant-size
  references.

The float/rune checkpoint additionally covers every admitted float32 and
float64 arithmetic, ordering, equality, and unary operation; float32
overflow, underflow, subnormal, signed-zero, infinity, and NaN behavior; and
canonical rune values independent of source spelling. Mutations remove
float32 rounding, swap each operator class, and substitute source spelling for
checker value. Tests invoke generated Go-shaped call sites so arbitrary host
numbers cannot bypass an input conversion boundary.

The complex checkpoint additionally proves:

- `complex64` and `complex128` are statically incompatible without casts;
- imaginary literals and complex constants use checker values despite
  alternate source spellings;
- zero, construction, `real`, `imag`, unary `+`/`-`, `+`, `-`, `*`, `/`,
  `==`, and `!=` match the selected Go toolchain for both widths;
- `complex64` rounds each construction/result component and `complex128`
  preserves binary64 components;
- division matches the selected runtime algorithm for ordinary values, zero,
  signed zero, overflow-sensitive ratios, infinities, and NaNs;
- `complex`, `real`, and `imag` dispatch by exact `*types.Builtin` identity and
  preserve argument evaluation order;
- `real` and `imag` project a defined complex operand through its exact nominal
  value member, while raw complex carriers remain nominality-agnostic;
- each reached width emits exactly one nominal class, division emits one
  shared definition, and every use site remains O(1); and
- mutations remove a width brand, omit component rounding, replace robust
  division with the naive formula, swap real/imaginary parts, or dispatch a
  built-in by spelling, and each fails its owning gate.

Examples include:

- one-result versus comma-ok map indexing;
- one RHS producing multiple assignment results;
- direct tuple declaration/return/forwarding, blank-result stores, and one
  multi-valued call supplying a complete argument list;
- direct callable parameters, returned named functions, immediately invoked
  literals, lexical captures, and cross-package function values, with the
  exact reached declaration set joined by `types.Object` identity;
- named single and multiple results, zero initialization, bare return, explicit
  return, and a nested literal proving the inner result context replaces the
  outer one;
- explicitly typed local constants, expressionless switches, and each admitted
  direct simple-statement position;
- side effects in parallel assignment and short-circuit expressions;
- nil pointer receiver methods that do and do not dereference;
- embedded methods whose Go static selection differs from TypeScript virtual
  dispatch;
- interfaces with many implementers, proving constant-size call sites;
- nested function literals requiring branch-local versus file-level placement.

Interface-adapter certification additionally proves:

- a concrete type with many unrelated exported and private receiver methods,
  boxed into a two-method interface, emits exactly two receiver forwards;
- boxing that type only into `any` emits zero receiver forwards;
- two demanded interfaces selecting one shared method emit that method once;
- an interface conversion, assertion, and type-switch case discovered before
  or after the concrete adapter each reconstruct the same final method union;
- a transition source widens only adapters whose reachable-contract set
  already contains that source; a concrete type boxed only into `any` stays
  unwidened merely because it also implements the source and target;
- promoted pointer/value methods retain exact checker-selected receiver paths;
- adding an unrelated receiver method is byte-stable, while deleting a demand,
  widening selection to the complete concrete method set, or omitting a
  statically possible assertion target fails the shape, differential, or
  mutation gate; and
- adapter bytes and methods are reported by concrete type, demanded contract,
  and selected method, with the twenty largest adapters inspected.

A repeated-transition scaling fixture holds unique adapters and contracts
constant while multiplying identical assertion occurrences. Admitted
adapter/contract pairs and generated adapter bytes remain constant, and
construction work grows only with occurrences; it must not multiply
occurrences by adapters or transitions.

The callable-value family additionally mutates the callee from a named
declaration to a function-valued expression, changes a signature parameter or
result, drops one lexical capture reference, replaces direct invocation with
`.call`, aliases the signature through `any`/`unknown`, omits a reached
cross-package function, and leaks an outer named-result context into a nested
literal. Each mutation must fail at the target-structure, strict-type,
identity-closure, differential, or broad-search gate. Construction and output
size are measured over doubled parameters, nested literals, captures, and call
sites; call-site size must remain constant per argument and must not grow with
the number of possible function values.

The nil/defined-callable extension additionally proves that callee and
arguments execute in Go order before the nil panic, known non-nil calls remain
byte-stable and direct, every defined value is a stable class whose nil-capable
payload alone may be `undefined`, aliases add no wrapper, and distinct defined
callables remain statically incompatible. Mutations make the class reference
optional, restore static wrap/project helpers, remove the nil-payload guard,
move it before argument evaluation, replace the nominal wrapper with an
intersection/string brand, mutate the function object with `Object.assign`, or
invoke through `.call`; each must fail its owning shape, differential, or
broad-search gate.
Generic defined-callable fixtures additionally require one parameterized class,
exact represented type arguments at every reference, payload projection before
call/range, strict execution for two distinct instantiations, and a mutation
that restores the rejected raw-wrapper invocation.

The recursive-struct extension additionally proves:

- same-spelled local named component types in different lexical declarations
  never canonicalize together;
- identical anonymous structs reuse one declaration, while field tag, blank
  field, unexported-field package, embeddedness, and named-component changes
  remain distinct exactly where `types.Identical` says they do;
- a forced fingerprint collision cannot unify non-identical Go types;
- a shape containing a local named type is emitted only where that type is
  nameable;
- legal pointer/slice/map recursion converges, while definitions grow with
  shape once and zero/copy/equality/hash use sites remain constant-size; and
- mutations restore spelling-only identity, choose first-encounter placement,
  expose a blank field, or inline a field walk at every use, and each fails.

The aggregate-map extension additionally proves key copy on insertion, value
copy on store and lookup, exact collision equality, nil operations, comma-ok,
and one operation owner reused by multiple values. Generated artifacts and a
mutation gate reject function-valued hash/equality/copy fields, strategy
objects, JSON/text keys, reflection, and target object identity.

It also proves:

- number and BigInt profiles both strict-typecheck and execute differentially
  against Go for aggregate keys and values;
- reversed roots produce byte-identical artifacts, and an unreachable map
  shape creates no artifact or target-name perturbation;
- `types.Identical` is the join authority: forced full artifact-key and
  truncated target-name collisions cannot merge non-identical maps, while
  mutating a derived TypeScript declaration spelling cannot change canonical
  Go-type identity;
- an anonymous-struct key and value demand their canonical struct artifacts,
  including exactly one key hash member, through the same generated-artifact
  fixed point rather than a map-only identity or placement graph;
- shapes containing local named types remain immediately after the exact local
  type anchor in a nested block, nested function literal, or package
  initializer, and never create compilation-support output;
- a multi-LHS `types.Initializer` remains one reconstruction owner while no LHS
  variable can masquerade as that owner; checker-produced blank targets remain
  admitted and foreign nonblank targets fail closed; and
- doubling equal-shape use sites keeps one byte-identical specialization class
  and grows source use-site bytes linearly rather than duplicating shape logic.

The first named-struct family additionally proves:

- empty structs and grouped field declarations use the same single
  representation owner;
- two field-identical named Go structs remain incompatible in strict
  TypeScript through an erased nominal brand;
- zero values allocate fresh nested records;
- assignment, initialization, arguments, results, and value receivers do not
  alias mutable struct storage;
- borrowed values are copied once at each admitted boundary while fresh
  composite/call results and single-result returns transfer ownership without
  a duplicate field walk;
- pointer-free assignment rebinds to one copied value without a speculative
  destination-identity helper;
- positional, keyed, and omitted-field composite literals have exact values;
  the default `direct` profile contains no field or argument temporaries added
  solely for keyed source order, while `preserve-go` retains source evaluation
  order when field declaration order differs;
- equality is field-wise and recursive rather than target object identity;
- a field whose equality needs prerequisite statements, such as an array,
  executes those statements inside the static struct equality operation and
  short-circuits in field order; scalar-only structs retain the compact direct
  conjunction;
- concrete value-receiver calls use the exact `go/types.Selection` and the
  selected owner's native class member, never an unqualified virtual
  redispatch; and
- tags, pointers, interfaces, generics, and unsupported field representations
  fail at their typed owner.

Mutations replace a requested copy with direct assignment, replace requested
field equality with `===`, remove the private brand, add source-order captures
to `direct`, remove them from `preserve-go`, detach a receiver method into a
top-level function, duplicate a receiver body, emit an unrequested static value
operation, duplicate an operation, make a value operation an instance member,
route a method contribution to the method's source file instead of its type
artifact, or admit an unsupported field. Each fails its owning structural,
strict-type, differential, placement, or unsupported-boundary gate. The
`preserve-go` fixture uses call-valued field expressions; the `direct` artifact
test treats constants and calls identically, proving that no purity heuristic
silently changes profiles. A scaling fixture doubles fields and proves
definition size/work grows linearly while copy, assignment, equality, and
method call sites remain constant-size. An ownership mutation adds a callee
prologue copy or copies a fresh composite/call result; artifact and operation
counts must reject the duplicate boundary work.

The package-state and initialization family additionally proves:

- reaching one declaration reaches every variable initializer in that package,
  including an otherwise unreferenced effectful initializer;
- state fields join exactly to package-scope `types.Var` identities, with one
  field and no source-file-local mutable binding per variable;
- every field receives its representation owner's exact zero before the first
  initializer executes;
- initializer assignments match `go/types.Info.InitOrder` exactly across
  source files and do not use lexical file order;
- same-package reads, stores, compound assignments, and increments mutate one
  state object;
- package assemblies are passive and contain no top-level Go initialization
  statements;
- one `program.ts` exact-joins reached packages with initialization work,
  consumes the selected import graph, and calls them in Go's global
  import-path-sorted dependency order exactly once; blank-import-only packages
  participate identically;
- admitted `init` functions run after variable initialization in the selected
  loader's file order and declaration order, with no filesystem rescan or
  independently sorted file list;
- multiple legal `init` declarations retain identical target names and call
  order across fresh loads even when cross-file token allocation differs;
- the state module has no runtime import from a source-file module, no
  initializer, and no `any`/`unknown`/cast-created empty object;
- strict ESM execution is differential against Go for cross-file order,
  dependency order, a branching graph where ESM depth-first order differs from
  Go global order, zero-observing initializers, mutation, and repeated reads;
  and
- state/assembly bytes, typed AST nodes, imports, assignments, typecheck
  time/RSS, and runtime grow linearly with variables and initializers.

Mutations emit a package variable as a source-file `let`, omit an unreferenced
initializer, use file order instead of `Info.InitOrder`, skip zeroing, duplicate
a state field, execute initialization at package-module top level, derive order
from ESM depth-first traversal, omit/duplicate/reorder a `program.ts` call,
import a dependency source module instead of its passive assembly, add a
runtime source import to `state.ts`, or replace the state class with a cast
empty object. Each must fail at the ownership, structure, strict-type,
differential, or scaling gate.

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
runtime range or overflow. Every integer gate runs under an explicit
compilation-wide integer selection. The default `number` gate inspects direct
arithmetic, bitwise and shift operators, ordinary numeric literals, and the
absence of routine casts, `Math.imul`, and wrapping operators; its differential
values remain inside the declared exact number and 32-bit bitwise domain.
Checker-valid wide-number constants and bitwise fixtures prove direct
typed-AST shape and strict TypeScript acceptance, but are not evidence of
Go-exact runtime behavior. A mutation restoring a safe-integer rejection must
fail this gate. The `bigint` gate inspects BigInt aliases and literals,
executes values beyond JavaScript's safe-number range, and proves that no
number carrier leaks into integer operations.
Both gates also execute variable shifts whose count is a defined integer type;
artifact inspection requires one nominal payload projection at the shift and
no cast, dynamic dispatch, or duplicated count evaluation.

Mutations reintroduce a routine literal cast or redundant inferred annotation,
emit a wrapped default multiplication, mix representations across generated
files, or print a number literal in BigInt mode. Each fails its owning AST,
strict-type, differential, or profile-coherence gate. Reports explicitly state
that implicit fixed-width overflow is deferred and that the number profile's
wide bitwise operators use JavaScript coercion; neither profile may be
described as proving behavior outside its declared boundary.

Each checkpoint records every compilation-profile axis plus the exact Go
toolchain, pinned TS-Go revision/schema, JavaScript runtime, and GoToTS
support/runtime revision used by proof. No external transpiler, target
compiler, plugin, or unrelated product may supply missing semantics or
verification.

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

Before a full-corpus compile checkpoint closes, generate the same selected
corpus twice from the clean revision and byte-compare every target artifact.
Strict-typecheck the complete generated module graph, not a published subset.
Report represented Go bytes, generated bytes and ratio, category totals,
largest twenty files, generation wall/RSS, and strict-typecheck wall/RSS.
Attribute every material parent delta to a typed owner; a zero-diagnostic
result cannot conceal an unexplained size tail.

The determinism mutation assigns semantically earlier provider declarations and
methods later absolute token positions, perturbs request/load/root order, and
requires byte-identical source modules, provider modules, class members,
exports, and support artifacts. Source-artifact scheduling must still preserve
stable module-path plus physical declaration order: the assignment golden and
constant-call-site scaling fixture fail if semantic-name order replaces that
source order. Restoring cross-file position-first ordering must fail before the
corpus comparison.

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

### Milestone 3A Value-Foundation Gate

Each of the six value families has an independent focused gate and mutation
battery. Integration additionally runs one mixed-family, multi-package
program. The required evidence includes:

| Family | Independent behavioral proofs | Required mutations |
|---|---|---|
| integers and numeric conversions | every source/destination width and profile disposition; narrowing, sign change, integer/float, float-width, and complex-width boundaries; constants; NaN, infinity, signed zero, and representability; BigInt division/remainder for nonzero and zero divisors | width alias collapse; unsafe number literal; missing narrowing/bounds operation; conversion spelling lookup; ordinary-call fallthrough; duplicated conversion operand; direct host divide/remainder or non-finite BigInt conversion bypass |
| strings | arbitrary-byte literal, concat, comparison, byte length/index/slice and bounds; integer/rune UTF-8 encoding; `[]byte`/`[]rune` conversions including invalid and truncated sequences | Unicode-code-point literal; UTF-16 indexing; host text codec; invalid-sequence width drift; missing bounds check |
| arrays | length-distinct target types, fresh zero/copy, compiler-lowered recursive equality, index/store/len/cap, direct and pointer-to-array slicing with bidirectional aliasing | erased length; shared zero; shallow copy where element policy forbids it; copied array slice; duplicate bounds owner; runtime zero/copy/equality callback or target object identity |
| slices | nil/empty distinction, aliasing, reslice, append reuse/reallocation, copy overlap, aggregate fresh zero/copy/clear, contextual elided nested literals, distinct-defined spread, `[]byte` plus string spread, and large spread without host argument expansion | bare-array substitution; lost capacity; always-reallocate append; explicit-type requirement for an elided literal; semantic strategy field/parameter; JavaScript spread of a Go slice; unconditional append-spread/clear surface |
| maps | nil write, missing zero, comma-ok, aliasing, direct bool/integer/string keys, owner-private defined-key projection/reification, delete/len, and explicit floating-key rejection | plain-object substitution; missing-value `undefined`; copy-on-assignment; callsite storage-key leakage; wrapper object identity; floating-key admission through native SameValueZero |
| pointers | nil/new/read/store/alias/equality; local/parameter/result/receiver/package/field/array/slice addresses; reassignment through projections; pointer receiver nil/copy cases | fresh wrapper on copy; wrapper identity instead of canonical location; nil dereference success; unrelated-local cell wrapping; wrong requirement owner |

Conversion certification additionally proves that represented struct
conversion emits one destination-owned `$convert` definition under repeated
uses, recursively copies aggregate fields, ignores tags only where
`go/types.ConvertibleTo` permits it, and never accepts a structurally similar
but Go-inconvertible source. Slice-to-array value tests prove short-length
panic timing, one operand evaluation, fresh array identity, recursive element
copy, defined source/destination composition, and zero callback/cast paths.
Slice-to-array pointer tests separately prove short-length panic timing and
one evaluation, nil-versus-empty length-zero results, canonical equality by
backing plus offset, bidirectional aliasing, whole-array assignment, recursive
element-copy isolation, defined slice/array composition, and the absence of
nominal-wrapper forwarding methods.
Array-slicing tests prove direct, defined, plain, pointer, and defined-result
forms; low/high/max evaluation order; ordinary slice-bound panic behavior;
bidirectional aliasing; and exact demand shape for one array-location facet
and one array-to-slice helper. A no-array-slicing artifact contains neither
facet. Contextual-literal tests prove nested elided slice and map elements use
checker types and produce the same strict target structure and behavior as
their explicit-type equivalents.
Mutations inline the field walk at every use, omit one field copy, admit an
inconvertible struct, move the length check after allocation, alias slice
backing for the value conversion, copy backing for the pointer conversion,
key the pointer by slice-descriptor identity, collapse nil and non-nil empty
zero-length slices, copy an array during slicing, add a second array-bounds
path, require explicit nested literal syntax, restore defined-array forwarding
methods, or evaluate the slice twice; each must fail its owning shape or
differential gate.

Pointer-conversion certification additionally proves:

- the generated pointer contract has distinct logical and canonical-storage
  type arguments, with no erased payload or runtime semantic callback;
- conversions accepted by `go/types.ConvertibleTo` preserve one canonical
  address and typed storage accessor pair across scalar, tagged-struct,
  aggregate-field, and nested pointer-field base types;
- writes through either logical view are visible through the other, and a
  round trip compares equal by canonical location;
- array- and slice-element addresses use direct storage when logical and
  storage types coincide, and one compile-time-selected typed projection when
  they differ;
- cross-file storage aliases are imported as type-only bindings, emitted once
  by their declaration owner, and absent when no address/conversion demands
  them; and
- mutations collapse the two pointer type arguments, copy the pointee during
  conversion, compare facade identity, replace `go/types` convertibility with
  a hand-recursive field rule, omit a storage projection, or expose storage
  conversion as a runtime callback, and each fails its owning strict,
  differential, artifact-shape, or broad-search gate.

Open-generic representation certification uses a matrix in which one source
type parameter is instantiated by a direct named-struct pointer, scalar
pointer, conversion-selected struct carrier, defined array, and nested generic
type. It proves:

- `*new(T)` is emitted as the exact generic zero operation with no pointer or
  storage facet, while a direct `pointer.Field = value` store emits no
  interior-pointer construction; restoring either unnecessary pointer route
  fails strict typechecking and the artifact-shape gate;
- a logical-only declaration emits no storage or pointer facet;
- `*T` transport adds exactly one pointer facet, whole-value storage adds the
  whole-storage facet, and array/slice slots add the distinct
  container-storage facet plus only their demanded closed operation signatures;
- each concrete instantiation exact-joins source type arguments to its logical,
  whole-storage, container-storage, and pointer target arguments in canonical
  order;
- `Bag[T]` with unaddressed `[]T` uses plain `Item` container storage, while
  `Arena[T]` with `&slice[i]` selects canonical carrier storage; both
  strict-typecheck and execute for direct-class `Item` and scalar `int32`;
- replacing a slice or array slot after taking its address remains visible
  through that pointer, proving the class-object shortcut is not admitted;
- a nested `Outer[T] -> Box[T] -> *T` demand propagates one facet through the
  artifact fixed point, terminates under recursion, and never creates a second
  body;
- one exact concrete pointer type has one compilation-wide representation, so
  generic and nongeneric users cannot select direct and carrier forms
  simultaneously;
- every pointer to an instantiation of one generic named declaration joins the
  origin's one parameterized pointer family, so declaration methods, concrete
  calls, interface adapters, and nested fields cannot acquire incompatible
  direct/carrier ABIs; and
- 1x/2x/4x instantiations grow by distinct reached facet contracts rather than
  call count, with unchanged logical-only artifacts byte-identical.

Mutations restore `Storage(T)=T`, conflate whole and container storage,
substitute `GoPointer<T,T>` for the selected pointer facet, bypass the indexed
address capability, omit or reorder a concrete facet argument, add every facet
speculatively, key a facet by target spelling, keep both direct and carrier
forms, or transport a semantic converter on a runtime pointer. Each fails at
the exact facet join, strict target, differential alias/copy, convergence, or
artifact-shape gate.

The generic-storage matrix additionally reads, writes, takes addresses of, and
constructs fields whose concrete logical and storage facets differ. Artifact
shape proves one whole-storage projection per reached class regardless of
field count, zero per-field static accessors, caller-owned conversion,
explicit canonical construction arguments, and no converter field or logical
cache on an instance. Mutations perform conversion inside an instance member,
store a callback on the value, read canonical storage as the logical type,
write the logical type directly, omit one consumer dependency, add per-field
accessors, or retain both direct and storage-property routes; strict
typechecking, differential alias/copy behavior, the observable-facet fixed
point, source-size scaling, and broad searches must catch them.

Map certification exact-joins every public map type and operation against
semantic `K,V`, while separately inspecting private storage choices. The
matrix covers primitive keys, defined-basic keys, aggregate keys, generic
`map[K]V`, `M ~map[K]V`, defined generic maps, and map range. Mutations expose
the storage key in `GoMapValue`, project/reify at a callsite, return storage
keys from `keys()`, restore defined-basic native-map selection, omit a
defined-key wrapper, store a projection callback, or construct the wrong
concrete owner. Strict typechecking, Go-versus-TS execution, owner-level AST
shape, exact facet joins, and 1x/2x/4x source-size gates must each catch their
owned class.

For every family:

1. the focused test is first observed failing at the typed unsupported owner;
2. TS-Go AST shape is asserted before printing;
3. printed output strict-typechecks before execution;
4. Go and generated ESM execute differentially;
5. the runtime export request exact-joins one definition and fails for omitted,
   duplicated, wrong-module, cyclic, or dependency-incomplete symbols;
6. generated source-module bytes, fixed support/runtime bytes, encoded target
   AST bytes/nodes, and definition counts are reported separately and measured
   for 1x/2x/4x use sites;
7. the twenty largest changed bodies and all runtime declarations are
   inspected; and
8. broad searches prove no text emission, duplicate representation switch,
   native-JS approximation, compatibility path, or erased payload remains.

The cross-family owners have additional blocking proofs:

1. the runtime dependency closure emits one `runtime/panic.ts`, and each
   demanded integer/string/pointer/array/slice/map module imports exactly one
   `GoPanic` binding through its contract;
2. only the panic runtime owner creates or initially throws a `GoPanic`;
   replacing one family guard with a host exception or removing one dependency
   fails; the only other admitted target throw is a callable envelope rethrowing
   an unrecognized caught host exception unchanged;
3. array, slice, and map stores all enter the same setter transaction, which
   differentially proves receiver/index/key/right-side order with both direct
   and prerequisite-bearing operands;
4. `new`, `make`, `len`, `cap`, `append`, `copy`, and `delete` enter one
   `*types.Builtin` dispatcher, while family sub-owners cannot rediscover the
   builtin from spelling; and
5. the mixed-family fixture spans at least two source packages, strict
   typechecks all emitted modules under the default `number` profile and the
   `bigint` override, exact-compares ordinary Go/ESM output, and observes the
   shared panic carrier for BigInt divide-by-zero, string bounds, nil pointer
   dereference, array bounds, slice bounds, and nil map store.

Deletion mutations restore a family-specific assignment route, a second
builtin resolver, a native family throw, or an undeclared runtime dependency;
each must fail at the owning architecture, artifact, or differential gate.

Expression-completion certification also requires:

1. a variadic declaration artifact with one represented slice parameter and no
   target rest parameter;
2. differential calls for zero/one/many variadic arguments, defined
   `slice...`, a tuple supplying fixed plus variadic positions, and a valid
   large slice that would exceed a host spread-call limit; an aggregate
   variadic element whose logical and container-storage types differ must
   exact-join the container-storage owner before slice construction;
3. `append` spread differentials for aliasing reuse, reallocation, two
   distinct defined slice types with identical elements, aggregate elements,
   `append([]byte, string...)`, and package-declared untyped string constants
   through both string-special `append` and `copy`; restoring the raw untyped
   child expectation must fail at the constant-use boundary;
4. aggregate array/slice artifacts containing direct element operation calls
   and no function-valued zero/copy parameter, field, or argument;
5. runtime artifact pairs proving append-spread and clear declarations are
   absent without demand and present exactly once with demand;
6. logical `&&`/`||` expressions whose right operand has prerequisite
   statements, proving those statements remain inside the selected
   short-circuit branch; and
7. mutations that restore tuple-before-variadic rejection, slice
   assignability admission, stored semantic callbacks, host spreading, shared
   aggregate zeros, shallow aggregate growth/copy, or unconditional runtime
   members.

The compile-only unsafe-boundary gate additionally proves that
`(*T)(unsafe.Pointer(p))` uses one nullable nominal contract parameterized by
the exact represented pointer type, for both direct named-struct pointers and
`GoPointer` carriers, round-trips nil differentially, strict-typechecks without
`any`, `unknown`, or casts, and throws the named unresolved placeholder for a
non-nil conversion. Removing the nil branch, hard-coding the carrier layout,
accepting `uintptr` through the pointer route, or replacing the placeholder
with target-memory guessing must fail at this gate.

The integrated Wave-3 matrix additionally compiles, TS-Go-encodes, prints,
strict-typechecks, and executes one cross-family program under the default
`number` profile and the `bigint` override. It includes a defined map literal
at a call boundary, predeclared booleans projected to a defined boolean, and
`real`/`imag` over a defined complex value. Artifact-shape checks require the
single nominal map wrapper, nominal boolean construction, and explicit complex
value projection; reverting any underlying-family decision fails before the
differential comparison. The matrix freezes total and largest-file byte bounds
and rejects erased types, casts, reflective call helpers, and dynamic imports.

### Milestone 3C Structured-Control Gate

The integrated fixture runs under the `number` and `bigint` integer profiles.
It must encode and print through pinned TS-Go, strict-typecheck before
execution, and match Go for arrays, pointer arrays, slices, arbitrary-byte
strings, maps, integer ranges, structured `for`, labels, expression switches,
fallthrough, and local multi-result declarations.

Blocking structural evidence includes:

1. a constant-length one-variable pointer-array range contains no operand
   evaluation, while an array-returning call is captured and invoked exactly
   once;
2. map range materializes one key snapshot and performs one live comma-ok
   lookup per considered key, with deletion suppressing an unseen entry;
3. exact `*types.Label` identity survives source-spelling mutation and missing
   label-use evidence fails at the branch owner;
4. custom-equality and prerequisite-bearing switches use one selection
   variable and one execution switch, and each clause body occurs once;
5. a default clause may fall through when it is not last, while every
   non-fallthrough clause receives one implicit target break;
6. structured loop prerequisites are declared once outside the target loop,
   but invoked only at their Go condition/post boundary; and
7. no `ForInitializerEmission`, alternate clause dispatcher, generic control
   IR, or collection-sized generated loop remains.

A 1x/2x/4x fixture independently measures source bytes, printed target bytes,
and encoded TS-Go nodes. Range-loop count remains one as source collection
length grows; custom switch checks and output grow linearly with source case
count. Mutations restore operand skipping for a real call, remove the key
snapshot or live lookup, route labels by spelling, duplicate a switch body,
lose fallthrough, or restore the superseded header-only API; each must fail its
own differential, shape, identity, scaling, or broad-search gate.

Addressability has an additional exact matrix:

1. direct local, parameter, named-result, receiver, package-state, nested
   closure, struct-field, fixed-array-element, slice-element, composite-literal,
   and `&*pointer` cases strict-typecheck and execute differentially;
2. a pointer to a struct field observes later whole-struct assignment, while a
   pointer projected from a pointer variable is not retargeted when that
   pointer variable changes;
3. slice aliases produce equal element pointers from the same backing/index
   and unequal pointers for different indexes or reallocated backing;
4. address/index/receiver operands execute once and in Go order;
5. only exact requested `*types.Var` identities become cells; shadowed
   same-spelling variables and a deterministic ordinary sample remain
   byte-identical and unwrapped;
6. a nested literal captures the selected outer cell and owns any selected
   inner cell at the inner lexical declaration;
7. package `init` and a checker-produced package initializer containing an
   address-taking function literal are each reconstructed by the ordinary
   artifact scheduler, with no fabricated function or variable owner;
8. pointer-receiver calls on values and pointers, and value-receiver calls
   through pointers, preserve nil, argument evaluation, and copy behavior
   through class-owned members without top-level receiver twins, `.call`,
   `.apply`, or `.bind`;
9. adding storage changes only the function body contract, causes zero caller
   reconstruction, and reaches a deterministic fixed point; and
10. 1x/2x/4x address sites keep each use constant-size and leave an unaffected
    declaration's generated TS-Go bytes unchanged;
11. the assignment package has no concrete storage or pointer-runtime import,
    while the root emitter installs exactly one typed storage capability;
12. ordinary `*S`, `new(S)`, `&S{}`, `&local`, field access, pointer equality,
    and pointer-receiver calls use `S | undefined` with zero `$Storage`,
    `$make`, `$storageOf`, `$fromStorage`, or `GoPointer.dereference` surface;
13. whole assignment to an addressed `S` mutates the one stable location while
    unaddressed `S` assignment still rebinds a Go copy; and
14. adding the first exact pointer conversion or non-class location demand
    reconstructs only the involved pointer/type artifacts into the one carrier
    form, while removing the last demand reconstructs them back.

Production-path mutations drop a storage requirement, key it by spelling,
select a shadow sibling, wrap every local, compare pointer wrapper identity,
mis-key a field/index projection, key a slice by descriptor identity, skip the
required `&*p` nil check or read its stored value, copy a pointer receiver,
omit the value-receiver copy, attach a package-initializer literal's local to a
fabricated function or package-variable owner, bypass package-`init` artifact
reconstruction, wrap an ordinary struct pointer, rebind an addressed struct
location, keep a storage facet after its last demand disappears, or dirty callers after an
unchanged callable facet. Each fails at its owning structure, artifact,
strict-type, or differential gate.

### Milestone 3D Concrete-Method Gate

The integrated fixture runs under the `number` and `bigint` integer profiles.
It must encode and print through pinned TS-Go, strict-typecheck before
execution, and match Go for methods on represented defined and struct types,
embedded values and pointers, nested promotion, field read/store/address,
direct calls, method values, method expressions, anonymous embedding,
variadics, and nil-sensitive receiver adjustment.

Blocking evidence includes:

1. every concrete selector exact-validates `Selection.Kind`, `Obj`, `Recv`,
   result type, and complete `Index()` path against the selected checker graph;
2. mutating only selector spelling leaves encoded target AST byte-identical,
   while substituting another valid selection object fails at the selector
   owner;
3. a promoted call from `Base.CallName` remains `Base.$Name(base)` even when
   the embedding type declares its own `Name`, proving no accidental virtual
   dispatch;
4. method-value formation evaluates and captures the receiver once, copies
   value receivers once, and preserves pointer identity;
5. nil-safe pointer receiver values may form and execute, while a value method
   selected through a nil pointer and a promoted field through a nil embedded
   pointer panic at the same boundary as Go;
6. direct value-method expressions are typed native-member arrows, direct
   pointer-method expressions are class-owned static-member references, and
   promoted method expressions are typed arrows with an explicit first
   receiver parameter;
7. embedded fields remain owned class fields, including anonymous structs and
   reserved/unexported member names; generated support imports their defining
   source modules with collision-safe package-qualified aliases; environment
   struct contracts likewise retain an unexported embedded promotion spine
   while omitting unrelated provider-private fields; and
8. generated artifacts contain no class `extends`, top-level receiver twin,
   prototype patch, `.call`, `.apply`, `.bind`, erased carrier, reflection, or
   per-method/per-implementer dispatch switch; and
9. a method declared in another Go file is present exactly once inside the
   declaring type's class, with its imports owned by that class artifact and no
   statement emitted in the method source module; and
10. a generic method selected from a different source file resolves its
    declaration receiver through the method origin, joins the exact same
    source-owned pointer-representation family as the declaration, passes the
    concrete carrier directly without rendering the origin's foreign type
    parameter, and emits no carrier-to-logical bridge; the same proof runs
    through an interface adapter and preserves all receiver prerequisite
    statements.

A 1x/2x/4x embedding-depth fixture independently measures source bytes,
printed target bytes, and encoded TS-Go nodes. Each use contains one selected
owner-qualified member call, and all three measures grow linearly with source
depth. Production mutations remove an index component, replace the selected
object, select by spelling, omit a value copy, skip an embedded-pointer nil
check, reevaluate a receiver, restore a top-level receiver function, or route a
method contribution through the wrong source/public assembly; each fails its identity,
differential, strict-type, shape, naming, or scaling owner.

### Milestone 3E Interface Gate

The integrated interface fixture runs under both integer profiles and covers
named and anonymous interfaces, embedding, exported and package-private
methods, value and pointer dynamic types, promoted methods, implicit and
explicit conversions, calls, method values, method expressions, assertions,
comma-ok, panicking assertions, ordered type switches, equality, and interface
map keys.

Blocking evidence includes:

1. nil interface and interface-containing-typed-nil remain distinct in emitted
   AST and differential execution;
2. concrete boxing copies value payloads once and preserves reference payload
   identity;
3. each exact concrete dynamic type has one adapter class regardless of
   interface count or call count, plus one canonical compilation-scope
   non-string dynamic-type token;
4. every adapter payload, method parameter, and result is statically typed,
   and every native method directly invokes the exact class-owned member
   selected from `go/types`;
5. exported and package-private method contracts exact-join by semantic
   identity and receiver-free signature, never source spelling;
6. interface calls contain one nil guard and one native call, with no
   implementer-count switch;
7. concrete assertions use a typed predicate over exact canonical dynamic-type
   token identity, interface assertions use the target contract's type
   predicate, and type-switch cases preserve source order and case-variable
   type;
8. same-type comparable payloads use their existing equality/hash owner,
   different dynamic types compare unequal, and non-comparable dynamic values
   panic only at the Go-required equality or map-key boundary; and
9. generated method-token, dynamic-type-token, anonymous-interface, and
   adapter artifacts reconstruct transactionally and expose only the facets
   their consumers subscribe to; and
10. the same local Go type boxed by separate invocations asserts equal in
    dynamic type, compares by its payload owner, and resolves as the same
    interface map key even though its lexical adapter class declaration
    executes separately; and
11. pointer-only fixtures that never hash a pointer contain neither
    `goPointerHash` nor `GoMapHash`, while an interface-map fixture with a
    pointer dynamic key exact-joins one optional pointer-hash definition and
    executes alias-equivalent pointer lookups correctly; and
12. ordinary arguments, panic payloads, interface equality operands, map
    keys, and captured multi-result elements all reach the same value-transfer
    owner. A `(T, bool)` call expanded into `func first[T](T, ...any)` must
    contain one typed adapter around the captured `bool`; removing that
    transfer must fail strict TypeScript, while root expression dispatch
    contains no interface-boxing route.

Production mutations collapse typed nil, omit the boxing copy, substitute a
same-spelling private method from another package, change a method signature,
drop a method token, enumerate implementers at a call or assertion, read an
adapter payload before token-predicate narrowing, substitute adapter
constructor identity for the canonical token, replace the token object with a
string or truncated hash, reorder type-switch cases, return false instead of
panicking for equal dynamic uncomparable types, or hash an uncomparable
payload. Each must fail its owning identity, strict-type, differential,
mutation, shape, or artifact gate.

A 1x/2x/4x fixture varies implementer count while holding interface call sites
constant. Call-site printed bytes and TS-Go node counts must remain constant;
adapter bytes may grow only with the reached concrete method sets. A separate
method-count fixture proves assertion work grows with the asserted interface
contract, not with possible implementers. The twenty largest changed target
declarations and calls are inspected before closure.

### Milestone 3F Generic And Iterator Gate

The integrated generic fixture covers generic functions, aliases, named and
recursive types, generic receiver methods, explicit and inferred
instantiations, instantiated function values, recursive and mutually recursive
generic calls, cross-package calls, type-parameter zero/copy/equality/hash,
operators/conversions/indexing/methods/interfaces, and all three
range-over-function signatures.

Blocking evidence includes:

1. every generic identifier occurrence exact-joins one
   `types.Info.Instances` record to the selected declaration, ordered type
   arguments, and instantiated signature/type;
2. each declaration has one target generic body and one hidden function
   parameter per distinct `(typed operation selection, exact signature)`
   emitted by that body; repeated source sites exact-join, same-signature
   distinct constraint methods remain distinct, and a cross-parameter operation
   remains one function rather than a per-parameter object;
3. a concrete operation-selection/instantiated-signature pair has one
   reconstructible function definition, while a still-generic signature
   projects and forwards the enclosing function without boxing or runtime
   lookup;
4. recursive capability propagation converges through callable-facet
   dependencies; an unchanged facet performs zero downstream reconstruction
   and an oscillating contract fails;
5. every operation function delegates to the existing concrete semantic owner,
   and mutations that replace it with direct target arithmetic, shallow copy,
   object identity, spelling selection, or a throwing unsupported member fail;
6. aliases add no runtime identity, instantiated named values retain exact
   nominal/copy behavior, and generic receiver methods use the same ordered
   capability ABI;
7. instantiated function values are typed arrows capturing capabilities once
   and contain no `.bind`, `.call`, or `.apply`;
8. iterator range evaluates its source once, copies yielded values at the
   iteration boundary, maps continue/break to true/false, rejects a post-false
   yield, composes with represented generic values, and routes blocking work
   through the exact yield callable ABI so callback, provider, iterator
   invocation, and enclosing source callable become cooperative as one
   dependency chain while a distinct synchronous ABI remains byte-identical;
   and
9. encoded TS-Go AST reparses, strict TypeScript passes under both integer
   profiles, and Go-versus-generated-ESM output and panic classes match.

A 1x/2x/4x instantiation fixture holds one generic body fixed and proves its
printed and encoded size is constant. Concrete capability growth is bounded by
distinct exact operation-selection/signature pairs, not call count or
source-site count.
A separate recursive-call fixture records contract revisions and proves
convergence. The twenty largest generic bodies, capability declarations,
instantiated types, and calls plus strict-typecheck time/RSS are inspected.

Production mutations drop or reorder an instance argument, use a defaulted or
spelling-derived type, omit or add an operation function, key an operation by
source position or diagnostic token spelling, duplicate same-signature hidden
parameters, duplicate a body per instantiation, erase a payload, replace
forwarding with a concrete descriptor, suppress a callable-facet dependency,
call yield after false, reevaluate the iterator, or move yielded copy work
outside the callback. Additional mutations attribute callback blocking directly
to the enclosing source facet, leave the generated callback synchronous, or
invoke the iterator outside the callable-value owner. Each must fail its
identity, contract, strict-type, differential, convergence, shape, or scaling
owner.

Lexical generic-operation identity mutations give two functions the same local
type spelling and scope shape, relocate one function across unrelated source
lines, and pair a local type with a foreign function root. Exact
owner-qualified keys must respectively differ, remain stable, and reject the
foreign root.

### Milestone 3G Function-Control Gate

The integrated control fixture covers source `panic`, recover outside defer,
direct deferred recover, recover one call below, nested and replacement
panics, immediate callee/receiver/argument evaluation, value copies, LIFO
order, ordinary and named results, class-owned receiver members, function and
method values, interface calls, generic functions, labels, fallthrough, labeled
break/continue, forward and backward goto, non-structural goto, and
goto/defer/range composition.

Blocking evidence includes:

1. the source call and every admitted runtime fault throw the same non-erased
   `GoPanic` class; only `runtime/panic.ts` creates or initially throws that
   carrier, while a callable envelope may only rethrow an unrecognized caught
   host exception unchanged;
2. recovered `panic(nil)` and generated runtime faults satisfy the canonical
   `error`, `interface { Error() string }`, and `runtime.Error` contracts;
   only `panic(nil)` has the canonical `*runtime.PanicNilError` dynamic identity
   and represented payload;
3. `recover` consumes one typed invocation-local authority only in the direct
   deferred invocation; ordinary calls and one-call-below cases return nil;
   every direct, function-value, method-value/expression, interface, generic,
   field, pointer, and container transport consumes the same exact-signature
   callable ABI rather than a carrier-specific recovery identity;
4. defer registration evaluates the callee, selected receiver, and copied
   arguments once and in source order, while invocation is LIFO at every exit;
5. a deferred panic replaces the pending panic without skipping older
   deferred calls, and named results are stored before and read after unwind;
6. exact `*types.Label` identity survives spelling mutation and missing or
   mismatched definition/use evidence fails at the label or branch owner;
7. direct structural edges contain no state machine; genuinely
   non-structural goto contains one callable-local linear machine and no
   persisted source/control model;
8. block, switch-clause, and type-switch-clause statement lists use the same
   sequence owner; a nested state machine preserves source loop/range
   `continue`, switch `break`, and switch `fallthrough` targets rather than
   capturing them in its generated dispatch;
9. a no-demand function's encoded TS-Go AST is byte-identical to its parent
   checkpoint artifact; and
10. TS-Go encode/print/reparse, strict TypeScript, and Go-versus-ESM behavior
   and panic classes pass under both integer profiles.

Mutation gates remove immediate capture, reverse the defer stack, shallow-copy
an argument or value receiver, make recovery ambient, forward authority to an
ordinary nested call, swallow or fail to replace a panic, return named results
before unwind, alias panic-nil and generic-runtime dynamic identities, replace
the canonical runtime method contract with spelling/signature matching, key a
recovery contract by a variable/field/pointer/container identity, key a label
by spelling, bypass the shared clause-sequence owner, let generated dispatch
capture source `break`/`continue`, misroute a goto, force all functions through
the envelope, or restore dynamic invocation. Each fails its owning shape, identity,
strict-staticness, differential, byte-stability, or artifact gate.

A 1x/2x/4x source fixture measures emitted bytes and TS-Go nodes for defer
registrations, labels, and non-structural states. Growth is linear; one source
statement is emitted once; runtime support remains constant. Inspect every
runtime control declaration and the twenty largest changed functions.

### Milestone 3H Cooperative Concurrency Gate

The selected race-free cooperative profile exits only when focused fixtures
prove through TS-Go encode/print, strict TypeScript, and Go-versus-generated
execution:

1. channel directions; nil, unbuffered, and buffered construction; capacity;
   canonical equality; and exact zero/copy transfer;
2. FIFO ordering within and across direct/select senders and receivers,
   receive expression, comma-ok and discarded receive, close/drain/zero/ok
   behavior, and close/send panic classes;
3. channel range and its break/continue/label integration;
4. immediate source-order goroutine callee/argument evaluation and copying,
   cooperative execution, main-return abandonment, uncaught goroutine panic,
   and deadlock detection;
5. ready, blocking, default, nil, cancellation, one-commit, and fairness
   properties for `select`, including delayed receive-target evaluation,
   select-to-select unbuffered rendezvous, fair same-channel blocked-case
   registration, and selected-send close failure attributed to the selecting
   operation rather than the closer. A
   pure default-bearing select's strict artifact is a synchronous,
   non-`Promise`, non-`async`, non-`await`, scheduler-free function; a
   no-default select retains the blocking cooperative path, while independently
   blocking operand evaluation still propagates cooperation;
6. source-facet propagation for direct functions/methods/literals and one
   storage-independent exact-signature callable ABI for first-class values
   transported through locals, parameters, results, package variables,
   fields, pointers, arrays/slices/maps, interface assertions/calls, generic
   aggregates, and recursive cycles, while deterministic synchronous
   contracts remain byte-identical;
7. cooperative constraint-method capabilities reconstruct the exact hidden
   operation function and generic caller, while synchronous instantiations
   retain one adapted typed path and no second generic body;
8. deferred cooperative direct, function-value, method, interface, and generic
   calls are captured immediately, awaited in LIFO order, complete before
   return or panic propagation, and retain direct-only recovery authority; and
9. immediately invoked function literals observe their exact literal facet
   without a first-class callable-ABI adapter, while transported literals still
   use that ABI; and
10. generic function and generic receiver-method uses exact-join callable
    leaves in declaration and instantiated signatures through `go/types`
    evidence. Synchronous and blocking concrete callbacks select distinct
    statically named callable-profile variants only when both profiles are
    reached. The synchronous declaration remains non-`Promise`, while the
    cooperative variant awaits its selected callback ABI. Nested container and
    result callables, deferred calls, instantiated function values, and generic
    method calls/values/expressions follow the same correspondence. A generic
    function that intrinsically returns a blocking callable propagates that
    declaration ABI to each concrete result ABI while retaining the ordinary
    source name. Mutations collapse profiles into declaration-wide widening,
    reverse a concrete ABI into the declaration baseline, drop the
    declaration-to-concrete result propagation, select by runtime thenable
    inspection, omit a selected profile, or create one variant per call; each
    must fail strict-staticness, differential, byte-stability, identity, or
    scaling evidence. A cross-package case imports the selected variant only
    through the provider package assembly and proves that dropping its
    re-export fails strict typechecking; and
11. a package variable initialized by a generic call with a synchronous
    function literal remains synchronous when another instantiation selects a
    cooperative callable-profile variant. A separate initializer whose own
    call selects that variant reconstructs through its exact
    `types.Initializer` facet, makes only its package `$initialize` async, and
    is awaited by `program.ts` before the next package. Mutations await the
    synchronous initializer, await every package, or key the requirement by an
    LHS variable; and
12. environment-owned generic callable profiles produce one typed ambient
    declaration per distinct reached profile without entering the source
    emitter or fabricating a body. A `slices.Values` result used with a
    cooperative range callback carries the exact Promise-returning yield ABI
    while the `Values` call remains synchronous. Strict typechecking and
    artifact inspection prove the base declaration remains unchanged and the
    selected declaration is deduplicated. Mutations omit the declaration,
    route the environment owner through source reconstruction, widen the base,
    infer the outer effect from nested callable shape, or create one ambient
    declaration per use; and
13. a generic function returning a named callable value keeps its ordinary
    nested literal and wrapper synchronous while a distinct reached profile
    owns a cooperative nested literal and exact closed result ABI. The named
    wrapper has one hidden value-facet parameter defaulted to its ordinary
    represented callable type; declaration-origin analysis gives every
    reference the same arity, while a generic map whose concrete argument's
    nominal fields contain a callback retains only its declared type
    parameters. Mutations drop the lexical profile from literal identity,
    globalize its ABI request, omit profile-to-closed-ABI propagation, restore
    a result intersection or cast, infer hidden arity from an instantiated
    argument's transitive fields, or add the hidden facet to every named type;
    each must fail
    differential behavior, strict staticness, artifact shape, ordinary-profile
    byte stability, or source-size evidence; and
14. callable-ABI identity-domain isolation: a declaration-scoped callable
    containing the generic owner's type parameter remains profile-local, an
    unrelated canonical synchronous named function used inside that profile
    adapts to its cooperative canonical target ABI, and neither change widens
    the ordinary generic declaration. Environment fixtures additionally prove
    that a non-generic `sync.WaitGroup.Go` callback and an environment
    interface method consume their canonical cooperative contracts, while an
    environment-owned generic baseline stays declaration-scoped. Mutations
    erase ABI scope, classify every ABI in a profile as profile-local, classify
    every ABI as canonical, omit the environment function-value observation,
    or omit the environment interface-method observation; each must fail
    artifact shape, strict typechecking, differential behavior, or
    ordinary-profile byte stability; and
15. the selected-profile boundary: concurrency fails while the profile is
    disabled; the cooperative selection and all output evidence name that it
    does not reproduce asynchronous preemption. The checked-in busy-goroutine
    counterexample receives no yield/preemption workaround and is never run as
    if it were admitted exact behavior; and
16. generic nominal-field callable ownership: a generic constructor stores both
    synchronous and channel-receiving callbacks in the same declared function
    field, and a generic receiver method invokes that field. The field
    declaration, constructor transport, receiver invocation, and reached closed
    ABIs exact-join through checker field identity and structural position.
    Strict output has one Promise-returning field contract, awaits the direct
    field call, statically adapts the synchronous provider, and matches Go for
    both uses. An unrelated callable remains synchronous. Mutations key the
    relation by spelling, omit the constructor edge, omit the field-read edge,
    reverse only one side, add a hidden per-instance effect parameter, or restore
    a generated-site cast; each fails identity, strict typechecking,
    differential behavior, artifact shape, or broad-search evidence. A
    copy-only generic function reads the field into a new instance while the
    field contract is cooperative; it remains synchronous and has no generated
    cooperative profile. A mutation that descends through the named
    declaration's underlying struct creates that profile and fails the shape
    gate.

Interface callable ownership has a separate exact-identity gate:

- adding a cooperative unrelated `func() bool` leaves the emitted
  `interface{ IsDir() bool }` declaration, adapter and direct call byte
  unchanged;
- making the selected concrete `IsDir` implementation cooperative changes the
  one canonical method-token callable facet and reconstructs its declaration,
  adapter, direct/deferred calls, method-value bridge and generic
  constraint-method capability;
- changing the method name while retaining its receiver-free signature does
  not join token identities;
- alpha-renaming `Value[T].Get() T` to `Value[U].Get() U` preserves the
  generic callable-family identity, while `Value[int32].Get` and
  `Value[string].Get` receive distinct closed runtime tokens;
- an open `Value[T].Get` direct call observes the generic family and a mutation
  that requests a runtime token fails at the naming boundary;
- one adapter demanded through multiple compatible generic and non-generic
  interfaces selects every target facet when any selected implementation is
  cooperative;
- `Value[T].Change(func(T))` exact-joins its declaration-owned callback ABI
  with every reached closed or ambient selected callback ABI; a blocking
  `Value[int32]` callback makes the interface declaration, closed call,
  adapter, and selected provider agree on `Promise<void>`, while an unrelated
  callable remains byte-identical;
- a generic receiver implementation invoked by an adapter receives the exact
  ordered type arguments and operation capabilities from the shared
  selected-method plan; and
- restoring interface-boundary `ValueContract`, `ValueCall`,
  `SourceValueContract`, or a raw adapter `SelectedMethodCall` fails the
  architecture wall and the unrelated-function mutation.

Artifact inspection reports synchronous and cooperative interface methods,
adapter bytes, and the twenty largest adapter methods separately. The gate
rejects duplicated token facets, direct-interface dependencies on
first-class-callable ABI artifacts, missing reverse dependencies, and
signature/body disagreement.

Mutations remove one nested-callable correspondence, reverse only one
cooperative edge, pair leaves by name instead of structural position, or let
an adapter emit its provider-local callback shape. Each must fail strict
typechecking, the exact artifact-shape assertion, differential behavior, or
the unrelated-callable byte-stability check.

Production mutations catch transfer without copy, LIFO queueing, close that
drops buffered values, incorrect closed-channel `ok`, eager receive-LHS
evaluation, duplicate select commit, uncanceled alternatives, default beating
ready communication, split direct/select queues that reorder live waiters,
source-order blocked-case registration, nil becoming ready, omitted
cooperative dependency,
all-function async conversion, a storage-shaped callable facet, scheduler
routing through the channel semantic builder, deadlock checking before Promise
settlement cleanup, swallowed goroutine panic, absent deadlock detection, and
routing a pure default-bearing select through the blocking Promise path.
Additional mutations remove the hidden-operation cooperative dependency,
leave a blocking generic capability synchronous, invoke a deferred Promise
without awaiting it, or widen every defer stack to Promise-returning; each
must fail its contract, strict-type, differential, byte-stability, or scaling
owner.

Static checks reject `any`, `unknown`, casts, reflection,
`.call`/`.apply`/`.bind`, dynamic imports, text fragments, runtime operation
tags, alternate task runtimes, and semantic spelling dispatch. Measurements
report source/generated bytes and AST nodes, typecheck time/RSS, generation and
runtime time, largest changed bodies, runtime size, O(1) send/receive sites,
and O(case-count) select growth.
Queue evidence additionally proves O(1) live-offer cancellation and storage
bounded by buffered values plus currently blocked operations, not historical
send/receive/select traffic.

### Milestone 3I Language-Closure Gate

Verification derives the selected toolchain's complete `go/ast`, `go/token`,
and `types.Universe` domains at test time. It exact-joins every non-recovery
declaration, expression, and statement form to the one root dispatcher,
fingerprints upstream shapes and identities, and classifies type syntax,
clauses, fields, comments, and punctuation as parent/toolchain-owned rather
than adding duplicate root handlers. Mutations remove a selected arm and add a
neighboring-category arm; each must fail the exact join.

Valid construct fixtures must contain every selected AST form and every
semantic operator token. Command-package proof includes alias, dot, and blank
imports, local scopes, comments, and reached blank-import initialization.
Every supported predeclared function is selected by exact `*types.Builtin`
identity and passes TS-Go encode/print plus strict typecheck. `print` and
`println` must instead return their exact typed environment boundary in both
ordinary and deferred contexts; changing source spelling while retaining
checker identity must not change that decision.

Bodyless functions must return an unresolved obligation carrying the exact
`*types.Func` and `*types.Signature`, not a guessed body. Number-profile `/`,
`%`, `/=`, and `%=` must share one runtime owner, truncate toward zero, preserve
remainder sign, and enter the common Go panic ABI on zero; Go-versus-generated
ESM tests cover positive, negative, expression, compound-update, and panic
paths, including signed-zero normalization.

Blank-identifier proof is a contextual disposition matrix rather than an
identifier or AST-kind fixture. It covers package and local blank constants and
types, blank functions and methods, ordinary and literal parameters, receivers,
generic type parameters, results with bare returns and defers, variables,
assignments, range targets, and multi-result targets. Direct TS-Go AST
assertions prove that compile-time blanks own no declaration, required callable
slots use deterministic target-only identifiers, blank results retain exact
zero/return behavior, and discard targets retain all evaluation. Differential
execution covers repeated blanks in one scope, `iota`, side effects, tuple
position, and interface calls. Mutations that reserve a compile-time blank,
drop a required callable slot, expose a blank result as a Go binding, omit a
discarded RHS, or preallocate a blank method must fail at the owning gate.
Verification exact-joins every selected-corpus blank occurrence to one of these
parent-role dispositions; node-kind coverage alone is insufficient.

The gate reports actual file/byte/encoded-node/largest-artifact values and
enforces absolute bounds. It broad-searches for production inventories,
alternate dispatch, raw target text, forbidden dynamic/erased operations, and
stale unsupported paths. Environment implementations remain outside this
milestone.

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
