# Translation Contract

## Total Contextual Dispatch

Every selected Go declaration, statement, expression, and parent-owned syntax
form is either handled by exactly one semantic owner or rejected with a typed
unsupported diagnostic. Root dispatch is category-based and non-recursive.
The parent supplies the child role, order, expected type/arity, lexical scope,
and legal placement.

For example, the same identifier syntax has different contracts:

```go
x = value      // x is a store target
use(x)         // x is a loaded value
T(x)           // T is a conversion owner, not a function value
```

`go/types` identifies the selected object and whether `T(x)` is a
conversion. The assignment/call parent supplies the role. The identifier
handler never guesses from spelling or scans source text.

Production handlers enumerate meaningful direct children. Verification derives
the selected toolchain's complete AST domain and exported node-bearing fields,
then exact-joins them to handler contracts or explicit parent-owned
dispositions.

## Emission Shapes

An expression handler returns:

- one typed TS-Go expression;
- prerequisite statements that must execute before it, if any;
- root requests and typed diagnostics.

A statement handler returns ordered typed TS-Go statements plus requests.
A declaration handler contributes one revisable target artifact.

Parents decide whether prerequisite statements can be placed at the current
boundary. Consider a hypothetical source expression that needs setup:

```text
call(makeValue())
```

If `makeValue()` lowers to setup plus a value, the call owner emits:

```ts
const temporary = prepare();
call(temporary);
```

If the expression appears where statements are illegal, the owner chooses an
exact expression construction or fails; it never inserts into an arbitrary
ancestor. Imports and preferred-static declarations request file scope.

## Declarations And Names

Names are reserved by exact `types.Object` identity in deterministic package
and source order. Target-only collision suffixes are stable and readable.
Source spelling never acts as semantic identity.

Target host intrinsics have one closed identity owner. Value references use
`globalThis.<name>`, so a valid Go declaration such as
`func String(string) Token` keeps its source-shaped name without shadowing the
target `String.fromCharCode` used by an unrelated conversion. Host type names
remain idiomatic when TypeScript defines an unambiguous global type: generated
async signatures use `Promise<T>`, and a conflicting Go type declaration is
deterministically target-renamed. TypeScript's forbidden target class name
`Object` is reserved by the same closed owner. Bare host value intrinsics are
forbidden in source-facing generated modules. A structurally isolated support
or runtime module that admits no Go declaration may use the direct host
identifier; the artifact-placement owner, not a spelling heuristic, selects
that form.

### Constants

A typed constant emits one binding at its selected type. An untyped constant
has no universal runtime representation and is projected from its canonical
`go/constant.Value` at each use's checker-selected contextual type.

A package constant whose selected type needs a package-local runtime
constructor is not eagerly constructed in a source-file ESM module. Its source
artifact is one private, statically typed value thunk; generated Go uses invoke
that thunk, and the package assembly invokes it once to expose the ordinary
public `const`. This preserves the Go-facing API while preventing a legal Go
same-package file cycle from becoming an ESM temporal-dead-zone failure.

```go
// compact.go
type ID uint16
func Use() uint16 { return Value.Value() }

// tables.go
const Value ID = 7
```

```ts
// tables.ts: no eager `new ID` during cyclic module evaluation
export function Value$constant(): ID { return new ID(7); }

// compact.ts
export function Use(): uint16 { return Value$constant().Value(); }

// package.ts: the handwritten-facing surface remains a value
export const Value = Value$constant();
```

Primitive package constants stay direct immutable bindings, and local
constants stay at their lexical execution boundary. The compiler does not
inline repeated large values merely to avoid a module cycle: constant payload
size remains `O(value bytes + uses)`, not `O(value bytes * uses)`.
Inlining all uses is rejected for that scaling reason; a deferred mutable
binding is rejected because it would expose initialization state and mutation;
coalescing every Go file into one target module is rejected because it moves a
local constant issue into unbounded package-module and typecheck size. The
thunk is deleted if the defined-basic representation ceases to require runtime
construction, and its cost owner is reopened if measured call overhead becomes
material on representative constant-heavy code.

```go
const Huge = 1 << 63
func F() uint64 { return Huge }
```

