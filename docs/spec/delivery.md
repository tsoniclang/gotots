# Delivery

## Milestones Are Not Compiler Phases

The compiler always uses the one direct architecture described in
`architecture.md`. The milestones below sequence implementation and proof; they
must not become inventory, semantic-model, planning, or lowering stages.

## Construct-Driven Development

Delivery proceeds by construct case:

```text
focused Go test and minimal fixture
    -> observed typed unsupported failure
    -> one semantic-owner handler and closed child contract
    -> typed official TS-Go protocol AST
    -> pinned TS-Go print/reparse and strict typecheck
    -> Go/generated-ESM differential execution
    -> mutation and applicable cost proof
```

The test is written and run before production support. A semantic owner may
close several inseparable cases atomically, such as parallel-assignment
evaluation, but must not silently admit neighboring contextual variants.

Production directories follow:

```text
internal/emit/<domain>/<semantic-owner>/<sub-owner-as-discovered>
testdata/constructs/<domain>/<semantic-owner>/<case>
```

Neither hierarchy is pre-populated. New nesting records an evidenced ownership
boundary, not an attempt to predict every Go construct.

## 0. Clean Baseline

Retire the previous implementation and contradictory authority. Preserve its
history only on pushed `archive/*` branches.

Exit:

- repository contains governance, this specification, license, and empty Go
  module metadata only;
- no old compiler, tests, fixtures, dependencies, or generated artifacts;
- direct architecture and forbidden paths are stated consistently;
- documentation links and policy identity pass.

## 1. Native Target Contract And Minimal Loader

Pin one exact TS-Go tool revision and its official external AST schema,
protocol, encoder contract, and `printNode` API. Generate total node-specific Go
bindings/factories and the exact binary encoder. Start a persistent pinned
`tsgo --api --async` process and prove that TS-Go's real decoder, factory, and
printer accept and print a minimal source file.

Do not create a local formatter, generic target node tree, hand-maintained
target profile, inferred wire format, or fork of TS-Go internals.

Load one selected Go package with one coherent `go/types` graph.

Install the contextual dispatcher and coverage test. Unsupported constructs
fail explicitly. Prove one trivial declaration end to end:

```go
func Negate(value bool) bool { return !value }
```

The first accepted artifact includes a GoToTS-owned support module and a
generated package module, both constructed as TS-Go AST and printed by TS-Go:

```ts
// runtime/scalars.ts
export type bool = boolean;
```

```ts
// modules/example/logic.ts
import type { bool } from "../../runtime/scalars.js";

export function Negate(value: bool): bool {
    return !value;
}
```

The basic representation owner may also request `int32` or `int64` according
to the loaded `types.Sizes`. The default compilation-wide integer
representation is `number`, preserving the selected Go-width name in target
source:

```ts
export type int32 = number;
export type int64 = number;
```

The CLI may instead select the `bigint` representation for the complete
dependency closure, in which case the aliases target `bigint` and integer
literals use BigInt syntax. Neither initial representation reproduces implicit
fixed-width Go overflow. The default also accepts JavaScript's 32-bit coercion
for wider bitwise and shift operations; the `bigint` override owns exact wide
bitwise behavior. Those boundaries are deliberately outside the default
integer contract rather than being scattered through ordinary arithmetic.
Every checker-valid integer constant within its selected Go carrier remains
compilable in either profile: `number` emits the canonical decimal directly,
including magnitudes beyond JavaScript's safe-integer range, while `bigint`
emits the exact BigInt literal. The default does not add per-use wrapping,
casts, helpers, or safe-integer rejection for behavior already outside its
declared exactness boundary.
Explicit narrowing conversions and a future fixed-width profile require their
own complete construct proof.

The independent evaluation-order selection defaults to `direct`. It emits
direct target expressions and does not add temporaries solely because a keyed
composite's source order differs from positional constructor order. The CLI may
select `preserve-go`, which captures those values in Go source order before
construction. `direct` is intentionally not exact when reordered expressions
have observable effects; reports must name that boundary. There is no
purity/call heuristic and no per-site automatic mode.

Bootstrap in dependency order:

1. pinned TS-Go tool/contract test;
2. generated binding and upstream-encoder differential test;
3. native `printNode` round-trip test;
4. loader coherence test;
5. unsupported-dispatch test; and
6. the first construct test, observed failing at that unsupported boundary.

