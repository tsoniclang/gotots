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

When declarations originating in different Go files share one target file,
their canonical order is the semantic source path followed by position within
that file, then declaration identity. A raw `token.Pos` never orders across
files: its numeric base reflects parser allocation timing rather than source
semantics. Relocated receiver methods therefore remain byte-stable regardless
of package-loading concurrency.

Every generated binding inserted into a source-bearing lexical scope is
allocated by the file name owner. The owner reserves the portable target
spellings of every explicit and implicit Go binding in the complete package,
every selected import alias, and every generated binding already allocated;
it therefore avoids a source name even when that declaration occurs later or
inside a descendant scope. For example, a source parameter named
`__gotots_field_0` forces the first composite-literal capture to use
`__gotots_field_1` or the next available name. Generated member names use the
struct member owner under the same no-duplicate rule. The owner reserves the
lowercase member `then`: JavaScript uses that property for Promise assimilation,
which has no Go-language counterpart. A legal Go field named `then` therefore
receives the next deterministic collision-safe member name, and every access
uses that one identity. A literal target-only name is admitted only inside a
structurally closed synthetic scope whose owner defines the complete binding
set and uses the `$` namespace that Go source cannot spell. Raw counters are
forbidden in scopes that contain authored bindings.

A compilation-global artifact keeps an imported derived export unqualified
when that export is unique in the selected source universe. The naming owner
preindexes exact declaration identities and adds the package qualifier only
when two distinct source modules can provide the same local derived export, or
when the local target scope is already occupied. Collision safety must not add
hash or package suffixes to every ordinary generated reference.

Target host intrinsics have one closed identity owner. Value references use
`globalThis.<name>`, so a valid Go declaration such as
`func String(string) Token` keeps its source-shaped name without shadowing the
target `String.fromCharCode` used by an unrelated conversion. Source callables
never emit the host `Promise` type. TypeScript's forbidden target class name
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
constructor is never eagerly constructed during ESM module evaluation. Its
source artifact is one private, statically typed value thunk and generated Go
uses invoke that thunk. The package assembly declares the ordinary public
value without an initializer, then assigns it once at the start of the package
`$initialize` function, before package variables and Go `init` functions. The
program initializer runs packages in Go dependency order. This preserves the
Go-facing value API while preventing a legal generated module cycle from
becoming an ESM temporal-dead-zone failure.

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

// package.ts: no constructor executes during ESM evaluation
export let Value: ReturnType<typeof Value$constant>;
export function $initialize(): void {
  Value = Value$constant();
}
```

Primitive package constants stay direct immutable bindings, and local
constants stay at their lexical execution boundary. The compiler does not
inline repeated large values merely to avoid a module cycle: constant payload
size remains `O(value bytes + uses)`, not `O(value bytes * uses)`.
Inlining all uses is rejected for that scaling reason; generated Go code never
reads the delayed public binding, and consumers cannot assign an imported ESM
binding. The only writer is the package initializer that already owns Go
package-variable and `init` ordering. An untyped or dynamically recovered
binding is rejected because it would expose initialization state without an
exact contract;
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

### Source-Implementation Pointers

Provider TypeScript uses a direct ABI that is independent of the product's
canonical marker contract. For example, a provider implementation of:

```go
func Parse(text string) *Entry
func Increment(value *int)
```

has source-implementation shapes equivalent to:

```ts
export function Parse(text: gostring): Entry | undefined;
export function Increment(value: ProviderPointer<int> | undefined): void;
```

`Entry` is the location for `*Entry`; the scalar location is the certified
`{ value: int }` carrier. A generated result boundary emits canonical intent:

```ts
const raw = provider.Parse(text);
return raw === undefined
  ? undefined
  : bindPointer(raw, () => raw, value => EntryOperations.$assign(raw, value));
```

For `*int`, the closures read and write `raw.value`. This is canonical
Tsonic-flavored TypeScript; GoToTS typechecks it but does not execute marker
calls. TSTS finalizes the exact marker facts and the selected target lowers
them before runtime execution. Provider source itself never imports or calls
`bindPointer`, `loadPointer`, or `storePointer`.

### Basic Types

Go basic identities map to GoToTS-owned aliases over TypeScript primitives.
The default integer profile uses `number`; the explicit `fixed64-bigint` and
`bigint` profiles use `bigint` for their selected wide carriers. No runtime
marker compensates for missing semantics. Integer overflow not preserved by a
selected number carrier is a declared profile tradeoff, not silently described
as exact.

Each BigInt-carrier profile is exact for operations whose selected carrier is
`bigint`. Their arithmetic, bitwise, shift, unary, compound-assignment, and
increment results are normalized to the selected Go width before they become
observable. The demand-generated `goInt64` and `goUint64` runtime operations
are the only wide-result wrappers; each delegates to the corresponding
`globalThis.BigInt.asIntN` or `asUintN` operation. For example, `uint64(max) +
1` emits `goUint64(max + 1n)` and produces zero. Number-carried aliases retain
direct operations and the declared number-carrier overflow tradeoff; neither
explicit profile adds wrappers to ordinary `int32` expressions, and
`fixed64-bigint` does not add them to native `int`. This policy is owned by the
integer value family and is never selected by package, function, or identifier
spelling.

A product whose reached behavior depends on exact fixed-width wide-integer
values, such as a `uint64` hash used as a cache identity, cannot claim runtime
parity under the `number` profile. It must select at least `fixed64-bigint`
explicitly. A product whose behavior additionally depends on native 64-bit
integer overflow must select `bigint`. GoToTS neither changes a profile
heuristically nor adds a local override for the reached package.

The alias name always follows the selected Go type, while the carrier follows
the profile and selected native width:

```text
Go type                 number       fixed64-bigint    bigint (64-bit build)
int8..int32, uint8..32  number       number            number
int64, uint64           number       bigint            bigint
int, uint, uintptr      number       number            bigint
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

