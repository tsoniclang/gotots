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
| canonical generated-type identity | one generated-artifact registry in `internal/emit/naming`, indexed and named by a full canonical Go-type digest that never includes generated target spelling, then exact-joined by `types.Identical` |
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
- opaque immutable composition nodes used only to retain ordered root-request
  leaves without copying transitive child lists;
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
or `preserve-go`. The initial concurrency-semantics axis selects `disabled`
(the default) or the explicitly requested `cooperative` profile. Handlers query
the typed profile carried by their context; they never inspect CLI flags or
independently select behavior. A generated file set may not mix selections.
Every selected axis is part of reproducibility evidence and any eventual
output manifest.

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

Root intent is a closed typed part of the request and is not discarded after
object selection. Whole-file coverage, exported Go API, ordinary declaration
demand, and explicit constant-representation demand are distinct. This matters
when a source declaration has no single runtime form: an unused exported
untyped constant is a compile-time-only Go contract and therefore has no target
declaration, while an explicit constant-representation root names and
materializes exactly one concrete projection. A generic representation request
for an untyped constant is invalid. Only a typed compile-time-only disposition
may publish an explicit empty observable contract; an empty concrete
representation root is a gate failure.

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

### Value Transfer

One value-transfer owner is the only authority for moving an emitted value
across a typed Go boundary. Its input is the source occurrence, actual
`go/types.Type`, destination `go/types.Type`, closed copy-ownership mode, and
typed TS-Go expression. The copy mode records whether this boundary performs
the Go value copy or only adapts representation because the value is already
fresh or a typed destination operation (such as aggregate map storage)
performs the copy exactly once. It first requires
`types.AssignableTo(actual, destination)`, then composes, in order:

1. the exact source representation projection when actual and destination
   representations differ;
2. the destination Go value-copy operation at its declared single owner; and
3. the exact destination representation construction when required.

Arguments, returns, assignments, declarations, aggregate elements, map
entries, channel transfers, and generated generic calls all use this owner.
They do not call a target-only copy function and do not independently
special-case defined callable, slice, map, pointer, channel, basic, or array
families. Explicit Go conversions remain owned by the conversion handler and
are not inferred from assignment.

For `type Value []byte`, passing `Value` to a `[]byte` parameter is admitted by
`go/types` and produces the static payload projection `value.$value`; passing
an unnamed `[]byte` to a `Value` destination produces one `new Value(...)`.
Passing one defined slice type to a different defined slice type is rejected
before target construction. No cast, structural target assignability,
spelling comparison, or dynamic wrapper inspection participates.

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

Composing child results does not flatten and recopy every transitive root
request. The request API owns one opaque immutable persistent sequence whose
leaves are the typed requests above. Composition copies only immediate child
roots, preserves exact source order, and neither deduplicates nor attaches
semantic meaning to tree shape. The sequence is request transport, not source
IR or a second fact store. Only root consumers walk its atomic leaves, once,
before placement, scheduling, and artifact-dependency admission. A sequence
node is never interpreted as an import, declaration requirement, or artifact
dependency, and no mutable backing storage is exposed.

Each atomic request is a bounded immutable handle to one constructor-validated
payload. Copying or composing results shares that payload; it does not copy the
request's owner, TS-Go import nodes, requirement, or dependency. Payloads are
never mutated after their constructor returns.

A use-dependent target obligation is a declaration requirement keyed by the
authoritative `types.Object` and a closed requirement kind. It is owned by the
semantic handler and generated source-file module containing that declaration,
not by the first caller. The first admitted requirement family is a named
struct's static zero/copy/equality operation, keyed by its exact
`types.TypeName` and closed operation kind. The operation is incorporated into
the owning class and callers use the statically selected Go type
(`Box.$copy(value)`), never an instance. A defined-basic wrapper instead
composes equality directly from its underlying value and has no operation
requirement. Applying one requirement may produce further typed requests, such
as `Box` copying requesting `Point.$copy`.

An interface adapter's callable surface is also use-dependent. The adapter is
owned once by the exact concrete dynamic Go type, while each concrete-to-
interface conversion requests the exact completed target interface contract.
An interface-to-interface conversion or assertion records the exact static
source interface and target interface. Direct concrete boxing seeds an
adapter's reachable-contract set. A transition may add its target only when
its source contract is already in that adapter's reachable set and
`go/types` proves the concrete type implements the target. The target-name
owner computes this monotone closure for existing and future adapters. Mere
implementation of the transition's source interface is not reachability and
must not widen an adapter that was boxed only into another contract. The final
adapter method surface is the deterministic union of the reachable exact
contracts. It is not the concrete type's complete receiver method set, an
implementer union, a value-flow graph, or a runtime method lookup table.
The adapter payload retains the exact representation selected for its concrete
dynamic type. Each adapter method separately consumes the selected provider
method's origin-owned receiver ABI. If those representations differ, the one
receiver-selection owner emits the same typed, nil-preserving prerequisites as
an ordinary selected call, and the adapter places them before invocation.
Adapters never assume that a concrete payload and a generic method origin have
the same pointer representation, rediscover the receiver ABI, or discard
receiver prerequisites.
Construction admits each `(concrete adapter, contract)` pair at most once and
propagates only newly reachable pairs through transitions indexed by source
contract. Repeated occurrences of an already-known conversion or assertion
perform no global adapter scan and schedule no duplicate reconstruction.

