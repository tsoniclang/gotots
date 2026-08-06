# Verification

## Proof Principle

Every construct family is test-first and closes through independent evidence:

1. a smallest valid Go fixture;
2. a focused test observed failing at the owning unsupported boundary;
3. exact TS-Go AST shape assertions;
4. strict TypeScript typechecking;
5. Go-versus-generated-ESM differential execution where executable;
6. an independent structural/semantic comparison;
7. a mutation that the owning gate catches;
8. generated artifact and cost inspection;
9. broad deletion searches for the superseded path.

Implementation tests alone do not prove architecture. Generated artifacts are
mandatory evidence.

## Baseline Gates

Every checkpoint runs, in dependency order:

1. `gofmt` check;
2. schema fingerprint and generated-binding drift check;
3. architecture walls and package-layer registry;
4. focused tests for changed owners;
5. `go build ./...`;
6. `go vet ./...`;
7. `go test -count=1 ./...`;
8. race tests for materially concurrent compiler/runtime owners;
9. representative TS-Go encode/print and strict TypeScript checks;
10. applicable differential/runtime and artifact-size gates.

Heavy jobs run one at a time in `.temp/`, with explicit timeout,
`GOMEMLIMIT`, low `GOMAXPROCS`, disk-backed logs, and breadcrumbs. Failure
artifacts remain available so timeout, OOM, type error, and semantic mismatch
are distinguishable.

## Construct Coverage

Verification derives from the selected toolchain:

- all active `go/ast` node forms;
- every exported node-bearing child field and deterministic visit order;
- all relevant `go/token` operators;
- predeclared `types.Universe` objects.

The derived domain exact-joins:

- one production dispatcher owner;
- one closed child role or explicit parent-owned disposition;
- one supported or typed-unsupported semantic contract.

Mutations remove a dispatch arm, child edge, child order, role, operator, or
built-in identity. Each must fail with exact node/file/span evidence. Parser
recovery forms remain rejected.

This is a verification inventory, not a production source model.

## TS-Go Target Proof

The pinned TS-Go schema is copied under `schema/tsgo/`. Generation derives:

- kind IDs and discriminants;
- field names and optionality;
- aliases/unions;
- factory/visitor ordering;
- encoder validation.

Tests exact-join generated bindings to the pinned schema and send actual
protocol values through pinned `tsgo --api printNode`. Required-field
deletion, kind renumbering, field reordering, local object-shape substitution,
or text-emitter fallback must fail.

Broad walls reject:

- target text fragments/templates/formatters;
- handwritten target AST shapes;
- TypeScript internal-memory assumptions;
- post-print text mutation;
- imports from unpinned TS-Go internals;
- `any`, `unknown`, unchecked casts, reflection, and dynamic shape tests.

An adversarial target-intrinsic fixture declares a Go `String` object while a
conversion in the same module requires the target `String.fromCharCode`.
Strict artifacts retain the Go name and qualify the host value through
`globalThis`. A second fixture declares a Go generic type named `Promise` while
emitting a cooperative callable: the Go type is target-renamed, while the
callable retains idiomatic global `Promise<T>`. A closed contract test
enumerates every supported host value and reserved host type identity.
Replacing a qualified host value with a bare identifier, or removing the host
type reservation, must fail strict typechecking or the intrinsic-shape gate.
The same artifact inspection declares a Go `Object` type: the declaration is
target-renamed because TS-Go rejects a class named `Object`, its source module
uses `globalThis.Object.freeze`, and independently generated support modules
use direct `Object.freeze`. Moving either form across that lexical boundary
must fail the strict, shape, or source-size/AST-node gate.

## Closed Child Proof

For each contextual handler:

- ordinary, contextual, and adversarial source forms exercise every child role;
- a direct-child exact join proves each meaningful child is consumed once;
- parent-owned syntax is explicitly listed and cannot also dispatch;
- prerequisites are inspected at their execution boundary;
- illegal placement fails rather than moving silently.

Mutations duplicate a child, omit one, reverse order, supply a wrong expected
type/arity, or move an arm prerequisite outside short-circuit control.

## Source-Shape Conservation Gate

Every emitted callable claiming a Go source identity is mechanically joined to
its selected `go/types.Signature`.

For each function, literal, concrete method, interface method, function type,
method value/expression, provider/environment callable, and generic
concretization, record:

- exact Go identity;
- receiver mapping;
- ordered value parameters and variadic bit;
- ordered results;
- ordered source type parameters;
- selected cooperative result mapping;
- target declaration identity.

The gate asserts:

- value parameter count/order equals Go after the documented receiver mapping;
- source type-parameter count/order equals Go;
- variadic representation is one semantic slot;
- every source call supplies exactly the source argument cardinality;
- package exports expose each demanded cross-package representation binding
  exactly once and expose no private or undemanded support declaration;
- indirect callable/interface ABI changes only result Awaitability;
- private deferred/concretized/facade helpers claim no source identity.

Static searches reject source-facing occurrences of:

- `$go$recovery` or equivalent recovery parameters;
- `$go$binary_*`, `$go$copy`, `$go$zero`, constraint-method, or operation
  parameters;
- provider policy/bridge/capability arguments;
- public `$Value`/storage/effect type parameters;
- digest-named cooperative/profile exports.

Required mutations append each forbidden parameter/type parameter, publish a
private helper, alter receiver placement, duplicate a variadic slot, or select
an unjoined provider signature. The signature gate must fail before printing.

Representative artifact proof includes:

```go
func Add(left, right int) int
func (v Value) Read() int
func (p *Value) Set(int)
func Apply[T any](T, func(T) bool) bool
```

and verifies human-shaped target declarations with no additional source
parameters.

## Revisable Artifact Proof

Tests create a declaration, then discover a later demand that changes:

- implementation only;
- callable signature;
- class member surface;
- exported value surface.

An implementation-only change reconstructs the provider but does not requeue
users. A changed observable facet requeues exactly reverse subscribers;
transitive signature changes continue until fixed point. Identical requests
deduplicate. Oscillating mutations fail.

Tests additionally prove:

- class `$copy` appears only after an exact copy request;
- imports/helpers discovered late are placed once;
- reconstruction replaces, rather than patches, the complete AST;
- old requests/dependencies disappear;
- sealed AST cannot be edited.
- a late-reached exported generic declaration settles all intrinsic
  requirements before its package export aggregate is published; a private
  kernel replacing its temporary source facade neither leaks the facade nor
  trips the genuine-oscillation gate.

## Value-Family Proof

Each type/value family has focused differentials and mutations:

| Family | Required cases |
|---|---|
| constants | typed/untyped, contextual projection, iota, inherited specs, cross-package, alternate spelling |
| basic numbers | both profiles, shifts, division/remainder, boundaries, declared number-profile tradeoffs |
| structs | zero, keyed/unkeyed literals, copy demand, field mutation, nested values |
| arrays | length, zero, value copy, index/address, nested elements |
| slices | nil, make, append, capacity, overlap copy, slicing, bounds, element storage |
| pointers | nil, alias, read/store, equality, local/field/index addresses, read-only direct proof |
| unsafe pointers | typed-pointer round trip, live safe/unsafe aliasing, array/struct reinterpretation, slice/string headers, virtual-address arithmetic, fabricated/out-of-range rejection |
| maps | nil, set/get/comma-ok/delete/clear, key equality/hash, zero-on-miss, iteration |
| strings | bytes/runes, indexing, range, slicing, conversions |
| defined types | identity, projection/wrap, methods, nil-capable families |

Integer-profile proof pins the append-only scalar alias identities, including
distinct `int`, `uint`, and `uintptr`, and checks their carrier matrix under
32-bit and 64-bit `types.Sizes`. A mutation that restores native-to-fixed alias
collapse fails before printing. All complete runtime packages are generated
through TS-Go AST and strict-typechecked; artifacts must show source alias
names unchanged and only their primitive carrier changed.

Both BigInt-carrier profiles additionally differentially prove fixed-width
overflow for wide signed and unsigned binary operations, shifts, unary
operations, division's minimum-value edge, compound assignments, and
increments. Narrow and native number carriers remain direct where selected and
are checked against an artifact-shape gate.
Artifact inspection requires normalization at the wide result owner, not at
selected call sites, and requires exactly one demand-generated definition for
each selected signedness rather than repeated intrinsic spellings. An
independent source-available hash fixture must produce distinct, byte-exact Go
hash values for multiple inputs; a mutation removing result normalization must
collapse or change those keys and fail before product runtime certification.
The same fixture under `number` is recorded as profile boundary evidence,
never as parity.

Embed fixtures exact-join parsed directives, selected toolchain patterns,
selected files, variable identities, and immutable payload bytes. They cover
quoted names, ordinary versus overlapping `all:` trees, string, byte-slice,
and filesystem declarations. Emitted string fixtures include NUL and invalid
UTF-8, then print through TS-Go, strict-typecheck, and compare executed bytes
with Go. Mutations remove selected-file evidence, zero an embedded variable,
or leak hidden files through an ordinary pattern; each fails at its owner.