Generic provider kernels retain the generated caller's representation facets.
For example, `cmp.Or(0, 7, 9)` passes its generated `RuntimeSlice<int>` to
`CmpOrKernel<int, int>`; the `[]T` contract is generic-owned and is not rebuilt
as a provider `RuntimeSlice<bigint>`. In contrast, a concrete non-generic `int`
parameter such as the count in `slices.Grow(values, count)` is converted to the
provider's `int64` carrier at the private boundary. Callable parameters are
adapted recursively rather than treated as opaque merely because they mention
a type parameter.

Container-transforming kernels also retain Go assignment semantics. A call
such as `slices.Clone(records)` receives typed conversion, copy, storage, and
inverse-storage operations for the concrete `Record`; the kernel emits a new
slice whose elements are copied values rather than shared target storage
objects. In-place operations use the same operations while preserving the
original backing descriptor. The compiler never selects behavior from the
element's TypeScript spelling or assumes that a generic `T` is reference-safe.

Canonical container results are also recursive. For example, provider
`fs.ReadDir` returns a provider slice of provider `DirEntry` contracts; generated
code receives a `RuntimeSliceProjection` whose element functions are the
certified `DirEntry` profile bridges. A canonical `error` parameter, however,
already has the generated interface ABI and is passed unchanged so generated
`Is(error)` and `Unwrap()` implementations remain observable.

Go strings retain their exact byte sequence in target storage: one target code
unit represents one Go byte. Crossing to a host text API requires the canonical
UTF-8 decoder; crossing to a host byte API requires the canonical raw-byte
projection. Thus `File.WriteString("\xe2\x80\x9c")` writes the three source
bytes, not the six bytes produced by UTF-8-encoding three Latin-1-looking host
characters. NUL and invalid UTF-8 remain legal. Provider code may not pass a
canonical Go string directly to a Node string-writing API or define a second
byte/text conversion helper.

### Defined Types And Aliases

Aliases reference the selected target representation directly. A defined type
keeps nominal identity when Go method sets, conversion, or representation
requires it. Operations project to the underlying family and wrap the result
through one type owner. Public generated names remain readable; internal
operation artifacts are not source declarations.

A source-owned, non-generic defined type whose underlying type is `int8`,
`uint8`, `int16`, `uint16`, `int32`, or `uint32` uses one plain named scalar
alias regardless of its method sets. Go methods do not give numeric values
object identity. The alias declaration and source-facing references retain the
source identity in the TS-Go AST; ordinary values remain native scalars.
Methods on this representation are emitted as source functions: a value
receiver is the first value parameter, while a pointer receiver is the first
`Pointer<T> | undefined` parameter. Direct method calls, method expressions,
method values, promoted selections, and interface adapters all select that one
function identity; no wrapper member route remains.

```go
type NodeFlags uint32

func (flags NodeFlags) Has(mask NodeFlags) bool { return flags&mask != 0 }
func Add(left, right NodeFlags) NodeFlags { return left + right }
```

```ts
export type NodeFlags = uint32;

export function NodeFlags_Has(flags: NodeFlags, mask: NodeFlags): boolean {
  return (flags & mask) !== 0;
}

export function Add(left: NodeFlags, right: NodeFlags): NodeFlags {
  return left + right;
}
```

This representation allocates no wrapper and does not restrict the legal
underlying numeric values. The selected Go checker—not TypeScript structural
assignability—owns source validity. It is selected from `go/types`
identity, underlying kind, and generic arity; source spelling and package
identity never select it. Generic types, profile-dependent native-width
integers, strings, booleans, floats, complex values, and all other defined
families retain their exact existing representation. Type-representation
demands whose storage is already the scalar value consume no class marker or
auxiliary carrier.

A conversion to an ordinary basic or another defined numeric type is a direct
scalar expression. Local TypeScript inference needs no corrective annotation:

```go
position := int32(node.Pos())
```

```ts
let position = node.Pos();
```

No wrapper or unrelated annotation is emitted. Explicitly typed declarations
already carry their selected type; non-declaration uses are checked by their
parent context.

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

### Struct Construction And Value Demand

Source-owned structs use declaration names at construction sites. For example:

```go
type Point struct { X, Y int32 }

func NewPoint() Point {
    return Point{Y: mark(2), X: mark(1)}
}
```

maps to the source-ordered ordinary shape:

```ts
class Point {
  constructor(public X: int32, public Y: int32) {}
}

function NewPoint(): Point {
  const field0 = mark(2);
  const field1 = mark(1);
  return new Point(field1, field0);
}
```

The constructor's parameters are the complete declaration field set and create
the fields directly without an argument-object allocation. Under the
preserve-Go profile, the call captures only the expressions needed to retain
source evaluation order before placing arguments in declaration order. The
direct profile omits those captures by explicit project choice. If a later
semantic demand selects canonical storage, every subscribed construction is
rebuilt to the stable public storage factory instead:

```ts
return Point.$fromStorage({ X: field1, Y: field0 });
```

An empty storage-backed value uses `Empty.$fromStorage({})`; neither source
construction nor a package consumer calls a storage constructor directly.
Static `$zero`, `$copy`, `$equal`, `$hash`, `$convert`, `$storageOf`, `$fromStorage`,
`$zeroStorage`, and `$assign` members exist only when their exact semantic use
requests them. `$zeroStorage` constructs canonical field storage directly; a
storage consumer must not allocate `$zero()` and immediately discard its wrapper
through `$storageOf(...)`. For example, a nested `Inner` field in `Outer{}` uses
`Inner.$zeroStorage()`, not `Inner.$storageOf(Inner.$zero())`; array and slice
storage initialization applies the same rule through the one container-storage
zero owner. A
certified provider may expose a positional `$make` operation; ordinary
generated structs never gain that compatibility factory.

### Arrays And Slices

Arrays have fixed length and Go value-copy semantics. Slices have descriptor
identity over backing storage, length, and capacity. Append, slicing, copy,
overlap, nil, and bounds behavior belong to one slice runtime owner. Element
copy and zero are demanded from the element family only where the operation
requires them.

An ordinary array literal remains a readable `GoArray.literal` call. A static
literal with more than 4096 explicit elements must not expand each source
entry into separate target index and value nodes. When its element is a plain
8-, 16-, or 32-bit integer represented by `number` and every entry has exact
checker constant evidence, the array owner emits one demand-generated
`goArrayPacked` call. Its base-36 payload records exact index/value pairs, so
keyed sparse literals retain zero-filled holes without constructing an
expanded target AST:

```go
var table = [8193]int32{0: -5, 8192: 12 /* plus many constants */}
```

```ts
const table = goArrayPacked<int32, 8193>(
  8193, 0, 4097, "0,-5,...,6bk,c" /* abbreviated one-node payload */,
);
```

The fixed readable-node ceiling is a compiler resource-safety invariant, not
a corpus exception or target optimization switch. Unsupported element
representations, aggregate values, and nonconstant entries retain their
existing exact emission path. The decoder validates entry count, safe integer
syntax, and indexes before publishing the array. It uses no dynamic type
recovery, filesystem artifact, text patch, or product-specific implementation.

### Maps

Map representation is selected once from the exact closed key and value
types. A map whose key already has an identity boolean, integer, or string
primitive representation and whose value is runtime-basic uses the canonical
`GoMap<K,V>` owner. A map that needs an explicit key projection or static zero,
copy, hash, or equality operations uses one deduplicated support specialization
for its semantic shape, never one class per use site.

When the selected key storage is an exact boolean, integer, or string
primitive, a specialized map stores tuple cells in a native JavaScript `Map`.
This includes a defined key only when its canonical storage mapping is
bijective: either the identity representation or an explicit projection.
Public lookup/store/delete/keys operations apply and reverse any explicit
projection at the specialization boundary. The cell
distinguishes a present `undefined` value from an absent key. Floating-point,
complex, pointer, interface, aggregate, and non-bijective defined keys retain
the typed hash/equality bucket representation. Both paths preserve nil
behavior, zero on miss, comma-ok, deletion, clear, assignment copy, and the
iteration envelope through the same Go-shaped map contract.

A native specialization performs its final write through the one canonical
generic `goMapStore<K,V>(values, key, value)` runtime callable. That callable
evaluates its three arguments once in order, executes exactly
`values.set(key, value)`, and returns `void`; hashed storage does not request
it. This is an executable canonical boundary, not a target decision. A
selected target may inline the checked callable only under its own explicit
exact-declaration contract, while an unoptimized consumer executes the same
GoToTS-owned behavior directly.

That common contract is one exported nominal abstract class, not a structural
interface:

```ts
abstract class GoMapValue<K, V> {
  declare private readonly then?: never;
  abstract lookup(key: K): V;
  abstract store(key: K, value: V): void;
  // the remaining map operations are abstract members of this same contract
}

class GoMap<K, V> extends GoMapValue<K, V> { /* direct implementation */ }
class MapOfPointToItem extends GoMapValue<Point, Item> { /* specialization */ }
```

The nominal root is emitted before every same-module subclass, each subclass
calls `super()`, and the erased private member is inherited exactly once. This
lets a checked consumer prove that a returned map cannot participate in
JavaScript Promise assimilation without recognizing `GoMapValue` by spelling.
A public structural `then?: never` is not substituted: a declaration boundary
could hide a runtime accessor, so it would not establish the same closed
contract. The abstract root emits one empty JavaScript class; no map instance
gains a field or per-operation wrapper from this rule.

