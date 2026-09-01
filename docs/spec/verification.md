# Verification

## Proof Principle

Every construct family is test-first and closes through independent evidence:

1. a smallest valid Go fixture;
2. a focused test observed failing at the owning unsupported boundary;
3. exact TS-Go AST shape assertions;
4. strict canonical-source checking and total finalized marker facts;
5. exact-node target-AST transformation assertions;
6. strict executable-target typechecking;
7. Go-versus-target differential execution where executable;
8. an independent structural/semantic comparison;
9. a mutation that the owning gate catches;
10. canonical and executable artifact/cost inspection;
11. broad deletion searches for the superseded path.

Implementation tests alone do not prove architecture. Generated artifacts are
mandatory evidence.

## Information-Loss Closure Gate

Production conservation is owner-local. Existing closed AST/type/runtime
dispatchers provide the denominator for their own semantic class; tests mutate
each dispatcher and fact constructor to prove omitted, duplicated, and unknown
forms fail closed. Runtime and generated-artifact assemblers exact-join their
requested declarations, members, and fact applications. The selected target
then exact-joins planned source-attribute applications and metadata-only import
bindings to the nodes it erases or lowers.

The TypeScript-target control may consume a fact by exact-erasing its
compile-time attribute while retaining the canonical executable carrier. Its
proof exact-joins applications, metadata-only imports, and removable fact
declarations, and preserves local same-spelled calls and every live binding.
Native targets may not use this control as proof for a changed representation.

Independent verification uses bounded fixtures that derive each semantic
family through a structurally different path and compare exact emitted AST and
finalized facts. It must not build a second production semantic inventory,
re-run the checker, or claim universal whole-program coverage from a duplicate
Go graph walk.

The matrix includes:

- every basic kind under 32-bit and 64-bit sizes, named and alias forms;
- embedded/tagged/blank/generic structs and fixed arrays of distinct lengths;
- nil and non-nil slices, maps, pointers, interfaces, functions, and channels;
- direct and interface methods, variadics, multiple results, typed nil, and
  implicit interface implementation;
- addressable versus non-addressable values, aliases, copies, mutation,
  escaping captures, and loop captures;
- evaluation-order-sensitive calls and assignments;
- panic/defer/recover and goroutine/channel/select operations;
- reflection identity, unsafe/layout boundaries, and selected provider/body
  implementations.

TSTS must finalize every emitted marker on its exact TS-Go AST subject. For
GoToTS-owned attributes, tests select the attribute type by exact canonical
declaration identity and inspect its typed arguments; a local same-spelled
attribute class and helper are foils. Every selected target then exact-joins all
reached facts to either a lowering disposition or one typed unsupported
diagnostic. Silent ordinary-carrier fallback fails.

Required mutations omit one owner disposition; double-own one field in a shared
and Go companion fact; change primitive width/signedness; drop defined-type or
field-tag identity; collapse two array lengths; erase channel direction; turn a
typed nil into an untyped nil; change a copy into an alias; move an
evaluation-order prerequisite; replace a fact-bearing goroutine operation with
an unmarked direct call before fact finalization;
select a target implementation in canonical source; classify a runtime helper
by spelling; and reuse a lossy target artifact as another target's input. Each
must fail at its single owner.

Broad searches reject semantic source comments, helper-name recognition,
target-side Go parsing/checking, a serialized source-operation manifest,
duplicate fact stores, direct-evaluation canonical mode, and an unmarked
canonical serial-concurrency route. Artifact review shows source -> disposition -> exact
TS-Go node -> finalized fact -> target decision for every fact family.

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
9. representative TS-Go encode/print and strict canonical-source checks;
10. TSTS fact totality and exact AST-node ownership;
11. selected-target AST transform and strict output checks;
12. applicable differential/runtime and artifact-size gates.

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

## Exact Tool Selection Proof

Focused fixtures select an arbitrarily named Go executable and both certified
TS-Go forms through configuration and CLI overrides. They prove one immutable
selection reaches package loading, gostdlib and external certification,
source-implementation certification, strict TypeScript compilation, and AST
printing. The selected Go's reported version/root/default target—not
`runtime.Version`, `runtime.GOROOT`, or a later `PATH` lookup—must determine the
build profile and standard-library contract.

Adversarial mutations change the selected Go bytes during identity discovery,
the sealed executable after selection, a root-snapshot candidate before
publication, a reopened snapshot's `compile`, standard-library source, or a
non-source asset such as `lib/time/zoneinfo.zip`. Each fails at the tool owner.
Changing the mutable source root after resolution cannot alter a command,
because only the sealed snapshot remains reachable. Relocated byte-identical
roots, including absolute in-root symlinks rewritten to the relocated root,
retain one semantic identity; an escaping symlink or changed root does not. A
hostile ambient `PATH` and temporary root must not enter a selected subprocess.
Cgo without an exact external-tool contract fails before loading.

A non-vacuous full-root-walk census records the count immediately after
selection, then performs repeated Go commands, an overlay-capable
`go/packages` driver load, TS-Go resolution, and strict TS-Go compilation. The
count must remain unchanged. The one explicit pre-publication compilation
boundary must add exactly one complete snapshot verification. Restoring a
per-command complete-root hash, skipping exact verification when reopening a
digest root, or publishing a mutated candidate fails this gate.

The exact package-driver proof uses an overlay that introduces a dependency
absent on disk and enables test variants. It exact-joins roots, packages,
files, imports, `Dir`, `Module`, and `ForTest` from the one overlay-selected
graph. Dropped/duplicated evidence, malformed trailing JSON, a second JSON
value, or any non-overlay metadata route fails.

TS-Go mutations cover a foreign module, wrong module checksum, replaced
module, development build without exact VCS evidence, wrong revision,
`vcs.modified=true`, build-Go mismatch, selected-byte drift, and selected-Go
identity drift. Semantic evidence is searched for selected/cache/GOROOT paths;
resolved configuration is separately allowed to report those operational
paths. Broad walls reject hardcoded production `go`, `runtime.GOROOT`, ambient
tool lookup, provider-local resolver fields, and production calls to a default
TS-Go resolver.

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

## Marker And Target Boundary Proof

Every canonical marker occurrence owns one finalized TSTS fact selected by
provider declaration identity and keyed to the exact node in the checked
TS-Go-contract AST. The selected target transforms that same AST; it never
builds or joins a second tree. Tests cover nested calls, multiple calls on one
line, comments, multiline syntax, CRLF, BMP and non-BMP text, overlays, and
synthetic nodes. Mutations substitute a different node, duplicate or omit one
selected fact, classify a local same-spelled call, reparse source, or introduce
a range join; each must fail at its sole owner.

