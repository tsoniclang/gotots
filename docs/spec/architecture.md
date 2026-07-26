# Direct Compiler Architecture

## Design Rule

Use the simplest architecture that remains exact and scalable:

```text
Go syntax/type truth
┌─────────────────────────┐
│ ast.File / ast.Node     │
│ types.Info / Package   │
└────────────┬────────────┘
             │
             ▼
    contextual emitter
             │
             ▼
typed values generated from the pinned
official TS-Go external AST protocol
             │
             ▼
pinned tsgo --api printNode
┌─────────────────────────┐
│ real TS-Go AST decoder  │
│ real NodeFactory        │
│ real Printer            │
└────────────┬────────────┘
             │
             ▼
       TypeScript text
```

The contextual emitter is a traversal and construction service, not a phase
pipeline. It may follow authoritative references to relevant Go declarations,
query the existing checker graph, and inspect package relationships while
handling a source occurrence. It must not materialize those answers into a
general-purpose intermediate program.

## Authoritative Owners

| Concern | Sole owner |
|---|---|
| selected packages, files, overlays, build tags | `internal/load` using `go/packages` |
| Go syntax and source positions | standard `go/ast` and `go/token` |
| binding, type, constant, instance, method selection | one coherent `go/types` graph |
| contextual traversal and translation | `internal/emit` |
| external target node kinds, fields, encoding, protocol version | pinned TS-Go schema/protocol under `schema/tsgo` |
| typed target protocol values and factories | generated `internal/target/tsgo` |
| target lexical placement and deduplication | scoped builders in `internal/emit` |
| target decoding and formatting | pinned `tsgo --api` `printNode` |
| Go primitive target names and support declarations | representation owner in `internal/emit` plus generated `support/scalars.ts` |
| standalone runtime behavior not expressible directly | GoToTS-owned modules under generated `runtime/` |
| output paths and atomic writes | `internal/output` |
| runtime/manual/external ownership | explicit contracts under their named roots |
| independent checks | `internal/verify` |

No second package may recreate one of these truths.

## Source Admission And Owner-Directed Traversal

The selected Go toolchain is the executable language authority:

- `go/parser` admits syntactically valid selected files;
- `go/ast` supplies the typed syntax shape; and
- one coherent `go/types` graph proves bindings and semantic validity.

Any selected-package parse or type error blocks emission. Parser-recovery
`BadExpr`, `BadStmt`, and `BadDecl` nodes are diagnostic evidence only and
never enter a translation handler.

The written Go specification and its EBNF explain the language. GoToTS must not
copy that grammar into a production parser or generate a second production
visitor from it. Invalid syntax fails before emission. For example,
`somefunc(if condition {})` cannot produce a call argument because
`CallExpr.Args` contains `ast.Expr` values and `ast.IfStmt` is an `ast.Stmt`.

Production traversal uses these components:

```text
Emitter
  owns the session, shared services, and category dispatchers

Dispatcher
  routes one explicitly requested declaration, statement, expression, or type
  to exactly one semantic-owner Handler; it never descends automatically

Handler
  interprets the complete construct case, accounts for every direct field,
  consumes inseparable syntax, delegates independent children with explicit
  roles and context, and constructs target values

ChildEmitter
  narrow callback implemented by Emitter and used by handlers to request
  contextual child dispatch without importing the root or sibling handlers
```

A handler owns a closed child contract. Every direct source field is handled in
one of five ways:

1. syntax inseparable from the parent rule is consumed directly by the owner;
2. a semantically independent child is delegated exactly once with a closed
   role and context;
3. an optional absent child is explicitly accepted;
4. non-semantic metadata is assigned to its named owner; or
5. an impossible shape fails with a typed malformed-AST diagnostic.

Direct owner consumption is not a hidden recursive visit. The owner names the
field and its semantic use explicitly. For example, a call owner may consume
callee syntax as a conversion target or built-in identity rather than routing
it through ordinary value-expression handling.

Use `go/ast`'s static child type when it is already exact. Narrow explicitly
when an AST field is broader than the grammar or semantic role. For example,
`IfStmt.Else` has static type `ast.Stmt`, but its handler accepts only `nil`, a
block, or another `if`; it must not send an arbitrary value to the general
statement dispatcher.

No `ast.Walk`, `ast.Inspect`, generated visitor, or equivalent generic walker
may drive production emission or recurse on behalf of a handler. A bounded
read-only query may inspect the authoritative Go graph for a representation
decision, but it must not emit nodes, duplicate source state, or bypass the
owning handler. Generic visitors are reserved for independent verification,
selected-toolchain contract reconciliation, and other non-producing checks.

