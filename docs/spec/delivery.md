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
    -> Go/authoritative-consumer differential execution
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
func Add(left, right int) int { return left + right }
```

On a selected 64-bit Go target, the first accepted artifact is constructed as
TS-Go AST and printed as:

```ts
import type { int64 } from "@tsonic/core/types.js";
export function Add(left: int64, right: int64): int64 {
    return left + right;
}
```

The `int64` import is a selected-width target primitive contract, not a source
spelling heuristic. A 32-bit Go target requests `int32` instead. Plain
TypeScript `number`, unqualified `bigint`, and `BigInt.asIntN` are not accepted
substitutes: the first loses Go integer exactness and the latter two do not
carry the finalized Tsonic primitive evidence required by the first consumer.
The selected Tsonic virtual declaration may use `number` as its TypeScript
checker carrier while attaching an exact `int64` target fact. GoToTS preserves
the canonical imported type identity; it does not replace it with that carrier.

Bootstrap in dependency order:

1. pinned TS-Go tool/contract test;
2. generated binding and upstream-encoder differential test;
3. native `printNode` round-trip test;
4. loader coherence test;
5. unsupported-dispatch test; and
6. the first construct test, observed failing at that unsupported boundary.

The output must be built as typed official TS-Go protocol AST, printed by
TS-Go, strict-typechecked, executed against Go through the authoritative
consumer for the represented semantics, and size-attributed. Direct Node
execution is sufficient only for behavior whose selected Tsonic source carrier
has identical JavaScript semantics. Width, overflow, and other target-owned
primitive behavior require a selected Tsonic target build and native
differential execution; an ordinary-value Node check is supporting evidence
only.

## 2. Core Direct Emission

Implement declarations, types, expressions, statements, lexical scopes,
imports, deterministic names, placement, evaluation order, and multiple
results directly from Go AST/type evidence. Add the identity-keyed demand
scheduler for executable, API, test, extension, initialization, function-value,
and callback roots without constructing a parallel source graph.

Close each construct family independently. Finish with several ordinary
multi-package programs and no runtime/manual/external dependency.

## 3. Go Semantic Families

Add exact representations for:

- values, copies, zeros, pointers, maps, slices, arrays, and strings;
- methods, embedding, classes, interfaces, assertions, and equality;
- generics and constraints;
- initialization, defer/panic/recover, concurrency, channels, and `select`;
- reflection-visible and `unsafe`/`cgo` boundaries where supported.

Each family must satisfy the source-shape and cost gates before the next
dependent family proceeds. Interface dispatch must remain constant-size.

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
typechecking, runtime/performance gates, and the self-host/Tsonic bootstrap.

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