This demand state is target-construction state: closed interface types and
canonical type-identity keys enter through typed root requests and are consumed
only to reconstruct generated adapter AST. It never answers a Go semantic
question; `go/types.Implements`, completed interface method sets, and exact
method selections remain the sole truth. Adding or removing an unrelated
concrete receiver method cannot alter an adapter unless that method enters a
demanded interface contract.

A generated artifact that consumes a generic source declaration resolves that
declaration's ABI identities through the canonical source owner, not through
the consumer's lexical scope. For example, `OrderedSet[T]` calling
`OrderedMap[K,V].Set` observes the receiver representation owned by the
`OrderedMap.Set` origin and the concrete representation owned by
`OrderedMap[T,struct{}]` separately. Relaxing lexical identity, copying the
foreign parameter spelling `K`, or assigning a consumer-local identity is
forbidden.

Addressable local storage is the second admitted requirement family. An
address expression identifies the exact `*types.Var` whose storage becomes
observable and requests that storage from the enclosing reconstructible
artifact: either the exact source `*types.Func` or the exact checker-produced
package initializer. That owner reconstructs its whole typed TS-Go artifact
with a cell only at each requested variable's lexical declaration. Parameters
and receivers keep their observable callable parameters and receive one
body-local cell; local variables and named results are declared directly as
cells. A function-literal local inside a package initializer therefore requests
the package-initializer artifact while retaining its cell inside the literal.
Reads, stores, captures, and address formation all query that one
identity-keyed selection. No source prepass, escape-analysis copy, blanket
local wrapping, or second body emitter may select storage.

The concrete storage representation is installed once by the root emitter as
a typed `api.Context` capability. Construct handlers consume only that
capability; in particular, the generic assignment owner imports neither the
storage implementation nor the pointer runtime. This keeps assignment ordering
independent of the selected value representation while retaining one storage
truth owner.

Logical value representation and addressable storage representation are
separate facets of the same value-family owner, but separation is not emitted
until an exact use requires it. A named-struct object is already a stable,
mutable JavaScript location. Therefore an ordinary `*S` whose reached uses are
nil, identity, field access, `new(S)`, `&S{}`, `&local`, or receiver calls is
represented directly as `S | undefined`. `&local` makes that exact local keep
one stable `S` object; later whole-value assignment copies fields into that
object instead of rebinding it. Unaddressed values continue to rebind at Go
copy boundaries. No `$Storage`, `$make`, `$storageOf`, `$fromStorage`, or
`GoPointer.dereference` surface is emitted for this ordinary class-reference
case.

`GoPointer<Logical, Storage>` remains the one explicit location protocol where
direct class identity is insufficient: pointers to scalars, pointers to
pointers, array or slice elements, interior field locations, and pointer
conversions that require a typed representation view. `Logical` preserves Go's
named-type identity, while `Storage` is the canonical mutable representation
shared by every admitted pointer type whose Go base types are identical under
the selected toolchain's pointer-conversion rule. The runtime reads and writes
only `Storage`; load and store sites ask the value-family owner to convert
between `Logical` and `Storage`. The runtime never receives semantic conversion
callbacks.

A named or generated struct gains a canonical storage record and logical
getters/setters only when a reached conversion or location operation proves
that direct class identity cannot preserve the required alias. The declaration
is then reconstructed once from that complete demand set. A pointer conversion
requests the storage facet from both endpoint declarations and is admitted only
when their independently constructed canonical storage types match the exact
Go conversion evidence. Struct tags do not enter that storage facet; field
identity, order, package identity, and recursively represented field storage
do. The selected representation is compilation-wide for each exact pointer
type, so one `*S` never has both direct-class and carrier forms.

```text
Go *Left / *Right (underlying bases identical ignoring tags)
        |
        v
GoPointer<Left, LeftStorage> --typed view--> GoPointer<Right, RightStorage>
        |                                      |
        +-------- LeftStorage == RightStorage--+
```

The view operation may allocate a constant-size pointer facade, but it reuses
the same opaque canonical address and the same typed storage accessors. It
does not copy the pointee, inspect a value shape, recover an erased payload, or
carry a source/target conversion function. Removing the final conversion or
interior-location demand reconstructs the declarations back to the direct
class-reference form; demand is not a permanent widening flag.

A defined array follows the same split without growing its nominal wrapper.
Its generated class is the logical type and contains only its brand,
constructor, and underlying value. Its canonical pointer storage is the
underlying `GoArray<Element, Length>`. Indexing, pointer projection, and
whole-array stores operate on that storage; they do not demand duplicate
`get`/`set` methods on the nominal class.