The output must be built as typed official TS-Go protocol AST, printed by
TS-Go, strict-typechecked, compiled to ESM JavaScript, executed directly against
Go within the selected representation contract, and size-attributed. Evidence
must state the selected integer representation and evaluation order and may not
describe overflow, wide-number behavior, or reordered effectful expressions as
exact when that behavior is outside the selected profile.

## 2. Core Direct Emission

Implement declarations, types, expressions, statements, lexical scopes,
imports, deterministic names, placement, evaluation order, and multiple
results directly from Go AST/type evidence. Add the identity-keyed demand
scheduler for executable, API, test, extension, initialization, function-value,
and callback roots without constructing a parallel source graph.

Before later semantic families add more use-dependent target obligations,
install the generic pre-seal artifact lifecycle: semantic-owner reconstruction,
mechanically projected observable TS-Go contract facets, typed consumer edges,
exact unchanged suppression, deterministic transitive propagation, atomic
replacement of requests/dependencies, convergence failure, and seal gating.
The existing named-struct static-operation requirements are the first real
producer; direct function/type/value references are the first consumers. This
coordination remains target assembly and must not become a source or semantic
IR.

Close each construct family independently. The exit boundary includes:

- non-generic, non-variadic function declarations, function literals, and
  unnamed function types over already-supported value representations;
- function values used as parameters, locals, arguments, results, direct
  callees, and lexical closures, with reached declarations scheduled by exact
  `go/types` object identity;
- unnamed and named result variables, including exact zero initialization and
  bare returns where every result is named;
- explicitly typed local constants without `iota` or inherited expressions;
- expression and expressionless switches without fallthrough; and
- a single-binding short declaration, single-target supported assignment,
  `++`/`--`, or discarded direct call in each Go-legal `if`, `switch`, and
  three-clause `for` simple-statement position. A scoped `if`/`switch`
  initializer may retain an already-supported multi-statement transaction;
  a `for` header rejects any form that cannot remain one target initializer or
  expression without prerequisite statements.

Function nil values, defined or declared-alias function types, variadics,
generics, method values/expressions, interface dispatch, strings and
containers, `defer`, `go`, labels, `goto`, fallthrough, range, type switches,
and other runtime/value-model families remain Milestone 3 obligations. Their
absence does not justify a helper or erased callable protocol in Milestone 2.

Finish with several ordinary multi-package programs that exercise direct calls,
callbacks, returned closures, named results, package initialization, and
cross-package demand without a runtime/manual/external dependency.

Before proceeding beyond the constant/float/rune checkpoint, close the
constant-context subsystem as one end-to-end capability:

- bare named constants use one identity-based owner for identifier,
  package-selector, and dot-import routes;
- concrete projection type comes from the occurrence when concrete, otherwise
  from a validated owning-parent expectation, never `types.Default`;
- enclosing checker-constant expressions such as `Scale + Scale` materialize
  their folded value without runtime child evaluation;
- local projections remain at their lexical `const` declaration;
- cross-package projection imports are keyed by package constant identity plus
  representation;
- whole-file and exported-Go-API roots retain the compile-time-only disposition
  of unused untyped constants, while an explicit constant root must name and
  materialize one concrete representation; and
- float32 operation rounding, the complete admitted float operator matrix, and
  rune value materialization have focused shape, strict-type, differential,
  mutation, and cost evidence.

## 3. Go Semantic Families

Add exact representations for:

- values, copies, zeros, pointers, maps, slices, arrays, and strings;
- methods, embedding, classes, interfaces, assertions, and equality;
- generics and constraints;
- initialization, defer/panic/recover, concurrency, channels, and `select`;
- reflection-visible and `unsafe`/`cgo` boundaries where supported.

Each family must satisfy the source-shape and cost gates before the next
dependent family proceeds. Interface dispatch must remain constant-size.

### 3A. Value-Model Foundation

The first Milestone 3 checkpoint installs six bounded families together
because they share zero, copy, assignment, equality, nil, bounds, and runtime
support ownership. The families remain separate semantic owners; they do not
create a value IR or a generic operation registry.

