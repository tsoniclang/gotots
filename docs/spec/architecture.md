# Architecture

## Design Rule

GoToTS is a direct, context-aware Go-to-canonical-TypeScript compiler:

```text
selected Go packages
    -> one Go AST and one coherent go/types graph
    -> one owner-directed emission walk
    -> typed values of the pinned TS-Go external AST protocol
    -> pinned TS-Go decoder, factory, and printer
    -> canonical strict ESM Tsonic-flavored TypeScript
```

The Go AST and `go/types` graph are the only source-semantic model. Typed
TS-Go protocol values under construction are the only target model. There is
no source inventory, semantic IR, operation IR, call graph, lowering plan,
handwritten TypeScript AST, text emitter, or post-print rewrite.

Canonical output is consumed through a separate exact target boundary:

```text
immutable canonical text
    -> TSTS checking, TS-Go-contract AST, and finalized facts on exact nodes
    -> target-owned transformation of that same TS-Go-contract AST
    -> stable printer boundary and target runtime
```

The target boundary is not a second Go semantic model. TSTS selects marker
facts by canonical declaration identity. Targets consume those facts and must
not recognize marker spelling, scan source text, join source ranges, reparse,
reread files, or re-enter the checker. The first TypeScript target transforms
the checked TS-Go-contract AST directly. Bootstrap printing uses the pinned
TS-Go decoder/factory/printer through a framed adapter; a later TSTS-native
printer may replace that adapter without changing target semantics.

Handlers may inspect the current Go node, its ancestors, the selected package
graph, and existing checker evidence. They may create typed TS-Go AST nodes and
closed root requests. They must not copy source meaning into a second model or
rerun the checker.

## Authoritative Owners

| Truth | Sole owner |
|---|---|
| selected source files and package classes | selected toolchain plus `go/packages` metadata |
| `//go:embed` patterns, selected files, and exact payload bytes | selected toolchain plus loader-owned immutable evidence joined to the declaration's `go/types.Var` |
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
| selected environment object, closed use demand, and sole implementation route | root environment scheduler and settled environment builders |
| final settled environment-use evidence | the root's immutable post-fixed-point obligation projection |
| provider implementation disposition and private dependency closure | provider certification over the strict checked provider project |
| provider/generated conversion | generated static facade for the selected Go callable or type |
| target AST shape and ordering | pinned TS-Go schema and generated protocol bindings |
| target-neutral marker identity and fact meaning | shared Tsonic/TSTS contract |
| exact checked source snapshot, AST node, and marker fact | finalized TSTS source program |
| executable representation, target AST transformation, and target runtime | selected target |
| bootstrap TypeScript printing | pinned TS-Go printer adapter |

One fact may have many references but one producer. A second workaround in the
same semantic class reopens its owner.

## Source Selection

Every compilation records an immutable Go build profile:

- selected Go executable and toolchain fingerprint;
- `GOOS`, `GOARCH`, and `CGO_ENABLED`;
- sorted build tags and explicit build flags;
- module/workspace roots and overlays.

### Exact Tool Authority

The project configuration is the sole selector for both compiler tools.
`tools.go` selects an optional Go executable, `tools.tsgo` selects an optional
TS-Go executable, and `tools.cache` selects the project-owned `.temp/cache`
root used to seal executables, snapshot `GOROOT`, and hold subprocess scratch. The corresponding
CLI overrides are `--go`, `--tsgo`, and `--tool-cache`. An omitted Go selection
means the ordinary `go` found for the invocation; an omitted TS-Go selection
means the pinned tool resolved by that exact selected Go executable in the
configured distribution module. Selection happens once before loading.

Resolved operational evidence keeps the selected executable paths, sealed
`GOROOT`, and cache path so users can inspect what ran. Semantic project
evidence contains no machine path. Its Go identity consists of the selected
`GOVERSION`, sealed executable digest, and digest of the complete selected
`GOROOT` contract. Symlinks in that root are admitted only when they resolve
inside the root and are fingerprinted by normalized root-relative target, so
relocating the same root does not change identity. The contract includes the
actual compile/link tools, standard-library source/export inputs, and all
other root assets that selected Go commands may consume.

`ResolveGo` verifies the selected source executable immediately before and
after identity discovery, inventories the complete reported root, and copies
that exact normalized manifest into a content-addressed root under
`tools.cache`. File bytes, executable modes, empty directories, and normalized
in-root symlink targets are part of the manifest; escaping links, special
members, copy drift, and candidate drift fail before publication. An existing
digest root is reopened only after its manifest and every member exact-join.
After resolution, commands execute only the sealed executable with `GOROOT`
pointing at the sealed snapshot; the mutable source root is neither retained
nor read.

Per-command verification hashes the sealed executable and checks the sealed
root's opaque handle and small seal document. It never walks or hashes the
complete root. One complete snapshot verification runs after all compilation
subprocesses and before the staged output transaction is published. Thus root
integrity and command cost are simultaneous requirements rather than a choice
between exactness and repeated whole-toolchain I/O.