Under the `bigint` integer profile the return occurrence emits the exact
`9223372036854775808n` projection. Under the default number profile, an
unrepresentable exact integer fails at the constant-value owner. Alternate
source spellings of one checker value produce identical output.

`iota`, inherited constant expressions/types, multiple names, blanks, and
cross-package selectors use the same declaration and occurrence evidence.

### Variables And Initialization

Package variables live in package state. Local variables live at the narrowest
legal lexical scope. Every value begins with the exact Go zero unless an
initializer supplies it. Package initializer order comes directly from
`types.Info.InitOrder`; file order and target imports do not reconstruct it.

A selected `//go:embed` declaration is a compiler-supplied package-storage
initializer, not a source initializer and not a zero value. The loader-owned
declaration-to-file join supplies its exact bytes. For example:

```go
//go:embed defaults.json
var defaults string
```

initializes the emitted `defaults` storage from the selected file bytes before
ordinary package initialization. String emission uses the canonical Go-string
value owner so arbitrary bytes survive TypeScript printing and execution.
Unsupported embed storage families fail at the declaration owner; they never
silently retain zero.

Assignments evaluate all right-hand values and required left-hand addresses in
Go order before stores. Parallel assignment never observes an earlier store.
The default `direct` evaluation profile avoids extra temporaries where
ordinary TypeScript is equivalent. `preserve-go` requests the exact
temporaries only at affected boundaries.

### Functions And Literals

One source function/literal artifact owns its declaration and body. Its
source-facing signature is the direct selected `go/types.Signature` mapping.
Function literals remain inline when target syntax and placement permit; they
are hoisted only when a demanded declaration, recursion, deferred entry, or
legal placement requires it.

Go has no overload selection by target spelling. Every reference resolves from
checker identity.

## Calls

The call owner distinguishes, from syntax plus `go/types`:

- ordinary function calls;
- concrete method calls;
- interface calls;
- function values;
- method values and expressions;
- conversions;
- built-ins;
- generic instantiation;
- variadic expansion.

Arguments are evaluated once in source order and converted/copied at the
selected parameter boundary.

### Source Arity

```go
func Add(left, right int) int { return left + right }
```

emits a callable with two source parameters:

```ts
export function Add(left: int, right: int): int {
  return left + right;
}
```

It must not become:

```ts
function Add(
  operation: (left: int, right: int) => int,
  left: int,
  right: int,
): int;
```

The operation is selected by the body owner, not transported through the
source ABI.

### Variadics

A Go variadic declaration owns one final semantic slice parameter. Calls with
zero/one/many arguments construct that represented slice; `slice...` uses the
selected represented slice directly after required projection. A TypeScript
rest parameter is legal only when it is the exact one-slot representation and
all declaration/value/defer/interface/provider forms agree.

### Multiple Results

Go multiple results use one typed tuple at the target boundary. Single-result,
comma-ok, assignment, return, and argument-expansion contexts are selected by
the parent.

```go
value, ok := values[key]
```

requests the map index comma-ok operation; `value := values[key]` requests
zero-on-miss. The index expression does not decide this alone.

## Types And Values

### Basic Types

Go basic identities map to GoToTS-owned aliases over TypeScript primitives.
The default integer profile uses `number`; the explicit `bigint` profile
uses `bigint` where required. No runtime marker compensates for missing
semantics. Integer overflow not preserved by the default profile is a declared
profile tradeoff, not silently described as exact.

The alias name always follows the selected Go type, while the carrier follows
the profile and selected native width:

```text
Go type                 number profile       bigint profile (64-bit build)
int8..int32, uint8..32  number               number
int64, uint64           number               bigint
int, uint, uintptr      number               bigint
```

Thus `var index int` emits `let index: int`, never `let index: int64`; changing
`GOARCH` changes the native alias carrier, not its identity. Conversions use
the source and destination Go widths and signedness from `go/types`; alias
spelling never selects behavior.