Every test inspects generated source and reports bytes/AST nodes. A mutation
that always emits copy carriers/helpers, uses JavaScript identity for Go map
keys, drops nil checks, or restores a target non-null assertion must fail.

Unsafe-pointer proof runs under all integer profiles and at the selected
GOOS/GOARCH. It exact-compares `unsafe.Sizeof`, `Alignof`, and `Offsetof` facts
used by each codec against the selected Go toolchain; different source
spellings of the same type must select the same artifact. Differential cases
mutate a value through a safe alias and an unsafe alias in both orders, cross
array elements by virtual-address arithmetic, reinterpret nested/padded
structs, and exercise slice/string data headers. Mutations swap field offsets,
detach writes from the source location, allocate a second codec for one type,
accept a fabricated integer, recover a payload through `object`/`unknown`, or
restore the universal runtime-throwing conversion path. Each must fail at its
owning gate. Product evidence reports conversion counts by source/target type,
codec count and bytes, largest codecs, virtual-memory runtime bytes, strict
typecheck time/RSS, and representative execution results.

## Struct, Receiver, And Embedding Proof

Fixtures cover:

- value and pointer receiver entry;
- nil pointer methods that handle nil and that dereference;
- method values/expressions;
- promoted fields and methods at multiple depths;
- shadowing, ambiguous promotion, unexported methods, addressable receivers;
- inheritance-eligible and composition-required spines.

Artifact assertions require value receiver as `this`, pointer receiver as
one explicit first parameter, exact `go/types.Selection` ownership, and no
prototype patch or duplicate top-level twin. Mutations force `extends` across
a field-shadow/copy/nil counterexample, use virtual target selection for an
ordinary concrete call, or copy every receiver unconditionally.

Defined-over-generic fixtures cover concrete and generic derived structs.
Strict artifacts use the basis storage field types, concrete accessors project
one field directly, and generic selections use canonical storage without
restoring a logical basis. Restoring `$basis.field`, omitting a required field
storage conversion, or giving a generic `$make` a logical field parameter must
fail strict typechecking or the artifact-shape gate.

## Interface Proof

The differential matrix covers nil interface, typed nil, value/pointer
payloads, assertions, comma-ok, type switches, conversions, equality,
comparability panic, map keys, and method values.

Scaling fixtures hold one call site constant while implementer count grows.
Call-site bytes/AST nodes must remain constant. Adapter growth is attributed by
exact concrete type and demanded contract. Mutations restore implementer
switches, duplicate adapters, emit complete concrete method sets, use
constructor identity, erase payloads, or bypass selected method ownership.

Under cooperative mode, synchronous and blocking concrete methods both satisfy
one `Awaitable` interface declaration. Indirect calls are awaited without
thenable tests. A mutation creating method-profile variants or widening
source type arity fails.

## Reflection Proof

The focused matrix covers basic, defined, pointer, array, slice, map, struct,
interface, function, and recursive types. It compares Go and generated strict
ESM for:

- `TypeFor`, `TypeOf`, descriptor equality, kind/name/package/string identity;
- element/key/length, size/alignment/bits, comparability, and implementation;
- field count/index/name/tag/embedding/export and recursive field types;
- `ValueOf`, nil/invalid/addressable/settable state, pointer `Elem`, fields,
  indexes, maps, scalar projections, mutation, zero, and interface recovery;
- reflective map/slice/pointer construction and iterator behavior;
- open generic `TypeFor[T]` through exact private capability or
  concretization, with unchanged source value arity.

An independent descriptor extractor walks the selected `go/types` graph and
exact-joins canonical type identity, kind, relations, fields, tags, methods,
and architecture sizes against generated descriptor records. Runtime tests
assert `TypeOf(boxed T) == TypeFor[T]` and that reflective mutation changes the
same Go storage observed by ordinary generated code.

Required mutations omit or duplicate a descriptor, key it by display spelling,
change kind/field order/tag/method token, eagerly inline a recursive descriptor,
disconnect an interface adapter from its descriptor, recover a payload through
`any`/`unknown`, or add a hidden value parameter to `TypeFor`. Each fails at the
descriptor join, staticness wall, source-shape gate, strict typecheck, or
differential test. Scaling holds call sites constant and bounds descriptor
bytes/AST nodes by canonical types plus fields/methods, with no type-pair or
call-site cross-product.