The selected Go's reported `GOVERSION`, `GOROOT`, default `GOOS`, and default
`GOARCH` create the build profile. Because GoToTS directly uses the
`go/ast` and `go/types` implementation compiled into its executable, admission
requires that executable's frontend version to equal the selected
`GOVERSION`; selecting a different command cannot replace the in-process Go
frontend. Runtime defaults never substitute for the selected tool's reported
profile. Selected commands receive only the sealed Go directory on `PATH`,
the selected `GOROOT`, and scratch variables rooted in `tools.cache`.
`GOTOOLCHAIN` is local and `GOENV` is disabled. A cgo profile fails before
loading until an exact external-tool contract is explicitly selected; ambient
C compilers and host `PATH` are never inherited.

`go/packages` runs through one exact self-driver using that same selection,
profile, build flags, tests setting, and overlay bytes. The public driver JSON
schema omits `Dir`, `Module`, and `ForTest`; the driver therefore writes those
facts from the same loaded graph to one request-scoped cache evidence file.
The consumer exact-joins roots, package identities, files, and import edges
before attaching the omitted fields. It never launches a non-overlay metadata
query or constructs a second package universe.

The TS-Go identity binds its sealed executable digest, selected-Go identity,
build Go version, and one of exactly two pinned forms: the pinned module
version and checksum, or a clean source build with the pinned module path,
exact VCS revision, `vcs.modified=false`, and no replacement. A development
binary without that exact VCS evidence is foreign. AST printing, provider
certification, source-implementation certification, and strict TypeScript
compilation all consume the same resolved TS-Go object; none resolves a tool
again.

Ambient shell values do not silently change selection. The loader uses
toolchain metadata, not import-path spelling, to distinguish:

- workspace and source-available dependency packages, which are translated
  unless an exact certified package implementation is selected;
- standard-library packages, whose declarations come from the selected
  `GOROOT` and whose behavior comes from `gostdlib`;
- toolchain/language pseudo-packages, which have explicit compiler ownership;
- true native, cgo, platform, or unavailable boundaries, which receive exact
  obligations or typed unsupported diagnostics.

A load fails on parse/type errors, inconsistent package identity, an
unavailable selected file, or build-profile drift.

The loader requests the selected toolchain's embed patterns and files, joins
each parsed `//go:embed` directive to its exact package variable identity, and
reads each selected payload once. Emission consumes that immutable evidence;
it never rescans source text, expands patterns independently at emission time,
or reads payloads from the host at runtime. Ordinary patterns exclude hidden
and underscore-prefixed files while `all:` patterns include them, including
when both forms overlap in one package.

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

The callable-signature facet includes one ordered canonical mapping for every
source parameter and result. Its authoritative key is the selected Go callable
object, not a package/name string. A Go `*T` maps to
`Pointer<T> | undefined`; GoToTS does not replace it with `T` because a
particular body appears read-only. Receiver and variadic mappings remain the
only source-shape transformations explicitly owned by this specification.

A certified authored implementation is checked against the same canonical
surface. It may replace an algorithm or private storage under its equivalence
envelope, but it cannot select a target representation by changing a pointer
parameter to a scalar. Such scalarization is a selected-target optimization
over finalized facts and must rewrite the complete location flow there. No
GoToTS config key, annotation, package name, function name, or caller allowlist
selects it.

Every direct call, method value/expression, function value, callback, interface
adapter, deferred entry, provider bridge, package export, and definition
subscribes to the same settled signature facet. A structural projection change
reconstructs those consumers through the ordinary reverse-dependency graph;
an implementation-body-only change does not. A plain value projection never
silently discards a source-observable write. Such a callable retains a location
unless a separately specified input/output projection owns an explicit
write-back rule.

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
- Their target representations may use only the settled closed callable
  projections above; this does not add, remove, reorder, or hide a parameter.
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

Portable algorithms receiving that canonical ABI must preserve it directly.
They may remove avoidable recursion, allocation, and iterator layers, but they
must not inspect whether a result is a Promise. The canonical cooperative
generic path uses one bounded-work iterative merge over two dense buffers and
awaits every comparison.

A reached private generic concretization may instead select one separately
certified synchronous provider kernel when every callback argument named by
that kernel resolves statically to a synchronous callable facet. The effect is
part of the concretization identity, all call sites and adapters use the same
exact synchronous contract, and an unresolved, escaped, or cooperative
callback retains the canonical `Awaitable` path. This is not a public callable
profile: it adds no source parameter, public overload, runtime result test, or
spelling-based selection. A direct callback's internal defer/recover envelope
remains in its ordinary entry and therefore does not require the deferred-call
registry at this exact ordinary invocation. Native `Array.sort` remains
reserved for a provider kernel whose exact synchronous contract and observable
ordering are certified. Captured method values and function variables remain
on the canonical path; the first captures receiver state and the second may
have escaped or changed.

Possibly nil indirect calls have one target owner. The emitter captures the
callee, captures every argument in Go order, then calls
`(callee ?? GoPanic.raiseRuntime("call of nil function"))`. The statically
typed nullish expression adds no helper invocation, source parameter, erased
type, host-shape inspection, or semantic dispatch. Statically non-nil callees
omit it. No conditional, target assertion, or alternate nil-call path coexists.

## Values And Representations