At a gostdlib call, the source-facing signature remains unchanged. For example,
under the `number` product profile a Go `uint64` argument remains `uint64`
(`number`) in generated source and the private facade converts it to the
provider's exact `uint64` (`bigint`). Under the `bigint` product profile the
same facade crossing is an identity. Provider results take the inverse path.
These conversions occur only at the certified provider boundary and are never
passed as callable parameters.

### Defined Types And Aliases

Aliases reference the selected target representation directly. A defined type
keeps nominal identity when Go method sets, conversion, or representation
requires it. Operations project to the underlying family and wrap the result
through one type owner. Public generated names remain readable; internal
operation artifacts are not source declarations.

### Structs

```go
type Position struct {
    Line int
    Name string
}
```

normally emits:

```ts
export class Position {
  Line: int = 0;
  Name: gostring = "";
}
```

If a selected use copies `Position`, artifact reconstruction adds the exact
class-owned copy operation before seal. Unused copy/zero/hash helpers are not
emitted.

Anonymous fields remain storage. Native `extends` is admitted only for a
proved semantic spine; otherwise promoted access follows composition.

A defined struct whose basis already has canonical storage projects that
storage directly. A complete visible field layout constructs the storage
object once; a partial layout starts from the basis zero storage so hidden
fields retain Go zero values. Concrete field accessors convert only the
selected field, while generic selections remain storage-shaped. The derived
owner never reconstructs a logical basis merely to read or write one field.

### Receivers

```go
func (animal Animal) Speak() string
func (animal *Animal) Rename(name string)
```

maps to:

```ts
class Animal {
  Speak(): gostring { /* value receiver body */ }

  static Rename(animal: Animal | undefined, name: gostring): void {
    /* nil may enter; Go body decides where dereference panics */
  }
}
```

Concrete calls use exact owner qualification. A method value creates a wrapper
with the receiver-free source signature. No prototype patching or
`.call/.apply/.bind`.

### Arrays And Slices

Arrays have fixed length and Go value-copy semantics. Slices have descriptor
identity over backing storage, length, and capacity. Append, slicing, copy,
overlap, nil, and bounds behavior belong to one slice runtime owner. Element
copy and zero are demanded from the element family only where the operation
requires them.

### Maps

One `GoMap<K,V>` runtime preserves nil behavior, key equality/hash, zero on
miss, comma-ok, deletion, clear, assignment copy, and iteration contract.
Generated source uses ordinary names:

```ts
const counts = GoMap.make<gostring, int>();
counts.set("a", 1);
```

It does not emit per-site classes such as `GoMap$9e138...`.

### Pointers

`&x` identifies a location. `*p` reads or writes it and preserves nil
panic. The compiler uses direct values where no location identity can be
observed and introduces one carrier at the highest exact addressable owner
where aliasing/mutation requires it.

```go
func Read(p *int) int { return *p }
func Set(p *int, v int) { *p = v }
```

`Set` necessarily consumes a mutable location. `Read` may use a direct
read-only representation only when every selected boundary and alias fact
proves that doing so cannot change Go behavior; uncertainty retains the
carrier. Representation is selected once and propagated to all users through
observable facets.

Storage demand is joined back to the declaration of every addressable binding,
including parameters, locals, named results, range variables, and implicit
type-switch case bindings. The binding owner emits the carrier; a later address
or implicit pointer-receiver call may request it, but may never fabricate a
carrier name at the use site.

### Interfaces

An interface value is nil or a canonical dynamic-type token plus represented
payload. Concrete boxing copies value payloads and preserves reference
payloads. One adapter per reached concrete type/contract exposes only demanded
methods. Calls are native constant-size member calls; implementer switches are
forbidden.

Assertions, comma-ok, type switches, equality, comparability, and map keys use
typed runtime metadata. No constructor-name test, reflection, spelling table,
`any`, or `unknown`.

Provider-created dynamic values use the same contract. For example, a provider
may return an `error` whose concrete provider class also implements
`Unwrap() error`. A reached `errors.Is`, direct assertion, or type-switch case
demands that optional interface from the canonical `error` bridge. The bridge
uses its certified provider capability view, conditionally carries the exact
`Unwrap` method token, and converts the delegated result back through the same
`error` bridge. It does not assume that every provider error unwraps, test for
an `Unwrap` property, or special-case `os.IsNotExist`.

