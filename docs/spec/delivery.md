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
   concatenation, ordered/equality comparison, `len`, indexing, and two-index
   slicing are exact over arbitrary Go string bytes. Rune conversion,
   iteration, formatting, and external text encoding remain separate
   boundaries.
3. Unnamed fixed-length arrays of represented scalar elements support zero,
   copy, equality, literals, indexing/stores, `len`, and `cap`. Target types
   retain the array length. Named arrays and recursively aggregate elements
   remain unsupported until their identity and copy contracts are proved.
4. Unnamed slices of represented scalar elements support nil, `make`, literals,
   indexing/stores, two- and three-index slicing, `len`, `cap`, `append`, and
   `copy`. The representation preserves backing-store aliasing, offset, length,
   capacity, append reallocation, and fresh zero slices.
5. Unnamed maps with represented scalar comparable keys and represented scalar
   values support nil, `make`, literals, lookup, comma-ok lookup, stores,
   `delete`, and `len`. Missing lookup returns the exact value zero; writes to a
   nil map fail at runtime; map iteration remains deferred.
6. Pointers to represented scalar values support nil, `new`, dereference read
   and store, assignment, return/argument passing, and equality. Address-of
   existing variables, fields, and indexed elements remains unsupported until
   stable addressable-storage identity can be installed without cell-wrapping
   unrelated locals.

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
`copy`, and `delete`. One setter-store transaction owns Go evaluation order
for array, slice, and map stores. Family owners provide typed operands and
members; they do not rediscover builtin identity or install assignment routes.
No checked-in TypeScript, source fragment, template, raw export spelling,
handler-local duplicate implementation, or family-specific store transaction
is allowed.

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