The representation owner chooses the smallest exact direct TypeScript shape.
Defaults prefer readable source and no semantic machinery:

- integer profile defaults to `number`; `fixed64-bigint` and `bigint` are
  explicit profiles;
- evaluation defaults to direct TypeScript evaluation; `preserve-go` enables
  additional ordering temporaries;
- copy, pointer, interface, map, channel, and runtime support is demanded only
  when a selected occurrence requires it.

GoToTS-owned scalar aliases preserve every selected Go basic identity. In
particular, `int`, `uint`, and `uintptr` remain distinct aliases rather than
being rewritten as `int32`/`int64` or `uint32`/`uint64`. Their carrier width is
derived once from the selected package graph's `types.Sizes`; inconsistent
width evidence in one compilation fails before emission. The `number` profile
maps every integer alias to `number`. The `fixed64-bigint` profile maps
`int64` and `uint64` to `bigint` while retaining `number` for native-sized and
narrower aliases. The `bigint` profile additionally maps 64-bit native
`int`, `uint`, and `uintptr` to `bigint`. Generated output imports no unrelated
compiler.

Profile selection also fixes the overflow contract. The direct `number`
profile retains its declared precision/overflow tradeoff. Both explicit
BigInt-carrier profiles normalize every `int64` and `uint64` operation to its Go
width at the integer value owner. The `bigint` profile does the same for
64-bit native integers; `fixed64-bigint` deliberately retains the direct
number-profile tradeoff for native `int`, `uint`, and `uintptr`. A reached
identity or control decision that depends on exact fixed-width 64-bit
arithmetic therefore requires at least `fixed64-bigint`; one that depends on
exact 64-bit native-integer overflow requires `bigint`. There is no package
allowlist, adaptive representation heuristic, or hidden per-call profile.
Wide-result normalization requests one of the two demand-generated integer
runtime operations; repeated source sites do not duplicate the target
intrinsic expression.

Defined-value representation is selected once from the complete `go/types`
declaration, never from a use site. A non-generic source-owned defined
fixed-width integer with no value or pointer methods uses a nominal TypeScript
numeric enum and direct numeric storage. The enum's one value member provides a
typed no-op wrap for conversions and operation results, eliminating per-value
objects without erasing nominality. Method-bearing, generic,
profile-dependent, non-numeric, provider-owned, and otherwise non-identity
families retain their canonical wrapper or provider representation. All
consumers query this one representation owner; none infer it from target
spelling or structural assignability.

When an identity conversion from that nominal enum initializes a target whose
TypeScript type would otherwise be inferred, the declaration owner emits the
selected converted basic type annotation. This is a static inference boundary,
not a second representation or runtime conversion; declarations and contexts
that already preserve the selected type emit no annotation.

Nil-capable values use `undefined` unless their family requires a distinct
carrier. Zero, copy, equality, hashing, conversion, and mutation are each owned
once by the value family. A class gains `$copy` only when the compilation
requests copying that class; revisable artifact reconstruction adds it before
seal.

Pointers identify typed writable locations, not merely values. Canonical source
always preserves that meaning with `Pointer<T> | undefined` from
`@tsonic/core/types.js` and the accepted `addressOf`, `allocatePointer`,
`loadPointer`, `storePointer`, and `equalPointer` operations from
`@tsonic/core/lang.js`. GoToTS emits explicit Go nil checks before loads and
stores and emits `equalPointer` for pointer comparison. Passing, returning,
storing, or copying a pointer preserves the same location identity.

Addressable locals, fields, package variables, parameters, named results,
array/slice elements, and pointer-to-pointer values remain distinguishable
through the selected operation facts. An address fact retains the complete
typed location path and its evaluation boundary: replacing a value-struct root
retargets an interior field location, while reassigning a pointer-valued root
does not. Slice-element locations retain the selected backing store even when
the slice variable is later rebound. GoToTS emits the exact marker occurrence
from its Go AST and `go/types` evidence; it does not decide whether a target
uses a class object, scalar snapshot, `{ value: T }` location, native pointer,
or another representation.

TSTS owns canonical marker recognition and retains the exact call, operands,
source types, typed location path, and mutability evidence. GoToTS does not
emit a parallel storage cell, storage-name requirement, or generic
cell/load/store capability: ordinary declarations remain ordinary TypeScript
declarations and location identity exists only in the selected marker facts.
The selected target owns flow-wide representation choice. It may eliminate a
location only when every definition/reference in that location flow is
rewritten and observable alias, mutation, nil, identity, and lifetime behavior
remain exact. A local decision at one call site is forbidden.

For example, the TypeScript target may turn a complete non-escaping read-only
`Pointer<int>` flow into a `number`, while a flow with two mutating aliases uses
one shared runtime location. Canonical source is unchanged in both cases, and a
native target consumes the same finalized facts without reverse-engineering a
Node-oriented carrier.

### Unsafe Pointer Memory

Raw addresses are a different semantic class from typed locations. The shared
contract owns only opaque raw-pointer identity: `RawPointer`,
`bindRawPointer`, `equalRawPointer`, and `hashRawPointer`. GoToTS may convert a
safe typed pointer or a certified provider raw-pointer result to that identity,
preserve `undefined` as nil, copy or box it, and use the canonical equality and
hash operations. The marker and its target runtime expose no address or
pointee.

