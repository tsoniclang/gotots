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
| mutable package storage and package-local initialization bodies | package-state and passive package-assembly builders in `internal/emit`, driven by the selected `go/types.Info.InitOrder` |
| whole-program package initialization order | one static program-initialization builder consuming the selected `types.Package` import graph |
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
- typed root requests for placement, declaration requirements, and artifact
  dependencies;
- canonical encoded TS-Go observable contract facets plus their deterministic
  reverse-dependency and dirty sets;
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
identifier runes are escaped deterministically. Escaped spellings are not
assumed to be injective: every target namespace, including package import
qualifiers, allocates from the escaped candidates as one globally checked
collision domain with deterministic Go-identity ordering.
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

Multiple package `init` declarations are the one legal same-spelling
package-level definition family that is not represented as ordinary lookup
members of the package scope. The same name service assigns those exact
`*types.Func` identities in canonical source-module-path and physical
declaration order before scheduling. Cross-file `token.Pos` allocation,
checker map iteration, root order, and demand order never select which
initializer receives a collision suffix.

Representation consistency has one owner inside `internal/emit`. On the first
request for a Go type, object, or method, that owner queries the complete
selected Go AST/type graph, chooses the exact target form, and creates or
reserves its TS-Go declaration. Every later use retrieves that same target
state by authoritative Go identity. Source-semantic and base representation
decisions are final when selected; they are never serialized into a plan or
consumed by a later lowering stage. A genuinely use-dependent target
obligation may nevertheless reconstruct an open target artifact. If that
reconstruction changes a consumer-visible TS-Go contract facet, the root
reconstructs only consumers that recorded a dependency on that exact facet
before sealing.

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

Reaching any source-available package also reaches that package's complete
initialization obligation. This includes every package variable, including an
unexported or otherwise unreferenced variable whose initializer has effects,
and every admitted `init` function. Reachability may prune packages, but it
must not prune initialization within a reached package. The selected
`go/types.Info.InitOrder` is the sole variable-initializer order; the emitter
does not infer order from files, imports, references, or source positions.

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

The root emitter owns mutable target builders, declaration assemblies, the
artifact dependency graph, and the placement service. Handlers cannot mutate
an arbitrary target ancestor. They return typed TS-Go protocol AST values and,
when additional target statements, declarations, imports, changes to an owned
declaration, or dependencies on another artifact are required, typed requests
with:

1. exact target nodes or an exact declaration identity plus closed requirement;
2. legal scopes;
3. the preferred scope;
4. the execution/evaluation constraint; and
5. a typed deduplication owner where repetition is legal; and
6. for a dependency, the authoritative provider identity and one closed
   observable facet.

Name reservation and placement are separate operations. The authoritative
name owner indexes stable target names from the selected `go/types` scope tree
before emission; a handler reserves or references that indexed name while
constructing its target node. The handler returns any associated placement
request in its emission result. Only the root placement owner consumes requests
and mutates a target builder. No `Context` capability may silently install an
import or hoisted declaration while a child is being translated.

A use-dependent target obligation is a declaration requirement keyed by the
authoritative `types.Object` and a closed requirement kind. It is owned by the
semantic handler and generated source-file module containing that declaration,
not by the first caller. The first admitted requirement family is a named
struct's static zero/copy/equality operation, keyed by its exact
`types.TypeName` and closed operation kind. The operation is incorporated into
the owning class and is called through the statically selected Go type
(`Box.$copy(value)`), never through an instance. Applying one requirement may
produce further typed requests, such as `Box` copying requesting
`Point.$copy`.

Addressable local storage is the second admitted requirement family. An
address expression identifies the exact `*types.Var` whose storage becomes
observable and requests that storage from the enclosing reconstructible
`*types.Func` artifact. The function owner then reconstructs its whole typed
TS-Go declaration with a cell only for each requested variable. Parameters and
receivers keep their observable callable parameters and receive one body-local
cell; local variables and named results are declared directly as cells. Reads,
stores, captures, and address formation all query that one identity-keyed
selection. No source prepass, escape-analysis copy, blanket local wrapping, or
second body emitter may select storage.