Reachability tests exercise both discovery orders: adapter before reflection
demand and reflection demand before adapter. They exact-join the descriptors
against concrete adapters reaching the observed interface contract and prove
that an adapter reaching only an unrelated contract emits no descriptor. A
mutation restoring a compilation-wide dynamic-type sweep must fail the exact
descriptor set and generated-byte bound.

Descriptor-shape tests reject repeated default-valued properties, explicit
top-level `[ordinal]` field indexes, and self-only relation arrays. They compare
the provider-materialized defaults against Go for named/unnamed, exported/
unexported, tagged/untagged, embedded, nonzero-offset, and non-comparable
cases. Product evidence reports descriptor roots, recursive relation types,
field records, descriptor bytes, and the largest descriptor module separately.

## Generic Proof

Fixtures cover:

- generic functions/types/aliases/methods;
- explicit and inferred instantiation;
- recursive instantiated types and calls;
- basic, defined, aggregate, interface, and callable type arguments;
- operations that are directly expressible over open parameters;
- operations requiring exact concretization;
- iterator functions.
- builtin operations whose accepted type set has disjoint target
  representations, including `append([]byte, B...)` for
  `B ~[]byte | ~string`.

The gate exact-joins every concretization to one
`(declaration identity, ordered exact type arguments)` instance and proves
its source-substituted signature. Direct generic bodies remain one definition.
Concretization count grows with distinct necessary instances, not calls.

The same gate inspects storage, container-storage, and pointer associated-type
facets. It proves source generic arity is unchanged, each demanded concrete
marker has one type owner, identity/default cases emit no marker, and every
private kernel signature uses the canonical projection. Strict compilation is
rerun after deleting a required marker; the mutation must fail at the concrete
representation mismatch. Representation-only demand must not create a
runtime kernel or concretization.

Mutations add an operation parameter, operation object, public specialized
export, target-spelling key, cast/erased payload, duplicate instance, or
concretize a transport-only body. Additional mutations fabricate a hidden
representation type parameter, omit a demanded associated-type marker, or
materialize a marker for an undemanded identity case. Open unsupported exports
and intentionally unbounded recursive instantiation must fail
deterministically.

For representation-disjoint builtins, mutations bypass the internal operation,
collapse its concrete families, add it to the source facade, or select it by
runtime type/spelling. Each must fail at AST shape, strict typecheck,
differential behavior, or the generic-operation ownership gate.

## Function-Control Proof

The integrated fixture covers:

- panic values, runtime panic values, and `panic(nil)`;
- recover outside defer, direct deferred recover, and recover one call below;
- direct, function-valued, method-valued, interface, generic, and provider
  deferred calls;
- immediate argument/receiver copy;
- LIFO order, named-result mutation, panic replacement;
- ordinary/named returns;
- labels, fallthrough, break/continue, structural and non-structural goto;
- composition with loops, range, switches, and concurrency.

Strict target compilation enables `noFallthroughCasesInSwitch`. A differential
fixture covers fallthrough into a returning clause, fallthrough through a
middle default, source `break`, and `continue` to an enclosing loop. Removing
the callable's unreachable-end guard must restore a missing-return diagnostic;
ordinary result functions ending in a direct return must not gain the guard.
A nested-control fixture labels a non-breakable `if` containing both a switch
and a loop while non-structural gotos target the label. Artifact inspection
requires exactly one emitted target label on the direct `if`; propagating its
target-label capability to either nested breakable child must fail the gate.
The inverse nesting matrix also covers a loop inside a fallthrough-lowered
switch and a switch inside a labeled loop. Mutations that retain the outer
break target when entering either inner construct must fail before product
execution.

Signature inspection proves ordinary callable/function/interface/provider
contracts contain no recovery parameter. Private deferred entries are present
only for recover-capable callables. Typed registries exist only for demanded
dynamic signatures and fall back to ordinary invocation when a value has no
registered private entry. Provider-facet tests prove both certified presence
and certified absence; absence must not create a public/bridge-wide recovery
entry or fail an otherwise valid call.

Mutations make recovery ambient, pass authority through an ordinary source
call, forward it one call deeper, omit registry registration, key the registry
by storage location/spelling, reverse defer order, capture arguments late, or
swallow a host exception. Each fails differential behavior or structural
walls.

## Cooperative-Concurrency Proof

The selected race-free cooperative profile exits only with:

- unbuffered/buffered/nil/closed channel cases;
- send/receive/range and element copy;
- direct/select waiter FIFO interaction;
- ready/default/blocking select, fair choice, cancellation, same-channel and
  select-to-select rendezvous;