1. Every predeclared integer type receives a width-preserving GoToTS-owned
   target alias and the operators admitted by the selected `number` or
   `bigint` profile. The default `number` profile emits direct JavaScript
   bitwise and shift operators; 8/16/32-bit carriers normalize while wider
   carriers explicitly accept JavaScript's 32-bit bitwise coercion. The
   `bigint` profile owns exact wide bitwise and shift behavior. Fixed-width
   implicit overflow remains a separately selected future profile rather than
   hidden ordinary-expression baggage. Both profiles use constant-size checked
   division and remainder operations.
2. Go strings use one byte-preserving target representation. Source literals,
   concatenation, ordered/equality comparison, `len`, indexing, two-index
   slicing, integer/rune encoding, and byte/rune slice conversions are exact
   over arbitrary Go string bytes. String range, formatting, and external text
   encoding remain separate boundaries.
3. Fixed-length arrays of admitted recursively represented elements support
   zero, copy, equality, literals, indexing/stores, `len`, and `cap`. Target
   types retain the array length. A defined array has one minimal nominal
   wrapper around the same exact array storage; aliases add no wrapper.
   Aggregate-element zero, literal, and copy operations are requested only
   when the element representation requires structural copying, while the
   scalar-only runtime artifact remains byte-identical. The compiler emits
   direct typed loops that call the selected element operations; runtime array
   values receive no semantic function.
4. Unnamed slices of admitted recursively represented elements support nil,
   `make`, literals, indexing/stores, two- and three-index slicing, `len`,
   `cap`, `append`, and `copy`. The representation preserves backing-store
   aliasing, offset, length, capacity, append reallocation, fresh zero values,
   aggregate-element copying, and overlap-safe copy. Aggregate operations are
   demand-selected; scalar-only slice artifacts do not acquire aggregate
   methods, strategy fields, or helper declarations. Aggregate behavior is
   emitted as direct typed TS-Go AST at the source operation and uses only
   structural descriptor members. Defined slice identity remains a separate
   named-type obligation.
5. Unnamed maps with represented scalar comparable keys and represented scalar
   values support nil, `make`, literals, lookup, comma-ok lookup, stores,
   `delete`, and `len`. Missing lookup returns the exact value zero; writes to a
   nil map fail at runtime; map iteration is deferred from this checkpoint to
   Milestone 3C.
6. Pointers whose element has an admitted complete value representation
   support nil, `new`, dereference read and store, assignment,
   return/argument passing, and canonical-location equality. The current
   `new` matrix covers represented primitives, pointers, scalar slices, scalar
   maps, fixed arrays, and named structs. Function zero/nil behavior is not
   admitted, so `new(func(...))` remains a typed failure with that whole
   callable-nil family. Demand-driven addressability covers locals,
   parameters, named results, receivers, package variables,
   direct/indirect named-struct fields, fixed-array elements, slice elements,
   composite literals, and `&*pointer`. Only exact addressed variables become
   cells. Package `init` bodies use the same reconstructible artifact
   lifecycle. Direct pointer-receiver calls and value-receiver calls through
   pointers preserve Go nil, addressability, and copy behavior. Interface
   dispatch remains its separately proved family.

The generated runtime boundary is demand-driven and typed. Each family owns
its runtime module and exported operations; source handlers request imports by
closed semantic identity. One runtime-symbol contract owns every symbol's
module, export, type/value phase, and closed dependencies. Program assembly
computes the cycle-free transitive closure, emits each symbol once, and creates
canonical static ESM imports between runtime modules. Runtime files are
constructed as TS-Go AST and printed by the pinned TS-Go printer.

Runtime modules and primitive aliases are assembled as one local
`@gotots/runtime` package rooted at `runtime/`. Its explicit package exports
are generated from the exact emitted module set. The selected `gostdlib`
contract contributes its closed runtime-symbol and primitive-alias
requirements before closure, and its peer imports resolve to the same physical
modules used by generated source. The provider test runtime is generated by
this owner; a handwritten substitute or copied runtime class is not an
acceptable fixture.

All admitted runtime failures enter one generated non-generic `GoPanic`
carrier. `runtime/panic.ts` is the only runtime module that creates and
initially throws that carrier; integer, string, pointer, array, slice, and map
modules request it through the dependency graph. The function-control
milestone adds source `panic`/`recover`, exact recovered runtime-error
contracts, and a distinct canonical `*runtime.PanicNilError` dynamic identity.
A callable envelope may rethrow an unrecognized host exception unchanged, but
may not interpret it as a Go panic.