GoToTS strict-typechecks canonical marker-bearing TypeScript against the
resolution-only declaration contract but never executes that module's
JavaScript. A direct Go-versus-Node differential remains valid only for an
artifact proven to contain no canonical marker call. For marker-bearing
artifacts, GoToTS records native-Go evidence and canonical AST/typecheck
evidence; TSTS fact finalization plus the selected target's lowering and
runtime differential is the sole executable proof. A test that executes a
resolution-only marker body, treats a no-op marker implementation as behavior,
or reports canonical-only evidence as a runtime differential must fail review.

For typed pointers, focused proof covers `addressOf`, `allocatePointer`,
`loadPointer`, `storePointer`, and `equalPointer`; nil, alias identity, mutation
through two aliases, argument/return transport, fresh allocation, and pointer equality;
canonical marker AST shape; TypeScript-target location lowering; strict output;
and Go-versus-output execution. A mutation that scalarizes one alias without
rewriting the complete flow must change behavior and fail. Broad searches must
find no target-side marker spelling recognition, filesystem source reread,
checker re-entry, or second pointer fact store.

The canonical pointer boundary is exact-joined before target optimization.
Tests enumerate every row owned by the architecture table (`*T`, address,
allocation, load, store, equality, hash, provider binding, and projection),
inspect their TS-Go AST nodes, and join each
source-facing pointer parameter/result and direct caller to the settled
callable-signature facet. Mutations change one marker identity, operation kind,
pointee type, source parameter/result, nil union, location identity, provider
read/write binding, projection, or caller
subscription; each must fail at marker finalization, signature conservation, or
the complete-flow target gate. Canonical GoToTS tests never accept target
scalarization, a target cell, or an authored scalar signature as evidence for
this boundary.

An adversarial target-intrinsic fixture declares a Go `String` object while a
conversion in the same module requires the target `String.fromCharCode`.
Strict artifacts retain the Go name and qualify the host value through
`globalThis`. A second fixture declares a Go generic type named `Promise` and
proves that the closed target-intrinsic owner applies its deterministic target
rename without making the source callable Promise-bearing. A closed contract
test enumerates every supported host value and reserved host type identity.
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

Every generated concrete interface adapter and provider-interface bridge is
also exact-joined to the declaration-referenceable Go-facing TypeScript
surfaces selected from its canonical demand graph. For source-owned interfaces,
the join compares named/instantiated Go type identity, emitted heritage target,
type arguments, and multiplicity; a structural method-set key is not a
substitute. An anonymous or predeclared interface fixture proves that its
canonical generated contract is present exactly once in file-level heritage. A
function-local named interface fixture proves that its runtime contract remains
represented while no illegal file-level heritage reference is emitted. For
provider-routed interfaces, the join compares the exact canonical generated
contract used by translated callers and every reached capability contract
carried by that bridge, never the provider's implementation-facing ABI.
Fixtures include two distinct source-owned named interfaces with the same
methods, a generic instantiation, generated and provider-profile capability bridges,
interface-to-interface demand, and reversed discovery order. Mutations drop a
heritage entry, add an unrelated entry, collapse distinct source-owned
surfaces, change one type argument, or omit one reached provider capability;
each must fail before output is sealed.

The adapter-shape gate separately partitions ordinary adapters by their exact
demanded Go method set. A zero-method ordinary adapter must be one typed call to the canonical
interface-value adapter factory plus its concrete equality/hash/format
operations; it must contain no repeated class body or per-adapter method-token
set. A method-bearing adapter must remain a concrete class and contain exactly
its demanded methods. A fixture with many zero-method concrete types bounds
generated bytes and TS-Go AST nodes linearly by concrete operation bodies, not
by duplicated common machinery. Mutations force a zero-method type through
the class path, drop one concrete operation, or route a method-bearing type
through the zero-method factory; each fails the shape, strict-typecheck,
differential, or method-demand join before output is sealed.

Reflection-created adapters use the same demanded-method shape, but their
requirements are released by the quiescent reflection-interface join. A
fixture presents many matching assertion contracts before and after one
exposure and proves one reconstruction per quiescent wave, one requirement per
distinct matched contract, no unasserted concrete methods, and output growth
linear in selected methods. A foil reflected `*sync.Mutex` must not demand
`TryLock` when no reached assertion contains it.

For each function, literal, concrete method, interface method, function type,
method value/expression, provider/environment callable, and generic
concretization, record:

- exact Go identity;
- receiver mapping;
- ordered value parameters and variadic bit;
- ordered results;
- ordered source type parameters;
- direct synchronous result mapping;
- target declaration identity.

The gate asserts:

- value parameter count/order equals Go after the documented receiver mapping;
- source type-parameter count/order equals Go;
- variadic representation is one semantic slot;
- every source call supplies exactly the source argument cardinality;
- package exports expose each demanded cross-package representation binding
  exactly once and expose no private or undemanded support declaration;
- indirect callable/interface ABI preserves the direct result contract;
- private deferred/concretized/facade helpers claim no source identity.

Static searches reject source-facing occurrences of:

- `$go$recovery` or equivalent recovery parameters;
- `$go$binary_*`, `$go$copy`, `$go$zero`, constraint-method, or operation
  parameters;
- provider policy/bridge/capability arguments;
- public `$Value`/storage/effect type parameters;
- digest-named effect/profile exports.

Storage-backed struct fixtures additionally exact-check that nested named-struct
zero values use the demanded `$zeroStorage` owner. A mutation restoring
`$storageOf($zero())` must fail the generated-shape gate, while strict
typechecking and Go/TypeScript differential execution prove identical field
zeros for nested and generic structs. The same fixture initializes aggregate
slice storage and rejects an outer `$storageOf($zero())` composition, proving
that container initialization uses the same direct owner.

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
| arrays | length, zero, value copy, index/address, nested elements, bounded static payloads |
| slices | nil, make, append, capacity, overlap copy, slicing, bounds, element storage |
| pointers | canonical marker facts, nil, alias, read/store, equality, local/field/index addresses, target flow lowering |
| unsafe pointers | opaque bind/nil/copy/interface/equality/hash/map identity; reinterpretation, arithmetic, pointer/integer, and provider-input boundaries |
| maps | nil, set/get/comma-ok/delete/clear, key equality/hash, zero-on-miss, iteration |
| strings | bytes/runes, indexing, range, slicing, conversions, host text/raw-byte boundaries |
| defined types | identity, native fixed-width numerics with value/pointer methods, projection/wrap, method calls/expressions/values, interfaces, nil-capable families |