Open generic declarations use the same representation decisions without
pretending that a source type parameter is its own whole-value storage,
container-slot storage, or pointer type. Each source `*types.TypeParam` always
owns one logical target type parameter. A declaration gains a whole-storage
facet only when a canonical struct/location record can differ from the logical
type, a container-storage facet only when an array or slice slot can differ
from both, and a pointer facet only when an exact `*T` value is represented.
These are immutable target type parameters keyed by the source parameter
identity and facet kind, not runtime descriptors:

```text
Go T                         -> target T
whole storage of T needed    -> target T$Storage
array/slice slot of T needed -> target T$ContainerStorage
Go *T needed                 -> target T$Pointer
```

At each concrete instantiation the ordinary representation owner supplies the
exact selected facet types. For a direct named struct `Item` with no
carrier-requiring use, `T$Storage = Item$Storage`,
`T$ContainerStorage = Item`, and `T$Pointer = Item` when the pointer facet is
otherwise demanded. If a reached use requires a canonical location carrier,
container storage and pointer become `Item$Storage` and
`GoPointer<Item, Item$Storage>`. For a scalar, whole and container storage are
the scalar and the pointer is its exact carrier. The pointer facet is the
non-nil representation; an exact Go `*T` occurrence is
`T$Pointer | undefined`. A declaration that demands no additional facet
retains only `T`.

Whole-storage and container-storage conversion are distinct closed generic
operations. Indexed address formation is another closed operation over the
source collection, index, and result pointer. It always requests the canonical
array/slice location carrier: a slot may later be rebound, so returning its
current class object would make an existing pointer observe the old value.
Generic pointer load, store, construction, equality, conversion, and indexed
address formation therefore use exact typed signatures; they do not inspect a
facet, carry semantic callbacks, recover an erased payload, or introduce a
second generic body.

For example:

```go
type Item struct{ Value int32 }
type Arena[T any] struct{ data []T }
func (a *Arena[T]) At(i int) *T { return &a.data[i] }
```

```ts
class Arena<T, T$ContainerStorage, T$Pointer> {
  data: RuntimeSlice<T$ContainerStorage>;
  At(
    i: int,
    indexAddress: (
      value: RuntimeSlice<T$ContainerStorage>,
      index: int,
    ) => T$Pointer | undefined,
  ): T$Pointer | undefined {
    return indexAddress(this.data, i);
  }
}
```

`Arena<Item>` supplies `Item$Storage` and
`GoPointer<Item, Item$Storage>` because `At` takes a slot address.
`Arena<int32>` supplies `int32` plus `GoPointer<int32, int32>`. Both concrete
capabilities return a canonical backing/index carrier, and one generic body
remains strictly typed. A generic `Bag[T]` that stores and reads `[]T` without
taking an element address instead supplies plain `Item` for
`T$ContainerStorage`; it does not acquire whole storage speculatively.

Facet demand propagates structurally through reached generic declarations and
instantiations. For example, if `Box[T]` contains `*T`, every represented
`Box[X]` supplies the exact pointer facet for `X`; if `Outer[T]` contains
`Box[T]`, its use of `Box` requests and forwards that same facet. The canonical
facet ordering is source parameter order and, within one parameter, logical,
whole storage when demanded, container storage when demanded, then pointer
when demanded. Structural comparison of that closed list drives the existing
artifact fixed point. No conditional target type, erased payload, universal
representation bag, per-use wrapper, or declaration-wide speculative facet is
admitted.

When a reached generic struct requires canonical storage and a field's logical
and storage facets may differ, the class stores only the storage facet. It
publishes one typed static whole-storage projection independent of field
count; the field-selection owner accesses the selected property and applies
the declaration-level `to-storage` or `from-storage` capability at the use
site. The class does not emit per-field accessor methods, retain converter
callbacks or descriptors, keep a second logical cache, or inspect a type
argument at runtime. Construction similarly converts each field once at the
caller and invokes the storage-valued constructor surface with explicit
canonical type arguments. A storage-facet surface change requeues field and
construction consumers through the existing observable artifact facets; no
unconditional rescan or per-instantiation class body is introduced.

A slice-to-array pointer conversion is one typed region view over the slice's
existing backing store. The slice runtime validates the requested length and
returns canonical backing identity plus absolute offset; the array runtime
constructs an offset-aware fixed-length view; the pointer runtime keys the
location by that same backing and offset. A whole-array store copies the
already value-copied source into the region. No layer copies the conversion
operand, passes a semantic callback, or creates a second address identity.
For length zero, a nil slice produces a nil pointer while a non-nil empty slice
produces a non-nil pointer, matching Go.

Slicing an array value or pointer-to-array uses the inverse view over that same
canonical storage. The array owner exposes one demand-only typed
`backing + offset` location; the slice owner constructs one descriptor with
the array length as its initial length and capacity, then applies the ordinary
slice-bounds operation. The result aliases the array in both directions and
does not copy elements. Programs without array slicing emit neither the array
location facet nor the array-to-slice view facet.

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

