# Delivery

## Milestones Are Not Compiler Phases

The compiler always uses the one direct architecture described in
`architecture.md`. The milestones below sequence implementation and proof; they
must not become inventory, semantic-model, planning, or lowering stages.

## 0. Clean Baseline

Retire the previous implementation and contradictory authority. Preserve its
history only on pushed `archive/*` branches.

Exit:

- repository contains governance, this specification, license, and empty Go
  module metadata only;
- no old compiler, tests, fixtures, dependencies, or generated artifacts;
- direct architecture and forbidden paths are stated consistently;
- documentation links and policy identity pass.

## 1. Target Contract And Minimal Loader

Pin the TS-Go schema, generate typed Go bindings/factories, install the sole
formatter adapter, and load one selected Go package with one coherent
`go/types` graph.

Install the contextual dispatcher and coverage test. Unsupported constructs
fail explicitly. Prove one trivial declaration end to end:

```go
func Add(left, right int) int { return left + right }
```

The output must be TS-Go-AST-built, strict-typechecked, executed against Go,
and size-attributed.

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