One builtin-object dispatcher owns `new`, `make`, `len`, `cap`, `append`,
`copy`, `delete`, and `clear`. One accessor-store transaction owns Go evaluation order
for array, slice, and map stores, including the captured getter/setter location
used by compound defined-value updates. Family owners provide typed operands
and members; they do not rediscover builtin identity or install assignment
routes.
No checked-in TypeScript, source fragment, template, raw export spelling,
handler-local duplicate implementation, or family-specific store transaction
is allowed.

The expression-completion checkpoint additionally treats a variadic signature
as one represented Go-slice parameter. Calls perform Go's tuple adjustment
before variadic packing, project defined slices for `slice...`, and never use
host argument spreading for a source slice. `append` spread follows Go
element-identity admission rather than slice assignability and includes the
language-defined `[]byte` plus string case. `new(x)` evaluates `x` once and
initializes a fresh pointer cell with the selected value copy; `new(T)` remains
the distinct type-form zero-value operation.

The same checkpoint admits represented struct conversion through one
demand-owned `$convert` member on the destination class and slice-to-array
value conversion through one source-site typed loop. It also treats checker-
constant `len`/`cap` as non-evaluating expressions, including array and
pointer-to-array operands, while nonconstant pointer-to-array expressions are
evaluated exactly once. Pointer reinterpretation and slice-to-array-pointer
conversion use the canonical pointer-storage representation. Ordinary
named-struct pointers instead use the class object directly as `S | undefined`;
`new(S)`, `&S{}`, `&local`, pointer equality, field access, and receiver calls
therefore add no carrier or storage surface. An addressed struct local retains
one stable object and whole-value stores copy into it. A pointer carries
distinct logical and storage type arguments only when a scalar/interior
location or exact conversion proves that direct class identity is
insufficient. Value-family owners then project and restore storage at the typed
load/store site, while the pointer runtime owns only address identity and
storage access. Named/generated structs reconstruct to a storage-backed private
layout only when that facet is demanded. Casts, erased payloads, shape tests,
and semantic read/write adapters remain forbidden. A defined array's canonical
storage is its underlying `GoArray`, so its nominal wrapper never grows
pointer-only forwarding methods.
Slice-to-array pointers are offset-aware aliases of existing slice backing,
preserve nil-versus-empty behavior at length zero, panic before construction
when short, and copy only when Go later assigns an array value through the
pointer.

Open generic declarations complete the same representation model before this
checkpoint exits. A source type parameter always has one logical target
parameter and gains distinct whole-storage, container-storage, and pointer
facets only from exact reached uses. Array/slice element address formation uses
one closed generic operation whose concrete capability always selects the
canonical location carrier; a rebindable slot cannot be represented by the
class object currently stored there. Concrete instantiations supply every
demanded facet from the one representation owner, nested generic declarations
forward them in canonical order, and the artifact fixed point rejects missing,
duplicate, reordered, conflated, dual-representation, or nonconvergent facet
contracts. No target conditional type, erased descriptor, runtime semantic
callback, or speculative all-facet signature is accepted.

The checkpoint exits only when all six families:

- use the one `go/types` graph and the existing parent-directed walk;
- install one representation owner and migrate every admitted context;
- retain typed unsupported failures for every named neighboring boundary;
- strict-typecheck and execute differentially in ordinary multi-package
  projects;
- prove nil, aliasing, copy, bounds, evaluation order, and mutation behavior;
- keep every operation/use site constant-size and runtime modules linear in
  exported behavior; and
- pass an integrated mixed-family, multi-package project through strict
  typechecking and Go-versus-generated-ESM execution for ordinary results and
  every admitted runtime-failure class without compatibility paths.

The addressability extension additionally exits only when a declaration with
no addressability or pointer-identity demand has a byte-identical TS-Go
artifact when address sites are added elsewhere, storage reconstruction
converges, its unchanged callable contract triggers zero reverse-consumer
reconstructions, and package `init` uses no non-artifact requirement path.

### 3B. Defined And Recursive Values