- goroutine argument capture, main return, panic, deadlock, and settlement;
- direct calls, first-class values, interfaces, callbacks, generic callables,
  package initializers, and deferred calls.

Callable shape proof requires:

- direct synchronous callables remain synchronous;
- direct blocking callables become Promise-bearing;
- all transported values of one exact Go signature use one
  `Awaitable` ABI;
- all generated interface methods use the equivalent canonical ABI;
- indirect calls unconditionally await;
- no per-method/combinatorial profile variants, hidden effect type parameters,
  result runtime tests, or transport/dataflow graph.

Provider contract validation separately proves that every boundary certificate
is uniform: all transported methods are synchronous for direct mode or all are
`Awaitable` for cooperative mode. A Go identity admits at most one certificate
per mode. Mutations adding a second certificate in either mode or mixing the
two effects inside one certificate fail before emission.

The certifier independently derives a total directional obligation multiset
over every provider callable. It recursively records inward interface-method and
callable-value effects by source parameter root, including nested callbacks,
then exact-joins that multiset to ordinary bindings and private facade
certificates. Result roots are verified as conversions on the selected target
but do not enter the inward obligation set. Mutations remove `sort.Sort`'s
interface facade, remove `sort.Search`'s callback facade, change a callback from
`Awaitable` to synchronous, add an unneeded result-only facade, or add a second
certificate for one effect; each must fail certification with the exact Go
callable and root identity.

The same gate distinguishes unnamed direct callbacks from named function-value
representations. A mutation that inserts a nested callable below a direct
callback must either produce a typed nested-path certificate or fail at contract
generation; observing only the outer function result is not sufficient proof.

Mutations split direct/select queues, use historical queue storage, omit
cancellation, bias source order, treat nil as ready, settle before pending
Promises, globalize all functions as async, restore callable-profile variants,
or test a result for Promise shape.

Measurements report queue storage, runtime bytes, call-site AST size,
typecheck time/RSS, and representative runtime.

## Provider Proof

Provider certification independently loads the selected `GOROOT`, resolves
exact Go objects/signatures, and inspects the strict TS provider project.
It exact-joins each public export and private compiler kernel to one owner.
Architecture walls reject `go/ast` or `go/types` imports from immutable
provider-document files; only the focused source-contract and certification
owners may inspect the selected Go graph. A mutation that restores any of the
callable, interface-method, or protocol-resolution helpers to the document
package must fail the wall.

Required source and artifact fixtures include ordinary functions, value/pointer
methods, interfaces in both directions, callbacks, nested containers,
provider-created values, constants, package state, generic operations,
cooperative calls, and deferred recovery.

Nested-container proof mutates a provider slice and its generated projection in
both directions, then reslices, appends with and without reallocation, and uses
`copy`. The observed backing aliases, nil state, length, and capacity must match
Go. Artifact inspection proves equal-representation slices remain direct and a
differing element emits one `RuntimeSliceProjection<F,T>` with reciprocal typed
element conversions. Replacing the projection with an eager copy, omitting the
reverse conversion, or projecting an equal carrier must fail.
Projected contiguous-region proof verifies that nil stays nil and a non-nil
request reaches the projection's explicit typed unsupported boundary rather
than inheriting the direct slice's raw-backing implementation.

For each generated static facade, proof records:

- selected Go callable identity/signature;
- canonical generated parameter/result types;
- provider module/export;
- every static bridge/guard/private kernel import;
- effect and exact generic instance;
- produced facade AST.

Scalar-boundary fixtures cross every signedness, fixed-width, and native-width
class in both directions. Equal product/provider carriers must produce no
conversion AST. Differing carriers must produce the one width-aware static
conversion and no extra source parameter. Differential cases include signed
minimums, unsigned maximums, values above JavaScript's safe-integer range, and
multi-result tuples. Mutations change the provider profile or native width,
remove one conversion, replace unsigned normalization with signed
normalization, or restore a product-scalar import in provider source; each
fails at the runtime contract, strict typecheck, artifact gate, or execution
differential.

Provider-certificate generation invokes the pinned TS-Go compiler and rejects
every diagnostic before sealing provider evidence. The provider is therefore
independently strict-typechecked against its certified scalar module. Exact
64-bit algorithms (binary encoding, parsing, bit operations,
hash folding, and counters) execute against Go. Node/host boundaries have
focused range tests; unchecked `bigint`-to-`number` narrowing is forbidden by
source-shape and mutation gates.