Offsets, reinterpretation, raw-pointer-to-typed-pointer conversion,
pointer/integer conversion, and raw pointer input to a provider remain typed
boundaries until separately accepted contracts own those operations. GoToTS
emits no substitute virtual address, JavaScript cast, identity extraction, or
target-specific codec in canonical source. Safe pointer semantics are never
routed through a legacy raw-memory implementation to keep a corpus compiling.

Maps have one representation owner and three storage modes. An exact built-in
boolean, integer, or string key with a runtime-basic value uses the canonical
`GoMap<K,V>` runtime. Closed map shapes that need static zero, copy, hash, or
equality operations use one deduplicated support specialization for each exact
semantic shape.

A specialization uses native JavaScript `Map` storage only when its selected
key is an exact built-in boolean, integer, or string. Tuple cells preserve the
difference between an absent key and a present `undefined` value. All other
specialized keys use typed hash/equality buckets because JavaScript `Map`
identity is not Go equality for those classes. Both representations retain the
same nil, zero-on-miss, comma-ok, copy, mutation, and iteration contracts;
neither is selected by a source spelling or use site.

Native storage compares the already-selected primitive carrier. Consequently,
wide integer keys under the `number` profile retain that profile's declared
precision collision envelope, including `uint64` values immediately above
`2^53`. Exact fixed-width 64-bit map identity requires `fixed64-bigint` or
`bigint`; the map owner does not add a hidden hash, conversion, or profile
override.

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

Runtime type constructors compose from that same graph. For example, when a
runtime-flowing descriptor `typ` reaches `reflect.PointerTo(typ)`, the portable
reflection owner canonicalizes `*typ` by descriptor identity, derives its
method set from compact, collision-checked identities produced by the existing
interface-method identity owner, and preserves the selected provider profile's
pointer size and alignment. The compact set is a representation of canonical
method tokens, not a spelling lookup or a second relation table. The compiler
does not guess a finite pointer closure, require `*T` to appear in source, or
install a product-specific descriptor. Repeated composition (`T`, `*T`,
`**T`) remains canonical and grows only with the types actually composed at
runtime.

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

Open generic bodies refer to storage representation through the closed
associated facets owned by the representation classifier:

```ts
type GoStorage<T> = T extends GoStoredValue<infer S> ? S : T;
type GoContainerStorage<T> = T extends GoContainerStoredValue<infer S> ? S : T;
```

Concrete generated classes declare only the demanded zero-runtime
`unique symbol` marker members. The marker target is the already-selected
logical, storage, or container-storage representation; it never
chooses representation itself. A provider certificate must prove an
equivalent marker for a non-default provider representation. Identity storage
uses the conditional fallback and needs no marker. A generic pointer is the
canonical target-neutral `Pointer<T>` and therefore has no generated associated
pointer facet.

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

Every cooperative task belongs to a typed settlement owner. Provider
synchronization primitives may transport a rejected task result only to that
owner and rethrow the exact failure when the corresponding wait settles; they
must not discard a launched Promise or inspect/recover its payload. In
particular, `WaitGroup.Go` task failure cannot let `WaitGroup.Wait` report
success.

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

`internal/contracts/externals/certify` is the sole transient owner that reloads
and exact-joins source declarations for an external-provider certificate. It
may read the selected Go AST and type graph while certifying; the finalized
external contract and every downstream consumer remain graph-free.
External module bindings use the same certified provider scalar ABI as
`gostdlib`; portable source redirects remain in the selected product ABI.
The compiler emits type-directed scalar conversion only around a module
binding whose provider and product carriers differ. The external certificate
therefore records the provider integer representation, not a product integer
or concurrency profile.

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

A provider interface certified as sealed-native is already one canonical
`GoInterfaceValue`; it receives no wrapper or open-interface bridge. Its
generated method-token contract remains the runtime assertion truth. Where a
TypeScript control-flow boundary must recover the sealed provider type (for
example an interface assertion or reflective interface-field assignment), the
interface-value owner emits a typed predicate that delegates to that canonical
contract guard and narrows to the certified provider type. It does not cast,
inspect host shape, repeat the method test, or introduce a second admission
rule.

A direct provider-owned Go struct remains one object identity across the
boundary, but each exported field is independently certified against its
selected Go field identity and type. A field whose provider and product
representations differ is projected at every semantic operation: reads convert
provider to product, writes convert product to provider, and field addresses
use one bidirectional pointer projection that retains the original Go address.
Construction converts each supplied field to the provider representation;
zero and copy use the struct's certified provider operations. Equal-
representation fields stay direct and acquire no adapter. The compiler may not
create a shadow struct, duplicate field state, infer a field from spelling, or
let the provider carrier leak into generated source semantics.

