# Direct Compiler Architecture

## Design Rule

Use the simplest architecture that remains exact and scalable:

```text
Go syntax/type truth                          TypeScript syntax truth
┌─────────────────────────┐                  ┌─────────────────────────┐
│ ast.File / ast.Node     │                  │ pinned TS-Go schema     │
│ types.Info / Package   │                  │ generated Go bindings   │
└────────────┬────────────┘                  └────────────▲────────────┘
             │                                            │
             └──────── contextual emitter ────────────────┘
```

The contextual emitter is a traversal and construction service, not a phase
pipeline. It may walk to relevant Go declarations, query the existing checker
graph, and inspect package relationships while handling a source occurrence.
It must not materialize those answers into a general-purpose intermediate
program.

## Authoritative Owners

| Concern | Sole owner |
|---|---|
| selected packages, files, overlays, build tags | `internal/load` using `go/packages` |
| Go syntax and source positions | standard `go/ast` and `go/token` |
| binding, type, constant, instance, method selection | one coherent `go/types` graph |
| contextual traversal and translation | `internal/emit` |
| target node kinds, fields, factories, visitor order | generated `internal/target/tsgo` |
| target lexical placement and deduplication | scoped builders in `internal/emit` |
| formatting | one TS-Go formatter adapter |
| output paths and atomic writes | `internal/output` |
| runtime/manual/external ownership | explicit contracts under their named roots |
| independent checks | `internal/verify` |

No second package may recreate one of these truths.

## Allowed Emission State

An emission session may retain:

- selected package/file references and the shared `types.Info`;
- a stack of source context and target lexical builders;
- deterministic mappings from typed Go objects to target names/declarations;
- already-created TS-Go AST nodes;
- typed import/helper/declaration placement requests;
- source-map/provenance links; and
- typed diagnostics.

It may not retain:

- copied source occurrences or a parallel parent graph;
- a catalog-produced inventory of the current program;
- normalized semantic operations;
- a target-independent semantic model;
- a representation or whole-program plan consumed by a later lowering pass;
- target source fragments; or
- a second checker-derived fact store.

A narrow memoized answer is allowed only when its key is the authoritative Go
object, its value is directly needed target state, and recomputation would
produce the same answer. It is a cache, not a new identity domain.

Representation consistency has one owner inside `internal/emit`. On the first
request for a Go type, object, or method, that owner queries the complete
selected Go AST/type graph, chooses the exact target form, and creates or
reserves its TS-Go declaration. Every later use retrieves that same target
state by authoritative Go identity. The decision is final before dependent
target nodes are emitted; it is never serialized into a plan or consumed by a
later lowering stage.

## Definition Scheduling

Emission is demand-driven from explicit executable, exported-API, test, and
extension roots plus Go package initialization roots. A resolved reference
enqueues its authoritative Go object for emission. The scheduler stores only
`pending` and `emitted` Go identities and direct links to their target
declarations; it does not copy call edges into a source graph.

Function values, interface conversions, callbacks, reflection contracts,
generic instantiations, initialization, runtime helpers, manual obligations,
and external contracts add work through explicit typed rules at the construct
that exposes them. A definition is emitted at most once per required target
specialization. Cycles reserve target declarations before bodies are emitted.

Consequently, every generated/manual/external obligation reached by the work
queue is product-reachable and must be resolved. Unselected declarations need
not become placeholders merely because they share a source package. A
library-generation request makes its selected public API an explicit root.

## Context And Placement

Each handler receives an immutable contextual view:

```text
source package/file/node
parent-supplied grammatical role
expected Go type and result arity
expression/statement/declaration position
lexical and control-flow boundary
current target scope builders
shared go/types evidence
target factories and placement service
```

Children do not walk upward to guess their role. Parents already know why a
child exists and pass that fact explicitly.

Handlers return TS-Go AST nodes. If a source expression requires additional
target statements, declarations, or imports, it returns typed placement
requests with:

1. the exact target nodes;
2. legal scopes;
3. the preferred scope;
4. the execution/evaluation constraint; and
5. a typed deduplication owner where repetition is legal.

One placement service applies the policy:

- imports always enter file import scope; dynamic imports are forbidden;
- reusable static declarations prefer file scope;
- function-wide declarations enter the function prologue only when their
  lifetime is function-wide;
- evaluation-dependent temporaries remain immediately inside the branch,
  loop, short-circuit arm, deferred closure, or statement where they execute;
- no request may cross a semantic execution boundary merely because a wider
  scope is syntactically legal.

This permits hoisting without maintaining a separate target IR.

Call and composite handlers consume child emissions in Go evaluation order. If
a later child has prerequisite statements, already-evaluated earlier children
are captured in target temporaries before those statements. Package-level
initializers use the selected checker graph's exact initialization order and
append their TS-Go statements to one package-initialization builder; no second
dependency graph is constructed.

## Target Construction

`schema/tsgo/` pins the exact TS-Go schema revision. A generator creates typed
Go node/factory bindings in `internal/target/tsgo`. Production emission may
construct target syntax only through those generated bindings.

The formatter accepts a validated TS-Go source-file node. No production code
may concatenate TypeScript, inject raw expressions/statements, patch formatted
text, or carry an alternate target AST. Imports, trivia, escaping, precedence,
and punctuation are formatter concerns.

## Package And Output Shape

The source tree is organized by responsibility:

```text
cmd/gotots/                  CLI composition only
schema/tsgo/                 pinned TS-Go schema and generator manifest
internal/load/               project/toolchain selection
internal/emit/               direct contextual emitter
internal/target/tsgo/        generated schema bindings and formatter adapter
internal/output/             deterministic paths and atomic writes
internal/contracts/          typed environment/manual/external contracts
internal/verify/             independent gates
runtime/                     minimal reusable Go-semantics runtime
gostdlib/                    reusable manual standard-library behavior
testdata/                    construct and project fixtures
```

Within `internal/emit`, files are named for semantic construct families, for
example:

```text
context.go
placement.go
file.go
declaration_function.go
declaration_type.go
statement_assignment.go
statement_control.go
expression_call.go
expression_selection.go
expression_composite.go
type.go
method.go
interface.go
```

There is no `ir/`, `plan/`, `lower/`, `catalog/`, `inventory/`, `legacy/`,
`compat/`, `fallback/`, `util/`, `helpers/`, or `misc/` production package.

A generated product uses deterministic ownership:

```text
<output>/
  modules/<module-key>/<package>/<source-file>.ts
  gostdlib/<toolchain-key>/<import-path>/<source-file>.ts
  externals/<contract-key>/<import-path>/index.ts
  runtime/<runtime-module>.ts
  manifest.json
```

Source-available dependencies go under `modules`, regardless of whether they
come from the workspace, module cache, vendor tree, or replacement directory.
Standard-library and external routing uses resolved metadata and explicit
contracts, never path spelling.

## Extension Boundary

An extension may:

- claim an explicit typed package/declaration/operation contract;
- inspect the same read-only Go node and `go/types` evidence supplied to the
  ordinary handler;
- construct exact TS-Go AST nodes through the same factories; and
- submit the same typed placement requests.

It may not rerun loading or typechecking, scan source text, patch output text,
invent target node shapes, or create an alternate translation pipeline.

## Complexity

The default traversal is O(number of selected Go AST nodes plus emitted TS-Go
AST nodes). Indexed object/name/import lookup must be O(1) average or O(log n).
No emitted call or declaration may grow with the number of interface
implementers, packages, or unrelated definitions.

Any broader analysis or helper mechanism requires a concrete Go counterexample,
one owner, an asymptotic/storage bound, generated-size evidence, and a deletion
condition. If it duplicates the Go graph or could be replaced by a local
`go/types` query during emission, it is rejected.
