# Architecture

## Design Rule

GoToTS is a direct, context-aware Go-to-TypeScript compiler:

```text
selected Go packages
    -> one Go AST and one coherent go/types graph
    -> one owner-directed emission walk
    -> typed values of the pinned TS-Go external AST protocol
    -> pinned TS-Go decoder, factory, and printer
    -> strict ESM TypeScript
```

The Go AST and `go/types` graph are the only source-semantic model. Typed
TS-Go protocol values under construction are the only target model. There is
no source inventory, semantic IR, operation IR, call graph, lowering plan,
handwritten TypeScript AST, text emitter, or post-print rewrite.

Handlers may inspect the current Go node, its ancestors, the selected package
graph, and existing checker evidence. They may create typed TS-Go AST nodes and
closed root requests. They must not copy source meaning into a second model or
rerun the checker.

## Authoritative Owners

| Truth | Sole owner |
|---|---|
| selected source files and package classes | selected toolchain plus `go/packages` metadata |
| Go syntax and validity | selected toolchain parser and type checker |
| bindings, types, constants, instances, selections, signatures | the one selected `go/types` graph |
| construct meaning in context | the semantic handler selected by the parent-owned child role |
| Go callable value/type-parameter shape | selected `go/types.Signature` and declaration |
| target representation of a Go type | one representation owner keyed by canonical Go type identity |
| zero, copy, equality, hash, conversion, and arithmetic | the corresponding value-family owner |
| target declaration and later revisions | one declaration assembly keyed by exact Go identity |
| imports, placement, sealing, and printing | root emitter |
| cooperative callable effect | callable artifact plus canonical indirect-call ABI |
| panic carrier and deferred recovery entry | `runtime/panic.ts` plus the callable's private deferred entry |
| provider boundary meaning of selected Go types | `internal/contracts/gostdlib/sourcecontract` |
| immutable provider certificate documents | `internal/contracts/gostdlib` |
| provider implementation | certified `@gotots/gostdlib` export |
| provider/generated conversion | generated static facade for the selected Go callable or type |
| target AST shape and ordering | pinned TS-Go schema and generated protocol bindings |

One fact may have many references but one producer. A second workaround in the
same semantic class reopens its owner.

## Source Selection

Every compilation records an immutable Go build profile:

- selected Go executable and toolchain fingerprint;
- `GOOS`, `GOARCH`, and `CGO_ENABLED`;
- sorted build tags and explicit build flags;
- module/workspace roots and overlays.

Ambient shell values do not silently change selection. The loader uses
toolchain metadata, not import-path spelling, to distinguish:

- workspace and source-available dependency packages, which are translated;
- standard-library packages, whose declarations come from the selected
  `GOROOT` and whose behavior comes from `gostdlib`;
- toolchain/language pseudo-packages, which have explicit compiler ownership;
- true native, cgo, platform, or unavailable boundaries, which receive exact
  obligations or typed unsupported diagnostics.

A load fails on parse/type errors, inconsistent package identity, an
unavailable selected file, or build-profile drift.

## Owner-Directed Dispatch

The root has category dispatchers for declarations, statements, expressions,
types, and parent-owned syntax. A dispatcher selects one handler and never
recursively walks the node.

Each handler owns a closed child contract:

```text
parent node
    -> child field
    -> semantic role
    -> dispatch category
    -> order
    -> expected result shape and legal placement
```

The parent consumes inseparable syntax itself and delegates each independently
meaningful child exactly once. For example, a call owner decides which child is
the callee, which are ordinary arguments, whether the last argument is
variadic expansion, and the expected parameter type of each argument. An
identifier child never scans upward to rediscover that role.

The selected Go parser/checker rejects grammatically or semantically illegal
children. Production does not duplicate the Go grammar. Verification derives
the selected toolchain's AST fields and forms and exact-joins them to handlers
and parent-owned dispositions.

## Project Structure

The directory tree grows by semantic ownership, not by a flat file per syntax
node:

```text
cmd/gotots/                         CLI only
internal/load/                      selected toolchain/package graph
internal/contracts/gostdlib/        immutable provider certificate documents
  sourcecontract/                   selected-Go-type boundary interpretation
  certify/                          Go/provider exact joins and generation
internal/emit/
  root/                             orchestration, placement, convergence
  declaration/
    function/
    variable/
    constant/
    type/
  expression/
    call/
    selector/
    index/
    operator/
    literal/
  statement/
    assignment/
    branch/
    loop/
    range/
    defer/
    select/
  type/
    basic/
    defined/
    aggregate/
    signature/
    interface/
  value/
    zero/
    copy/
    equality/
    conversion/
  provider/
    contract/
    facade/
    obligation/
  runtime/
    panic/
    pointer/
    map/
    channel/
internal/target/tsgo/               generated protocol bindings/encoder
internal/verify/                    independent walls and product gates
schema/tsgo/                        pinned external TS-Go contract
gostdlib/                           standalone manually maintained provider
```

Nesting is introduced when a semantic owner has multiple independent
sub-owners. Root files orchestrate; they do not accumulate feature logic.
Maintained non-generated files stay focused and below 600 physical lines.

## Emission Context And Results

A handler receives immutable context:

- current lexical and artifact owner;
- parent-assigned role and expected result shape;
- exact `go/types` evidence;
- target scope capabilities;
- selected compilation/build profiles;
- a narrow child-emitter interface.

It returns typed TS-Go AST values plus closed requests:

- imports and support modules;
- placement at a preferred legal scope;
- declaration requirements;
- observable-facet dependencies;
- typed diagnostics.

It does not receive arbitrary mutable target ancestors. The root owns file,
class, function, block, and expression builders.

### Placement

Every prerequisite names:

1. its semantic execution boundary;
2. all legal scopes;
3. its preferred scope.

The root inserts at the preferred legal scope, not merely the nearest legal
scope. Static imports always go to file scope even if discovered in a function.
Evaluation-sensitive temporaries remain at the exact expression/statement
boundary. When an expression needs statements, its result carries those
prerequisites so the parent can either place them legally or choose an
expression-only construction. Dynamic imports are forbidden.

## Revisable Target Artifacts

Emission is open until every requested target artifact reaches a fixed point.
One declaration assembly owns the complete pre-seal TS-Go AST for one Go
definition. A later semantic demand may request a revised constructor, member,
modifier, copy operation, interface adapter, helper import, or callable effect.

References subscribe to closed observable facets such as:

- callable signature;
- instance surface;
- static surface;
- constructor surface;
- exported value surface.

Body-only changes do not invalidate users. When a facet changes structurally,
the root requeues only its reverse subscribers. Reconstruction transactionally
replaces the artifact's complete AST, requests, dependencies, and facet
contract. Identical facets do nothing; oscillation fails. Final printing occurs
only after convergence.

Derived aggregate artifacts, including package export assemblies, are
published only after declaration discovery, declaration requirements, reverse
dependency reconstruction, package initialization discovery, and requirement
removal have reached quiescence. An aggregate may then subscribe to the final
facets of its members and be reconstructed by a later real member-facet
change. The root must not commit incomplete intermediate membership as an
aggregate revision: a declaration that first exposes a source facade and then
settles on a private kernel must not create a false `absent -> present ->
absent` package-export cycle.

There is no text patching, shared mutable TS-Go node mutation, spelling-keyed
dependency, unconditional rescan, or duplicate old/new path.

## Source-Shape Conservation

Every target callable that claims a Go source identity directly projects the
selected `go/types.Signature`.

- Ordinary value parameters preserve order and cardinality.
- A value receiver may become TypeScript `this`.
- A pointer receiver may become one explicit first parameter so nil reaches the
  Go method body.
- A variadic parameter remains one semantic slot, represented either as the Go
  slice value or one TypeScript rest parameter.
- A cooperative result may become `Promise<R>` or `Awaitable<R>`.
- Source type parameters preserve order and cardinality.

No source-facing declaration, function value, interface method, environment
callable, package export, or ordinary call gains a recovery authority,
operation function, provider policy, bridge set, scheduler state, storage
facet, profile selector, or digest-named parameter/variant.

Compiler-owned private support definitions may have implementation parameters
only when they claim no Go source identity and are not exposed through package
assembly. Every generated source call still has the source argument list.