An artifact with a typed compile-time-only disposition may provide an explicit
empty contract while still consuming dependencies. This includes whole-file
and exported-Go-API accounting for an unused untyped constant. It is
reconstructed when a provider facet changes, but cannot dirty downstream
artifacts because it provides no facet. No generic declaration constructor may
emit an empty result. An absent contract is invalid, and an explicit concrete
representation root is forbidden from ending with an empty contract; source
accounting is therefore not confused with failed projection.

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

Cycles are permitted only when structural contracts converge. The root retains
the current exact canonical bytes and a losslessly compressed reverse delta
for each distinct changed contract. A deterministic non-semantic fingerprint may index
historical candidates, but the root must reconstruct and compare their exact
canonical bytes before declaring a repeat; a hash alone never establishes
equality. Repeating a prior non-current contract is a typed non-convergence
error, not an arbitrary iteration limit or silently accepted result. Unchanged
reconstruction adds no history. Append-only or locally changed contracts retain
only a lossless encoding of their exact changed regions rather than a full
snapshot per revision.
Dirty owners live in one identity-deduplicated deterministic priority queue.
Each wave is ordered provider-before-consumer from the exact current facet
edges, so a fan-in consumer observes all settled providers once; unrelated
ready owners use exact Go-object order as the tie-breaker. A cyclic remainder
uses that same order to choose one deterministic break point and remains
subject to exact oscillation detection. A wave completes against one applied
requirement snapshot; requirements discovered during reconstruction are
identity-deduplicated and applied before constructing the next wave. They never
cause the current dependency order to be rebuilt after each artifact. Queue and wave construction are
`O((artifacts + consumed facet edges) log artifacts)`, not repeated full-set
minimum scans. Sealing requires empty declaration, requirement, initialization,
placement, and dirty-artifact queues.

This graph is target-assembly coordination, not a semantic IR or call graph. It
contains only authoritative Go definition identities, closed target-contract
facets, and reverse invalidation edges discovered while constructing actual
TS-Go references. It does not copy source statements, infer whole-program
meaning, perform name lookup, or decide reachability.

## Generic Capability Fixed Point

Go type parameters remain TypeScript type parameters. They are never erased,
boxed into a universal payload, or expanded into one body per reached
instantiation. Operations that TypeScript cannot perform while preserving the
exact Go type travel through a statically typed target capability parameter:

```go
func Add[T ~int32](left, right T) T { return left + right }
```

```ts
export function Add<T>(
  $go$binary_add: (left: T, right: T) => T,
  left: T,
  right: T,
): T {
  return $go$binary_add(left, right);
}
```

This operation function is target ABI, not source analysis. The one ordinary
emission walk discovers a closed operation demand while constructing the actual
TS-Go body. That demand is a typed declaration requirement owned by the exact
generic `types.Object`. Its identity is `(typed operation selection, exact
receiver-free signature)` within that owner. The selection is the closed
operation kind plus exact semantic evidence that kind requires; a constraint
method includes its selected `*types.Func`, so two same-signature methods do not
collapse. Source position is not identity, so repeated uses exact-join one
hidden parameter. Reconstruction adds the corresponding hidden function
parameter to the same declaration. No prewalk, copied constraint model,
operation IR, capability object, or runtime registry exists.

The program session exposes only the current canonical generic callable
contract to a call handler. A call records a `CallableSignature` dependency,
reads the exact ordered `types.Info.Instances` arguments, and supplies one
hidden operation function per demanded semantic signature before ordinary
arguments. A concrete instantiation requests one reconstructible function
artifact keyed by `(typed operation selection, exact instantiated signature)`.
A signature
still containing the caller's type parameters projects one hidden operation
function on that caller and forwards it. Cross-parameter operations such as
`Shift[T, U](T, U) T` therefore remain one exact `(T, U) => T` function; they
are not forced into a per-parameter object. Recursive and mutually recursive
calls use the ordinary artifact fixed point. Identical contracts do not
propagate, and a repeated non-current contract remains a convergence failure.

Each concrete operation artifact delegates to the existing authoritative
value/operator/method owner. It may not restate zero, copy, equality, hashing,
numeric, conversion, indexing, method, interface, channel, or iterator
behavior. Constructed signatures containing type parameters compose by
forwarding exact enclosing operation functions; they do not create an erased
descriptor, growing object, capability factory, or per-use semantic closure.

`go/types` remains the sole authority for constraints, type sets, inference,
core types, method selection, and admissible operations. Target constraints are
not reconstructed from Go syntax. The hidden callable list contains exactly the
distinct operations demanded by the emitted declaration body. An unused
function, duplicate same-signature function, optional operation, throwing
invalid-operation function, string lookup, or universal operation bag is
forbidden.

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
- demanded named-struct static operations are incorporated into their owning
  class in a fixed operation order; no top-level sibling helper or instance
  operation is emitted;