## Allowed Emission State

An emission session may retain:

- selected package/file references and the shared `types.Info`;
- one immutable compilation-wide profile containing closed, independent
  semantic-policy axes;
- a stack of source context and target lexical builders;
- deterministic mappings from typed Go objects to target names/declarations;
- already-created typed TS-Go protocol AST values;
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

Target names are keyed by `types.Object`, not source spelling. A declaration
that shadows an ancestor receives a deterministic distinct target name because
JavaScript/TypeScript lexical declarations have a temporal dead zone before
their textual declaration while the new Go variable is not in scope during
the right side of its short declaration. The lexical name service emits the
portable ASCII identifier subset accepted by strict TypeScript. Non-ASCII Go
identifier runes are escaped deterministically.
Every pinned TS-Go keyword spelling and strict-binding name is escaped at this
same boundary; the keyword set is generated from the pinned `SyntaxKind`
contract rather than copied into the emitter.
Names are distinct across the same or ancestor/descendant lexical declaration
spaces in either source order so a target declaration remains valid throughout
the complete enclosing block. Sibling scopes and distinct functions may reuse
a target name.
Shadow suffixes and compiler-created prefixes are distinct, and allocation
excludes every portable source-name base in the loaded package before emission;
there is no claim that a valid cross-target identifier namespace is impossible
for Go source to spell. Compiler-created names and all collision decisions are
owned by this one service. The service derives its target-name index once from
the authoritative `go/types` scope tree in outer-to-inner order. This is an
O(n log n) target-name cache keyed by `types.Object`, not a source inventory or
semantic plan; declaration and reference lookup during emission is O(1)
average.

Representation consistency has one owner inside `internal/emit`. On the first
request for a Go type, object, or method, that owner queries the complete
selected Go AST/type graph, chooses the exact target form, and creates or
reserves its TS-Go declaration. Every later use retrieves that same target
state by authoritative Go identity. The decision is final before dependent
target nodes are emitted; it is never serialized into a plan or consumed by a
later lowering stage.

Compilation policy is selected once at the compilation entry point and is
immutable for the complete dependency closure. Each axis is a closed typed
choice. The initial integer-representation axis selects `number` (the default)
or `bigint`. The initial evaluation-order axis selects `direct` (the default)
or `preserve-go`. Handlers query the typed profile carried by their context;
they never inspect CLI flags or independently select behavior. A generated file
set may not mix selections. Every selected axis is part of reproducibility
evidence and any eventual output manifest.