## Callable Representation

### Direct Calls

A statically selected function, method, or immediately invoked literal uses its
exact callable artifact. If the body blocks under the cooperative profile, its
result is Promise-bearing and exact direct callers emit `await`. Synchronous
direct paths stay synchronous.

### Indirect Calls

The compiler does not build a transport/dataflow graph to predict which
function implementation reaches a variable, field, interface, or container.
Under the cooperative profile, each receiver-free Go function signature has
one canonical indirect target ABI:

```ts
type Awaitable<T> = T | Promise<T>;
type GoFunc<A, R> = (argument: A) => Awaitable<R>;
```

The same rule applies recursively to callable parameters/results and to
interface methods. An indirect call unconditionally `await`s the result.
This is statically typed and performs no thenable inspection. A synchronous
implementation is directly assignable; an asynchronous implementation is also
assignable. Under the disabled-concurrency profile the ABI is synchronous and
blocking constructs fail at their owner.

Named function types retain exactly their source type parameters. They do not
gain a hidden payload/effect type parameter. Method values and method
expressions create typed wrappers with the exact receiver-free Go signature;
they never use `.call`, `.apply`, or `.bind`.

Within cooperative compilation, this single ABI replaces callable-profile
matrices, public cooperative variants, effect-dependent named-type arity, and
runtime Promise detection.

## Values And Representations

The representation owner chooses the smallest exact direct TypeScript shape.
Defaults prefer readable source and no semantic machinery:

- integer profile defaults to `number`; `bigint` is an explicit profile;
- evaluation defaults to direct TypeScript evaluation; `preserve-go` enables
  additional ordering temporaries;
- copy, pointer, interface, map, channel, and runtime support is demanded only
  when a selected occurrence requires it.

GoToTS-owned scalar aliases preserve Go type identity for future consumers
while mapping to ordinary TypeScript primitives. Generated output imports no
unrelated compiler.

Nil-capable values use `undefined` unless their family requires a distinct
carrier. Zero, copy, equality, hashing, conversion, and mutation are each owned
once by the value family. A class gains `$copy` only when the compilation
requests copying that class; revisable artifact reconstruction adds it before
seal.

Pointers identify addressable locations, not merely values. A direct value is
used when no address/alias semantics are demanded. A pointer carrier is
introduced at the owning addressable location only when mutation, aliasing,
address comparison, or nil pointer behavior requires it. Read-only scalar
arguments do not become carriers merely because Go's source type is a pointer
when the checker and use contract prove direct representation exact.

Maps use one canonical generated `GoMap<K,V>` runtime type because JavaScript
`Map` does not preserve Go key equality, zero-on-miss, comma-ok, copy, or
iteration semantics for all Go keys. Source-facing variables keep their Go
names; runtime class names are stable semantic names, never per-site hashes.

## Structs, Methods, And Embedding

A Go struct is emitted as a TypeScript class when that preserves its selected
value and method behavior. Fields are initialized to Go zero values.

- value-receiver methods are instance methods using `this`;
- pointer-receiver methods are class-owned static/unbound methods whose first
  parameter is the represented receiver;
- nil pointer receivers enter the body; dereference panics only where Go would;
- ordinary concrete calls use the exact `go/types.Selection`, not target
  virtual lookup.

Embedding does not automatically mean `extends`. Native inheritance is used
only when one proved spine preserves promoted field/method selection, nil
behavior, value copying, construction, shadowing, and addressability. Otherwise
the embedded field remains composition and promoted access follows the exact
selection path. A type may `implements` an interface only as a target
declaration aid; Go interface satisfaction remains structural and checker-owned.

## Interfaces

Every reached concrete-to-interface conversion requests one typed adapter for
the exact concrete dynamic type. The adapter owns:

- canonical runtime type metadata;
- exact demanded method tokens;
- the represented payload;
- Go equality/comparability behavior;
- native constant-size dispatch methods.

Interface calls are O(1) and do not emit implementer switches. Adapter methods
invoke the exact concrete owner, preserving value-copy and pointer semantics.
Interfaces carry no `any`, `unknown`, reflective lookup, source-name tests,
or erased payload recovery.