- every reached concrete receiver body is incorporated into its exact named
  type's class, even when the Go method and type declarations are in different
  files. The method source artifact owns one typed class-member contribution;
  the type artifact owns the one reconstructed class declaration. A value
  receiver contributes an instance member. A pointer receiver contributes a
  class-owned static member with an explicit selected pointer parameter so nil
  and argument-evaluation timing remain source-controlled. No top-level
  receiver function, prototype assignment, partial class, wrapper twin, or
  second method body survives;
- one canonical generated artifact represents each exact reached anonymous
  struct or specialized map shape. A shape with no local named component enters
  deterministic compilation support; a shape containing local named
  components is emitted immediately after the deepest declaration that makes
  every component legally nameable, inside the exact reconstructible source
  artifact;
- a package initializer is a first-class artifact owned by its selected
  `(*types.Package, *types.Initializer)` identity. Its first LHS variable is
  only a source-declaration anchor, never its owner; multi-result initializers
  remain one artifact, and checker-produced blank LHS targets remain legal;
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

A class-member contribution is immutable typed TS-Go AST, not a declaration
patch. Its method owner retains the source body, callable requirements, and
imports; the containing type subscribes to that contribution's exact observable
facet and reconstructs the complete class in deterministic semantic-method
order. Changing only a method body reconstructs the type artifact but does not
requeue callers whose subscribed callable signature is unchanged. Changing a
method signature changes that facet and requeues its exact users. This is the
same canonical artifact graph used for static value operations, not a parallel
method registry or source prepass.

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

## Cooperative Concurrency

Concurrency is disabled unless the compilation entry explicitly selects the
**race-free cooperative Go** profile. Disabled compilation fails at the exact
channel, goroutine, or select construct owner. Selecting the cooperative
profile is an explicit source-program precondition, not an inferred property:
execution may switch at modeled synchronization operations, but the compiler
does not claim asynchronous Go preemption.

For example, this race-free program is outside the cooperative profile:

```go
started := make(chan struct{})
go func() {
    close(started)
    for {
    }
}()
<-started
```

Go may preempt the looping goroutine and let the receiver return; the
cooperative target cannot. Soundly detecting that requirement would need the
forbidden whole-program effect/dataflow analysis, and emulating it would need
preemption. GoToTS therefore neither admits a hidden yield heuristic nor
describes this program as exact under the cooperative selection.

Two separate semantic owners assemble declarations into the same
demand-selected `runtime/channel.ts` output module:

- the channel owner controls channel identity, value transfer, queues, close,
  direction views, and select alternatives; and
- the scheduler owner controls goroutine lifecycle, blocked/runnable counts,
  program settlement, first uncaught panic, main return, and deadlock.

Sharing an output module does not merge these responsibilities.
`GoChannel<T>` is the canonical runtime identity for a channel value.
Send-only and receive-only target views expose only operations admitted by the
selected `types.ChanDir`; conversion between views does not allocate or change
identity. The channel owner contains:

- capacity, a FIFO buffer, one insertion-ordered live sender-offer set, one
  insertion-ordered live receiver-offer set, and closed state. Direct
  operations and `select` alternatives enter the same typed queues, so arrival
  order cannot be changed by operation kind;
- exact element zero and copy functions selected from the ordinary value
  owner, including generic operation functions when `T` is a type parameter;
- send, receive, close, equality-by-canonical-identity, and select
  registration/commit operations.

Each queued sender owns typed success and close-failure commits; each queued
receiver owns one typed value/`ok` commit. A select alternative adds its
atomic claim and O(1) cancellation to that same offer. There is no parallel
listener registry, direct-versus-select queue, historical head-index storage,
operation tag, or erased task. Removing a selected alternative deletes its
live offer, so storage is bounded by current buffer contents and current
blocked operations rather than historical traffic.

Nil channels are represented by `undefined`. Send or receive on nil blocks;
closing nil, sending on closed, and closing closed channels raise the one Go
panic carrier. Closing wakes blocked receivers with buffered values first and
then `(zero, false)`, and wakes blocked senders with the send-on-closed panic.
No payload crosses `any`, `unknown`, a cast, or an erased task queue.

Blocking changes existing observable artifact facets rather than creating a
call graph. A concrete source function, method, or literal body owns one source
callable facet. A body that directly blocks, or calls a cooperative callable,
receives one closed declaration requirement; its `CallableSignature` becomes
`Promise`-returning and its body becomes `async`. Exact direct source calls
subscribe to that concrete source facet and add `await` only in its reverse
closure.

Each checker-produced `types.Initializer` is also an exact callable owner.
Its package-initializer facet is keyed by the existing
`(types.Package, *types.Initializer)` artifact identity; no package variable,
synthetic `types.Func`, or package-wide effect flag substitutes for it. If an
initializer directly blocks or invokes a cooperative direct/generic callable,
that initializer reconstructs with `await`. The passive package
`$initialize` function becomes `async` exactly when one initializer or source
`init` function in that package is cooperative. The program-initialization
module awaits exactly those package calls in Go package order. Thus
cooperation crosses initializer, package, and program assembly through the
same closed facet requirements without a call graph or unconditional async
tax.