Provider-created dynamic-interface proof additionally exact-joins every
configured capability view to its base Go interface, target Go interface,
provider parameter type, optional provider result type, target method set,
module/export, implementation owner, and fingerprint. Differential fixtures
cover a provider value with and without each capability, direct assertions,
comma-ok assertions, type switches, and facade guards. Error fixtures include
`Is(error) bool`, `Unwrap() error`, and `Unwrap() []error`; filesystem fixtures
include every provider-created optional interface actually implemented by the
selected provider.

Sealed-native interface proof performs an ordinary Go type assertion and a
reflective interface-field read/write through the public pipeline. The
generated artifact must contain one statically typed predicate to the certified
provider type, backed by the canonical method-token guard, and no wrapper,
cast, host-shape check, or duplicated contract. Removing that predicate must
fail strict typechecking before execution.

Certification also exact-joins each view's closed usage. Artifact inspection
proves provider-internal views never enter generated bridges and each demanded
generated-bridge view is evaluated once per provider crossing. Mutating the
usage, substituting a direct-profile view for a canonical-profile view, or
removing the sole ABI-compatible candidate must fail before printing generated
source. Direct and cooperative differential fixtures inspect the emitted view
calls and reject the opposite ABI even when its Go method contract is identical.

Mutations remove a view, change its base or target type, return the target
unconditionally, drop or add a method token, omit delegated result conversion,
make two incompatible same-name capability views select one value, bypass the
reverse demand, or restore member-spelling/shape inspection. Each fails at its
certificate, artifact-shape, strict-typecheck, or differential owner. Artifact
inspection proves ordinary bridges contain no undemanded capability members and
each demanded call remains constant-size as provider implementer count grows.

The source-shape gate verifies facade arity. Artifact inspection proves calls
contain only source arguments and support is imported at module scope.
Provider-owned named-callable fixtures prove that generated annotations retain
the source type-parameter arity and canonical `Awaitable` result recursively.
They also inspect the provider facade and prove that a differing provider
callback ABI is adapted there, not exposed as a public type parameter, default,
constraint, profile alias, or generated call argument. Removing the bridge or
substituting the private provider ABI must fail strict typechecking at the
callback parameter or result boundary.

The facade-support gate exact-joins the ordered private suffix after source,
receiver, and canonical value parameters: guards, contracts, then
provider-to-generated bridges. Mutations swap two support classes, substitute
one structurally similar marker, omit one dependency, or add a policy-object
parameter; each fails contract generation at the exact parameter identity.

For every generic provider kernel, certification also exact-joins the kernel's
outer effect and callback-parameter effects to the public binding document.
Mutations change either side between synchronous and `Awaitable`, restore a
separate cooperative kernel, or expose a capability as a public source
parameter; each fails before emission.

Generic-boundary differentials include concrete and open instances. A
`cmp.Or[int]` fixture proves that generic-owned `[]T` remains
`RuntimeSlice<int>` and selects `CmpOrKernel<int, int>`; a mutation that applies
the provider bigint slice projection must fail strict typechecking. A callback
fixture proves callable shells are still recursively adapted when their
parameters mention `T` and their concrete leaves require provider conversion.

Canonical-boundary differentials include a container of profile-owned
interfaces and an already-canonical interface leaf. `fs.ReadDir` must project
each `DirEntry` in its result slice, including nested `FileInfo.Size` scalar
behavior. `errors.Is` must retain generated custom `Is`, single `Unwrap`, and
multi-`Unwrap` behavior. Mutations either skip the nested element projection or
route canonical `error` through the ordinary provider bridge; each fails its
Go-versus-TypeScript differential.

An allocation-order mutation inserts an unrelated computed `unique symbol`
member and must leave every unaffected provider fingerprint byte-identical
even when TS-Go's diagnostic/display spelling changes.

Mutations add a runtime policy object, omit/misroute a bridge, select by Go or
target spelling, accept structural assignability without certification,
duplicate a provider owner, append generic/recovery operations to the public
call, expose a private kernel, discover a missing profile only during source
emission, or silently fall back to ambient mode.

`gostdlib` runs focused Go-versus-provider differentials and strict ESM tests
independently. Generated product linkage proves the same physical runtime
module is used by provider and generated code.

Provider-source strict typechecking uses the generated direct-profile runtime
harness. Runtime generation tests separately inspect direct and cooperative
contracts, and the linked cooperative product must strict-typecheck provider
declarations against its own generated cooperative runtime. Mutating either
profile selection, or substituting the direct harness into the cooperative
product, must fail at the exact interface method effect.

Provider packaging begins from an empty emitted-output tree. A package
artifact contains exactly current source-owned outputs; deleting a source
module and rebuilding removes its emitted JavaScript and declaration files.