Direct source implementations have one certified pointer ABI. A pointer to a
named Go struct is the provider object itself: its object identity is the Go
location, reads observe that object, and writes use the struct's certified
stable-assignment operation so existing aliases observe whole-value
replacement. Every other pointee uses the provider-owned
`ProviderPointer<T> { value: T }` carrier from the runtime contract. Provider
source imports that carrier, never canonical Tsonic markers. At the generated
boundary, a provider result becomes one canonical `bindPointer` marker over
the provider identity plus exact read and write closures; nil remains
`undefined`. A canonical named-struct pointer input is loaded and passed as
the provider object. A canonical non-object pointer input is a typed
unsupported boundary until shared Tsonic owns an exact inverse external-
location transport contract. A detached wrapper, cast, identity cache, or
package-specific pointer rule is forbidden.

For example, under the product `number` profile and the provider `bigint`
profile, Go `runtime.MemStats.Alloc uint64` is observed as a generated `number`
but stored in the provider object as `bigint`. Reading performs the checked
provider-to-product conversion, assigning performs the reciprocal conversion,
and `&stats.Alloc` projects those same conversions over the provider field's
pointer. Conversely, `runtime/metrics.Description.Name string` remains a
direct string field. `unicode.RangeTable{LatinOffset: 5}` converts only the
`int` constructor argument; its equal-representation slice fields remain
direct.

When a slice element has different provider and generated representations, the
boundary emits one typed, bidirectional slice projection. Reads convert from
provider storage, writes convert back to provider storage, and nil, length,
capacity, reslicing, append, copy, and ordinary pointer identity continue to
refer to the source backing store. An eager element copy is forbidden because
it loses observable Go aliasing. Ordinary equal-representation slices remain
the existing `RuntimeSlice<T>` with no projection, branch, or per-element cost.
The companion pointer-storage projection is a demanded runtime facet; ordinary
pointer runtimes do not carry it.
The projection is a runtime demand, not a provider-symbol exception. It owns
its contiguous-region operation explicitly: nil remains nil, while a non-nil
projected region raises the typed unsupported boundary until an exact projected
region exists. It may not inherit the direct slice's null backing, materialize
a detached array, or recover through a cast. A product claiming that operation
as supported must replace the explicit boundary with an exact projected-region
model and differential proof.

The provider's scalar ABI is a separate certified fact from the product's
selected integer profile. For a selected provider build, the runtime contract
records one provider integer representation and one native integer width. The
provider imports its GoToTS-owned scalar aliases from a provider-owned module;
it never imports product-selected scalar aliases from `@gotots/runtime`.
Provider implementations therefore strict-typecheck once against a stable,
exact ABI. For the 64-bit `bigint` provider ABI, `int64`, `uint64`, `int`,
`uint`, and `uintptr` are `bigint`; narrower integer aliases are `number`.

Generated static facades compare the source Go type's product carrier with the
certified provider carrier. Equal carriers cross unchanged. A differing
carrier is converted in the facade with typed TS-Go AST (`BigInt` plus
`BigInt.asIntN`/`asUintN`, or `Number` according to the declared `number`
profile tradeoff). The source-facing callable retains exactly its Go
parameters; no profile, policy, converter, or hidden scalar argument is added.
Host APIs that require JavaScript `number` receive it only at a provider-owned,
range-checked host boundary. Provider algorithms whose Go contract requires
64-bit arithmetic operate on `bigint` and never narrow through `Number`.

A provider bridge is revised by the same exact interface-contract demand graph
as a generated concrete adapter. If a provider-created value of a base Go
interface can also implement another reached Go interface, the provider
certificate owns one static capability view:

```ts
function asProviderUnwrapper(
  value: ProviderError,
): ProviderUnwrapper | undefined;
```

The view parameter is the certified provider representation of the base Go
interface. Its non-nil result is the certified provider representation of the
target Go interface. Certification exact-checks both identities, the complete
target method contract, one synchronous view call, and the implementation
owner. The view implementation may use provider-owned classes or other static
provider facts; it may not inspect member spelling or host object shape.

Every exported capability view has one closed certified usage. A
`provider-internal` view supports a hand-maintained provider algorithm and is
never visible to generated bridge reconstruction. A `generated-bridge` view is
eligible for reverse Go-interface demand. It may certify either the ordinary
direct provider ABI or the selected callable profile's canonical ABI. Bridge
construction selects the view whose complete certified base interface equals
the active direct provider ABI, or whose method effects equal the active
canonical generated ABI; a demanded contract with no exact candidate fails.
The compiler indexes only `generated-bridge` views; neither export naming,
TypeScript assignability, nor a source/profile spelling may choose a view or
promote a provider-internal view into generated output.

Only a reached source assertion, type switch, or certified facade guard creates
the corresponding bridge demand. Bridge reconstruction evaluates each demanded
view once per crossing, adds the target generated method tokens only for a
non-nil view, and delegates demanded methods through the stored typed view.
Absent capability evidence means provider-created values do not implement that
target; it never triggers inference. Same-name incompatible Go method
signatures are represented by exact TypeScript overloads, and capability views
for those mutually exclusive Go contracts must not both select one value.
Capability views and method tokens remain private bridge mechanics and never
change a translated source signature.