The next value checkpoint completes aliases and defined types whose underlying
value families are represented here, anonymous and legal recursive structs,
recursive arrays/slices/maps, aggregate map keys and values, and
nil-capable/defined callable values. Generic non-struct defined families use
the same nominal owner with exact declaration parameters and instantiated
target arguments. Interfaces and channels retain their later explicit
boundaries.

It extends the existing family owners rather than introducing a value IR,
generic operation registry, runtime strategy object, or second type-identity
model:

- exact `*types.TypeName` identity owns each defined nominal class; aliases add
  no runtime owner;
- `types.Identical` owns anonymous-struct canonicalization, with fingerprints
  used only for indexed lookup and collision-checked artifact naming;
- zero/copy/equality/hash/address capabilities are static demanded members
  owned once per exact represented type;
- aggregate-map lookup/store/delete use one canonical statically specialized
  owner per represented map shape; map instances carry storage, never semantic
  callbacks;
- every public map contract uses semantic represented key/value types, while
  key projection and reification remain private to the exact native or
  generated map owner;
- nil callable values use `undefined`; defined non-nil callables use one
  minimal wrapper whose `$value` is invoked directly after the nil guard; and
- local component types constrain generated declarations to the highest scope
  where every referenced type is legally nameable.

This checkpoint exits only when recursive requirements converge, scalar
artifacts remain byte-identical, definitions grow `O(type shape)`, use sites
remain constant-size, aggregate stores/lookups preserve Go copies, and the
twenty largest changed type and operation artifacts pass strict, differential,
mutation, source-size, typecheck, runtime, and broad-deletion review.

### 3C. Structured Control And Range

Complete classic `for`, expression switch, local declaration transactions,
empty statements, labeled break/continue, and range over arrays,
pointer-to-arrays, slices, strings, maps, and integers. Channel and iterator
range remain with their semantic dependencies.

This checkpoint replaces narrow header-only and primitive-switch boundaries
atomically. Parent statement owners select direct target forms when exact and
construct one source-proportional structured form otherwise. It introduces no
control-flow IR, generic statement visitor, unrolled collection loop, or
parallel clause-emission route.

Exit requires both integer profiles to pass typed TS-Go encode/print, strict
TypeScript, and Go-versus-generated-ESM differential tests for:

- constant and evaluated array-range operands, copy versus alias behavior,
  per-iteration addresses, blanks, assignment targets, and integer limits;
- UTF-8 string byte indexes including invalid encodings;
- map nil/deletion/copy behavior using a key snapshot plus live lookup;
- direct and prerequisite-bearing loop clauses, including labeled continue;
- direct and custom-equality expression switches, ordered case prerequisites,
  default placement, and fallthrough; and
- local multi-result and blank declaration transactions.

Generated loop size is independent of collection length, switch construction is
linear in source cases, each body is emitted once, exact label identity ignores
source-spelling mutations, and the superseded target-header-only result API is
absent.

### 3D. Concrete Methods, Embedding, And Promotion

Complete methods on represented non-generic named types, named and anonymous
embedded fields, promoted field reads/stores/addresses, promoted concrete
calls, method values, and method expressions.

One selection-path owner consumes exact `go/types.Selection` evidence for all
of those contexts. Reached receiver bodies are immutable typed class-member
contributions assembled into the exact declaring type's one reconstructed
class, including when source method and type declarations are in different
files. Value receivers are instance members; pointer receivers are class-owned
static members with an explicit selected pointer parameter. Embedding remains
class-field composition; ordinary concrete calls select the exact Go owner
before member invocation and never become accidental target virtual dispatch.
Method values capture their selected receiver once, while method expressions
use a typed native-member arrow, a direct static-member reference, or one typed
adapter when promotion/receiver adjustment requires it.

This checkpoint exits only when:

- every `types.SelectionKind` used by the admitted concrete family has one
  contextual owner;
- selection kind, object identity, receiver/result types, and complete index
  path are exact-joined to checker evidence;
- value-receiver copy, implicit address-taking, embedded-pointer dereference,
  and nil panic timing match Go;
- source-spelling mutation is byte-stable and mismatched selection identity
  fails closed;
- generated output contains no class `extends`, top-level receiver twin,
  prototype patch, `.call`, `.apply`, `.bind`, erased payload, or implementer
  switch;
- each use is constant-size apart from its selected embedding depth, and
  1x/2x/4x depth fixtures grow linearly; and