First-class function values use one different owner: a canonical generated
callable ABI artifact keyed by the exact receiver-free `go/types.Signature`
after represented generic arguments are selected. Every value call subscribes
to that ABI once, regardless of whether the value came through a local,
parameter, result, package variable, field, pointer, array, slice, map,
interface assertion, method value/expression, or generic aggregate. A blocking
provider used as a value selects the ABI cooperative. Synchronous providers
adapt statically when that exact ABI is cooperative. Storage locations never
receive callable facets and no AST-shape resolver, dataflow graph, or effect IR
tracks transport. Structural contract equality terminates recursive cycles.
An immediately invoked function literal is also statically selected: its call
observes the literal's exact source facet and never routes through the
first-class callable ABI. Only a function value that is transported through a
value location uses that ABI. Functions, immediately invoked literals, and
callable ABIs that remain nonblocking retain byte-identical synchronous
declarations and calls.

An interface method is not a first-class function-value ABI. Its callable
state is owned by canonical interface-method artifacts keyed by exact Go
method identity and receiver-free signature. A non-generic method has one such
artifact. A method of a generic interface has a parameter-ordinal-normalized
family artifact; a closed instantiation additionally has its exact concrete
signature artifact. Open signatures containing ambient type parameters never
receive runtime method-set tokens. Runtime tokens exist only for closed
signatures, while callable observation may use the generic family and closed
facets together. Callable-family artifacts are contract-only compiler state:
they participate in exact reverse dependencies but emit no TypeScript
declaration.

A concrete adapter observes the selected source-method facets and every
demanded target-interface facet. If the source or any demanded target facet is
cooperative, the adapter selects every demanded target facet as cooperative;
one generated adapter method cannot expose contradictory call contracts
through two interfaces. Interface declarations, direct interface calls,
deferred interface calls, adapters, and constraint-method capabilities observe
only interface-method callable facets. Runtime interface contracts additionally
request the closed token. An unrelated function or differently named method
with the same receiver-free signature cannot change that interface method.

Only taking a method value or method expression crosses from the
interface-method callable facet into the canonical first-class callable ABI.
That wrapper observes the exact callable family as its provider and widens the
function-value ABI when
necessary; widening never flows back from the function ABI into direct
interface dispatch. Interface-method callable state is an observable generated
artifact facet, so a synchronous-to-cooperative change invalidates exact
subscribers through the ordinary reverse artifact graph.

There is one selected-method invocation plan for source calls and generated
adapters. It owns generic receiver type arguments, exact generic operation
capabilities, recovery-control parameters, class-versus-environment target
selection, and cooperative source observation. Generated adapters may not
bypass that plan with a raw member call.

Generic instantiation does not create a second callable-effect model. At each
generic function or generic-receiver-method use, the selected
`go/types.Instance` supplies an exact structural correspondence between
callable leaves in the declaration signature and callable leaves in the
instantiated signature. That correspondence selects a closed callable-ABI
profile for that use. The ordinary profile keeps the source declaration's
ordinary target name and its declaration-owned callable-ABI baseline.
Declaration-intrinsic cooperation flows only outward to the corresponding
concrete ABI. For example, a generic function that always returns a
channel-reading literal has a cooperative returned-callable ABI for every
instantiation. A concrete call-site ABI never widens that baseline in the
reverse direction. Each distinct reached profile whose concrete callback
requires a cooperative ABI where the declaration baseline is synchronous
requests one additional source-owned variant keyed by the declaration
identity and the sorted canonical override-facet identities. The variant is
reconstructed from the same Go declaration through the ordinary handler with
only those deviations selected; it is not a copied semantic model or a
separately lowered body.

For example, a synchronous
`Apply[T](value T, predicate func(T) bool)` use selects `Apply`, while a use
whose concrete predicate ABI is cooperative selects a deterministic
`Apply$cooperative_<profile>` variant. Calls inside the variant await only the
callable ABI facets selected by its profile. The variant itself becomes
`async` only when its body actually reaches one of those calls or another
cooperative operation; merely accepting a cooperative callback does not make
an unused callback execute. A synchronous provider may be adapted statically
when a selected profile requires a cooperative callable ABI.

No declaration-wide cooperative bit may widen all instantiations. No call
returns `T | Promise<T>`, checks for a thenable, selects a variant at runtime,
or recovers a profile from target spelling. Profiles are demand-created,
deduplicated by canonical identity, and bounded by distinct reached callable
ABI profiles rather than call count. Callable leaves inside arrays, slices,
maps, pointers, structs, tuples, and nested signatures use the same
`go/types`-proven correspondence. A type parameter itself is an opaque
substitution boundary and does not imply a callable correspondence.
Every demand-created variant of an exported source declaration is part of that
package's value surface. Package assembly re-exports the exact variant name
from the declaration's source module; consumers never bypass package assembly
or infer a profile export from spelling. Export cardinality therefore grows
with distinct reached profiles, not calls or importing files.
Direct calls, deferred calls, generic receiver calls, instantiated function
values, and generic method values/expressions all select through this same
owner. A callable returned intrinsically cooperative by the declaration
selects the ordinary source name and propagates that fact to its concrete
value ABI; it does not create a duplicate variant.