`direct` evaluation emits target expressions without introducing temporaries
solely to preserve the source order of expressions that target reshaping
reorders. It applies equally to literals, calls, and other expressions; no
purity or side-effect heuristic selects behavior. For example, a keyed
`Point{Visible: visible(), X: x()}` represented by a positional
`Point(X, Visible)` constructor emits `new Point(x(), visible())`. This is an
intentional profile boundary and is not exact Go behavior when the calls'
effects expose their order. `preserve-go` evaluates `visible()` and `x()` into
source-ordered temporaries before constructing `Point`, preserving Go behavior.
Temporaries required by target syntax or independently selected semantics such
as parallel assignment are unaffected by this axis.

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
current target scope identities and capabilities
shared go/types evidence
generated TS-Go protocol factories
```

Children do not walk upward to guess their role. Parents already know why a
child exists and pass that fact explicitly. A parent may select a specialized
child entry point when ordinary expression dispatch would be semantically
wrong, such as a store target, condition, call callee, type expression, or
comma-ok result.

The root emitter owns mutable target builders and the placement service.
Handlers cannot mutate an arbitrary target ancestor. They return typed TS-Go
protocol AST values and, when additional target statements, declarations, or
imports are required, typed placement requests with:

1. the exact target nodes;
2. legal scopes;
3. the preferred scope;
4. the execution/evaluation constraint; and
5. a typed deduplication owner where repetition is legal.

Name reservation and placement are separate operations. The authoritative
name owner indexes stable target names from the selected `go/types` scope tree
before emission; a handler reserves or references that indexed name while
constructing its target node. The handler returns any associated placement
request in its emission result. Only the root placement owner consumes requests
and mutates a target builder. No `Context` capability may silently install an
import or hoisted declaration while a child is being translated.

A use-dependent operation on a named type is a typed companion-declaration
request keyed by the authoritative `types.TypeName` and a closed operation
kind. It is owned by the generated source-file module containing that type's
declaration, not by the first caller. Applying one request may produce further
typed requests, such as `Box` copying requesting `Point` copying. The root
placement owner resolves this work to a fixed point, deduplicates by typed
owner, and rejects ownership conflicts.

One placement service applies the policy:

- imports always enter file import scope; dynamic imports are forbidden;
- every emitted package declaration is exported from its generated source-file
  module for static intra-package linking, including declarations whose Go
  names are unexported; only the package assembly facade exposes the selected
  Go public API to consumers;
- reusable static declarations prefer file scope;
- named-type companion declarations are emitted immediately after their
  owning type declaration in a fixed operation order;
- function-wide declarations enter the function prologue only when their
  lifetime is function-wide;
- evaluation-dependent temporaries remain immediately inside the branch,
  loop, short-circuit arm, deferred closure, or statement where they execute;
- no request may cross a semantic execution boundary merely because a wider
  scope is syntactically legal.

This permits hoisting without maintaining a separate target IR.

Creating a typed TS-Go node is not finalizing or printing its containing file.
The root emitter seals each target file only after scheduling and placement
queues are empty. A sealed file accepts no further request. Already-created
class nodes are never reopened: genuine class members come from the complete Go
type contract when the class is constructed, while use-dependent generated
operations are top-level companion declarations. There is no target-text
patch, prototype assignment, mutable class reopening, or post-print insertion.

Under `preserve-go`, a keyed composite captures child values in Go source order
before consuming them in target constructor order. If any child emission has
prerequisite statements, its parent still places those statements at the
required execution boundary and captures values required to make the target
AST representable. Under `direct`, the composite itself creates no prerequisite
statements solely for field reordering, so an enclosing call remains a direct
call when its other children are direct. Package-level initialization order and
atomic multi-value operations are separate semantic contracts and remain exact
under both profiles. Package-level initializers use the selected checker
graph's exact initialization order and append their TS-Go statements to one
package-initialization builder; no second dependency graph is constructed.

## Target Construction

`schema/tsgo/` pins one exact TS-Go revision and its official external AST
schema, protocol version, encoder contract, and relevant API definitions
byte-for-byte. A generator mechanically creates node-specific Go types,
closed child interfaces, factories, and the binary encoder in
`internal/target/tsgo`. Generation is total over the pinned schema; there is no
manually selected target-node profile or second schema.

Schema-invalid target children must be unrepresentable through ordinary typed
factory calls. A generic handwritten `Node`, runtime kind-switch validator,
field-shape inference, or special-case wire classifier is not an acceptable
substitute. Encoding layout comes only from the pinned official protocol and
must not be inferred from coincidental field shapes.

GoToTS starts one persistent exact pinned `tsgo --api --async` process for a
compilation. It sends validated binary AST values to the official `printNode`
endpoint. TS-Go then decodes them through its real AST factory and prints them
through its real printer. GoToTS has no formatter and does not fork, import
through, or expose TS-Go's Go `internal` packages.

The TS-Go command is a pinned Go tool dependency, not a PATH lookup. Its module
revision is joined to `schema/tsgo/manifest.json` and its executable build
identity is checked before the first request. Missing or mismatched tools fail
closed; there is no compatible-version fallback.

No production code may concatenate TypeScript, inject raw
expressions/statements, patch formatted text, or carry an alternate target
tree. Imports, trivia, escaping, precedence, and punctuation are TS-Go
concerns.

Generated support modules obey the same rule. The representation owner requests
one canonical primitive alias by typed identity; the root emitter constructs
the corresponding exported TS-Go `TypeAliasDeclaration` in
`support/scalars.ts` and deduplicates it. Generated package files use canonical
relative type-only `.js` imports. No checked-in source template or external
package supplies these declarations. Any other marker-like support declaration
must have complete ordinary TypeScript semantics; a no-op declaration cannot
delegate missing behavior to an external tool.

Every compilation entry point returns the complete reachable target-file set,
including generated support/runtime modules and source dependencies. A
file-root convenience may choose roots from one Go file, but it must not return
only that file's TypeScript module or discard any requested artifact. Consumers
never reconstruct omitted support from imports.

## Package And Output Shape

Every source-available Go file has one checkout-independent target path:

```text
modules/<sha256(module-path NUL module-version)>/<module-relative-package>/<source-base>.ts
```

The full digest is an opaque semantic-module owner, not a shortened display
identifier. Generated value imports use canonical relative `.js` specifiers.
Every cross-package imported binding uses a deterministic package-qualified
local alias even when its source spelling is currently unshadowed. This keeps
the reference exact when a local declaration has the same spelling and prevents
unrelated lexical traversal order from selecting whether qualification occurs.
Same-package cross-file references retain their reserved package name.

The source tree is organized recursively by authoritative responsibility. This
is a growth rule, not a prediction of every construct. Create a directory only
when its first real owner is implemented:

```text
cmd/gotots/                         CLI composition only