A provider-owned named callable has the source type-parameter arity and the
canonical indirect callable representation. A provider-private callable ABI is
not a source type argument, default, constraint, or alternate public alias.
Certification derives that canonical representation for every methodless named
callable; no source-identity list selects it. A named callable with methods must
instead have an explicit operations owner, because erasing its nominal carrier
would erase observable behavior.
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
function type is instead certified once by its defined-value representation;
every generated use renders that certificate's canonical underlying callable
shape rather than importing a provider-private alias or reclassifying the type
at each parameter. If a recursively nested callable shape has no typed path
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

A private generic kernel is parameterized over the generated caller's certified
logical, storage, container-storage, and pointer facets. Those facets remain
caller-owned; they are not silently replaced with the provider's scalar
profile. A non-callable contract shape containing a Go type parameter is
therefore transported opaquely to the kernel, while a callable shell is adapted
recursively so its concrete non-generic parameter and result leaves still cross
the provider boundary. Only concrete source leaves outside a generic-owned
shape use ordinary provider scalar projection.

A canonical provider parameter or result recursively projects containers and
profile-owned interface/callable leaves. A canonical leaf already expressed in
the generated ABI remains unchanged rather than being reboxed through an
ordinary provider-interface bridge. Thus `[]fs.DirEntry` projects every entry
through the selected `DirEntry` profile bridge, while canonical `error` values
accepted by `errors.Is` preserve generated `Is` and `Unwrap` method sets.

When a private generic kernel transports callbacks, its outer effect and each
callback parameter effect are exact-joined to the public provider binding. A
second private synchronous kernel is admissible only as a certified narrowing:
it has the same Go identity, type projection, capability order, and source
value shape; every direct callback changes from `Awaitable` to synchronous;
the outer result changes from cooperative to synchronous; and the target
implementation is independently inspected. Selection belongs to an exact
private concretization and fails closed to the canonical kernel whenever any
callback is not statically synchronous. The kernel cannot introduce another
public profile or use runtime Promise detection.

Compile-only mode emits exact typed throwing placeholders and canonical
obligations. Linked mode uses certified provider facades. These are explicit
profiles, never fallback paths in one compilation. Publication requires every
reachable obligation to be implemented.

### Selected Environment Use And Provider Closure

A certificate proves linkage, not behavior. A certified provider binding whose
selected implementation still raises a typed unsupported boundary satisfies no
publication requirement. Reachable-obligation acceptance therefore extends to
provider-internal behavior: a satisfied generated obligation or a linked
certified binding is never, by itself, proof that the selected provider body
is implemented.

Every environment target selection—provider reference or facet, compiler
intrinsic, generated runtime facet, or explicit boundary—synchronously
observes its canonical `go/types` object at the root environment owner with
one closed monotonic use demand and exactly one implementation route before
the target is returned. A selection route that can produce an environment
target without that observation is invalid. Closed use demands are: type
contract, value, callable behavior, state, initializer, interface capability,
callback capability, and generated runtime facet. Repeated references
deduplicate the canonical definition while monotonically joining demand.

After the compilation fixed point, the root's immutable environment-obligation
projection carries, for every settled environment declaration: canonical
declaration identity, joined demand, sole implementation route, final
requirements and facets, and the exact toolchain, schema, provider, and build
digests. The projection is deterministic proof evidence. It is not a usage
ledger, side table, or second source model, and it is never read to drive
emission.

Provider certification assigns every behavior-bearing binding and private
provider facet exactly one mechanically derived implementation disposition:

- `implemented` — exact behavior is supplied and certified;
- `profile-boundary` — one named selected profile intentionally excludes the
  behavior;
- `placeholder` — a typed throwing boundary that is never publishable when
  used.

Dispositions and private dependencies come only from the strict checked
provider project: every value-level reference in a checked implementation body
is conservatively retained as a dependency and resolved to its exact symbol;
type-only references create no behavior edge; interface, callback, and host
operations terminate at certified capability or host boundaries; an unresolved
dynamic call or ambiguous symbol fails certification. Canonical placeholder
identity derives from checked caller/symbol evidence, never a free-form
string argument.

The used-provider closure is computed from the provider-routed settled roots
through certified implementation dependencies and capabilities under one exact
build and provider profile. Every join is by canonical identity and reports
both one-sided residual lists on mismatch. Compiler-intrinsic and
generated-runtime-facet rows join their existing artifact owners and must not
also demand a provider body; a dormant provider catalog entry for such an
identity stays outside the closure. Unused catalog entries remain outside the
closure; extra closure rows are invalid. Closure construction is linear in
nodes plus edges.

Product sealing fails on any used `placeholder` and on any used
`profile-boundary` whose exclusion the selected product profile cannot prove
statically; runtime non-execution and argument-value guessing are not proof.
The publication scope is the exact settled closure, not the entire standard
library.

## Package, Runtime, And Target Output

### Project Configuration And Source Implementations

One versioned project-local `gotots.json` is the configuration truth owner.
The command selects it with `-c` or `--config`; omission means exactly
`./gotots.json` in the invocation directory. No parent, home, global, ambient
environment, emitter-local, or package-name configuration path exists.
Relative source, implementation, output, and report paths resolve from the
configuration file's directory. Resolution order is typed defaults, project
file, then explicit CLI values. Unknown fields, versions, identities, and
conflicting owners fail.
Implementation bundles may live in the consuming project, a sibling checkout,
or any explicitly selected absolute location. Their location conveys no
semantic ownership: each contract owns source paths relative to its own
directory and is selected only by canonical package, module/version, build,
and compilation evidence. Repeated `--implementation-bundle` values replace
the configured bundle set as one deterministic CLI override.