Under the cooperative profile, interface method results use the canonical
`Awaitable` callable rule. A synchronous concrete method and an asynchronous
one satisfy the same generated method contract without profile variants.

### Go Reflection Metadata

Go reflection is a language/runtime observation over the selected Go type
graph; it is not JavaScript reflection. The compiler owns one canonical,
identity-keyed runtime type descriptor for every reached reflection-visible Go
type. The descriptor is derived directly from `go/types` and contains only the
facts required by the Go `reflect` contract: kind, defined identity, package,
display form, size/alignment, element/key/length relations, fields/tags, method
tokens, and typed value operations. Recursive relations are lazy references to
canonical descriptors, never copied nested descriptions.

The same concrete Go type identity owns interface dynamic identity and
reflection identity. `reflect.TypeFor[T]()` selects its canonical descriptor
statically from exact type-argument evidence and retains zero source value
parameters. `reflect.TypeOf` and `reflect.ValueOf` consume the descriptor and
typed payload already carried by the canonical interface adapter. A generic
body whose type argument is still open requests one private typed runtime-type
operation through the existing generic concretization/capability owner; it does
not attempt to recover an erased TypeScript type argument.

Reflection visibility follows typed value reachability, not compilation-wide
adapter existence. A static `TypeFor[T]` or concrete `TypeOf(value)` requests
the exact concrete descriptor. A `TypeOf` or `ValueOf` whose operand has an
interface type subscribes the existing canonical interface-contract demand;
every concrete adapter that reaches that contract, whether discovered before
or after the observation, requests its descriptor. Interface-to-interface
transitions preserve that closure. An unrelated adapter must not acquire
reflection metadata merely because the compilation contains another
reflection call. A registry-wide descriptor sweep, source-spelling scan,
points-to side model, and post-generation registration pass are forbidden.

Portable `reflect.Type` and `reflect.Value` behavior belongs to the `reflect`
provider implementation over those descriptors. Generated per-type adapters
own typed access to represented values, fields, elements, maps, pointers, and
setters. The provider never receives or recovers an erased payload. Runtime
constructor names, object-key enumeration, property spelling, source-name
tables, `any`, `unknown`, unchecked casts, and product/package-specific
descriptor overrides are forbidden.

Descriptor definitions are emitted once and use-site references are O(1).
Metadata growth is linear in reached canonical types plus their actual
fields/methods; it may not grow by type-pair, call-site, implementer, or package
cross-product. An unsupported reflection operation fails at its closed
operation owner rather than falling back to host reflection.

The descriptor record is sparse but named: mandatory identity/kind/text/size/
alignment facts are explicit, while closed contract defaults and absent
kind-specific facts are omitted. A top-level struct-field ordinal is derived
from its canonical field order unless an operation requires a different index
path. Empty tags, package paths, zero offsets, and false embeddedness are not
repeated. This is a source-size representation rule only; the provider exposes
the complete Go reflection result. Per-type self-only assignability,
convertibility, or implementation arrays are forbidden: those relations must
be derived exactly from the canonical type graph when supported, or the
operation must fail at its owner.

## Generics

Go type parameters remain TypeScript type parameters when the body is directly
expressible and exact:

```go
func Identity[T any](value T) T { return value }
```

```ts
export function Identity<T>(value: T): T {
  return value;
}
```

The source declaration never receives operation dictionaries/functions for
copy, zero, arithmetic, equality, conversion, methods, channels, or iteration.

When an operation cannot be expressed exactly over open `T`, the declaration
owner emits one internal kernel and one source-shaped facade for each exact
reached `go/types.Instance`. The kernel may receive the smallest statically
typed callable for each indivisible semantic operation used by the body. Each
facade constructs those callables from the existing concrete semantic owners
and retains only the source parameters after substitution. These callables are
internal artifact wiring: they never enter a source-facing signature, a
first-class operation bag, or runtime selection. Kernel, facade, and operation
identities derive from Go declaration identity plus exact type arguments, not
source position or target spelling.