Large-static-array proof includes a keyed sparse literal above the 4096-entry
readable-AST ceiling with negative and unsigned values, zero-filled holes, and
both integer profiles. Artifact inspection requires one string payload and no
expanded index/value array nodes; a small literal remains readable and a large
nonconstant literal retains the ordinary path. Go/TypeScript differential
execution proves decoded values. Entry-count corruption, malformed digits,
and an out-of-bounds decoded index must each fail at the packed-array runtime
gate. A product-scale generation delta reports packed call count, eliminated
numeric target nodes, source bytes, generation time/RSS, and downstream
parse/typecheck time/RSS.

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

Host-boundary string proof writes valid multibyte UTF-8, NUL, and invalid UTF-8
through the selected `os.File.WriteString` provider and exact-compares the raw
file bytes and reported byte count with Go. A direct codec round trip proves
the one-code-unit-per-byte representation. Passing the canonical Go string to
Node's string-writing overload must fail this gate by double-encoding the
multibyte bytes; conversion by decoded host text must fail the invalid-UTF-8
case.

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

An open-generic address fixture takes `&box.Value` where the canonical field
has storage type `GoStorage<T>`. The generated artifact must address that exact
slot and project it bidirectionally to `Pointer<T>` before any assignment,
call, or interface adaptation; generated-AST shape and strict typechecking
prove that the logical and storage contracts cannot be conflated. The pointer
runtime differential separately proves that a bidirectional projection keeps
the same mutable location. A mutation returning the raw
`Pointer<GoStorage<T>>`, projecting only one direction, or treating an open type
parameter as identity storage must fail at these owners.

Map proof additionally requires the common `GoMapValue<K,V>` surface to be one
nominal abstract class with the canonical private Promise-assimilation
exclusion. The direct map class and every native/hashed specialization must
extend that exact root, call `super()` once, and remain assignable through all
ordinary and generic map signatures. Same-module dependency ordering must emit
the root before `GoMap`; generated JavaScript must contain only the one empty
root class and no `then` field. Mutations restore a structural interface, make
the exclusion public, omit one subclass edge or `super()` call, or sort the
consumer before its root; each fails at AST shape, strict typecheck, target
effect settlement, or runtime initialization. The target control retains a
public structural `readonly then?: never` result, proving that nominal evidence
rather than member spelling owns the optimization.

The native defined-numeric fixture includes value and pointer methods, direct
calls, method expressions, method values, interface adaptation, explicit
conversion, and a switch whose listed constants do not exhaust the underlying
numeric domain. Its generated declaration must be one plain named alias over
the exact selected scalar carrier; arithmetic and every scalar conversion stay
direct. Replacing the alias with a finite enum must make the legal nonmember
value fail under the pinned checker. Replacing the exact scalar alias with an
intersection must fail the generated-AST shape gate: TypeScript structural
assignability is not a second owner for Go's already-checked nominal rule.
Full-product target-planning measurements remain corroborating cost evidence,
not a causal mutation proof. The open switch proves the alias preserves
subsequent control flow. Adding a runtime wrapper or coercion, restoring the
former empty-method-set restriction, or restoring a class-member method route
fails the same family gate.

Unsafe-pointer proof separates opaque identity from raw memory. Differential
fixtures convert the same and different typed locations to `unsafe.Pointer`,
cover nil, copies, interface boxing, equality, hashing, and map keys, and exact-
join the emitted canonical raw-pointer marker facts. Provider fixtures prove
that repeated certified provider identities bind to the same raw identity and
that nil remains nil. Target fixtures lower nested safe/raw marker calls in one
AST pass and prove that a local same-spelled function is untouched.

Separate negative fixtures cover offset arithmetic, reinterpretation,
raw-pointer-to-typed-pointer conversion, pointer/integer conversion, and raw
pointer input to a provider. They require a diagnostic carrying the exact Go
occurrence and selected source/target types. Mutations that restore the former
virtual-address runtime, use JavaScript equality or object hashing directly,
expose the provider identity, emit a cast, select by spelling, or fabricate a
target-neutral fact must fail. Broad searches prove that no legacy raw-memory
carrier or alternate raw-pointer route remains.

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
ordinary concrete call, or copy every receiver unconditionally. A provider-
represented addressable value is also forced into canonical storage by an
independent address use; its implicit pointer-receiver calls must consume that
same mutable storage. A mutation restoring the ordinary value-read/copy path
must fail the artifact assertion and the Go/TypeScript differential.

Defined-over-generic fixtures cover concrete and generic derived structs.
Strict artifacts use the basis storage field types, concrete accessors project
one field directly, and generic selections use canonical storage without
restoring a logical basis. Restoring `$basis.field`, omitting a required field
storage conversion, or giving a generic named-object constructor a logical
field where canonical storage is required must fail strict typechecking or the
artifact-shape gate.

The derived-layout matrix joins two opposite cases. A generated dependency
with an unexported underlying field must preserve that field in canonical
storage and expose it to reflection. A local type derived from an environment
contract must include only the contract's certified field surface and must not
print provider-private fields. Mutating either case to share the other's
ownership decision fails its artifact assertion and differential/typecheck
gate.

Source-owned named, anonymous, and derived fixtures exact-join each public
constructor parameter to its canonical declaration field, prove direct calls
allocate no argument object, and prove preserve-Go calls capture reordered
effects exactly once before declaration-order arguments. Canonical-storage
fixtures must instead call the class's public `$fromStorage` operation with one
named storage object; the empty-struct case must pass `{}` and strict-typecheck.
Generic canonical storage fixtures retain their one required named storage
object. The complete named-struct operation enum is mutation-tested one member
at a time: demanding
one of zero, copy, equality, hash, conversion, storage, or assignment must
materialize only that operation. Broad searches reject source-owned `$fieldN`
parameters and `$make` calls; the only admitted `$make` occurrence is attached
to a certified provider reference and is covered by a provider-boundary
fixture.

The generated-name gate enumerates the closed temporary-kind catalog, proves
that every kind has one distinct prefix, and exact-checks candidates against
all source, import, and already-generated bindings owned by the file. A
preserve-Go reordered-literal fixture places the would-be first capture name
in the enclosing function signature; generated TypeScript must select a
different binding, strict-typecheck, and remain differential-equal to Go.
Mutations that omit any namespace from the join, restore a raw source-scope
counter, duplicate a temporary prefix, or lose the allocated-name set across
artifact reconstruction must fail at this gate.