The resolved project is split before compilation into immutable loader,
compilation, implementation, and output contracts. Emitters receive only the
typed compilation and certified implementation contracts; they never read
JSON, flags, paths, or environment variables. The semantic project digest
includes the semantic Go and TS-Go identities, build and compilation profiles,
implementation contract, and source digests. It excludes selected
executable/cache paths and output/report paths.

The output contract is canonical strict ESM source. Project assembly writes a
root `package.json` with `type: module` and the exact selected physical package
dependencies (`@gotots/runtime` and any selected certified providers).
Every generated module outside the runtime package imports runtime declarations
through the canonical `@gotots/runtime/*.js` package identity, never through a
relative path. Runtime-package modules may use relative imports within that one
package. This keeps generated source and certified provider implementations on
one nominal TypeScript identity even when the installed package is a symlink to
the emitted runtime directory.
`@tsonic/core` is omitted because TSTS supplies that authoritative virtual
module rather than resolving a physical package. Top-level await is never made
valid by rewriting generated modules as CommonJS. GoToTS prints and seals that
source, but does not fabricate `@tsonic/core` declarations, emit a target
`tsconfig`, or invoke a TypeScript checker that lacks TSTS's authoritative
virtual marker modules. TSTS checks the exact immutable canonical source, finalizes every
selected marker fact, and the selected target then strict-typechecks the
executable artifact. The configured output directory is compiler-owned and is
reconstructed through a staged replacement; a successful build cannot retain
an artifact from an earlier canonical build.

Compilation-scoped generated support definitions retain their full semantic
artifact identities, but those identities do not each create a physical ESM
module. The output owner deterministically coalesces them by semantic family
and the first byte of the artifact digest, giving at most 256 shards per
family. A shard is only a placement container: every definition, dependency,
revision, and observable fingerprint remains keyed by the full artifact
owner. Lexical artifacts remain with their lexical owner. This bounded static
layout replaces one-module-per-artifact output; it uses no runtime registry,
dynamic import, bundler dependency, erased lookup, or source-name grouping.

Schema version 1 has the closed top-level sections `distribution`, `source`,
`go`, `semantics`, `providers`, `implementations`, `output`, and `tools`.
`distribution.root` identifies the installed GoToTS distribution that owns the
default pinned TS-Go module context and checked providers; it is operational
path evidence and is excluded from semantic identity. `tools.go`,
`tools.tsgo`, and `tools.cache` select the operational tools and cache root.
`source` selects one package pattern and root mode.
`implementations.bundles` contains exact package-contract paths. Every field
except `schemaVersion` has one registered CLI counterpart, including
repeatable `--tag` and `--implementation-bundle` flags.

A certified source implementation owns one exact source package's final target
module set. GoToTS certification joins canonical Go module, package, version,
selected build and compilation profiles, exact export identities,
implementation source digests, and the equivalence envelope. It
strict-typechecks the authored implementation project and structurally joins
the complete ordinary and installed canonical sets without inventing marker
declarations. TSTS checks both immutable canonical sets with its authoritative
marker modules and finalized facts. The selected target then lowers and
strict-typechecks both executable sets under one final module-resolution and
strictness contract. The installed target check is the authoritative proof
that every generated consumer accepts the replacement. TypeScript display text,
parameter names, and package-private representation shape are not semantic
package contracts and must not be compared as if they were. Selection is
settled once before target files are sealed. The final file set replaces the
complete generated package module set atomically; no generated body, package
state, initializer, compatibility wrapper, or fallback for the selected
package may survive. References keep the ordinary package assembly path and
source-facing contract.

Replacement uses two compilation sessions, not a filter over an assembled file
list and not rollback within one mutable graph. The first session settles the
ordinary canonical program solely for certification. It captures the complete
ordinary target set, immutable observable contracts for selected-package
source artifacts, and each contract artifact's exact outgoing support
requirements and observable dependency edges. The deterministic name and
generated-support identity registry is transferred as the one identity owner
from the completed first session to the final session. The sessions never use
it concurrently. Artifact revisions, liveness, builders, placement, requirement
scheduler state, and emitted declarations are not transferable. The first
session's remaining state is then discarded.