### Reflection

Reflection calls are selected by exact `go/types` object identity, never by the
spelling `reflect` or `TypeFor`. For example:

```go
type Entry struct { Name string `json:"name"` }

func Describe() (reflect.Kind, string) {
    typ := reflect.TypeFor[Entry]()
    return typ.Kind(), typ.Field(0).Tag.Get("json")
}
```

The call emits a reference to one canonical generated descriptor:

```ts
const typ: reflect.Type = $goReflectType_Entry;
return [typ.Kind(), typ.Field(0).Tag.Get("json")];
```

The descriptor is generated from the selected `go/types.Named` and
`go/types.Struct`; it is not assembled from target class names or target object
properties. Its `Entry` field record points lazily to the canonical string
descriptor and records the exact tag, package identity, offset, index, and
embedded/exported facts.

Dynamic observation uses the same identity:

```go
var value any = Entry{Name: "readme"}
same := reflect.TypeOf(value) == reflect.TypeFor[Entry]()
```

The interface adapter carries the exact typed `Entry` payload and its canonical
descriptor. `TypeOf` returns that descriptor and `ValueOf` creates a typed
reflective value view whose field/index/element operations delegate to
generated typed accessors. No host object inspection occurs.

For an open generic body:

```go
func KindOf[T any]() reflect.Kind { return reflect.TypeFor[T]().Kind() }
```

TypeScript generic erasure cannot implement `TypeFor<T>`. The body therefore
uses one private runtime-type operation supplied by the existing exact generic
owner or finite concretization. The source-facing callable still has zero value
parameters. A public hidden descriptor parameter, provider policy object, or
runtime type-name lookup is invalid.

## Generics

### Direct Generic Form

```go
func First[T any](values []T) T { return values[0] }
```

If the selected slice representation exposes exact typed indexing, this stays
one generic declaration:

```ts
export function First<T>(values: GoSlice<T>): T {
  return values.at(0);
}
```

Source generic/value arity is exact.

### Exact Concretization

```go
func Add[T ~int | ~string](left, right T) T {
    return left + right
}
```

TypeScript cannot express one exact `+` over open `T` without a cast. For
reached `Add[int]` and `Add[string]`, the owner may emit one internal typed
kernel and two source-shaped facades reconstructed from the same AST:

```ts
function Add$kernel<T>(
  add: (left: T, right: T) => T,
  left: T,
  right: T,
): T {
  return add(left, right);
}

function Add$int(left: int, right: int): int {
  return Add$kernel((a, b) => a + b, left, right);
}

function Add$string(left: gostring, right: gostring): gostring {
  return Add$kernel((a, b) => a + b, left, right);
}
```

Calls select the facades by exact `types.Info.Instances` evidence. The kernel
callable is individual, typed, internal, and selected statically; it is not a
public source parameter or an operation object. An unsupported open export
fails explicitly.

The same rule covers representation-disjoint builtin forms. For
`B ~[]byte | ~string`, `append(dst, src...)` requests exactly one internal
append-spread callable; concrete facades bind either the canonical slice
runtime operation or the canonical string-spread operation. The emitted source
surface still has exactly `dst, src`.

Generic named types keep only source type parameters. Storage/copy/callable
profiles never add target-only public type parameters.

### Associated Representation Types

A generic declaration keeps its source type parameters while referring to
closed representation projections:

```go
func ArrayAddress[T any](first, second T) T {
    values := [1]T{first}
    pointer := &values[0]
    values[0] = second
    return *pointer
}
```

The target kernel still has one `T`; it uses `GoContainerStorage<T>` for array
slots and `GoPointerType<T>` for the element address. If a concrete `Item`
stores as `Item$Storage`, its class declares
`GoPointerRepresentedValue<GoPointer<Item, Item$Storage>>`. A scalar or
identity-represented provider value uses the canonical fallback. No
`T$Storage` or `T$Pointer` parameter is fabricated.

Only the concrete type owner may emit an associated-type marker. Markers are
demand-created, type-only, and selected from the canonical representation
observation. They are not operation capabilities and cannot by themselves
cause a concretization or private runtime kernel.