For example, Go accepts `append(dst, src...)` when `src` has type parameter
`B ~[]byte | ~string`. TypeScript has no one static append expression over
both target representations. The function owner requests one internal
`append-spread` callable. Its `[]byte` facade binds the canonical slice-spread
operation; its `string` facade binds the canonical string-spread operation.
Neither facade adds a source argument, performs a runtime type test, or
reimplements slice behavior.

Direct generic bodies are emitted once. Internal kernels and facades grow only
with distinct reached instances that require them. Recursive requests
exact-join; an unbounded chain, unsupported open exported case, or
representation disagreement fails explicitly. No erased payload, broad
operation dictionary, hidden source parameter, runtime type switch, or
alternate semantic path exists.

Generic named types retain exactly their declared type parameters. Closed
representation operations may be private specialized artifacts; they do not
change public type arity.

Open generic bodies refer to representation through three closed associated
type facets owned by the representation classifier:

```ts
type GoStorage<T> = T extends GoStoredValue<infer S> ? S : T;
type GoContainerStorage<T> = T extends GoContainerStoredValue<infer S> ? S : T;
type GoPointerType<T> =
  T extends GoPointerRepresentedValue<infer P> ? P : GoPointer<T, T>;
```

Concrete generated classes declare only the demanded zero-runtime
`unique symbol` marker members. The marker target is the already-selected
logical, storage, container-storage, or pointer representation; it never
chooses representation itself. A provider certificate must prove an
equivalent marker for a non-default provider representation. Identity storage
and the canonical logical pointer carrier use the conditional fallback and
need no marker.

These facets do not add source type parameters, source value parameters,
runtime dispatch, casts, or operation kernels. A private kernel is selected
only when the body requires a runtime operation; representation-only demand
remains one direct generic declaration.

## Panic, Recover, Defer, And Control

`runtime/panic.ts` owns one typed Go panic carrier. Generated Go runtime
faults and source `panic` enter through that carrier; unrelated host
exceptions are rethrown unchanged.

A source callable containing `recover` has:

1. its ordinary source-facing entry with the exact Go signature, where
   `recover()` returns nil; and
2. one private deferred-entry artifact that accepts the invocation-local
   recovery authority.

Both may share a private body implementation. The private entry claims no Go
source identity and is never exported as the source callable.

A statically known `defer f(args)` captures copied arguments immediately and
selects `f`'s private deferred entry when one exists. Transported function
values use one exact-signature typed deferred-entry registry. A recover-capable
function/literal registers its ordinary value and private deferred entry when
the value is formed; defer lookup falls back to ordinary invocation when no
entry exists. Registries are generated only for demanded signatures and use no
`any`, `unknown`, dynamic properties, or `.call/.apply/.bind`.

Interface and provider deferred calls use equivalent private adapter/facade
entries. Source method/function signatures remain unchanged.

The invocation-local defer stack captures values at the statement, drains LIFO
on return or panic, preserves named-result mutation and panic replacement, and
awaits only entries whose typed internal contract is Promise-bearing.
`recover` one call below the deferred entry receives no authority.

Native target control is used where structurally exact. Only genuinely
non-structural `goto` selects a linear statement state machine assembled from
already-created TS-Go statements. No CFG or control IR is retained.

## Cooperative Channels And Goroutines

Concurrency is disabled by default. Selecting `cooperative` explicitly
accepts race-free cooperative semantics: execution may switch at modeled
synchronization operations but does not emulate asynchronous Go preemption.

One `GoChannel<T>` runtime identity owns capacity, FIFO buffering, close
state, and insertion-ordered live send/receive offers. Direct operations and
`select` alternatives use the same queues. Nil channels block; close and
closed-channel behavior use the Go panic carrier. Element zero/copy comes from
the concrete value owner; an unresolved generic operation concretizes its
owning callable.

The scheduler separately owns goroutine lifecycle, settlement, uncaught panic,
main return, and deadlock. A `go` statement evaluates and copies callee and
arguments immediately, then schedules one typed closure.

Send/receive sites are O(1) apart from value copy. A `select` is O(case
count), commits once, cancels alternatives in O(1) each, uses fair ready
selection, and retains storage only for buffered values plus live blocked
operations. A default-only ready decision stays synchronous.

## Provider And External Boundary