An environment-owned generic function has no retained body and therefore
cannot be reconstructed as a source variant. Its environment contract emits a
separate demand-created ambient declaration for each reached callable profile.
That declaration applies the same exact nested callable-ABI substitutions and
uses the same deterministic profile suffix, but contains no body and claims no
provider execution behavior. For example, a cooperative range callback passed
through the result of `slices.Values` selects an ambient
`Values$cooperative_<profile>` declaration whose returned iterator accepts the
Promise-returning yield ABI; calling `Values` itself remains synchronous.

The outer callable effect of an environment provider is never inferred merely
because a parameter or result contains a cooperative callable. A selected
`gostdlib` or external implementation contract must own that effect and the
matching implementation. Until then, the ambient declaration is an explicit
typed non-executable obligation. No source body is fabricated, no environment
package is entered into the source emitter, and no runtime Promise inspection
or union result is admitted.

Generic non-struct defined types retain one parameterized nominal wrapper.
Every target reference carries the exact represented `go/types` type arguments,
and every operation projects through the wrapper before applying the
underlying-family behavior. Rejecting a generic defined callable and then
treating it as a struct or raw function is forbidden: `iter.Seq[E]` is a
stable `Seq<E>` class value, and invocation uses its canonical payload
projection. Nil-capable families store `undefined` only in that payload; the
class reference itself is never optional. This gives every reached receiver
method one class-owned member surface without static wrap/project helpers.

A generated callback that inverts source control flow, such as the callback
implementing range-over-function, is not a new source callable and does not
attribute blocking work directly to the enclosing source function. Its
cooperative boundary is the canonical callable ABI of the exact callback
signature. Blocking work in the generated callback selects that ABI
cooperative; the iterator provider observes and awaits the same ABI; invoking
the iterator observes its own callable ABI and only then propagates cooperation
to the enclosing source callable. This is one ordinary facet-dependency chain,
not a callback-specific call graph. The generated callback is `async` exactly
when its ABI is cooperative, and unrelated callable signatures remain
byte-identical.

A `go` statement evaluates and value-copies the callee and every argument
immediately in source order, then schedules exactly one typed
`() => Promise<void>` closure. Main return stops the selected program without
waiting for remaining goroutines. A panic escaping any goroutine terminates the
selected program with the shared panic carrier.

`select` evaluates channel operands and send values exactly once in source
order. It creates one typed alternative per communication clause. The runtime
uniformly chooses one currently ready alternative, commits it once, and
cancels all other registrations. If no case is initially ready, alternatives
are registered in a fair permutation as active offers in the same channel
queues used by direct operations. This preserves FIFO ordering across direct
and selected waiters, permits select-to-select rendezvous, and prevents
same-channel source-arm bias. Closing a channel rejects a queued selected send
through that select's own typed completion path; it does not throw from the
goroutine performing `close`. A receive target location is evaluated only
after its clause wins. A select with a default invokes the channel owner's
synchronous fair ready-choice/commit operation: one ready communication wins,
or `default` wins when none is ready. Selection itself adds no `Promise`,
`async`, `await`, scheduler dependency, or cooperative callable requirement;
operand or send-value evaluation may independently require cooperation. A
select without a default invokes the blocking operation only when no
communication is ready and then participates in scheduler deadlock detection.
Nil-channel alternatives never become ready.

Channel send and receive call sites are O(1), apart from the selected element
copy. A select site and registration are O(number of clauses). Runtime source
size is independent of element type, channel count, goroutine count, and
select-site count. Queue storage is O(buffer capacity plus live blocked
operations), never O(historical operations).

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

Under the cooperative profile, an initializer that reaches a cooperative
callable instead emits structurally:

```ts
export async function $initialize(): Promise<void> {
  $state.Values = await Map(values, asyncValueAdapter);
}
```

Synchronous packages retain the byte-identical `(): void` form. Package
assembly derives this choice only from exact initializer and source-`init`
facets already closed by the artifact scheduler.

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

await $initialize_registry();
$initialize_y();
$initialize_b();
```

Only cooperative package calls receive `await`; the module uses ESM top-level
await so each package completes before the next Go-ordered package begins.

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

All generated runtime failures and source `panic` use one closed Go panic ABI:

```text
family runtime guard or source panic(value)
    -> GoPanic.raiseRuntime(message) or GoPanic.raise(value)
    -> one runtime/panic.ts owner
    -> one statically typed thrown GoPanic carrier