A late-demand fixture constructs a struct before taking its address in the
same Go function. Artifact inspection proves that the final class owns the
canonical-storage surface and that the earlier composite was reconstructed as
one `$fromStorage` call with exactly one typed storage object. Removing the
applied-demand resolver, the construction-facet dependency, or reconstruction
leaves positional input against the storage representation and must fail
artifact inspection or strict typechecking.

The request-liveness scaling gate attaches one immutable declaration-request
subgraph to many independently owned artifacts and proves storage remains
bounded by unique request nodes plus consumer roots, while final-consumer
removal retires every semantic requirement exactly once. A mixed
import/declaration graph proves projection preserves the original declaration
subgraph identity. Mutations that flatten the graph per consumer, rebuild the
selected subtree, skip the final release, or transfer copied outbound
requirements instead of certified request roots fail these gates.

## Interface Proof

The differential matrix covers nil interface, typed nil, value/pointer
payloads, assertions, comma-ok, type switches, conversions, equality,
comparability panic, map keys, and method values.

Scaling fixtures hold one call site constant while implementer count grows.
Call-site bytes/AST nodes must remain constant. Adapter growth is attributed by
exact concrete type and demanded contract. Mutations restore implementer
switches, duplicate adapters, emit complete concrete method sets for ordinary
conversion adapters, use
constructor identity, erase payloads, or bypass selected method ownership.

Every concrete method and generated interface declaration uses the same direct
synchronous result contract. A mutation that makes either side Promise-bearing,
creates method-effect variants, or widens source type arity fails.

## Reflection Proof

The focused matrix covers basic, defined, pointer, array, slice, map, struct,
interface, function, and recursive types. It compares Go and generated strict
ESM for:

- `TypeFor`, `TypeOf`, descriptor equality, kind/name/package/string identity;
- element/key/length, size/alignment/bits, comparability, and implementation;
- field count/index/name/tag/embedding/export and recursive field types;
- `ValueOf`, nil/invalid/addressable/settable state, pointer `Elem`, fields,
  indexes, maps, scalar projections, mutation, zero, and interface recovery;
- pointer `Elem` over interface, pointer, array, named and unnamed aggregate,
  function, channel, container, and scalar pointees, including nil pointers,
  direct mutation of the original location, interface dynamic-box replacement,
  aggregate copy isolation, repeated pointer depth, and `New` from the same
  canonical zero owner; deleting one pointee class or restoring an empty
  registration must fail before product replay;
- `Addr` over pointer elements, struct fields, and slice elements, including
  exact pointer-interface assertion, repeated-address equality, `Type`/`Elem`
  identity, `ValueOf(Addr().Interface())` type recovery, and mutation of the
  original storage without eager pointer-reflection closure; an address-only
  instantiated generic child whose pointer implements an asserted interface
  must carry that exact method contract and dispatch against the original
  location;
- reflective map/slice/pointer construction and iterator behavior;
- pointer and aggregate slice index/append/make/grow behavior, including
  capacity reuse versus reallocation, fresh aggregate zeros, value-copy
  isolation, pointer nilness, and no writes when grow reuses capacity;
- pointer and aggregate map key/value lookup, assignment, deletion,
  construction, key iteration, and value-copy isolation through each selected
  canonical map representation;
- interface map keys and values, including nil keys, present nil values,
  absent entries, explicit deletion, mutation, and iterator key/value typing;
- dynamic `PointerTo` composition with no source-level `*T`, canonical repeated
  lookup, repeated pointer depth, and value-versus-pointer method-set
  implementation checks;
- open generic `TypeFor[T]` through exact private capability or
  concretization, with unchanged source value arity.

Struct-field artifact proof partitions every field before inspecting output.
A field whose selected storage needs no value projection must emit one exact
property-key fact against the registration's typed storage resolver; a
settable aggregate additionally emits its canonical copy callback. Blank,
interface-valued, and representation-transforming fields must instead emit
their explicit typed callback form. Focused fixtures exercise both sides with
the same represented source types used by ordinary assignment.
Mutating a key, restoring a direct getter/setter closure, dropping a copy, or
forcing a transformed field through the property path must fail generated
shape, strict typecheck, or differential behavior at this gate. Product proof
reports the exact count and bytes of both dispositions and rejects a zero-row
property path as vacuous.

Mutating an address callback to omit reflection-interface exposure must
remove the address-only interface method and fail artifact inspection before
runtime. Mutating it to ordinary empty-interface membership, eager per-contract
reconstruction, complete concrete-method-set emission, or eager pointer-
descriptor demand must fail the bounded-closure gate. Registry proof covers
both discovery orders, exact-joins only implemented reached contracts, and
asserts that exposure itself yields zero contract, reflection-descriptor, or
value-operation requests before quiescence.

Constructor proof covers value- and pointer-method contracts on values created
by `Zero` and `New`, where no concrete source value previously crossed the
empty-interface boundary. Removing reflection exposure from a
value-operation registration must leave the constructed adapter without its
asserted method and fail artifact inspection before runtime.

Provider-facet tests independently certify the metadata operation target and
its concrete result target. The checked return type of `$create` must
exact-join the result export's type symbol, and the manifest must carry the
result export's sole implementation owner, target fingerprint, and behavior
sites. Generated-source inspection requires every descriptor declaration and
descriptor-returning callback to use that certified concrete target before an
ordinary Go-facing `reflect.Type` assignment widens it.

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

Facet-result mutations remove or rename the result export, substitute a
different checked provider type, alter its implementation owner or
fingerprint, widen `$create` back to the public interface, or widen one
generated descriptor callback. Each must fail at seed validation, provider
certification, manifest validation, generated-AST inspection, or strict
typechecking; no emitter-local spelling fallback is permitted.

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

Value-operation shape tests additionally reject a per-type adapter guard,
field-index switch, reflected-location wrapper, clone guard, or pointer nil
decision when the portable provider owns that invariant mechanic. A scaling
fixture grows reflected structs and pointers from 8 to 32 concrete types,
strict-typechecks both artifacts, and bounds incremental TypeScript bytes and
TS-Go AST nodes by the type-specific descriptors and callbacks while the
common provider implementation remains byte-identical. Mutations reorder one
field descriptor, substitute its adapter, drop a setter/copy callback, route a
provider-owned field through ordinary storage, or restore the per-type switch;
the descriptor-shape, strict-typecheck, native-Go differential, or generated-
size gate must fail before output is sealed. Broad artifact searches prove
that each migrated semantic family has one typed registration route and no
superseded direct `$registerValue` registration.