Native key equality is exact only within the selected primitive carrier. In
the `number` profile, wide integer keys retain the declared precision tradeoff:
for example, `uint64(9007199254740992)` and `uint64(9007199254740993)` have the
same JavaScript `number` key. A product requiring exact `int64` or `uint64` map
identity must select `fixed64-bigint` or `bigint`; native map storage neither
hides the collision nor changes the compilation profile.

### Pointers

`&x` identifies a typed location. `*p` reads or writes it and preserves nil
panic. Canonical output records that intent with the accepted neutral pointer
contract; it does not preselect a JavaScript carrier or scalarize a pointer:

```ts
import type { Pointer } from "@tsonic/core/types.js";
import {
  addressOf,
  allocatePointer,
  equalPointer,
  loadPointer,
  storePointer,
} from "@tsonic/core/lang.js";
```

```go
func Read(p *int) int { return *p }
func Set(p *int, v int) { *p = v }
```

Canonical source has the human-shaped contract:

```ts
function Read(pointer: Pointer<int> | undefined): int {
  if (pointer === undefined) goPanicNil();
  return loadPointer(pointer);
}

function Set(pointer: Pointer<int> | undefined, value: int): void {
  if (pointer === undefined) goPanicNil();
  storePointer(pointer, value);
}

let value: int = 1;
const pointer = addressOf(value);
const fresh = allocatePointer<int>(0);
```

GoToTS owns the explicit nil checks because they are Go behavior. Pointer
equality uses `equalPointer` over `Pointer<T> | undefined`; raw TypeScript
`===` is not the canonical source operation because two independently formed
addresses of the same storage must compare equal.
Passing, returning, or storing a pointer preserves the same location identity.
Every marker call is emitted as TS-Go AST and selected by exact semantic
evidence, never by marker spelling.

The TypeScript target may later transform a complete location flow to a native
value or one small `{ value: T }` location when that preserves all aliases. A
target optimization never changes canonical source, is never selected by
GoToTS, and must exact-join the finalized TSTS facts for every transformed
occurrence. The unoptimized/native-target build consumes the pointer facts
directly.

Opaque `unsafe.Pointer` identity uses the accepted target-neutral contract:

```go
pointer := unsafe.Pointer(&value)
same := pointer == unsafe.Pointer(&value)
lookup[pointer] = true
```

```ts
const pointer: RawPointer | undefined = bindRawPointer(addressOf(value));
const same = equalRawPointer(pointer, bindRawPointer(addressOf(value)));
lookup.store(pointer, true); // the map key facet uses hashRawPointer
```

Nil remains `undefined`. A certified provider raw-pointer result is bound by
its opaque object identity before entering generated code. Raw-address
arithmetic, reinterpretation, raw-pointer-to-typed-pointer conversion,
pointer/integer conversion, and provider raw-pointer inputs remain exact typed
boundaries. Canonical output never exposes or fabricates an address, and safe
typed pointers are not lowered through a legacy JavaScript virtual-address
representation.

### Interfaces

An interface value is nil or a canonical dynamic-type token plus represented
payload. Concrete boxing copies value payloads and preserves reference
payloads. One adapter per reached concrete type/contract exposes only demanded
methods. An adapter with demanded Go methods extends the canonical
interface-value root and calls its zero-argument base constructor. An adapter
with no demanded Go methods is a typed constructor produced by one canonical
demand-selected runtime factory:

```ts
export const IntAdapter = createGoInterfaceAdapter<int32>(
  intDynamicType,
  (left, right) => left === right,
  value => hashInt(value),
  false,
  (value, verb, flags, precision) => formatInt(value, verb, flags, precision),
);
```

The factory returns a statically typed constructor with the same `new
Adapter(value)` and `Adapter.$is(value)` ABI as a method-bearing adapter. It
contains the common box behavior once and does not inspect names or recover an
erased payload. Both shapes inherit the one Promise-assimilation exclusion
rather than redeclaring an incompatible private member. Calls are native
constant-size member calls; implementer switches are forbidden.

Every generated root class, including the canonical interface-value contract,
declares the erased nominal member
`declare private readonly then?: never`. JavaScript therefore cannot mistake a
generated Go value for a Promise-like value when a host Promise resolves it,
while structural TypeScript values cannot claim the guarantee. The
declaration emits no JavaScript field. Generated derived classes inherit the
one root declaration and never redeclare its private member. A source Go method
named `then` is unexported and keeps its package-qualified target member
identity; a source field named `then` is deterministically renamed by the
member-name owner. Neither occupies JavaScript's Promise-assimilation member.

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
export const $goReflectType_Entry: named_reflect.RuntimeType =
  named_reflect.ReflectTypeMetadataOperations.$create(/* exact metadata */);