- both integer profiles pass TS-Go encode/print, strict typechecking, and
  Go-versus-generated-ESM differential execution.

### 3E. Interfaces

Install one typed adapter per reached concrete dynamic type, canonical
non-string dynamic-type metadata, contract-demanded native methods, O(1)
dispatch, assertions, comma-ok, type switches, interface equality, and
interface map keys. Concrete receiver calls remain statically selected
class-owned members. Concrete conversions seed their exact target contract;
interface conversions and assertions propagate that target only from a source
contract already reachable on an adapter, with `go/types` proving the target.
Implementing a source contract without reaching it never widens the adapter.
Exit requires the complete
nil/typed-nil/copy/assertion/panic/equality matrix, constant-size call sites as
implementer count grows, adapter methods equal to the exact union of demanded
contracts, and zero erased payload, complete-concrete-method-set expansion,
constructor-identity, reflection, or implementer switches. Repeated identical
transition occurrences must not rescan all adapters or reschedule an already
admitted adapter/contract pair.

### 3F. Generics And Iterator Functions

Install generic functions, aliases, named types, instantiated methods,
explicit/inferred instantiation, instantiated function values, type-parameter
operations, and range-over-function.

One source declaration produces one strict target generic body. The body emits
typed operation-signature requirements into the existing
declaration-reconstruction fixed point. Repeated identical operation selections
exact-join one hidden function parameter. Concrete instantiations supply one
canonical function artifact per exact operation-selection/signature, and
generic callers project and forward the same typed functions.
`go/types.Info.Instances` and selected method/type evidence are the only
inference and constraint authority; constraint-method selections retain the
exact selected `*types.Func`.
There is no monomorphized alternate path, source-position identity, capability
object, source prewalk, copied type-set algebra, universal operation bag,
runtime registry, or erased payload.

Exit requires exact instance/capability joins; recursive generic convergence;
generic aliases and recursive instantiated types; generic receiver methods and
function values; operations over basic, defined, aggregate, interface, and
constructed type arguments; all three iterator signatures; and strict,
differential, mutation, tail-size, and scaling proof. Generic body bytes remain
constant as instantiation count grows, and capability definitions grow only
with distinct exact operation-selection/signature pairs.

### 3G. Function Control, Defer, Panic, And Goto

Install source `panic`, exact direct-call `recover`, immediate defer
registration with Go copies, LIFO unwind, named-result mutation, replacement
panic behavior, exact labels, and arbitrary valid goto. The exact callable
artifact owns one demand-selected target assembly; no control IR, CFG, source
inventory, or alternate walker is introduced.

Exit requires direct and function-valued calls, receiver and interface method
values, generic functions, nested function literals, ordinary and named
returns, panic replacement and recovery directness, labels, fallthrough,
labeled break/continue, direct structural goto, non-structural goto, and
goto/defer/range/switch/type-switch composition. Block and clause statement
lists share one control owner, and nested generated dispatch must not capture
an unlabeled source break or continue. The only exception carrier is
`runtime/panic.ts`; recovery authority is invocation-local and typed.
Deferred callable-value invocation consumes the one exact-signature callable
ABI installed by the callable-storage milestone: ordinary invocation omits its
optional recovery facet and deferred invocation supplies it. Function-control
must not create an ABI identity per variable, field, pointer, container, or
adapter, and must not introduce a storage/dataflow graph.
Functions without control demand remain byte-identical. State-machine size is
linear in source statements and is absent when direct target control suffices.

### 3H. Channels And Cooperative Concurrency

Add the closed concurrency-semantics profile axis, disabled by default and
explicitly selecting `cooperative`. Install separate typed channel-value and
scheduler-lifecycle truth owners that may assemble into one output module,
plus exact channel type/direction projection, `make`,
send/receive/close/equality/range, goroutine launch, and atomic `select`.
The channel owner uses one insertion-ordered typed live queue per direction
for both direct operations and selected alternatives, fair blocked-select
registration, O(1) cancellation, select-to-select rendezvous, and storage
bounded by live operations; separate direct/select queues or listener
registries do not satisfy this checkpoint.

