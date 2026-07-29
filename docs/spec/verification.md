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

Mutations remove a dependency edge, widen it to all facets, compare only a
hash, notify on equal contracts, ignore a changed contract, retain old
dependencies, process dirty work nondeterministically, mutate a provider node
through a consumer, or accept an oscillating cycle. Each must fail at the
contract, graph, lifecycle, determinism, ownership, or broad-search gate.
Measure declarations, graph vertices/edges, contract bytes, revisions,
reconstructions, final AST bytes, generation time, typecheck time/RSS, and
runtime. Current graph state must remain O(artifacts + consumed facet edges +
current contract bytes); convergence evidence additionally retains one exact
copy of each distinct changed contract and no entry for an unchanged rebuild.
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
byte-stable and direct, aliases add no wrapper, and distinct defined callables
remain statically incompatible. Mutations remove the nil guard, move it before
argument evaluation, replace the nominal wrapper with an intersection/string
brand, mutate the function object with `Object.assign`, or invoke through
`.call`; each must fail its owning shape, differential, or broad-search gate.

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
- concrete value-receiver calls use the exact `go/types.Selection` and a named
  receiver function, never a class method or virtual dispatch; and
- tags, pointers, interfaces, generics, and unsupported field representations
  fail at their typed owner.

Mutations replace a requested copy with direct assignment, replace requested
field equality with `===`, remove the private brand, add source-order captures
to `direct`, remove them from `preserve-go`, attach a receiver method to the
class, emit an unrequested static operation, duplicate an operation, make it an
instance member, emit a top-level helper, route it to the caller's file, or
admit an unsupported field. Each fails its owning structural, strict-type,
differential, placement, or unsupported-boundary gate. The
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
compilation-wide integer selection. The default `number` gate inspects direct arithmetic,
ordinary numeric literals, and the absence of routine casts, `Math.imul`, and
wrapping operators; its differential values remain inside the declared exact
number domain. The `bigint` gate inspects BigInt aliases and literals, executes
values beyond JavaScript's safe-number range, and proves that no number carrier
leaks into integer operations.

Mutations reintroduce a routine literal cast or redundant inferred annotation,
emit a wrapped default multiplication, mix representations across generated
files, or print a number literal in BigInt mode. Each fails its owning AST,
strict-type, differential, or profile-coherence gate. Reports explicitly state
that implicit fixed-width overflow is deferred; neither profile may be
described as proving it.

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
| maps | nil write, missing zero, comma-ok, aliasing, direct bool/integer/string keys, defined-key projection, delete/len, and explicit floating-key rejection | plain-object substitution; missing-value `undefined`; copy-on-assignment; wrapper object identity; floating-key admission through native SameValueZero |
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
2. only the panic runtime owner constructs a target `ThrowStatement`; replacing
   one family guard with a host exception or removing one dependency fails;
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
   large slice that would exceed a host spread-call limit;
3. `append` spread differentials for aliasing reuse, reallocation, two
   distinct defined slice types with identical elements, aggregate elements,
   and `append([]byte, string...)`;
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
7. package `init` is reconstructed by the ordinary artifact scheduler, with no
   non-artifact declaration requirement;
8. pointer-receiver calls on values and pointers, and value-receiver calls
   through pointers, preserve nil and copy behavior without class methods,
   `.call`, `.apply`, or `.bind`;
9. adding storage changes only the function body contract, causes zero caller
   reconstruction, and reaches a deterministic fixed point; and
10. 1x/2x/4x address sites keep each use constant-size and leave an unaffected
    declaration's generated TS-Go bytes unchanged;
11. the assignment package has no concrete storage or pointer-runtime import,
    while the root emitter installs exactly one typed storage capability.

Production-path mutations drop a storage requirement, key it by spelling,
select a shadow sibling, wrap every local, compare pointer wrapper identity,
mis-key a field/index projection, key a slice by descriptor identity, skip the
required `&*p` nil check or read its stored value, copy a pointer receiver,
omit the value-receiver copy,
bypass package-`init` artifact reconstruction, or dirty callers after an
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
3. a promoted call from `Base.CallName` remains `Base_Name(base)` even when the
   embedding type declares its own `Name`, proving no accidental virtual
   dispatch;
4. method-value formation evaluates and captures the receiver once, copies
   value receivers once, and preserves pointer identity;
5. nil-safe pointer receiver values may form and execute, while a value method
   selected through a nil pointer and a promoted field through a nil embedded
   pointer panic at the same boundary as Go;
6. direct method expressions are direct receiver-function references and
   promoted method expressions are typed arrows with an explicit first
   receiver parameter;
7. embedded fields remain owned class fields, including anonymous structs and
   reserved/unexported member names; generated support imports their defining
   source modules with collision-safe package-qualified aliases; and
8. generated artifacts contain no `extends`, `.call`, `.apply`, `.bind`,
   erased carrier, reflection, or per-method/per-implementer dispatch switch.

A 1x/2x/4x embedding-depth fixture independently measures source bytes,
printed target bytes, and encoded TS-Go nodes. Each use contains one selected
receiver-function call, and all three measures grow linearly with source
depth. Production mutations remove an index component, replace the selected
object, select by spelling, omit a value copy, skip an embedded-pointer nil
check, reevaluate a receiver, attach methods to classes, or route generated
private declarations through a public assembly; each fails its identity,
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
   and every native method directly invokes the exact top-level receiver
   function selected from `go/types`;
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
    executes alias-equivalent pointer lookups correctly.

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