const typ: reflect.Type = $goReflectType_Entry;
return [typ.Kind(), typ.Field(0).Tag.Get("json")];
```

The descriptor is generated from the selected `go/types.Named` and
`go/types.Struct`; it is not assembled from target class names or target object
properties. Its declaration and every generated descriptor callback preserve
the provider facet's certified concrete result type. Only the authored
Go-facing assignment above widens it to `reflect.Type`; the compiler does not
erase the concrete result at its provider boundary. Its `Entry` field record
points lazily to the canonical string
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

Pointer element locations remain exact when the pointee is itself an interface
or another non-scalar kind:

```go
var slot any = "before"
element := reflect.ValueOf(&slot).Elem()
element.Set(reflect.ValueOf(int64(7)))
```

The `*any` registration loads the existing interface box from `slot`, records
the static element descriptor as `any`, admits writes through the canonical
empty-interface contract, and stores the selected dynamic box back into the
same location. It does not wrap that box in a fabricated interface adapter or
fall back to JavaScript shape inspection. The same location path delegates
array copies and pointer/function/channel reference values to the ordinary
value-transfer owner rather than selecting pointee behavior from a reflection
kind whitelist.

Container operations retain the same assignment algebra as ordinary Go:

```go
type Cell struct { Count int }
cells := []Cell{{Count: 1}}
grown := reflect.Append(reflect.ValueOf(cells), reflect.ValueOf(Cell{Count: 2}))
pointers := reflect.MakeSlice(reflect.TypeOf([]*Cell{}), 2, 4)
```

The `[]Cell` descriptor copies existing aggregate elements if append allocates
a new backing array and stores a copied incoming `Cell`; it never aliases the
same target object as two independent Go struct values. The `[]*Cell`
descriptor instead stores pointer values directly and initializes new entries
to the typed nil pointer. `MakeSlice` and `Value.Grow` use the same canonical
slice-storage constructor and typed zero operation. Reflective maps similarly
select the compiler's ordinary scalar, native-key, or hashed map
representation from their exact Go key/value types rather than a basic-type
whitelist.

Per-type reflection output contains facts, not a copied reflection
interpreter. For example, an addressable source struct contributes one compact
typed registration equivalent to:

```ts
ReflectTypeMetadataOperations.$registerStruct(
  $goReflectType_Entry,
  () => $goInterfaceAdapter_Entry,
  entry => entry,
  fields => [
    fields.valueProperty(
      () => $goReflectType_string,
      () => $goInterfaceAdapter_string,
      "Name",
      storage => new $goInterfaceAdapter_ptr_string(addressOf(storage.Name)),
    ),
  ],
  value => Entry.$copy(value),
);
```

The portable provider owns the one adapter guard, field bounds check,
location wrapper, and clone wrapper used by every such registration. The
adapter resolver is retained lazily and evaluated only when reflection first
materializes the operation record, after ESM module initialization. The
storage resolver is likewise typed: for a direct class it is the identity; for
a selected canonical-storage class it is that class's exact `$storageOf`
operation. A field requiring Go value-copy semantics uses
`copyingValueProperty` with the existing canonical copy operation. A field
whose storage requires projection retains an explicit typed getter and setter
instead of pretending that a property key is sufficient. The emitted property
key is the target-name owner's exact member identity and is checked against the
inferred storage type; it is not a source-name lookup. No payload cast, runtime
shape test, `any`, `unknown`, or host reflection is permitted. Provider-
represented structs use a distinct typed opaque-field registration whose only
generated facts are field-order-preserving failure messages; they do not
retain the ordinary field-access path behind a fallback.

An addressable registration also carries its canonical pointer box callback.
For example, a field callback returns the exact adapter around
`addressOf(entry.Name)`, while a pointer-element callback returns the original
pointer box. The portable `Value.Addr` implementation consumes that callback;
it never invents a second cell or an unboxed pointer-shaped object. The exact
box token is registered against the canonical pointer descriptor at that
boundary, so a later `ValueOf` resolves the same type without eagerly demanding
reflection artifacts for every addressable child type. The pointer adapter
is also recorded at the reflection-interface boundary through which
`Interface` and `TypeAssert` expose it. Once ordinary emission is quiescent,
that owner exact-joins the recorded adapter to reached assertion contracts
using the selected checker graph and schedules all matches for that adapter as
one requirement batch. It does not enroll the adapter in ordinary
empty-interface reachability. Thus this address-only generic value:

```go
type Decoder interface { Decode([]byte) error }
type Cell[T ~string] struct { Value T }
func (cell *Cell[T]) Decode(text []byte) error {
    cell.Value = T(text)
    return nil
}