A fresh final session starts with empty artifact, liveness, requirement,
representation, placement, and target-builder state after taking ownership of
the one canonical name/support-identity registry. When a final consumer requests a
selected-package source artifact, the source owner publishes the captured
observable contract with no body, storage, initializer, class contribution, or
target declaration and reinstalls only that contract's captured outgoing
support requirements and observable dependency edges. A dependency whose
provider belongs to any selected replacement package is satisfied by that
authored package transaction and is not replayed as a generated source edge;
generated support and source dependencies outside the replacement set remain
exact. Selected-package source code is never traversed for body translation in
this session. The shared
checked program may be read only to index top-level declaration identity and
locate a captured source owner; neither operation emits a request or
representation. Therefore only requirements consumed by the final graph can
materialize shared support artifacts. All selected source-implementation
bundles form one replacement transaction. After final quiescence, every public
consumer is rebound while the complete final file set still consists only of
compiler-owned, inspectable TS-Go ASTs. Only after every bundle has been
rebound may the compiler remove all selected generated package module sets and
install all authored opaque TS-Go ASTs. Installing one authored bundle before
rebinding another is forbidden: an authored official source file is an output
artifact, not input to a later compiler transformation. The first session's
ordinary target set remains certification evidence. For example, `ErrOverflow` retains its
exact value contract without generated storage, while a private `worker`
reflection descriptor requested only by a Go body cannot enter the contract
session or the final graph. A genuine final consumer of a private type must
be satisfied by an exact body-free private contract module. Transferring either
session's artifact or liveness state, late import deletion, output-path
filtering as liveness, same-session withdrawal, and private generated runtime
shims are forbidden. The transferred registry may retain canonical interned
contract facts but cannot schedule or materialize output; the final requirement
scheduler is the sole liveness owner. Sharing either artifact graph, scheduler,
builder, liveness ledger, or emitted declaration is forbidden.

Registry transfer preserves semantic identity, not allocation identity. A
generated contract fact recreated by the final session exact-joins the
registry's existing fact by stable key, source owner, type arguments,
signature, placement, and lexical anchor, and the registry returns its one
canonical artifact. Equality of Go wrapper pointers is not evidence. The same
stable key with any differing semantic field fails closed.

Before ownership transfer, the registry deletes every first-session
observation set: interface transitions, adapter/bridge reachability,
provider-capability demands, reflection demands, and value-operation demands.
Those are liveness inputs, not identity facts, and the final session must
derive them only from its own consumers. Transfer and final-session claim are
each single-use operations.

For callable exports, the surface join is signature-exact rather than
name-only. GoToTS preserves the selected Go signature in ordinary canonical
callers and binds the authored module by exact export identity. TSTS is the one
owner that compares the generated and authored canonical parameter/result
types under the authoritative virtual marker modules. For example, Go
`Read(*int) int` requires authored
`Read(Pointer<int> | undefined): int`; authored `Read(number): number` is a
target-specific optimization and is rejected at this canonical boundary.

Each authored bundle contains exactly one executable package assembly.
Surviving imports of its public exports are rebound to that assembly in the
typed target AST. The set may additionally contain only the finite body-free
private contract modules still named by generated support artifacts. Each
private module is bound to one selected Go source-file identity, exact-joins
the imported private names to its checked exports, and contains no runtime
statement. It is not a retained generated body or a general forwarding shim.
An absent, extra, executable, value-imported-private, or profile-mismatched
private module fails before output is sealed.

Authored implementation source is not copied as output text. Pinned TS-Go
parses it, returns its official external AST encoding, and that typed
`SourceFile` is carried to the same `printNode` path as generated nodes. The
compiler never patches or reparses printed output.

An implementation may use private native JavaScript storage and a declared
equivalence envelope. The envelope is admissible only at the package contract
owner and must name every intentionally relaxed observable. Exact public
types, source arity, deterministic behavior, equality/collision obligations,
and failure boundaries remain mandatory. If a value escapes the proved
envelope, certification fails rather than silently selecting the replacement.

Each Go package emits deterministic ESM modules plus one package assembly.
Mutable package variables live in one state module. Checker
`types.Info.InitOrder` and the package import graph determine package and
program initialization order; target import order does not.
Each state field carries the value's storage representation, not its public
value wrapper. Reads, writes, initialization, and addresses use the same
storage projection owner, so a later-demanded struct storage facet revises the
state declaration and every dependent artifact together.
Compiler-supplied `//go:embed` values initialize their owning package storage
before source initializers through the same package-state assignment path.
String payload materialization preserves every Go byte, including NUL and
invalid UTF-8, rather than interpreting payload text as Unicode source.

The package assembly exports the exact observable contract of each exported
Go declaration. That contract contains the source-facing binding and any
demand-created associated representation binding required to use it across a
package boundary. For example, `type Item struct { Value int }` exports only
`Item` until a reached pointer or container operation materializes
`Item$Storage`; the assembly then exports both from the same defining module.
An exported non-generic Go interface always publishes its canonical runtime
contract and guard with the type (`Reader`, `Reader$contract`, and
`Reader$is`), because cross-package type assertions and reflection observe
those value bindings. A selected authored package must implement that same
closed public ABI; consumers never import the retired source module for them.
Private kernels, temporaries, and undemanded representation facets are never
published. The declaration handler owns this set; the package assembly does
not infer it from spelling or scan printed statements.

Generated support is GoToTS-owned under `runtime/` and demand-created. The
same physical runtime package is linked into generated code and `gostdlib`,
so class identity, panic carriers, maps, slices, pointers, channels, and
interface tokens are unique.

The runtime package manifest maps each public `.js` subpath directly to that
same `.js` path. This one stage-neutral export lets NodeNext resolve the
corresponding `.ts` file while canonical source is being checked and the
co-located `.d.ts` declaration after JavaScript emission. A manifest must not
name declarations that do not yet exist in the canonical source tree.

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