`@gotots/gostdlib` is a standalone, manually maintainable strict-ESM package.
Public subpaths mirror Go import paths and named exports preserve Go names:

```text
"strings" -> @gotots/gostdlib/strings.js
"io/fs"   -> @gotots/gostdlib/io/fs.js
```

Its public declarations preserve selected Go source arity. Host backends remain
private and do not alter public module names.

The selected `GOROOT` remains declaration truth. A provider certificate
exact-joins:

```text
Go package/object identity + go/types signature
    <-> public provider module/export
    <-> implementation owner and inspected target signature
```

The immutable certificate package contains no `go/ast` or `go/types` graph.
`internal/contracts/gostdlib/sourcecontract` is the sole focused owner that
interprets selected Go signatures as provider-boundary contracts; certifiers
and emitters consume that same owner. It distinguishes unnamed direct
callbacks from named callable values, describes interface methods, and resolves
typed provider protocols. No consumer repeats those decisions or moves them
into serialized documents.

The provider source is independently built against a generated direct-profile
runtime contract. A linked product owns its selected generated runtime package;
cooperative products consume the provider declarations through certified
cooperative facades rather than changing the provider's direct implementation
ABI. Both runtime profiles are generated from the same runtime contract owner.

Generated source never calls a provider kernel with extra source arguments.
When canonical generated values require conversion, guards, runtime tokens,
copy/zero operations, or a specialized generic implementation, the compiler
emits one static facade for that selected callable/type:

```ts
export async function WalkDir(
  fileSystem: FS,
  root: gostring,
  visit: WalkDirFunc,
): Promise<GoError | undefined> {
  return fromProviderError(
    await provider.WalkDir(toProviderFS(fileSystem), root, toProviderVisit(visit)),
  );
}
```

The facade has the source value parameters. Required bridges and operations
are statically imported at module scope or closed over by private
compiler-owned helpers. No policy object, bridge set, capability function,
profile key, or recovery authority appears at a generated call site.

Provider-created interface values cross one certified static bridge preserving
nil, dynamic identity, method tokens, equality, and canonical callable ABI.
Generated inputs cross the inverse bridge only where representation differs.
Nested callbacks, tuples, containers, fields, and results follow the same
type-directed boundary rule. Missing or ambiguous conversion fails
certification; TypeScript assignability alone is not semantic evidence.

A provider-owned named callable has the source type-parameter arity and the
canonical indirect callable representation. A provider-private callable ABI is
not a source type argument, default, constraint, or alternate public alias.
When provider parameter or result representations differ from generated ones,
the static provider facade owns one typed adapter at the crossing; ordinary
generated declarations and values never inherit the provider ABI.

Provider boundary certificates are mode-bounded, not method-profile matrices.
An ordinary direct call uses the source-synchronous ABI. A required semantic
protocol such as `errors.Is` may therefore have one direct certificate whose
transported methods are all synchronous. Cooperative compilation may have one
certificate for the same Go identity whose transported methods all use the
canonical `Awaitable` ABI. No source identity may have more than one certificate
for either boundary effect, and no certificate may mix synchronous and
`Awaitable` transported methods. Ordinary direct provider calls that need no
semantic protocol use their regular binding and require no duplicate profile.

Certification derives one total directional boundary-obligation set from the
selected Go signatures and the inspected provider project. Each source
parameter is inspected recursively for generated values that the provider will
invoke: named interface methods and callable values, including callables nested
inside another callable. Their canonical direct or cooperative effect is joined
exactly to one ordinary binding or one private facade certificate. A source
result is an outward conversion owned by that selected binding/facade; it never
creates an inward facade obligation by itself. Missing, duplicate, extra, mixed-
effect, or wrong-direction certificates fail contract generation before source
emission. Emission consumes this certified set and cannot discover profiles one
call site at a time.

An unnamed function parameter is a directly transported callable. A named
function type is instead certified once by its defined-value representation and
all uses reference that owner; it is not reclassified as a fresh direct callback
at every parameter. If a recursively nested callable shape has no typed path
representation in the current contract schema, certification rejects the
provider surface explicitly rather than flattening it into its outer effect.