field := reflect.ValueOf(&holder).Elem().Field(0)
decoder, ok := reflect.TypeAssert[Decoder](field.Addr())
```

causes the canonical `*Cell[string]` adapter to carry `Decoder`'s exact method
token and implementation. Unasserted methods on `*Cell[string]` are not added.
It does not demand a static pointer reflection descriptor merely to make the
assertion work. No marker spelling, eager reconstruction, method-name lookup,
or product-specific adapter list participates.

Provider-created reflected values use the same rule. For example,
`fresh := reflect.New(reflect.TypeOf(Label("")))` followed by
`reflect.TypeAssert[Decoder](fresh)` gives the generated `*Label` adapter the
`Decoder` contract even if no authored `*Label` value was converted to an
interface. The already-demanded pointer descriptor still owns construction;
the quiescent join adds only the selected method contract.

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

The concrete `(a, b) => a + b` operation is constructed directly in the
facade's TS-Go AST. Ordinary exact operations do not become sibling capability
files. A genuinely shared constraint-method or deferred-callable-registry
operation may remain a named private artifact because its effect or registry
contract, rather than a concrete arithmetic expression, owns the behavior.

The examples' `$int` and `$string` suffixes are canonical semantic names, not
abbreviated hashes. Method concretizations include the receiver, effects such as
the synchronous callback profile are explicit suffix components, and imports
are collision-checked against every visible authored, imported, and generated
binding before the TS-Go AST is sealed.

Compilation-generated types in one closed support family share that family's
bounded semantic module. Each definition keeps an injective semantic export
name whose named components use the package registry's unique readable
qualifier. For example, the specialization for
`map[chan int32]map[int32]<-chan int32` is emitted in:

```text
support/maps.ts
```

It exports
`$goMap$MapOf_ChannelOf_int32_To_MapOf_int32_To_ReceiveChannelOf_int32`.
A caller imports that exact export as the short local family name `GoMap` when
free. If `GoMap` is already visible, the caller first uses the full semantic
export name locally; only a real second collision adds the shortest exact
source-derived qualifier. A lexical anonymous struct likewise keeps its
complete `$goStruct$...` name. Artifact digests remain in manifests and
internal ownership only. A named component such as
`github.com/microsoft/typescript-go/internal/ast.SourceFile` appears as
`ast$SourceFile`; its full import path is not copied into every generated
identifier. Same-spelled local types receive the visibility-aware source name
already allocated for their lexical scope (`Local`, or `Local__shadow_...`
when the other binding is visible) and remain distinct through their exact
private Go-identity keys and lexical placement. Disjoint scopes may reuse the
same readable name.
Semantic contracts never carry a pre-rendered TypeScript suffix.
Unexported interface-method member names and token constants use that same
readable package qualifier; they do not independently encode `types.Id` paths.

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
slots and `Pointer<T>` for the element address. The canonical indexed-address
marker carries the exact container, index, logical element type, and location
semantics, so a concrete `Item` may retain its target-neutral logical type
while a selected target chooses the storage representation. It does not
request a generated pointer cell or a generic cell/load/store callable. No
`T$Storage` or `T$Pointer` parameter is fabricated.

Only the concrete type owner may emit a storage associated-type marker. Such
markers are
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

Because this defer site can execute at most once per invocation, its target
uses one fixed optional callable slot and invokes it from `finally`. By
contrast, a defer inside `for` or a conditional/goto-controlled region uses the
dynamic LIFO stack because the number and order of captured calls are runtime
facts. Both shapes preserve immediate capture and direct-recover authority;
the compiler never chooses the fixed form from source spelling alone.

Each defer-statement handler contributes its exact source-node identity to the
enclosing callable's root demand during the normal contextual emission walk.
The callable uses fixed slots only when that demand set equals the defer
statements directly present in its body list and goto control is absent.
Nested, conditional, repeated, missing, additional, or conservatively
unlocated sites therefore retain the dynamic stack without a recursive AST
scan.

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

Channel types use one typed runtime identity under fixed serial execution. A
goroutine call is emitted as a direct call; ready buffered send/receive and
ready/default select complete synchronously. A nil, unbuffered, full, empty, or
otherwise unready operation that would suspend raises the typed
serial-blocking panic instead of fabricating progress. The language path emits
no `Promise`, `async`, `await`, scheduler, waiter queue, or host task.
Explicit event-based provider APIs such as timers retain their separately
certified host callback behavior without changing the source call's direct
signature.

Source functions, methods, literals, callable values, interface methods,
generated callable contracts, package initializers, provider facades, and
selected generic kernels all have direct synchronous signatures at
construction. Provider bindings and stateful profiles are selected by exact
certified identity and must report a synchronous effect. A suspending provider
path fails before publication; no consumer repairs its declaration or call.

When a stateful provider type retains interface-typed state, it has one
certified synchronous target class and constructor profile. For example,
`bufio.NewWriter(w)` selects a class whose `Write` and `Flush` methods and
retained `io.Writer`/`error` contracts are synchronous. It has no awaitable
sibling and is never selected by target spelling.

Every possibly nil function value is evaluated once, its arguments are then
evaluated in source order, and one generic runtime check returns the identical
statically typed callable or raises Go's nil-function runtime panic:

```go
result := callback(argument())
```

```ts
const callee = callback;
const value = argument();
const result = (callee ?? GoPanic.raiseRuntime("call of nil function"))(value);
```

The nullish check adds no helper call, source ABI parameter, cast, dynamic
dispatch, or result adaptation. Statically non-nil declarations and literals
call directly. A conditional, assertion, or alternate nil-call path is
forbidden.

`go f(args)` evaluates/copies its callee and arguments immediately, then invokes
the call on the current stack. `select` evaluates operands once, chooses one
ready case fairly, and commits it once. It installs no registrations. Receive
assignment targets are evaluated only after their case wins.

## Packages, Standard Library, And Externals

Source packages translate normally. Standard-library references use generated
static facades linked to certified `@gotots/gostdlib/<go-path>.js` exports.

```go
failure := fs.WalkDir(fileSystem, ".", visit)
```

emits a call with the same three Go arguments:

```ts
const failure = io_fs.WalkDir(fileSystem, ".", visit);
```

The generated `io_fs.WalkDir` facade may statically import provider bridges
or a private provider kernel, but no call site supplies a policy object. Its
provider contract and visitor callback are certified synchronous.

A provider-defined callable type is joined by identity to the certified
synchronous effect of its target carrier. For example, an `iter.Seq[int]`
carrier accepts a direct generated sequence. A Promise-bearing carrier fails
before the call AST is constructed. The compiler never inserts a cast, tests a
result, or infers this contract from `Seq` spelling.

Every standard-library or toolchain reference selects exactly one
implementation route—provider binding, compiler intrinsic, generated runtime
facet, or explicit boundary—and records its canonical object and closed use
demand at the root environment owner before its target reference is produced.
A type-only reference demands the type and representation contract; it makes
no unrelated package function executable. Calling a function or method, taking
a function or method value, reading or writing provider state, package
initialization, a demanded interface or callback capability, a certified
private provider dependency, and a demanded generated runtime facet are
executable demand. A compiler-intrinsic or generated-runtime-facet
implementation—for example `reflect.TypeFor` and `reflect.TypeOf`—is that
declaration's sole route; the dormant provider catalog entry with the same Go
identity is not selected and must not acquire a duplicate provider demand.

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
fallback. A certified binding is linkage evidence only: publication also
requires the exact used-provider closure, in which every used provider
behavior joins one certified `implemented` body and any used placeholder or
unproved profile boundary fails before target files are sealed. Publication
fails if a reachable obligation remains.

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
  const result = unix.Syscall(
    BigInt.asUintN(64, goNumberToBigInt(trap)),
    BigInt.asUintN(64, goNumberToBigInt(a1)),
    BigInt.asUintN(64, goNumberToBigInt(a2)),
    BigInt.asUintN(64, goNumberToBigInt(a3)),
  );
  return [Number(result[0]), Number(result[1]), result[2]];
}
```