```

A family runtime must not throw a host `Error`/`RangeError` or depend on an
implicit JavaScript exception for Go panic behavior. The panic module is the
only generated runtime owner allowed to create and initially throw a
`GoPanic`. A callable unwind envelope may only rethrow an unrecognized caught
host exception unchanged; it may not inspect, convert, or recover that value.
The carrier owns one exact represented empty-interface payload. Runtime faults
and `panic(nil)` enter through distinct GoToTS-owned runtime-error dynamic
values. Both implement the canonical `error`, `interface { Error() string }`,
and `runtime.Error` method contracts, while `panic(nil)` alone carries the
canonical `*runtime.PanicNilError` dynamic identity and represented payload.
Source non-nil values enter through ordinary represented-interface conversion.
There is no generic or erased carrier payload.

`recover` consumes an invocation-local `GoRecovery` authority. Only the direct
call of a deferred function receives that optional, statically typed authority.
An ordinary call, a call made by that deferred function, and `recover` outside
deferred execution receive no authority and return nil. Function and method
values consume one canonical generated callable ABI contract keyed by their
exact `go/types.Signature`. Recovery is one optional final facet of that
contract: direct ordinary calls omit it and deferred invocation supplies it.
The ABI is not re-keyed for a variable, field, pointer, slice, map, interface
adapter, defined callable wrapper, or other carrier. Those locations store the
same callable value; they do not own recovery semantics. No per-storage
recovery identity, storage/dataflow graph, ambient global, module flag,
async-local store, dynamic property, cast, `.call`, `.apply`, or `.bind` may
select recovery.

Callable control is a demand-selected facet of the exact source function
artifact. Encountering `defer`, `recover`, or non-structural `goto` requests
reconstruction of that exact callable, including an independently anchored
function literal. The final callable owner assembles typed TS-Go AST in this
order:

```text
lexical declarations and addressable storage
    -> optional goto control structure
    -> return and named-result normalization
    -> defer/panic unwind envelope
    -> final typed TS-Go function body
```

This is target assembly in the existing declaration fixed point, not a source
prewalk or retained control model. No CFG, lowering IR, semantic plan, source
inventory, second checker, or second Go-AST traversal is retained. A callable
without a selected control facet has byte-identical target AST.

Emission context carries exactly one canonical `ArtifactOwner`. Addressable
storage derives its `*types.Func` from that owner rather than retaining a second
function-owner field. Callable-control requirements use the same owner and
carry the exact enclosing source artifact plus callable anchor; both the start
and end of a function literal must be contained by that enclosing artifact.
The selected callable map and current callable/facets are ephemeral
reconstruction facts only and are discarded with the emission context.

A `defer f(x)` registration evaluates and captures the callee, selected
receiver, and copied arguments immediately at the source statement. It pushes
one typed closure onto an invocation-local stack. The callable envelope drains
that stack LIFO on every normal return and panic exit. A later panic replaces
the pending panic without skipping older deferred calls; a direct successful
`recover` consumes the pending panic. Explicit returns in a named-result
function assign copied values to the named result locations before unwind, and
one trailing result read occurs after deferred mutation.

Labels are keyed only by exact `*types.Label` definition/use identity. Native
target labels own labeled `break` and `continue`, switch fallthrough remains at
the switch-clause owner, and structurally representable goto edges use direct
target labels. Only genuinely non-structural edges select a whole-function
linear state machine assembled from the already-created TS-Go statements.
State and generated size are linear in source statements and exact labels.
Every ordered statement list, including a block and a switch or type-switch
clause body, delegates to the same statement-sequence control owner. When a
generated state machine is nested inside a source loop, range, switch, or type
switch, that source construct receives a typed lexical target label so an
unlabeled source `break` or `continue` cannot be captured by the generated
dispatch loop or switch. This target is target-assembly context only; it is not
a persisted source-control graph.

Runtime classes may encapsulate JavaScript storage only when that storage is
the smallest exact representation of a Go value family. They may expose
ordinary statically typed methods, but no reflection, dynamic operation name,
erased payload recovery, target non-null assertion, `.call`/`.apply`/`.bind`,
per-use semantic closure, or stored semantic callback/strategy. Passing a
statically typed hash, equality, copy, zero, conversion, or dispatch function
into a runtime value still constitutes runtime semantic dispatch and is
forbidden. Nullable storage is narrowed by explicit control flow at the
runtime owner; a generated `value!` may not substitute for a proved invariant.
Compiler handlers still own Go evaluation order and copy boundaries; runtime
code does not rediscover source semantics.

Aggregate container operations therefore compose at the source operation
owner. That owner emits typed TS-Go loops which call the already-selected
zero/copy/equality/hash structures directly. Runtime array and slice classes
may expose only typed structural primitives needed to allocate, validate,
grow, and access storage. They may not receive a semantic operation as a
function parameter, retain one as a field, or create a second per-element-type
dispatch owner. Runtime members used only by `clear` or slice-spread append are
selected by closed demand and are absent from an artifact that does not use
the operation.

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
  naming/                           exact Go-identity target names and generated-artifact registry
    registry.go                     immutable target lookup and canonical generated-artifact ownership
    file_names.go                   one file/scope name service implementing `api.Names`
  placement/                        deterministic static import/request placement
  verification/                     public-surface program/session integration tests
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
      verification/                 end-to-end construct tests when the owner has a substantial matrix
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