Container-operation mutations restore the former basic-scalar whitelist,
reuse one aggregate zero object across slots, move aggregate storage without a
Go value copy, make every append reallocate, overwrite a latent tail during
`Value.Grow`, or bypass the canonical map representation. Each must fail at the
source-ownership wall, generated descriptor inspection, strict typecheck, or
Go differential. Broad source searches prove the reduced-slice route and
map/slice-local scalar-zero selection are absent.

Struct, opaque-struct, and pointer registration fixtures prove that adapter
resolvers are not evaluated at registration, are evaluated exactly once at
first operation materialization, and remain statically typed. Generated-source
shape gates require the resolver at every registration. A provider runtime
fixture recreates a legal indirect ESM cycle, proves that an eager adapter read
fails during module initialization, and then proves that the lazy registration
starts and materializes successfully. Mutations passing an adapter eagerly or
resolving the thunk during registration fail the source-shape, provider-build,
or runtime-cycle gate.

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

The same gate separately counts concrete inline operations and standalone
generic-capability files. Ordinary exact zero/copy/arithmetic/comparison/
conversion/storage/index/slice operations must contribute zero standalone
files; only a fixture's explicitly shared constraint-method or deferred
registry owners may remain. Every call, value, method, provider, and nested
generic path crosses the same concrete-operation seam, and an ordinary
operation reaching the generated-artifact builder fails. Mutations restoring
one concrete capability file,
replacing a semantic declaration/path with a digest suffix, omitting receiver,
method, effect, or ordered type arguments from a name, or colliding an import
with an authored binding fail the AST, layout, naming, or strict-typecheck
owner. A fixture also forces two distinct source owners through one candidate
module scope and proves canonical owner identity rejects the collision in both
insertion orders. Hidden identity hashes may be inspected internally but must be absent
from generated generic names and paths.

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

Artifact inspection separately exact-joins fixed and dynamic defer sites. A
top-level direct site that executes at most once owns one fixed typed slot and
no dynamic stack; multiple such slots drain in reverse source order. Loop,
conditional, and non-structural-goto sites retain the dynamic stack. Mutations
that drop or mis-parent an exact handler-produced site demand, restore a
recursive AST scan, classify a repeated/conditional site as fixed, restore a
stack for the single-entry class, or change the drain order fail the callable
control join, architecture wall, shape gate, or Go/TypeScript differential
behavior.

Mutations make recovery ambient, pass authority through an ordinary source
call, forward it one call deeper, omit registry registration, key the registry
by storage location/spelling, reverse defer order, capture arguments late, or
swallow a host exception. Each fails differential behavior or structural
walls.

Possibly nil callable fixtures inspect the TS-Go AST and generated source to
prove one callee capture, source-ordered argument captures, and one invocation
through the nullish nil-call boundary. Moving the boundary ahead of argument
evaluation must fail the Go/TypeScript differential; removing it must fail
strict typechecking and nil-call behavior. A pinned-checker fixture proves the
nullish expression narrows the callable without a helper call. Broad artifact
search rejects the former conditional and assertion paths.

## Serial-Execution Proof

The fixed execution contract proves immediate goroutine evaluation, ready
buffered channel send/receive, ready and default select, close/len/cap, closed
receive, and a loud typed boundary for every operation that would suspend.
Generated artifacts must contain no `Promise`, `async`, `await`, awaitable
union, execution scheduler, host task dispatch, blocked sender/receiver queue,
or callable-effect variant. Mutations that restore any one of those paths,
admit a silent unready operation, or fabricate an unbuffered rendezvous fail
the shape and differential gates.

The direct-callable proof covers every generated source function, method,
literal, callable value, interface method, named callable, package initializer,
deferred entry, provider facade, stateful provider profile, and generic kernel.
Declaration, storage, ordinary and detached invocation, interface transport,
defer capture, callback arguments, and nested callable parameters/results must
all exact-join the same synchronous source signature. Mutations that make only
one surface Promise-bearing, add an effect type parameter, create a second
callable profile, or defer incompatibility to target typechecking fail before
publication.

Provider source inspection and certification prove that every selected export,
method, projection, facade, kernel, callback, and nested callable contract is
synchronous. A Go identity admits at most one callable certificate. Mutations
return `Promise<T>`, return `T | Promise<T>`, mark a declaration `async`, add a
second effect certificate, mix effects within one profile, or substitute a
suspending stateful member; each fails at source inspection or exact contract
selection before generated source is sealed.

Channel and goroutine fixtures cover:

- argument evaluation/copy and immediate `go` invocation;
- buffered FIFO send/receive, range, element copy, close, len, and cap;
- nil, unbuffered, empty, full, closed, and double-close boundaries;
- one-case, multi-case, same-channel, ready, and default `select`, including
  one evaluation per operand and fair choice among ready alternatives;
- absence of blocked-operation storage and cancellation registration; and
- close-observer registration, unregistration, already-closed observation, and
  proof that the observer cannot transfer a value or wait for readiness.

Generic provider kernels including `slices.BinarySearchFunc`, `CompareFunc`,
`ContainsFunc`, `EqualFunc`, `IndexFunc`, `CompactFunc`, `DeleteFunc`, and
`maps.EqualFunc` are compared with the selected Go toolchain for values,
callback order, short-circuiting, nil-call panic behavior, and observable
mutation. `CompactFunc` additionally proves current-before-previous callback
order, in-place backing-store mutation, returned reslice, and zeroed tail. An
absent or suspending kernel fails closed. Changing projection, callback index,
source identity, capability, or direct effect fails contract certification.

The certifier independently derives the total directional obligation multiset
for every provider callable. It recursively records inward synchronous
interface-method and callable-value contracts by source parameter root,
including nested callbacks, and exact-joins them to ordinary bindings and
private facade certificates. Result roots remain outward conversions. Mutations
remove a required facade, add an unneeded result-only facade, reverse a
direction, omit a nested callable, or duplicate a certificate; each fails with
the exact Go callable and root identity.

Callable-field proof combines unnamed, defined, generic, addressed, and
reflection-demanded function-valued fields. Every declaration, storage
projection, ordinary/detached call, deferred capture, and generic-kernel
argument keeps the same direct function type. Reflection reports an unexported
field as non-settable and emits a panic-only setter with no assignment. A cast,
runtime result test, target-owned repair, or field-specific effect resolver is
forbidden.