Concrete bodies retain source callable facets. First-class function values use
one exact-signature generated callable ABI artifact independent of every
storage shape. Closed cooperative requirements on the existing artifact graph
make only exact blocking consumers and selected ABI contracts
Promise-returning; recursive cycles converge. Cover functions, methods,
locals, parameters, results, package variables, fields, pointers,
arrays/slices/maps, interface assertions/calls, and generic aggregates without
a call graph, prewalk, storage-facet hierarchy, erased queue, all-function
async tax, or yield heuristic.

Checker-produced package initializers are exact callable facets under their
existing initializer artifact identities. A cooperative initializer or source
`init` function makes only its passive package `$initialize`
Promise-returning, and the ESM program-initialization module awaits only those
package calls in Go order. No synthetic initializer function, package-wide
heuristic, or unconditional top-level await is admitted.

The same cooperative facet applies to hidden generic constraint-method
functions and deferred invocations. A concrete blocking constraint method
reconstructs its exact hidden operation function, the generic caller, and only
their reverse consumers. A deferred blocking call is captured immediately,
stored as a typed async defer entry, and awaited in LIFO order before function
exit; recovery authority remains invocation-local across the await. Neither
case may introduce a call graph, erased defer stack, or unconditional async
tax. Generic source declarations may have demand-created callable-profile
variants only when distinct reached instantiations require different static
callable ABIs. Such variants are keyed by canonical source and ABI-facet
identity, reconstructed by the ordinary source handler, and selected directly
at calls, deferred calls, function values, and generic method
calls/values/expressions. Declaration-owned callable cooperation propagates
directionally to each corresponding concrete ABI without creating a duplicate
variant; concrete call-site cooperation never widens the declaration
baseline. Declaration-wide widening, runtime Promise detection,
`T | Promise<T>` results, and per-call wrappers are forbidden.
Lexically nested callables inside a selected variant use profile-local
source-plus-AST facet identities. Their cooperation propagates only to the
structurally corresponding closed callable ABI. Named values containing a
callable representation that can vary independently of their declared Go type
arguments use one hidden, defaulted value-facet type parameter so each profile
carries its exact represented underlying callable type. The declaration
origin is the sole arity owner; transitive callable fields reached through a
type argument or nominal struct do not add a facet. Exit rejects globalized
nested literal facets, open-ABI widening, result intersections, casts,
instantiation-derived declaration arity, and hidden facets added to unrelated
named values.
If the source declaration is exported, package assembly re-exports every
reached variant from its owning source module. Export selection comes from the
same accepted profile requirements as declaration reconstruction; no consumer
import may name a binding absent from the package value surface.

Source-unavailable standard-library and external generic owners emit
demand-created ambient profile declarations through the environment-contract
owner, not source-body variants. Each declaration exact-joins the selected
nested callable ABI profile and deterministic name used by its consumers,
contains no body, and remains an explicit implementation obligation.
Environment callable effects are provider-owned and are never guessed from a
callable parameter or result. Exit includes `slices.Values` with a cooperative
range callback, strict typechecking of the consuming source, one declaration
per distinct reached profile, and mutations that route the owner through the
source emitter, drop the profile declaration, widen the base declaration, or
infer an outer Promise solely from nested type shape.

Immediately invoked function literals use their exact literal facet and bypass
the first-class callable ABI; transported literals still adapt through the ABI.
Exit includes synchronous and cooperative immediate invocations, a mutation
that restores ABI routing, strict output-shape inspection, and byte stability
for the synchronous case.

Exit requires the complete Milestone 3H differential, mutation, staticness,
artifact, scaling, runtime-cost, deadlock, panic, and synchronous-byte-stability
evidence under the selected race-free cooperative profile. A pure select with
a default must remain synchronous and scheduler-free; only a select without a
default may select the blocking operation, unless operand evaluation
independently requires cooperation.

### 3I. Language Closure

Derive the selected Go toolchain's AST forms, tokens, and predeclared
identities in verification code and exact-join them to production root
dispatch. This is not a production inventory. Parent-owned type/clause/field
forms remain with their semantic owners, parser-recovery forms remain rejected,
and every semantic operator and predeclared function has one admitted profile
or typed boundary.

Close package aliases, dot and blank imports, command packages, local scopes,
ordinary source comments, number-profile integer division/remainder, and the
selected bodyless-function and implementation-defined built-in boundaries.
Language closure identifies exact unresolved environment obligations but does
not install standard-library, external, `print`/`println`, cgo, or `unsafe`
implementations.

