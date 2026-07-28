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
// support/scalars.ts
export type bool = boolean;
```

```ts
// modules/example/logic.ts
import type { bool } from "../../support/scalars.js";

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
fixed-width Go overflow. That behavior is deliberately outside the initial
integer contract rather than being scattered through ordinary arithmetic.
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
   `bigint` profile. Division, remainder, shifts, bitwise operations, and
   explicit conversions are admitted only where the selected profile has an
   exact bounded implementation. Fixed-width implicit overflow remains a
   separately selected future profile rather than hidden ordinary-expression
   baggage. In this checkpoint, BigInt division and remainder use constant-size
   checked runtime operations; the `number` profile keeps them unsupported
   rather than approximating Go integer truncation.
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
   nil map fail at runtime; map iteration remains deferred.
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
   pointers preserve Go nil, addressability, and copy behavior; method values,
   method expressions, embedding/promotion, and interface dispatch remain
   their separately proved families.

The generated runtime boundary is demand-driven and typed. Each family owns
its runtime module and exported operations; source handlers request imports by
closed semantic identity. One runtime-symbol contract owns every symbol's
module, export, type/value phase, and closed dependencies. Program assembly
computes the cycle-free transitive closure, emits each symbol once, and creates
canonical static ESM imports between runtime modules. Runtime files are
constructed as TS-Go AST and printed by the pinned TS-Go printer.

All admitted runtime failures enter one generated `GoPanic<T>` carrier.
`runtime/panic.ts` is the only runtime module that constructs a target
`ThrowStatement`; integer, string, pointer, array, slice, and map modules
request it through the dependency graph. Source-level `panic` and `recover`
remain a later semantic family, so this checkpoint proves failure occurrence
and carrier identity, not yet recovered runtime-fault payload equivalence.

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
conversion use the canonical pointer-storage representation. A pointer carries
distinct logical and storage type arguments; value-family owners project and
restore storage at the typed load/store site, while the pointer runtime owns
only address identity and storage access. Named/generated structs reconstruct
to a storage-backed private layout only when the storage facet is demanded.
Casts, erased payloads, shape tests, and semantic read/write adapters remain
forbidden.

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
nil-capable/defined callable values. Interfaces, channels, and generic
underlying families retain their later explicit boundaries.

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
- nil callable values use `undefined`; defined non-nil callables use one
  minimal wrapper whose `$value` is invoked directly after the nil guard; and
- local component types constrain generated declarations to the highest scope
  where every referenced type is legally nameable.

This checkpoint exits only when recursive requirements converge, scalar
artifacts remain byte-identical, definitions grow `O(type shape)`, use sites
remain constant-size, aggregate stores/lookups preserve Go copies, and the
twenty largest changed type and operation artifacts pass strict, differential,
mutation, source-size, typecheck, runtime, and broad-deletion review.

## 4. Environment And Completion

Add deterministic module output, minimal runtime modules, selected-`GOROOT`
standard-library contracts, reusable manual `gostdlib`, true external
contracts, placeholders, structural manual completion, extensions, and
reachable-obligation checking.

Source-available dependencies continue through ordinary direct emission.
Reachable unresolved placeholders block publication.

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