## Expressions

Literals use checker values and selected target representations. Unary/binary
operators select exact Go operator families, including shifts, integer
division/remainder, string comparison, pointer/channel/interface equality, and
short-circuit behavior.

Selectors use exact `go/types.Selection` or package object identity. Index and
slice expressions use parent context plus selected family. Composite literals
delegate field/element order and implicit keys to their owner. Type assertions
use checker-selected target types and canonical interface metadata.

Short-circuit arms, conditional branches introduced by translation, and
function literals retain prerequisites inside the arm where Go evaluates them.
Go function literals emit TypeScript arrow functions because Go closures have
lexical receiver capture and no call-site-bound `this`. A JavaScript `function`
expression is not an equivalent fallback: inside a native receiver method it
would detach a captured Go receiver when passed as a callback.

## Statements And Control

Blocks and clause bodies share one ordered statement-list owner. It preserves
lexical scopes and parent-assigned break/continue targets.
An entered loop replaces both inherited implicit targets. An entered non-loop
breakable statement replaces only the break target and preserves the nearest
enclosing loop's continue target. Thus a loop inside a lowered switch cannot
break the switch, while a switch inside a labeled loop cannot turn its own
unlabeled break into a loop break.

- `if`, `switch`, and type switch preserve initializer scope and one-time
  evaluation;
- `for` preserves init/condition/post order;
- `range` selects array/slice/string/map/channel/function semantics from the
  checker type, including copies and iteration variables;
- labels use exact `types.Label` identities;
- native target control is preferred when exact;
- only non-structural goto uses the linear target state machine.

A source label is attached exactly once to its direct statement. A directly
labeled loop, switch, type switch, or select may consume the target-label
capability while emitting that statement. A non-breakable direct statement
such as `if` is wrapped by the label owner after its body is emitted; it must
not pass a reusable target label to nested breakable statements. The control
label used to resolve source `break`, `continue`, and `goto` identities remains
available independently and never selects a target label by spelling.

An explicit Go `fallthrough` never becomes an implicit TypeScript case fall.
The switch owner selects one clause, then executes ordered clause blocks under
one break label; fallthrough advances to the next block without re-evaluating
its case expressions. A Go `break` exits that label, while `continue` still
targets the enclosing source loop.

An expressionless Go switch whose case expressions have no prerequisites and
whose clauses do not fall through emits an ordered `if`/`else if` chain inside
that break label. It never emits `switch (true)`: TypeScript may incorrectly
narrow a Go boolean mutated through a closure or a prior loop iteration to the
literal `false`. Cases with prerequisites or fallthrough use the general
single-selection lowering above, which preserves conditional evaluation.

The selected Go checker proves that every result-bearing function has no
reachable end. If exact target control lowering hides that proof from the
TypeScript checker, the callable owner appends one unreachable `throw`; it does
so only when the emitted target statement sequence can still fall through.
The guard never changes the callable signature, returns a fabricated value, or
introduces a runtime policy parameter.

### Defer, Panic, And Recover

```go
func F() (result int) {
    defer func() { result++ }()
    return 1
}
```

The return stores `1`, deferred arguments/callee were captured when the defer
statement ran, the deferred literal runs LIFO, and the final result read
returns `2`.

A callable's ordinary target signature never contains recovery state. If its
body uses `recover`, its private deferred entry receives the authority:

```ts
export function recoverPanic(): GoAny {
  return recoverPanic$body(undefined);
}

function recoverPanic$deferred(recovery: GoRecovery): GoAny {
  return recoverPanic$body(recovery);
}
```

The body helper and deferred entry are private compiler artifacts. Ordinary
calls return nil from `recover`; only direct deferred invocation receives the
pending panic. A nested call does not.

Dynamic function-value defer consults the exact-signature typed deferred-entry
registry only for a demanded dynamic signature. A recover-capable value has a
registered private entry; every other value falls back to ordinary invocation.
This does not alter the function type or append an argument. A certified
provider recovery facet follows the same rule: presence selects its private
entry, while absence uses the ordinary provider callable.

### Channels, Goroutines, And Select