schema/tsgo/                        exact pinned TS-Go external contract
  manifest.json                     revision, protocol, paths, exact digests
  upstream/                         byte-exact upstream schema/protocol inputs

internal/load/                      project and selected-toolchain loading

internal/emit/                      session, scheduling, closed dispatch only
  emitter.go
  dispatch_declaration.go
  dispatch_statement.go
  dispatch_expression.go
  dispatch_type.go
  api/                              narrow handler contracts
    context.go
    role.go
    result.go
    children.go
  <domain>/                         declaration, statement, expression, type
    <semantic-owner>/               assignment, call, index, function, ...
      <sub-owner>/                  only after an evidenced ownership split

internal/target/tsgo/               official-protocol target boundary
  generate/                         schema/protocol binding generator
  *_generated.go                    typed nodes, factories, encoder
  client.go                         persistent tsgo API process
  print.go                          printNode request boundary
  contract_test.go                  pinned schema/binding totality
  encoder_test.go                   upstream encoder differential
  client_test.go                    native printNode round trip

internal/output/                    deterministic paths and atomic writes
internal/contracts/                 environment/manual/external contracts
internal/verify/                    independent gates and harnesses

runtime/                            minimal reusable Go-semantics runtime
gostdlib/                           reusable manual standard-library behavior

testdata/constructs/                minimal construct-case fixtures
  <domain>/<semantic-owner>/<case>/
    source.go
    expected.ts
testdata/projects/                  multi-file/package/project fixtures
```

`internal/emit` contains only orchestration and services that genuinely span
semantic owners. A handler package imports `internal/emit/api` and
`internal/target/tsgo`; it does not import the root emitter or a sibling
handler. The root emitter imports handlers and implements `api.ChildEmitter`.
This fixed dependency direction prevents cycles and hidden semantic coupling:

```text
internal/emit
    -> internal/emit/<domain>/<semantic-owner>
        -> internal/emit/api
        -> internal/target/tsgo
```

The `api` package contains immutable Go AST/type references, parent roles,
target results, placement requests, and typed diagnostics only. It must not
become a semantic IR, property bag, copied source model, or second dispatcher.

The split follows semantic ownership, not one file per AST type or construct
fixture. For example, `statement/assignment` owns ordinary, parallel,
compound, multi-result, and store-target assignment semantics until evidence
requires a coherent sub-owner. If `expression/call` later becomes substantial,
it may split into evidenced `builtin/`, `method/`, or `interface/` owners.

The first concrete owners therefore have ordinary focused files rather than a
file per contextual case:

```text
internal/emit/statement/assignment/
  handler.go
  handler_test.go

internal/emit/expression/call/
  handler.go
  handler_test.go
```

No directory is pre-populated. Before a directory would exceed twenty
maintained non-generated Go files, stop and review its truth owner; extract a
coherent semantic sub-owner rather than mechanically splitting filenames.
Maintained non-generated files remain below 600 physical lines.

There is no `ir/`, `plan/`, `lower/`, `catalog/`, `inventory/`, `legacy/`,
`compat/`, `fallback/`, `util/`, `helpers/`, `misc/`, `common/`, or `shared/`
production package. A new directory requires a named owner, dependency
direction, smallest motivating construct case, and split/deletion criterion.

Construct fixtures follow the same discovery-driven organization. The ordinary
typed Go test beside the handler names its fixture and supplies context. There
is no JSON case manifest, per-program inventory, or second coverage registry.
A fixture adds more files, packages, modules, or expected outputs only when the
case itself requires them.

A generated product uses deterministic ownership:

```text
<output>/
  modules/<module-key>/<package>/<source-file>.ts
  gostdlib/<toolchain-key>/<import-path>/<source-file>.ts
  externals/<contract-key>/<import-path>/index.ts
  support/scalars.ts
  runtime/<runtime-module>.ts
  manifest.json
```

Source-available dependencies go under `modules`, regardless of whether they
come from the workspace, module cache, vendor tree, or replacement directory.
Standard-library and external routing uses resolved metadata and explicit
contracts, never path spelling. `support/scalars.ts` contains only aliases
requested by the selected program, each defined once. `runtime/` contains only
GoToTS-owned behavior required for exact standalone execution; neither location
imports an unrelated compiler, transpiler, target, or product.

## Extension Boundary

An extension may:

- claim an explicit typed package/declaration/operation contract;
- inspect the same read-only Go node and `go/types` evidence supplied to the
  ordinary handler;
- construct exact typed TS-Go protocol AST values through the same generated
  factories; and
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