The concrete storage representation is installed once by the root emitter as
a typed `api.Context` capability. Construct handlers consume only that
capability; in particular, the generic assignment owner imports neither the
storage implementation nor the pointer runtime. This keeps assignment ordering
independent of the selected value representation while retaining one storage
truth owner.

Every executable Go function body, including each package `init` declaration,
is a reconstructible artifact. Package initialization may impose ordering and
an exported generated entry name, but it does not own a second emission or
requirement lifecycle. A body-only storage reconstruction leaves the
mechanically projected callable signature unchanged and therefore publishes no
consumer invalidation.

The root owner keeps one open declaration assembly per emitted definition.
Only that definition's semantic owner interprets its requirements and
reconstructs its typed TS-Go protocol nodes. Requirements are monotonic,
identity-deduplicated, and applied in deterministic closed-kind order;
incompatible requirements fail rather than selecting whichever was discovered
first. Reconstructing a declaration replaces its prior in-memory target nodes;
it never creates a second final definition. The root resolves declaration,
requirement, initialization, and placement work to a fixed point before file
sealing.

A fact already available from the Go declaration or selected `go/types`
evidence is not a legitimate late requirement. For example, source export
identity and the target module policy determine accessibility without waiting
for a later use. Late requirements are reserved for obligations that genuinely
arise from reached uses, such as a requested value operation, interface
adapter, generic specialization, callback adapter, or runtime-type carrier.

## Observable Artifact Propagation

Every source definition that can be reconstructed owns one target artifact
keyed by its exact `types.Object`. An artifact revision is a transaction:

```text
authoritative Go object + Go AST/type evidence + accumulated requirements
    -> one semantic-owner handler
    -> complete replacement TS-Go AST roots
    -> complete replacement root-request set
    -> complete replacement artifact-dependency set
    -> mechanically projected observable contract
```

No consumer receives a mutable provider node. A reference records a dependency
on the smallest closed provider facet it consumes:

- `CallableSignature` for a call or function value;
- `ConstructorSurface` for construction;
- `InstanceTypeSurface` for an instance type/member contract;
- `StaticSurface` for a statically selected class operation; and
- `ValueSurface` for an exported target value.

An artifact that emits only an executable body may provide an explicit empty
contract while still consuming dependencies. It is reconstructed when a
provider facet changes, but cannot dirty downstream artifacts because it
provides no facet. An absent contract is invalid; an explicit empty contract is
therefore not confused with failed projection.

The facet enum is closed. A new facet requires a concrete source example,
canonical projection, consumer rule, equality proof, and mutation test; string
facets and catch-all dependencies are forbidden.

The canonical contract is derived mechanically from the artifact's typed
TS-Go AST, never restated by a handler. Function and method projections retain
modifiers, names, type parameters, parameters, and explicit return types while
removing bodies. Class projections partition constructors, instance members,
and static members; member bodies and explicitly typed property initializers
are removed. Interface and type-alias structure is already contract-only.
Variable projections retain binding, modifiers, declaration flags, and
explicit type while removing initializers. A declaration whose visible target
type depends on TypeScript inference fails at this boundary: silently dropping
its body or initializer would make implementation bytes masquerade as a stable
contract. Exact encoded TS-Go structure is the equality authority. A hash may
only accelerate comparison; it cannot establish equality.

The root replaces an artifact's reverse edges when a reconstruction commits,
compares each old and new facet structurally, and queues only consumers
subscribed to changed facets. Duplicate edges and dirty entries collapse by
typed identity. Dirty artifacts are processed in deterministic Go-object
order. An initial publication establishes a baseline and does not notify
consumers. A body-only change therefore commits without rebuilding callers:

```go
func Value() int32 { return source }
func Use() int32  { return Value() }
```

Changing only `Value`'s emitted body leaves its `CallableSignature` projection
unchanged, so `Use` remains untouched. If a later target obligation changes
`Value` from `Value(): number` to `Value(flag: boolean): number`, the callable
facet changes, `Use` is reconstructed, and any resulting change to `Use`'s own
callable facet propagates to its callers.

Cycles are permitted only when structural contracts converge. The root records
each distinct committed canonical contract for the compilation, without a
second copy of the current revision. Repeating a prior non-current contract is
a typed non-convergence error, not an arbitrary iteration limit or silently
accepted result. Storage is bounded by current artifacts, consumed facet edges,
and the exact bytes of distinct changed contracts; unchanged reconstruction
adds no history. Sealing requires empty declaration, requirement,
initialization, placement, and dirty-artifact queues.

This graph is target-assembly coordination, not a semantic IR or call graph. It
contains only authoritative Go definition identities, closed target-contract
facets, and reverse invalidation edges discovered while constructing actual
TS-Go references. It does not copy source statements, infer whole-program
meaning, perform name lookup, or decide reachability.

One placement service applies the policy:

- imports always enter file import scope; dynamic imports are forbidden;
- one imported binding has one placement identity independent of whether a
  requester needs its type or value namespace; a value import dominates a
  type-only request for the same module, export, and local binding, in either
  request order, because the value import supplies both namespaces;
- every emitted immutable package declaration is exported from its generated
  source-file module for static intra-package linking, including declarations
  whose Go names are unexported;
- mutable package variables are fields of the package's one state object, not
  duplicate `let` declarations in source-file modules;
- same-package generated source modules import that state object directly,
  while cross-package references use the passive package assembly; consumers
  import the program-initialization module before using a selected package
  surface;
- reusable static declarations prefer file scope;
- named-struct static operations are incorporated into their owning class in a
  fixed operation order; no top-level sibling helper or instance operation is
  emitted;
- function-wide declarations enter the function prologue only when their
  lifetime is function-wide;
- evaluation-dependent temporaries remain immediately inside the branch,
  loop, short-circuit arm, deferred closure, or statement where they execute;
- no request may cross a semantic execution boundary merely because a wider
  scope is syntactically legal.

This permits hoisting without maintaining a separate target IR.

Creating a typed TS-Go node is not finalizing or printing its containing file.
The root emitter seals each target file only after declaration, requirement,
initialization, placement, and dirty-artifact queues are empty. A sealed file
accepts no further request. Before sealing, an already-created class or other
declaration may be replaced only by its one semantic owner using freshly
constructed typed TS-Go protocol nodes and the complete accumulated requirement
set. The replacement transaction also replaces that artifact's root requests
and dependencies and publishes its mechanically derived observable contract.
Other handlers can request a closed obligation or record a typed dependency but
cannot receive or mutate the target declaration.

This is controlled pre-seal target assembly, not mutable class reopening.
There is no target-text patch, prototype assignment, arbitrary AST callback,
post-print insertion, or editable on-disk AST authority. A future disk cache
may retain only immutable, fully fingerprinted sealed artifacts; any changed
input or requirement regenerates the declaration through its owner.

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
package-assembly function body. Whole-program order consumes the selected
package import graph directly; it does not infer order from target imports or
construct a parallel semantic graph.

## Package State And Assembly

TypeScript ESM imported bindings cannot be assigned by another module, while
Go package variables are mutable from every file in their package. Therefore
every reached package has one package assembly, and a package with mutable
variables additionally has one state module. Every compilation also has one
program-initialization module:

```text
packages/<module-key>/<module-relative-package>/state.ts
packages/<module-key>/<module-relative-package>/package.ts
program.ts
```

The state module owns storage. It emits one generated class whose public
`declare` fields are typed from the authoritative package-scope `types.Var`
objects, and one state instance:

```go
var Counter int
```

```ts
export class $PackageState {
  declare Counter: int32;
}

export const $state = new $PackageState();
```

`declare` is intentional: it is erased JavaScript and avoids importing runtime
source declarations into the state module. The state module may use type-only
imports and primitive-alias imports, but it must have no runtime dependency on
a generated source-file module. It contains no initializer and no cast from an
empty object.

The package assembly is passive at module evaluation. It imports the state and
required source declarations and exports one `$initialize` function when the
package has initialization work. That function assigns the exact Go zero to
every state field before any package initializer runs, then emits initializer
assignments in `go/types.Info.InitOrder`. Admitted `init` functions follow in
the selected toolchain loader's file order and declaration order within each
file; the emitter preserves that evidence and does not rescan the filesystem
or sort a second file list. For example:

```go
var B = 4
var A = B + 1

func Read() int { return A + B }
```

becomes structurally:

```ts
import { Read } from "../../modules/<module-key>/<package>/read.js";
import { $state } from "./state.js";

export function $initialize(): void {
  $state.A = 0;
  $state.B = 0;
  $state.B = 4;
  $state.A = $state.B + 1;
}

export { $state, Read };
```

The program-initialization module is the sole executor. Go does not use a
depth-first module walk: from all packages sorted by import path, it repeatedly
initializes the first package whose imports are already initialized. ESM
dependency evaluation is depth-first and can produce a different order for
independent branches. Therefore no package assembly executes initialization at
top level and target import order is never treated as Go initialization order.

The program builder applies Go's algorithm directly to the reached
`types.Package` identities and their selected `Imports()` edges, then filters
the resulting order to packages with initialization work. It emits static
imports and one direct call per such package:

```ts
import { $initialize as $initialize_registry } from "./packages/<key>/registry/package.js";
import { $initialize as $initialize_y } from "./packages/<key>/y/package.js";
import { $initialize as $initialize_b } from "./packages/<key>/b/package.js";

$initialize_registry();
$initialize_y();
$initialize_b();
```

There is no runtime registry, idempotence flag, callback table, dynamic import,
or alternate package graph. ESM evaluates `program.ts` once, so each direct
call executes once. A consumer imports `program.ts` for effects before using
bindings from a passive package assembly.

Generated reads and stores use the same field identity (`$state.A`,
`$state.A = value`, `$state.A++`). No per-variable getter/setter, mutable-cell
wrapper, duplicated source-file binding, or source-order emulation exists.
Same-package source modules import `state.ts`; cross-package generated code
imports the dependency state and declarations through passive `package.ts`.
Loading source and package modules cannot execute Go initialization early.

The package assemblies and `program.ts` are sealed only after the declaration
scheduler, package initialization work, and placement requests reach a fixed
point. A consumer that omits `program.ts` is outside the executable output
contract. Package-state, assembly, and program-initialization nodes are
constructed through the same typed TS-Go protocol factories and printed by the
same pinned printer as every other target file.

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

Runtime support follows the same typed placement model. A contextual handler
requests a closed runtime symbol from the name owner. That symbol fixes:

- its one `runtime/<family>.ts` output owner;
- its exported target name and allowed type/value uses;
- its closed runtime-symbol dependencies;
- the family builder that constructs its declaration as typed TS-Go AST; and
- its deterministic deduplication identity.

The source-file placement owner receives the resulting ordinary static import
at the requested type or value phase and upgrades type to value when both are
needed in one file.
The program target-file owner independently collects the tagged runtime
symbols, computes their cycle-free transitive dependency closure, exact-joins
every requested export to one generated definition, emits each demanded
runtime module once, and creates canonical relative static ESM imports between
runtime modules. A raw module/export string may not select runtime behavior,
and a runtime declaration may not be carried from a use-site handler as an
opaque duplicate AST payload.