Private provider facades have one certified support suffix after their source
parameters, receiver, and canonical value parameters. Its order is fixed:
interface guards, interface contracts, then provider-to-generated bridges.
Every support parameter has one named provider-owned marker type, and contract
generation exact-joins both the total arity and each ordered marker identity.
These are statically imported facade dependencies; they never form a policy
object or appear at a translated source call site.

For example, cooperative `sort.Sort(data sort.Interface)` requires one private
facade whose `Len`, `Less`, and `Swap` inputs use the canonical `Awaitable` method
ABI. Cooperative `sort.Search(n, predicate func(int) bool)` requires one private
facade whose predicate result is `Awaitable<bool>`. The public `Sort` and
`Search` exports remain direct and source-shaped, and neither facade adds a
source parameter or runtime policy object.

For example, Go `io/fs.WalkDirFunc` remains one non-generic generated callable
type whose result is `Awaitable<error>`. The provider may implement its private
visitor over provider `DirEntry` and `GoError` values, but the generated facade
statically bridges that private visitor to the canonical source type. It does
not expose `WalkDirFunc<Value = PrivateABI>` or require a generated use to
supply an implementation type.

Provider fingerprints use semantic member identities. A public computed
`unique symbol` member is keyed by the checker-resolved symbol and its project
owner; TS-Go's allocation-dependent display spelling is never persisted as
contract identity.

A provider recovery facet is an optional closed certificate for one exact Go
callable. Presence selects that provider's private recovery-aware entry for a
deferred call. Absence selects the ordinary source-shaped entry; it is not an
error and cannot trigger inference, a generated bridge-wide recovery method,
or a hidden argument. Public provider interfaces and ambient declarations
never contain recovery authority.

Representation-dependent provider generics use direct exact TypeScript when
possible. Otherwise a reached exact instance selects a private generated
facade/concretization or private provider kernel. Public/provider and generated
source callables retain source arity.

When a private generic kernel transports callbacks, its outer effect and each
callback parameter effect are exact-joined to the public provider binding.
The kernel cannot silently narrow an `Awaitable` callback to synchronous,
duplicate a cooperative implementation, or introduce another public profile.

Compile-only mode emits exact typed throwing placeholders and canonical
obligations. Linked mode uses certified provider facades. These are explicit
profiles, never fallback paths in one compilation. Publication requires every
reachable obligation to be implemented.

## Package, Runtime, And Target Output

Each Go package emits deterministic ESM modules plus one package assembly.
Mutable package variables live in one state module. Checker
`types.Info.InitOrder` and the package import graph determine package and
program initialization order; target import order does not.

The package assembly exports the exact observable contract of each exported
Go declaration. That contract contains the source-facing binding and any
demand-created associated representation binding required to use it across a
package boundary. For example, `type Item struct { Value int }` exports only
`Item` until a reached pointer or container operation materializes
`Item$Storage`; the assembly then exports both from the same defining module.
Private kernels, temporaries, and undemanded representation facets are never
published. The declaration handler owns this set; the package assembly does
not infer it from spelling or scan printed statements.

Generated support is GoToTS-owned under `runtime/` and demand-created. The
same physical runtime package is linked into generated code and `gostdlib`,
so class identity, panic carriers, maps, slices, pointers, channels, and
interface tokens are unique.

Every target file is built from exact generated bindings for the pinned TS-Go
external AST protocol. The encoder validates required fields and discriminants.
Pinned TS-Go decodes, constructs real nodes, and prints. No target text exists
before printing, and no post-print mutation is allowed.

## Complexity And Failure

Ordinary source growth must produce proportional target growth. In particular:

- concrete/interface calls are constant-size in implementer count;
- generic direct bodies do not grow with call count;
- concretizations grow with distinct necessary exact instances;
- provider facades grow with distinct selected boundary contracts;
- fixed-point work is bounded by changed artifact facets and reverse edges;
- runtime source is independent of runtime value count.

Unknown syntax, missing checker evidence, unsupported representation,
unbounded concretization, unresolved provider conversion, non-convergence,
schema drift, or blocked strict typing fails with a typed diagnostic at the
owning construct. There is no heuristic recovery, compatibility path, dynamic
semantic dispatch, or threshold increase in place of a fix.