Constant verification includes an executing two-file same-package cycle where
a defined-basic constant depends on its class while a method in the class file
uses that constant. The emitted source artifact must contain a deferred typed
value thunk, the package assembly must expose an ordinary `const`, strict ESM
must typecheck, and Node execution must match Go. A direct eager constructor in
the provider file is the required mutation foil. Large-value scaling separately
proves payload bytes are materialized once rather than once per use.

## Environment And Obligation Proof

Compile-only fixtures inspect deterministic package, environment, runtime, and
obligation files. Every selected bodyless/external declaration has one exact
typed throwing body and canonical identity. Linked mode removes each satisfied
placeholder and admits no dual path. Publication fails on any reachable
obligation.

External-function linkage tests begin from the unlinked obligation artifact,
then compile again with a target derived from that exact obligation. They inspect
the TS-Go AST and printed TypeScript for the static module or portable-source
call, strict-typecheck the linked module, execute it against the Go result, and
assert that the obligation and throwing body are absent. Independent mutations
duplicate a binding, reuse it under another build profile or module version,
select a mismatched source signature, omit the target export, and provide an
unreachable extra binding. Each fails at the binding join or strict module
boundary; no runtime lookup is exercised.

The production proof uses the verified provider certificate rather than a test
binding slice. Certification independently reloads the pinned Go modules under
the manifest build profile, re-derives source and portable-target contracts,
inspects module exports through TS-Go, fingerprints their public symbol shapes,
hashes the complete provider source tree, and reproduces canonical manifest
bytes. Mutations change the source module version, signature, target profile,
standard-library dependency digest, module export, target fingerprint,
implementation owner, and provider source bytes. Each is rejected before the
compiler can resolve an obligation.

Mutations classify by import spelling, omit an obligation, duplicate one,
execute a `declare`-only ESM binding, read an optional provider artifact
without proving it exists, or keep ambient fallback after a failed provider
lookup.

## Settled Environment And Provider-Closure Proof

Settled environment evidence is proved structurally and by exact join:

- every provider reference/facet route and every intrinsic or generated-facet
  handler invokes the non-optional root environment observation with its
  canonical object, sole route, and closed demand before returning a target;
- the settled environment builders' canonical object/demand set exact-joins
  the final immutable projection, and the provider-routed subset exact-joins
  the provider closure roots;
- the product header binds the selected Go toolchain, schema, provider, and
  build digests;
- type-only references do not become executable use; function and method
  values do.

Required environment-evidence mutations: remove root observation from one
provider target route; classify a callable use as type-only; omit the behavior
demand from a method-value route; change canonical identity while keeping
display spelling; fail to upgrade an earlier type-only demand at a later
callable use; drop one generated facade/capability use; alter a provider or
profile digest; record all catalog bindings as used; select both intrinsic and
provider implementations for one declaration; return a provider target while
the environment observer is absent. Each must fail at its owning gate.

Required provider-certificate mutations: mark a canonical placeholder as
implemented; move a placeholder call behind a private helper; omit a private
dependency edge; redirect an edge to a same-name symbol in another module; add
an unresolved dynamic call; reuse a certificate under another build profile;
duplicate an implementation owner; retain an old manifest reader beside the
replacement schema.

Required closure mutations: omit a used root; include an unused provider
export; stop before transitive dependencies; accept a used placeholder; accept
a used profile boundary from runtime argument assumptions; let a dependency
cycle loop or silently truncate; replace the canonical identity join with a
package/name join. Closure construction is measured linear in nodes plus
edges, with both one-sided residual lists reported on every mismatch.

## Artifact And Cost Review

### Project And Source-Implementation Proof

Configuration tests prove strict versioned decoding, exact `-c`/`--config`
selection, config-relative paths, defaults < file < CLI precedence, and the
absence of ancestor/home/environment lookup. A registry-totality test proves
every semantic CLI switch and JSON field resolve through the same owner.
Relocating unchanged source and implementation trees preserves the semantic
digest; changing implementation bytes or its envelope changes it.

The production `gotots build` command writes a canonical resolved project on
request, emits every TypeScript file through pinned TS-Go, records the semantic
digest and exact output membership in `gotots-manifest.json`, declares the
project ESM in a root `package.json`, and runs the strict generated-project
typecheck before reporting success. Removing `type: module` must make a
cooperative program with top-level await fail at that strict gate.