Under `cooperative`, channel send/receive and blocking select lower to typed
Promise operations. Direct call effects propagate through revisable callable
facets. Function values and interface methods use canonical
`Awaitable<R>` results and are unconditionally awaited.

```go
func Consume(next func() int) int { return next() }
```

maps in the cooperative profile to:

```ts
export async function Consume(
  next: () => Awaitable<int>,
): Promise<int> {
  return await next();
}
```

The source still has one parameter. No synchronous/cooperative public variants,
hidden effect parameters, or runtime Promise tests exist.

`go f(args)` captures/copies immediately and schedules one closure. `select`
evaluates operands once, chooses a ready case fairly, commits once, and
cancels every other registration. Receive assignment targets are evaluated
only after their case wins.

## Packages, Standard Library, And Externals

Source packages translate normally. Standard-library references use generated
static facades linked to certified `@gotots/gostdlib/<go-path>.js` exports.

```go
failure := fs.WalkDir(fileSystem, ".", visit)
```

emits a call with the same three Go arguments:

```ts
const failure = await io_fs.WalkDir(fileSystem, ".", visit);
```

The generated `io_fs.WalkDir` facade may statically import provider bridges
or a private provider kernel, but no call site supplies a policy object.

Selected bodyless or true-external declarations emit exact typed throwing
placeholders and one canonical obligation file grouped by semantic owner.
Their filesystem shape is deterministic:

```text
generated/
  packages/<module>/<package>/...
  environment/<go-import-path>.ts
  obligations/<go-import-path>.json
  runtime/...
```

Provider linkage replaces the placeholder owner atomically; it does not add a
fallback. Publication fails if a reachable obligation remains.

An external-function obligation binds the canonical Go declaration identity,
stable `go/types` signature, owning module path and version, selected Go build
profile, selected target representation profile, and source position. The
compiler accepts one verified external-provider certificate; it does not accept
per-function package/name overrides. The certificate is regenerated from the
pinned dependency modules and the strict TypeScript provider project, then
byte-compared with the checked manifest before compilation may consume it.
Linkage accepts exactly one of two static targets recorded by that certificate:

1. a certified ESM module export, called with the original Go value parameters;
2. a body-owning function in the same selected Go package with an identical
   signature, used as the portable implementation of a selected native entry.

For example:

```go
func Syscall(trap, a1, a2, a3 uintptr) (uintptr, uintptr, syscall.Errno)
```

linked to `@gotots/externals/golang.org/x/sys/unix.js` emits:

```ts
import * as unix from "@gotots/externals/golang.org/x/sys/unix.js";

export function Syscall(trap: uint64, a1: uint64, a2: uint64, a3: uint64) {
  return unix.Syscall(trap, a1, a2, a3);
}
```

No provider argument appears in the source-facing signature. A portable-source
binding emits the same direct wrapper call to the selected translated function.
Missing, duplicate, stale-profile, stale-module-version, incompatible-signature,
extra, or unconsumed bindings fail before target files are sealed. The initial
linkage contract covers nonvariadic nongeneric package functions; every other
bodyless callable remains an explicit obligation until its exact source ABI is
specified rather than being widened heuristically.

The certificate records the provider package/version/digest, its exact
`@gotots/gostdlib` manifest dependency, integer and concurrency profiles, each
source identity/signature/module version, and each target export fingerprint or
portable-source identity/signature. Compiler code joins only those typed facts;
it never selects a target from package spelling, function spelling, source-file
name, or call-site heuristics. Unused entries in a certified provider catalog are
not compilation overrides. A selected matching source declaration must consume
exactly one entry or remain an explicit obligation.

## Failure

Translation fails at the owning occurrence when:

- a construct/context has no handler;
- required checker evidence is absent or inconsistent;
- exact direct target representation is unavailable;
- generic concretization is open or unbounded;
- a source signature would require hidden source parameters;
- a provider contract/conversion is missing or ambiguous;
- target artifact reconstruction does not converge;
- TS-Go schema/encoding drifts;
- strict TypeScript fails.

No heuristic, source-name allowlist, cast, dynamic lookup, compatibility shim,
or widened threshold substitutes for the missing exact rule.
