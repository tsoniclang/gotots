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

TypeScript cannot express one exact `+` over open `T` without a cast or
operation parameter. For reached `Add[int]` and `Add[string]`, the owner may
emit two private exact implementations reconstructed from the same AST:

```ts
function Add$int(left: int, right: int): int {
  return left + right;
}

function Add$string(left: gostring, right: gostring): gostring {
  return left + right;
}
```

Calls select these by exact `types.Info.Instances` evidence. Neither function
is a second public generic ABI. An unsupported open export fails explicitly.

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