```text
contextual family handler
    -> Names.Runtime(closed symbol)
    -> static ESM import request + tagged definition requirement
    -> program-level dependency closure + exact symbol join
    -> internal/emit/runtime/<family> TS-Go AST builder
    -> runtime/<family>.ts
```

All generated runtime failures use one closed Go panic ABI:

```text
family runtime guard
    -> GoPanic.raise(typed value)
    -> one runtime/panic.ts owner
    -> one thrown GoPanic<T> carrier
```

A family runtime must not throw a host `Error`/`RangeError` or depend on an
implicit JavaScript exception for Go panic behavior. The panic module is the
only generated runtime owner allowed to construct a target `ThrowStatement`.
This boundary does not implement source-level `panic`/`recover`; it gives every
already-admitted runtime guard one stable carrier that those later constructs
can consume without replacing six family-specific exception protocols.

Runtime classes may encapsulate JavaScript storage only when that storage is
the smallest exact representation of a Go value family. They may expose
ordinary statically typed methods, but no reflection, dynamic operation name,
erased payload recovery, target non-null assertion, `.call`/`.apply`/`.bind`,
or per-use semantic closure. Nullable storage is narrowed by explicit control
flow at the runtime owner; a generated `value!` may not substitute for a
proved invariant.
Compiler handlers still own Go evaluation order and copy boundaries; runtime
code does not rediscover source semantics.

The pointer runtime is the one exception for typed location accessors because a
Go pointer is itself a first-class reference to mutable storage rather than a
copied value. Its closed factories may capture only statically typed read and
write access to an already-selected local, field, array element, or slice
backing element. They may not capture source nodes, names, type metadata,
operation tags, or semantic decisions. A canonical opaque address token owns
equality; payloads never pass through `any`, `unknown`, reflection, or dynamic
shape inspection.

Setter-backed stores have one target contract shared by arrays, slices, maps,
and future setter representations. The store-target owner returns typed
TS-Go receiver and argument expressions with their prerequisite statements;
the assignment owner alone orders and conditionally captures them before the
right side, then constructs the final member call. A family-specific
assignment handler is forbidden. Likewise, one builtin-object dispatcher owns
semantic dispatch for `new`, `make`, `len`, `cap`, `append`, `copy`, and
`delete`; family sub-owners consume the already-resolved `*types.Builtin`
rather than rediscovering it from spelling.

## Package And Output Shape

Every source-available Go file has one checkout-independent target path:

```text
modules/<sha256(module-path NUL module-version)>/<module-relative-package>/<source-base>.ts
```

Its package-owned assembly and conditional mutable state use the same semantic
module key and module-relative package under the separate `packages/` root:

```text
packages/<sha256(module-path NUL module-version)>/<module-relative-package>/state.ts
packages/<sha256(module-path NUL module-version)>/<module-relative-package>/package.ts
```

The compilation-owned initialization entry has the checkout-independent path
`program.ts`.

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
  dispatch.go                       emitter plus closed typed dispatch
  declaration_assembly.go           roots, demands, and artifact reconstruction
  artifact/                         target-contract coordination authority
    contract.go                     mechanical TS-Go observable projections
    graph.go                        facet edges, revisions, and fixed point
  target_files.go                   source/support/program file sealing
  package_state.go                  package storage and initialization owner
  storage/                          exact identity-selected local cell access
  api/                              narrow handler contracts
    context.go
    role.go
    result.go
    root_request.go
    artifact_dependency.go
    children.go
  <domain>/                         declaration, statement, expression, type
    <semantic-owner>/               assignment, call, index, function, ...
      <sub-owner>/                  only after an evidenced ownership split
  expression/address/               address formation and location projection
  runtime/pointer/                  one typed canonical-location runtime owner

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
  program.ts
  modules/<module-key>/<package>/<source-file>.ts
  packages/<module-key>/<package>/state.ts
  packages/<module-key>/<package>/package.ts
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