This example is the product `number` profile crossing the certified provider
`bigint` scalar ABI. Under the product `bigint` profile, the same wrapper calls
`unix.Syscall(trap, a1, a2, a3)` directly. No provider argument appears in the
source-facing signature. A portable-source binding emits the same direct
wrapper call to the selected translated function and requires no provider
scalar conversion.
Missing, duplicate, stale-profile, stale-module-version, incompatible-signature,
extra, or unconsumed bindings fail before target files are sealed. The initial
linkage contract covers nonvariadic nongeneric package functions; every other
bodyless callable remains an explicit obligation until its exact source ABI is
specified rather than being widened heuristically.

The certificate records the provider package/version/digest, its exact
`@gotots/gostdlib` manifest dependency, provider integer representation, each
source identity/signature/module version, and each target export fingerprint or
portable-source identity/signature. Compiler code joins only those typed facts;
it never selects a target from package spelling, function spelling, source-file
name, or call-site heuristics. Unused entries in a certified provider catalog are
not compilation overrides. A selected matching source declaration must consume
exactly one entry or remain an explicit obligation.

### Certified Source-Callable Implementations

Measured hot paths may occur inside packages whose remaining generated code is
already the correct implementation. Copying such a package into a product
implementation would create a second package owner merely to change one body.
Instead, a project may certify one exact source-callable body replacement.

For example, given:

```go
func (c *Checker) compareNodes(left, right *Node) int {
    // source algorithm
}
```

the canonical generated declaration retains its exact ABI and placement:

```ts
import { compareNodes as compareNodesImplementation }
  from "../../../../implementations/checker-hotpaths.js";

static compareNodes(
  checker: Pointer<Checker> | undefined,
  left: Pointer<Node> | undefined,
  right: Pointer<Node> | undefined,
): int {
  return compareNodesImplementation(checker, left, right);
}
```

The product-owned module exports that exact checked callable. Generated callers
still call `Checker.compareNodes`; no caller, package assembly, class shape, or
source-visible signature changes. The original translated body is deleted, not
retained behind a runtime branch. Selection is made once from the canonical Go
declaration identity and stable signature joined to the selected build,
compilation, and load-owned selected-source snapshot. The localized canonical
body digest identifies the selected declaration but is not treated as complete
freshness evidence: changing a referenced constant, helper, selected
dependency, embedded input, non-Go input, or effective per-file Go version
invalidates the source snapshot even when this body's text is unchanged.
Emission never asks whether a package or function name appears in
configuration.

The authored module may contain bound imports, interface declarations,
type-alias declarations, and function declarations, including private helpers.
It may not execute unrelated
module initialization or bypass checked evidence. For example, this source is
invalid:

```ts
recordStartup();

export function compareNodesImplementation(
  checker: Pointer<Checker> | undefined,
  left: Pointer<Node> | undefined,
  right: Pointer<Node> | undefined,
): int {
  return "not an int" as never;
}
```