The blank identifier is classified by its parent-owned semantic role, never by
spelling alone. A blank constant, type, function, or method is checked but owns
no target definition or target name. A blank value parameter, receiver, or type
parameter preserves its target signature slot under a deterministic target-only
identifier that is not a Go binding. A blank result preserves result arity,
type, zero initialization, bare-return, and defer behavior without creating a
source-visible binding. A blank variable, assignment target, range target, or
multi-result component preserves evaluation, conversion, copy, ordering, and
tuple position while omitting only the final store. No `types.Object` for `_`
may enter ordinary declaration reservation or reference lookup.

Exit requires exact selected-universe/dispatch joins, valid construct fixtures,
strict TS-Go-printed ESM artifacts, focused Go-versus-TypeScript differentials,
missing/widened-dispatch and spelling mutations, zero unknown valid in-scope
forms, no duplicate owner, and frozen source/generated/typecheck/runtime
bounds.

## 4. Environment And Completion

Add deterministic module output, minimal runtime modules, selected-`GOROOT`
standard-library contracts, reusable manual `gostdlib`, true external
contracts, placeholders, structural manual completion, extensions, and
reachable-obligation checking.

Build `gostdlib` first as the independently typechecked and executable
`@gotots/gostdlib` ESM package. Its public subpaths mirror Go import paths and
contain ordinary named exports with Go declaration names. The first backend is
Node.js, but `node` does not occur in the public package name or subpaths.
Portable and host-dependent implementation ownership remains internal.

Before compiler linkage, exact-join every provider export and implementation
against selected-`GOROOT` declaration identity and signature, execute focused
Go-versus-provider differentials, and prove the package contains no placeholder
or opaque public ABI spelling. The provider must be installable and vendorable
under the same module specifiers. Linkage is then one atomic compiler change:
replace ambient standard-library imports with namespace imports from
`@gotots/gostdlib/<go-import-path>.js`, migrate receiver and package-state
references, delete the ambient standard-library runtime path, and rerun the
whole-product gates. Do not combine an incomplete provider with compiler
linkage.

Before that atomic linkage, close requirement liveness and provider effects:
artifact reconstruction replaces its complete declaration-requirement set,
final-consumer removals wait until additions, reachability, dirty
reconstruction, and package initialization are quiescent, and removal
authority follows only exact changed artifact-dependency facets until each
affected reconstruction consumes it. A later-discovered consumer before the
sweep cancels an orphan without provider churn; later reintroduction after a
committed removal remains a convergence error. The manifest records the exact
outer effect of every reached profile, and the provider supplies all certified
private facets. The compile-only ambient profile remains separately selectable
but cannot satisfy executable-product proof. Linked compilation has no ambient
fallback.

Source-available dependencies continue through ordinary direct emission.
Reachable unresolved placeholders block publication.

The compile-only environment profile may materialize the typed throwing
placeholder for non-nil `unsafe.Pointer` conversion while preserving nil
exactly. This closes strict product typechecking; it is not an unsafe
implementation and cannot satisfy the no-placeholder publication gate.

## 5. Product Proof

Run broad project differentials, deterministic regeneration, source/toolchain
upgrade and relocation tests, generated-tail reviews, strict whole-product
typechecking, direct generated-program execution, runtime/performance gates,
and standalone installation tests with no undeclared external compiler or
target dependency.

Completion is one clean published revision with:

- every selected occurrence handled;
- no unknown construct or alternate translation path;
- no reachable placeholder;
- exact TS-Go AST ownership for every generated token;
- strict static TypeScript;
- differential behavior across selected projects;
- bounded source size, generated size, typecheck cost, generation cost, and
  runtime.

## Checkpoints

Commit and push coherent capability checkpoints. A checkpoint report states:

- exact revision and parent;
- construct families and contexts implemented;
- emitted/typechecked/executed denominators;
- artifact examples and twenty-item tail;
- mutations caught;
- source/generated/typecheck/runtime deltas;
- explicit unsupported boundaries;
- broad deletion/search results.

Do not report delivery-milestone percentages without naming the numerator and
denominator. Do not carry an earlier result forward after its owning boundary
changes; rerun it.