An architecture wall permits production class construction only through the
one generated-class owner, and a non-vacuous mutation restores a raw factory
call. Every root class is strict-typechecked with exactly one
`declare private readonly then?: never`; every derived class inherits the
contract and does not redeclare it. Emitted JavaScript is inspected to prove
that the declaration creates no field. An adversarial Go struct with a
lowercase callable field named `then` proves that the member-name owner chooses
a different stable property and that all accesses use it. A mutation that
restores one production raw class-factory call, omits the root member, adds it
again to a derived class, or stops reserving the authored field must fail at
the architecture, AST-shape, strict-typecheck, or runtime differential gate.

A target differential removes the nominal member and demonstrates that a host
Promise may assimilate an otherwise ordinary generated Go value with a hidden
callable `then`; restoring the member prevents that assimilation without
runtime result inspection.

Measurements report channel storage, runtime bytes, call-site AST size,
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
provider-created values, constants, package state, generic operations, direct
callback calls, and deferred recovery.

Nested-container proof mutates a provider slice and its generated projection in
both directions, then reslices, appends with and without reallocation, and uses
`copy`. The observed backing aliases, nil state, length, and capacity must match
Go. Artifact inspection proves equal-representation slices remain direct and a
differing element emits one `RuntimeSliceProjection<F,T>` with reciprocal typed
element conversions. Replacing the projection with an eager copy, omitting the
reverse conversion, or projecting an equal carrier must fail.

Generic slice-assignment proof uses aggregate elements with mutable fields and
exercises clone, concatenation, deletion, insertion, repetition, replacement,
and reversal, including overlapping source/replacement slices and both
capacity-preserving and reallocating paths. It exact-compares values, nilness,
length, capacity-envelope behavior, tail zeroing, and backing aliases with Go.
A mutation that removes logical copy, swaps raw storage references, omits tail
zeroing, or always reallocates must fail at the provider differential or the
generated-kernel artifact gate.
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

Provider-profile bridge identity proof feeds the same source interface through
two independently certified provider re-exports with different aggregate
provider fingerprints and requires one bridge artifact. Mutations change one
transported method effect or member and require distinct artifacts. A broad
artifact inspection also bounds the readable bridge export name by its compact
semantic shape; restoring the complete certificate descriptor to every
identifier must fail the source-size gate.

Callable-profile selection is exercised across compilation profiles that make
different proper subsets of one source signature ABI-incompatible. One target
may certify the union, but the generated crossing projection must equal the
currently affected subset. Mutations remove a needed certified root, convert
an unaffected root, remove a transitive interface certificate, or make a
second target match; each must fail at profile selection or artifact inspection
with the source callable and exact root/interface evidence.

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

Provider invocation-transport certification exact-joins every configured
source identity to one provider module/export/member, derives the exact
declaration path from the package export's `types` target, checks one callable
signature, derives its target type and fingerprint, and validates every
parameter index before sealing the provider manifest. Focused checked-source
fixtures cover direct callback ingress, callback-carrying results, one closed
state carrier with multiple writes and reads, and a carrier with an unrecognized
escape. Mutations alter the section schema, declaration path, target type, or
write index; duplicate a member; remove a provider body; use the same spelling
from another declaration; or introduce one unaccounted carrier reference. Each
must fail at certification, exact declaration selection, or closed-carrier
admission.

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
source. Differential fixtures inspect the emitted direct view calls and reject
an asynchronous ABI even when its Go method contract is otherwise identical.

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
the source type-parameter arity and direct synchronous result recursively.
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

For every generic provider kernel, certification exact-joins its synchronous
outer effect and synchronous callback-parameter effects to the public binding
document together with source identity, type projection, operations, source
arity, and callback indexes. Mutations make either side asynchronous, pair
non-identical callback indexes, duplicate the capability, or expose it as a
public source parameter; each fails before emission.

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

Provider-source strict typechecking uses the one generated synchronous
provider-certification harness. Runtime generation tests inspect that exact
contract and the demand-closed linked product runtime. Mutating an interface
method, callback, facade, kernel, channel surface, or stateful method to become
Promise-bearing must fail at the exact effect. A generated product must not
demand `Awaitable<T>` or any execution scheduler/waiter symbol; restoring such
a demand fails the runtime-assembly and generated-artifact shape gates.

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
digest and exact output membership in `gotots-manifest.json`, and declares the
canonical project ESM and exact selected physical dependency set in a root
`package.json`. Removing a selected runtime/provider dependency, adding
`@tsonic/core`, or changing a certified package version fails the package
manifest gate. It emits neither a target `tsconfig` nor a fabricated
`@tsonic/core` package. TSTS must strictly check
that exact immutable source using its authoritative virtual marker modules and
exact-join every selected marker to one finalized fact before a target runs.
The selected target must then strict-typecheck its complete executable output;
removing `type: module` makes an ESM-only generated artifact fail at that target
gate. Rebuilding over a populated output directory must replace the
complete compiler-owned artifact set: a seeded obsolete source file and an old
target `tsconfig` must both disappear, and the sorted manifest membership
(including the manifest itself) must exact-join the physical file set.
Inject a write failure after one staged artifact exists. The prior published
output must remain byte-identical, the partial artifact must remain absent from
that published output, and the typed error must identify a retained staging
directory containing the exact partial artifact. Mutating the lifecycle back
to unconditional deferred cleanup must make this gate fail.

The output-lifecycle gate proves a distinct compilation worker seals and
officially encodes the complete TS-Go AST, writes its validated handoff, and
exits before the parent starts any TS-Go consumer. The post-compilation plan may
retain only output identities, official encoded protocol payload locations and
digests, flattened source-implementation package/export contracts, the runtime
manifest, package dependencies, and semantic digest; a structural reachability
audit rejects source AST, `go/types`, loader, certificate, emitter-session,
mutable TS-Go node, or callback state. When source implementations are selected,
the parent first streams the staged ordinary and installed sets through a fresh
pinned generated-contract verifier, exact-joins their exports, closes that
process, and only then promotes the plan to the printer-only type. The printer
then streams one encoded final file at a time from a fresh process. A focused
round trip exact-compares direct `PrintNode` output with printing the staged
official encoding, including multiple files, payload mutation, and a printer
failure. Successful publication contains no worker, verification, or protocol
scratch. The handoff rejects the parent PID, unknown fields, path escape,
malformed digests, payload mutation, and a missing or duplicate file. The
guarded whole-product gate runs compiler, generated-contract verifier, and
printer under one process-tree ceiling and proves their three phase peaks do
not overlap. Mutations that compile in the parent, verify before worker exit,
bypass verifier-to-printer type-state promotion, retain one target node in the
plan, publish scratch, or accumulate every printed file in memory fail at the
lifecycle, contract, membership, or resource gate.