The top-level call and unchecked assertion each fail as an exact staged-source
violation before any generated or authored file is printed. Side-effect-only
imports, non-null assertions, TypeScript suppression directives, and explicit
or inferred semantic `any`/`unknown` fail at the same owner. Checked callable
values that are stored, passed, or returned are traversed through every
parameter, result, and constructor signature, so merely assigning `JSON.parse`
to a local function value does not hide its `any` result. A direct invocation
(call, construction, or tagged template) checks the callee value itself, every
authored argument, and the selected result; it
does not misclassify an unused broad parameter on an ambient overload as an
authored dynamic value (for example, `Number(exactBigInt)`). Generic
constraint/default TypeNodes are checked at their authored locations; TS-Go's
internal fallback
constraint for an unconstrained safe `T` is not misclassified as authored
`any`. Explicit `this` parameter TypeNodes are checked at the same authored
boundary; an implicit checker-only `this` fallback is not source evidence.

An explicit `kernel` variant may replace the one generated generic kernel while
leaving all finite facades and callers intact. Ordinary and kernel variants are
distinct closed identities. A selected value-receiver instance method,
deferred/recovery entry, missing target export, extra authored export,
incompatible checked signature, duplicate owner, unconsumed contract, or
package/callable ownership overlap fails before output is sealed.

### Certified Source-Package Implementations

A project-selected source implementation replaces a coherent package contract,
not isolated identifiers. It exports the exact selected package-assembly names
and satisfies every target type demanded by generated consumers. GoToTS checks
the authored project and structural replacement contract without fabricating
marker declarations. TSTS checks both the ordinary generated and installed
canonical sets with the same authoritative marker modules; the selected target
lowers and strict-typechecks both executable sets under the same final project
configuration.
Package-private storage and operation implementation may differ inside a
declared equivalence envelope;
that freedom does not extend to a source-visible callable ABI or to any type a
selected consumer requires. Callsites remain ordinary imports and calls; no
policy, bridge, digest, or implementation selector is added to a Go callable.

The checked authored signature preserves the canonical source contract. Given:

```go
// package fast
func Read(value *int) int

func Use() int {
    current := 41
    return fast.Read(&current)
}
```

the certified implementation receives the same typed location:

```ts
export function Read(value: Pointer<int> | undefined): int {
    if (value === undefined) goPanicNil();
    return loadPointer(value);
}
```

TSTS joins that authored signature to the canonical generated contract; callers
pass the pointer operation unchanged. A TypeScript target may later prove the entire flow
read-only and lower both definition and all calls to `number`. That decision is
not encoded in a source implementation signature and cannot be inferred by
GoToTS from the authored body.

For an internal hash package, Go source such as:

```go
hash := fasthash.Sum128(text)
if hash == previous { reuse() }
```

may be backed by a certified fast deterministic TypeScript hash when the
selected product proves the numeric hash is not otherwise observed. The target
still calls `Sum128(text)`, receives the exact generated result surface, and
compares through its normal equality operation. The translated original body
is absent. If the same value is printed, persisted, sent over a protocol, or
checked against specified vectors, that envelope is invalid and an exact
implementation is required. Concrete packages, implementation sources, and
product evidence belong to the consuming project, never the GoToTS
distribution.

Package replacement is selected only from the resolved project's certified
implementation set. Translation handlers never branch on import path,
package/function spelling, source location, or product identity.

The ordinary generated snapshot is certification evidence, not final
ownership. It is produced in a contract-only first session for every selected
package. The final session takes sole ownership of the deterministic canonical
name and generated-support identity registry, plus each selected source
artifact's immutable observable contract, outgoing support requirements,
accepted representation requirements, and observable dependency edges. Before
final roots emit, the accepted requirements are installed in an immutable
certified-selection ledger. They are query facts, not liveness: installation
queues no owner, schedules no declaration, and materializes no target artifact.
Thus an early `return T{}` uses `T.$fromStorage({...})` whenever the certified
ordinary contract selected storage, even when the selected-package Go body that
created the demand is absent from the final session. A later requirement batch
for that owner must be a subset of the certified set; a novel requirement
fails, while an unused certified capability remains inert. The final session
cannot read an implementation facet, builder, placement, artifact revision,
liveness ledger, scheduler queue, or generated declaration from the first
session. For example, an unexported `worker` used only by the Go body cannot
create final-session liveness merely because it appeared during certification.
A genuine final consumer of a private type requires an exact body-free private
contract module; a private value dependency remains an error. The final session
never emits a selected-package body and never infers liveness by retracting
already-emitted output.

When the final session recreates a demanded generic concretization, it joins
the transferred canonical artifact by stable generic key and exact owner,
arguments, signature, placement, and lexical anchor. A newly allocated but
semantically identical checker wrapper reuses that artifact; pointer identity
between compilation-session wrappers is never required.

Generated value support that is part of an exported declaration's observable
ABI remains public through the package assembly. In particular, an exported
non-generic interface's runtime contract and guard are package bindings, so a
cross-package assertion imports `Reader$contract` and `Reader$is` from the
same assembly that owns `Reader`. A source implementation provides and
certifies those bindings; retaining an import of its generated source file is
not an admissible private dependency.

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