For each source implementation, certification independently inspects the
selected Go package, the ordinary generated package assembly, and the authored
strict TypeScript project. It exact-joins exported identities, binds the build
and compilation profiles, parses every authored module through pinned TS-Go,
rebinds public generated imports to the package assembly, proves each residual
private contract module body-free and exact by imported name, and inspects the
final target file set. Before checker evidence is trusted, the complete ordinary
generated target set and the complete installed target set are independently
strict-typechecked with the same final module-resolution configuration. The
installed check proves all selected consumer requirements; callable source
signatures and projections are joined separately by the callable ABI owner.
Display-oriented checker strings, parameter spellings, and replacement-private
storage layouts are forbidden as package-contract equality evidence. Required
mutations alter a Go
signature, TypeScript export, package identity, build or compilation profile,
source digest, private module identity, or envelope; add an executable
private-module statement;
duplicate a generated and manual owner; retain one generated package file; or
feed text directly to the output writer. Each fails at its owning gate.

Callable source-implementation proof additionally exact-joins the selected Go
signature, ordered projection kinds, projected target parameter/result types,
and checked authored signature. The certificate fingerprint covers that join.
Focused artifacts prove both automatic and authored `*int -> number`
`pointee-value` selection: `Read(&x)` emits `Read(x)`, an existing pointer emits
one guarded current-value read, and no call-induced scalar cell is requested.
The complementary mutating fixture retains a location. Required mutations
change the Go parameter, authored parameter, nil shape, projection kind,
argument rewrite, method/callback adapter, or signature dependency; each fails
at its owning join or strict consumer. A body-only authored change proves that
the callable-signature facet and its callers remain unchanged.

The final broad search rejects any source-implementation name-only surface
join, package/function projection condition, projection config field, caller
allowlist, text patch, unchecked cast, duplicate signature store, or migrated
scalar-pointer carrier on a certified pointee-value path.

Pointer-runtime mutation tests require address construction to remain behind a
lazy accessor, ordinary value access to contain no unsafe branch, and composed
locations to call typed active access functions rather than recursively enter
public accessors. Differential fixtures compare repeated field addresses,
pointer equality and hashing, whole-parent replacement, writes through aliases,
safe/unsafe alias synchronization, and rebinding of an exposed carrier.
Deterministic structure checks prove that pointer creation cannot perform
address-map operations until an identity boundary is reached.

Field-path fixtures include two- and three-level aggregates, promoted fields,
whole-root replacement after taking an interior address, and pointer-valued
intermediate fields. Artifact checks require one location per maximal carrier
segment, require one root access plus one direct projection, and reject
restored nested field-location construction.

Runtime-feature tests exact-join the final feature requests to assembled class
members. A no-flat-path control retains the prior pointer member count, while a
flat-path fixture adds exactly the field-path member without adding an import
or standalone runtime definition. Dropping the request, emitting the member
unconditionally, or retaining a request from a replaced artifact fails its
owning gate and the generated-size/node bounds.

An equivalence envelope requires product-level evidence. For an internal hash,
tests prove equal-input stability, unequal-input behavior over an adversarial
corpus, streaming/one-shot agreement when exposed, collision-safe consumer
behavior, and unchanged externally observable compiler output. A mutation that
exports or persists the relaxed hash invalidates the envelope. Exact hash-vector
comparison is required only when the hash value itself is part of the selected
observable contract.

Artifact evidence reports removed generated files, source bytes, TS-Go AST
nodes, strict-typecheck time/RSS, runtime time/RSS, and the top profile frames.
The selected package must contribute one final implementation module, zero
translated bodies, and zero unsafe-pointer traffic from the removed package.

Every material checkpoint reports absolute values and parent deltas for:

- Go source and generated TypeScript bytes/tokens/TS-Go AST nodes;
- twenty largest and most-expanded bodies/types/calls;
- runtime, provider facade, adapter, concretization, and obligation bytes
  separately;
- duplicate definitions and identity collisions;
- forbidden dynamic/staticness counts;
- generation time/RSS;
- strict typecheck time/RSS;
- runtime results;
- focused/full/race test counts.

Aggregate improvement cannot hide a worsening tail. A material increase
without typed necessity reopens the owner; thresholds are not raised to absorb
it.

## Completion Language

Evidence binds to an exact clean pushed compiler revision, schema/toolchain
fingerprint, provider revision, selected build/compilation profiles, and
generated artifact digest. Reports state exactly what was generated,
typechecked, linked, and executed. A percentage names numerator and
denominator.

Before phase exit, broad searches prove no superseded helper, compatibility
route, fallback, hidden source ABI, stale test, or contradictory spec text
remains.