For each source implementation, GoToTS certification independently inspects the
selected Go package, the ordinary generated package assembly, and the authored
strict TypeScript project. It exact-joins exported identities, binds the build
and compilation profiles, parses every authored module through pinned TS-Go,
rebinds public generated imports to the package assembly, proves each residual
private contract module body-free and exact by imported name, and inspects the
final canonical file set. It does not invoke raw TS-Go to check canonical marker
imports. GoToTS strict-typechecks the authored implementation project and
structurally exact-joins the complete ordinary and installed canonical sets.
That final join consumes only the compiler worker's official encoded protocol
and flattened expected exports after the worker has exited; no live TS-Go
process or certificate crosses the compilation lifetime boundary.
For the shared implementation environment, mutate an admitted `.d.ts`
certification source after preparation, omit it from either implementation
verifier, duplicate it in the shared selection, select it with no
implementation consumer, and replace it with executable TypeScript. Digest
drift, unsealed evidence, an unused source, and a non-declaration source must
fail before output. Callable-local byte-identical declarations are materialized
once in scratch; no certification declaration appears in final output.
Mutate a selected Go callable body without changing its identity or signature.
Its canonical body digest must fail the post-load join before emission; omit or
change that digest across the compile-worker handoff and the parent exact join
must fail before staged TypeScript verification.
Before checker evidence is trusted, TSTS independently checks both sets with
its authoritative marker modules and exact-joins marker facts. The selected
target then lowers and strict-typechecks both complete executable sets with one
final module-resolution configuration. The installed check proves all selected
consumer requirements; TSTS exact-joins callable source signatures in that
canonical check.

The lifecycle fixture prepares authored projects before loading its Go program,
then proves that package identity, build profile, and every private Go-file
claim are still checked by the post-load join. Missing packages, changed module
identity, build drift, and missing private files fail only at that join with
the exact claimed identity. The prepared type exposes no emission certificate
and retains no loader/checker object. The guarded whole-product resource gate
is the mutation proof for phase ordering: moving TS-Go preparation after the
provider certificate or whole-product load must exceed the frozen process-tree
peak bound, while the prerequisite-first transaction stays below it with
identical sealed evidence.

The final-artifact gate additionally proves session isolation. A fixture with
a public package value and an unrelated private type must retain the public
observable contract while producing no selected-package storage/body and no
adapter/helper requested only by the ordinary implementation. A second fixture
with a genuine final private-type consumer must retain exactly its body-free
private contract. A multi-bundle fixture selects at least two source packages,
rebinds both consumer sets, removes both generated package module sets, and
installs both authored assemblies. A mutation that installs the first authored
official AST before rebinding the second must fail the lifecycle gate; the
compiler may never traverse an authored output artifact as transformation
input. Required mutations replace the canonical identity registry between
sessions, share an artifact graph, copy an implementation facet or
non-observable dependency into the final snapshot, drop one captured outgoing
support requirement, emit one selected-package source body, omit a demanded
observable contract, retain one unrequested ordinary generated support
declaration, or install replacement only after final assembly. Each fails at
the lifecycle, contract,
final-membership, private-value, or strict-consumer gate with the exact owner.
The replacement capture fixture feeds a selected-package implementation edge
through the dependency filter and proves it is absent, while generated support
requirements remain and an unselected-package dependency survives the same
filter. Retaining the selected edge must fail because the final graph has no
generated implementation facet for an authored declaration.
The two-session fixture also makes a later selected-package callable demand
canonical storage for an exported struct, then compiles only an earlier external
`return T{}` consumer in the final session. The consumer must call
`T.$fromStorage({...})`. The certified baseline must create zero pending
scheduler work and zero declaration batches. Omitting that baseline must
produce the positional `new T(...)` mutation and fail artifact inspection. A
foreign or duplicate captured requirement must fail atomic installation without
changing scheduler state. Removing storage from the certified set while a
final consumer still demands it must fail the subset join at the exact owner;
an unused certified capability must remain queryable without materializing its
owner.
The two-session fixture also reaches a generic call outside the replaced
package. Recreating its semantically identical concretization in the final
session must reuse the transferred canonical artifact. Mutations retain the
stable key while changing one type argument, signature, placement, or lexical
anchor; each must fail the registry join rather than create a sibling artifact.
The transfer fixture seeds every observation class, preserves representative
canonical name/type facts, and proves that transfer clears all observations.
A repeated transfer or second final-session claim must fail. A mutation that
retains one first-session interface or reflection observation must either fail
the exact final graph or be caught as an undemanded generated artifact.
The fixture includes a cross-package assertion against an exported interface:
the ordinary and installed assemblies must both export the interface type,
runtime contract, and guard; the final consumer must import the bindings only
from the installed assembly. Omitting either runtime binding, retaining its
generated source-module import, or accepting a private value shim fails.
Display-oriented checker strings, parameter spellings, and replacement-private
storage layouts are forbidden as package-contract equality evidence. Required
mutations alter a Go
signature, TypeScript export, package identity, build or compilation profile,
source digest, private module identity, or envelope; add an executable
private-module statement;
duplicate a generated and manual owner; retain one generated package file; or
feed text directly to the output writer. Each fails at its owning gate.

Callable source-implementation proof additionally makes TSTS exact-join the
ordinary generated parameter/result types and checked authored signature under
one canonical provider graph. The certificate fingerprint covers the immutable
authored source and export evidence consumed by that join. Focused artifacts prove that
`Read(*int)` remains `Read(Pointer<int> | undefined)` for generated and authored
implementations. Required mutations change the Go parameter, authored pointer
type, nil shape, method/callback adapter, or signature dependency; each fails at
the TSTS join or strict target consumer. A body-only authored change changes the
implementation digest without changing generated callers.

Callable-body implementation proof uses a source package containing ordinary,
pointer-receiver, generic-kernel, and adversarial same-spelled declarations.
The generated wrapper and authored export are inspected in one strict staged
project and must have the same checked callable shape. The final artifact must
contain one static import and direct call, preserve every generated caller and
declaration signature, contain the authored module exactly once, and contain no
node from the translated source body. Required source-snapshot mutations change
a referenced constant or helper while leaving the selected body unchanged,
change selected dependency source, change an effective per-file Go version,
and change selected non-Go or embedded bytes. Each must invalidate the
load-owned digest and fail the contract join before emission.

Required authored-module mutations add a top-level call, side-effect-only
import, executable top-level declaration, `as` assertion, angle-bracket
assertion, non-null assertion, `@ts-ignore`, `@ts-nocheck`, or
`@ts-expect-error`, explicit `any`/`unknown`, or checker-inferred semantic
`any`/`unknown`, including a function-valued reference whose result is inferred
as `any`. Each fails with its exact closed staged-source violation before
the ABI join. A positive control directly invokes an ambient callable whose
unused formal type is broad while its actual argument and selected result are
exact; mutating the actual argument, selected result, or escaping the callable
value restores the owning failure. Further mutations alter canonical Go
identity or signature, select the wrong ordinary/kernel variant, change one checked
parameter/result/type-parameter shape, add or omit an authored export,
duplicate or leave a contract unconsumed, collide an output path, retain the
translated body, or select both package and callable ownership. Each fails at
its owning join before printing. A differential executes the Go body, ordinary
generated body, and selected authored body over representative and adversarial
inputs; an equivalence envelope bounds every intentional algorithmic
difference.

Configuration mutations select schema 3, schema 2,
`implementations.bundles`, and `--implementation-bundle`; all must fail with
the current migration diagnostic.
A no-callable-implementation control proves the ordinary source digest and
output path remain singular and byte-stable.

The final broad search rejects a command-layer source digest,
`SourceForbiddenDynamicTypes`, a sibling callable-source policy file, schema-2
aliases, treating GoToTS's export-identity join as final
surface acceptance, any second name-only signature gate, package/function
projection condition, pointer-scalarization config field, caller allowlist,
text patch, unchecked cast, or duplicate signature store.

The provider pointer contract is independently certified as exactly one
writable `ProviderPointer<T>.value: T` member plus one
`providerPointer<T>(T): ProviderPointer<T>` factory, and its module identity is
part of the runtime-contract digest. Mutations remove either declaration,
change the value or factory type, add a member, make the value readonly in the
strict provider project, or redirect the module; each must fail at its owning
contract or strict-typecheck gate. Generated fixtures cover nil and non-nil
named-struct and scalar results and must contain `bindPointer` with one captured
provider identity and exact getter/setter closures. Mutations return a raw
provider object, replace stable assignment with rebinding, drop the setter, or
use a detached scalar cell; strict canonical checking or the selected target's
differential must fail. Broad searches prove provider source imports no
canonical marker module and GoToTS provider verification executes no
resolution-only marker JavaScript.

Pointer-target mutation tests require complete-flow lowering: changing one
definition without every reference, scalarizing one of two aliases, dropping a
nil check, or conflating two fresh allocations must fail. Differential fixtures
compare repeated addresses, pointer equality, whole-parent replacement, writes
through aliases, and rebinding of a location. Raw-pointer cases remain typed
unsupported until their shared contract exists.

Field-path and named-struct fixtures cover two- and three-level aggregates,
promoted fields, whole-root replacement after taking an interior address,
pointer-valued intermediate fields, parameter/receiver transport, nil panic,
pointer equality, interface boxing, copying before address-taking,
pointer-to-pointer rebinding, package variables, and array/slice elements.
Canonical artifacts must use one `Pointer<T>` contract and exact marker facts
for every address/load/store; no generated storage facet or JavaScript carrier
appears. TypeScript-target artifacts must use one shared location for every
observable alias group and may scalarize only a complete proven flow.
Mutations that omit an occurrence, attach the wrong addressable owner, detach a
write, cache an intermediate value, compare locations by pointee value, or
rewrite only one consumer fail at fact totality, exact join, differential, or
strict-target gates.

An equivalence envelope requires product-level evidence. For an internal hash,
tests prove equal-input stability, unequal-input behavior over an adversarial
corpus, streaming/one-shot agreement when exposed, collision-safe consumer
behavior, and unchanged externally observable consumer output. A mutation that
exports or persists the relaxed hash invalidates the envelope. Exact hash-vector
comparison is required only when the hash value itself is part of the selected
observable contract.

Artifact evidence reports removed generated files, source bytes, TS-Go AST
nodes, strict-typecheck time/RSS, runtime time/RSS, and the top profile frames.
The selected package must contribute one final implementation module, zero
translated bodies, and zero unresolved raw-pointer boundaries from the removed
package.

Generated-support topology proof exact-joins full internal artifact owners and
definitions to readable modules grouped by real semantic family and
source/type owner. Layout tests prove that every shared closed support family
retains one common module, every contained definition retains an injective
semantic export whose named components use the globally unique package
qualifier, different semantic owners do not merge, and
source-owned generic modules retain their exact semantic source owner. At an
import site the shortest family alias is used only when free; real collisions
use the full semantic export or the shortest deterministic source-derived
qualifier. Malformed semantic modules fail. Digests may occur in manifests and
diagnostics but never in ordinary declaration names or module paths.
Generated-size evidence also fails if full import paths are repeated inside
ordinary generated identifiers; a mutation replacing the package qualifier
with the complete path must exceed the owning source-size/line bound.
Naming-owner tests include same-named packages, same-spelled local types in
disjoint lexical scopes, and truly shadowing local types. Mutations that use a semantic contract's private key
as target spelling, move display suffixes into semantic APIs, or bypass the
registry qualifier/token owner—including the unexported interface-method token
path—fail before generated output is sealed.

Product evidence reports support definitions separately from physical support
modules, the largest semantic module, ESM startup time/RSS, and minimal-compile
time/RSS. Mutations restoring digest-byte or numbered sharding, an opaque
declaration suffix, source-name collision, or cross-family merging fail the
layout, ownership, source-shape, or strict-typecheck gate. Release evidence
compares startup and typecheck cost against the immediately preceding layout;
readability cannot hide an unbounded module, and a cost regression cannot
restore machine-named source.

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

The declaration-order gate places declarations from two source files into one
target module, deliberately reverses their raw `token.Pos` allocation order,
and requires canonical source-path order. Removing the source-path key makes
the reversed raw positions win and fails the gate. Whole-product regeneration
exact-joins every target-manifest member against the committed product so a
relocated method cannot drift between runs.

## Completion Language

Evidence binds to an exact clean pushed compiler revision, schema/toolchain
fingerprint, provider revision, selected build/compilation profiles, and
generated artifact digest. Reports state exactly what was generated,
typechecked, linked, and executed. A percentage names numerator and
denominator.

Before phase exit, broad searches prove no superseded helper, compatibility
route, fallback, hidden source ABI, stale test, or contradictory spec text
remains.
