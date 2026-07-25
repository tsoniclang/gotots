# Go Translation Contract

## Governing Rule

Every expressible Go form is represented in the language catalog before the
compiler claims support for a Go version. Each occurrence is resolved from
syntax plus typed context into exactly one closed `OccurrenceResolution`;
semantically executable forms own a typed operation, while structural,
declaration, binding, and type forms own their corresponding explicit semantic
record. A translation is then selected from whole-program evidence, never from
source spelling or the first corpus that exercised it.

The default output is the TypeScript a careful human would write while
preserving Go behavior. Stronger machinery is local to the observation that
requires it.

## Language Inventory

The machine catalog is exhaustive; this table defines its required domains.

| Domain | Required construct families |
|---|---|
| Files and packages | package clauses, imports, build constraints, files, package initialization |
| Declarations | constants, variables, type definitions, aliases, functions, methods, receivers, generic parameters |
| Types | basic, named, alias, pointer, array, slice, map, channel, struct, interface, function, tuple, type parameter, union/type term |
| Literals | exact constants, composite literals, keyed/unkeyed elements, function literals |
| Names and selection | identifiers, fields, embedded selection, methods, method expressions, method values, package selection |
| Calls | ordinary calls, conversions, generic instantiation, built-ins, variadics, multi-result forwarding |
| Operators | unary, binary, comparison, logical short circuit, shifts, receive, address, dereference |
| Storage | declarations, short declarations, assignment, compound assignment, increment/decrement, blank assignment |
| Index and slice | array/slice/string/map index, comma-ok lookup, two- and three-index slicing |
| Type dynamics | interface conversion, assertion, comma-ok assertion, type switch, dynamic equality |
| Control | blocks, labels, branches, `if`, all `for` forms, range, expression/type switch, fallthrough, return |
| Effects | `defer`, `panic`, `recover`, `go`, send, receive, `select` |
| Built-ins | `append`, `cap`, `clear`, `close`, `complex`, `copy`, `delete`, `imag`, `len`, `make`, `max`, `min`, `new`, `panic`, `print`, `println`, `real`, `recover` |
| Implicit behavior | zeroing, copying, assignment conversion, receiver adjustment, promotion, boxing, initialization, evaluation sequence, panic timing |

Version-specific syntax and built-ins carry explicit minimum/maximum Go
versions. The catalog version describes compiler capability, not permission for
every input file; occurrence admission uses that file's effective language
version. Directives and toolchain extensions receive typed dispositions;
unknown directives cannot silently disappear.

`gotots inspect constructs` reports Stage-1 syntax, role, definition, and
evidence-depth records. `gotots inspect semantics` reports, by canonical
occurrence identity, every Stage-2 semantic variant, resolution, operation,
support state, and evidence. The Stage-1 command never fabricates semantic
variants after checker finalization. Adding an unrecognized concrete AST form
or semantic variant must fail its owning stage before emission.

## Context Resolution Matrix

Syntax alone is insufficient. At minimum, analysis resolves these contextual
axes:

| Syntax | Contextual distinctions |
|---|---|
| `x[y]` | array/slice/string index, map one-result lookup, map comma-ok lookup, generic index |
| `<-ch` | value receive, comma-ok receive, discarded receive, select case |
| `x.(T)` | one-result assertion, comma-ok assertion, type-switch guard |
| `F(x)` | function/method call, built-in call, conversion, generic instantiation |
| `x.M` | field, promoted field, method value, package member, generic selection |
| `T.M` | method expression, package member, instantiated selection |
| `a = b` | variable/field/index/deref store, parallel store, blank effect, named-result store |
| composite key | field name, array/slice index, map key, element expression |
| function result | zero, one, or multiple values; named storage; bare return |
| `range` | array, pointer-to-array, slice, string, map, integer, channel, iterator function |

Expected type, result arity, and role flow from parent to child. The semantic
model records the result; lowerers do not repeat this matrix.

## Declarations And Source Shape

Packages, constants, variables, functions, methods, types, aliases, fields,
embeds, generic binders, constraints, and initialization dependencies retain
canonical identity. Output preserves source names when valid and unambiguous;
deterministic mangling handles collisions and reserved target words.

TypeScript modules mirror Go source files by default. Package visibility is
implemented through module exports/imports without changing semantic
ownership. A declaration is emitted once at its owning module.

Ordinary functions remain ordinary functions:

```go
func Keys[K comparable, V any](values map[K]V) []K
```

When no special generic operation is demanded at the call, source-shaped code
remains source-shaped:

```go
maps.Keys(values.Keys())
```

not:

```ts
maps$.Keys(values$.Keys(values, zero$P, eq$P, clone$P), rt$P, key$P);
```

## Values, Storage, And Evaluation

Semantic value and storage place are distinct. Places include variables,
eligible fields, array/slice elements, and pointer dereferences. Map elements
and temporaries are never made addressable because JavaScript happens to use
objects.

Go evaluation order is explicit in the semantic model. Assignment resolves
all left places and evaluates all right values before committing stores.

```go
i, values[i] = i+1, 7
```

The original `i` selects the element location. The plan introduces only the
temporaries needed to preserve that sequence.

Structs and arrays copy at every Go copy boundary: assignment, argument and
value receiver, result, interface conversion, map operation, channel operation,
append/copy, defer capture, and external/manual boundary. The copy may be
elided only by a whole-region non-observability proof.

Bindings use the exact Stage-1 occurrence that introduces their lexical scope,
their defining occurrence when spelled, a closed role, and a stable ordinal
derived from checker order. Source spelling is metadata. File-scoped imports,
generic binders on non-executable type declarations, shadowed locals, closure
captures, range variables, type-switch variables, and named results cannot
conflate or require a fabricated implementation definition.

## Types And Representations

The semantic type descriptor always preserves:

- alias versus defined identity;
- package, owner, and generic binders;
- underlying type, width, signedness, and constraints;
- nilability, comparability, copy class, and method set;
- fields, embeds, tags, array length, channel direction, and variadic state;
- instantiated arguments and normalized interface type set; and
- dynamic-type observations.

Canonical identity is total or returns an error. It never falls back to
`Type.String()`, a source name, truncated digest, or poison marker.

Storage is selected separately:

- `boolean` for bool;
- `number` for exact 8/16/32-bit integer regions and floating values;
- `bigint` for exact 64-bit integers and width-dependent integers unless a
  complete range proof permits `number`;
- direct strings where text operations suffice, with a byte-faithful boundary
  where Go byte semantics are observable;
- ordinary objects/classes for structs;
- direct typed references, cells, or locations for pointers according to
  escape/address analysis; and
- native or small runtime collections only according to required semantics.

Arithmetic plans preserve width, overflow, division, remainder, shifts,
conversion, `float32` rounding, NaN, signed zero, and complex behavior. Untyped
constants remain exact until context selects a type.

Zero construction is type directed. Mutable aggregate zeros are fresh. There
is no universal zero value. Nil uses one typed representation per nilable
semantic kind; nil interface and interface containing a typed nil remain
distinct.

## Structs, Receivers, And Classes

Go methods should appear as TypeScript methods whenever a class/object
representation preserves the receiver contract. Ordinary receiver calls remain
ordinary:

```go
type Counter struct { Value int }

func (counter *Counter) Add(delta int) {
    counter.Value += delta
}

counter.Add(2)
```

```ts
export class Counter {
  constructor(public Value: GoInt = 0n) {}

  Add(delta: GoInt): void {
    this.Value += delta;
  }
}

counter.Add(2n);
```

The semantic method has one body and an explicit receiver binding. The atomic
method plan chooses exactly one receiver-entry form:

| Entry form | Required shape | Selected only when |
|---|---|---|
| native instance | one class method containing the body | ordinary instance invocation is exact |
| checked invocation thunk | class method owns the body; one typed function checks the explicit receiver and invokes it | nil must panic without entering the body and virtual invocation is exact |
| explicit receiver body with facade | one typed function owns the body; one class method delegates `this` to it | nil may enter the body, exact concrete selection must bypass target virtual dispatch, or an explicit-receiver manual/external contract requires it |

These are planned representations, not emitter retries. Unknown receiver or
dispatch behavior selects the explicit form or remains unsupported; it never
guesses the native form. Value-receiver copying is an orthogonal entry plan and
occurs before the body unless a complete non-observability proof permits
elision.

An ordinary proven-non-nil call remains source shaped:

```ts
counter.Add(2n);
```

When a pointer receiver must panic on nil, a checked invocation thunk preserves
Go argument evaluation before method-body panic:

```go
func (animal *Animal) Rename(name string) { animal.name = name }

animal.Rename(makeName())
```

```ts
function Animal$Rename$invoke(
  animal: Animal | undefined,
  name: string,
): void {
  if (animal === undefined) goPanicNil();
  animal.Rename(name);
}

Animal$Rename$invoke(animal, makeName());
```

Function-call argument evaluation completes before the thunk body checks the
receiver. Direct property access on a nil receiver would fail too early and is
not equivalent.

When nil is observable inside the Go body, the explicit receiver function is
the sole body owner and the class method is only the dynamic-dispatch facade:

```go
func (animal *Animal) Speak() string {
    if animal == nil { return "<nil>" }
    return animal.name
}
```

```ts
function Animal$Speak$body(animal: Animal | undefined): string {
  if (animal === undefined) return "<nil>";
  return animal.name;
}

class Animal {
  Speak(): string {
    return Animal$Speak$body(this);
  }
}
```

Exact concrete selection uses the canonical receiver body when target
inheritance could otherwise dispatch to an override; interface selection uses
the facade dynamically:

```ts
Animal$Speak$body(animal); // exact concrete Go MethodID
speaker.Speak();           // dynamic Go interface slot
```

Method expressions and method values use ordinary typed functions or lambdas
that call the planned entry and preserve receiver-capture timing. Generated
code never uses `.call`, `.apply`, `.bind`, prototype lookup, or prototype
mutation. Names of body functions, thunks, and facades derive from canonical
method identity, and every non-native artifact has typed necessity and cost
evidence. The ordinary method class must produce no thunk or body-function
indirection.

A class does not erase Go value semantics; copies still occur where Go requires
them.

Go embedding means composition, not automatically TypeScript inheritance.
Planning selects `extends` only when all of these are proven:

- one embedded storage path is the unique physical/class spine;
- promoted methods and fields preserve exact selection and override behavior;
- construction, zero value, copying, addressability, and whole-value stores
  remain exact;
- no field/method/accessor collision changes meaning;
- the hierarchy is acyclic and has one TypeScript base; and
- interface conversions and runtime type identity remain representable.

`implements` is selected from Go method-set satisfaction and adds no storage.
Other embedded structs remain composed fields, with direct promoted access or
generated forwarding only where required.

For example:

```go
type Identifier struct {
    PrimaryExpressionBase
    FlowNodeBase
    Text string
}
```

does not by spelling imply two base classes. If analysis proves
`PrimaryExpressionBase` is the physical spine and `FlowNodeBase` contributes a
behavioral contract, a possible plan is:

```ts
export class Identifier extends PrimaryExpressionBase implements FlowNode {
  constructor(
    readonly flowNodeBase: FlowNodeBase,
    public Text: string = "",
  ) {
    super();
  }
}
```

If either proof fails, both embeds use composition. No project-specific list
chooses the spine.

## Interfaces And Dynamic Types

Method-set membership and method identity come from `go/types`, including
package-qualified unexported methods and generic-owner binders. A method slot
uses the complete canonical identity. Bare method names never select dispatch.
When distinct Go methods cannot share one TypeScript property spelling, the
plan allocates deterministic canonical properties once and every declaration,
call, method value/expression, adapter, and interface slot consumes those
properties directly.

Planning uses this preference order:

1. devirtualize when the target is statically unique;
2. use direct TypeScript interface/class virtual dispatch when the value can
   carry the required behavior exactly;
3. create one typed adapter at the interface-conversion boundary when Go value
   semantics, dynamic type, equality, or assertion metadata requires it; or
4. remain explicitly unsupported until an exact static representation exists.

Per-call exhaustive switches over all implementers are prohibited. A call
through an interface with 213 implementers must still be O(1) source shape:

```go
func (node *Node) Name() *DeclarationName {
    return node.data.Name()
}
```

```ts
Name(): DeclarationName | undefined {
  return goNilCheck(this.data).Name();
}
```

It must not become 213 repeated cases. Correlation between a payload and its
methods is established once by a class or typed conversion adapter, never
recovered from `unknown` at each use.

Interface representation preserves nil interface, typed nil, dynamic defined
type, assertion diagnostics, type switches, comparability, equality panic for
uncomparable dynamic values, and finite dispatch edges. Interface equality and
assertions use canonical runtime metadata owned once per concrete type or
conversion edge.

## Generics

Go type parameters, constraints, inference, instantiations, and exact operation
demands remain explicit. The planner chooses in order:

1. direct TypeScript generics when the body needs no unavailable operation;
2. operations already owned by the selected value/container/class
   representation;
3. a concrete operation at a statically known leaf call site;
4. shared factoring that does not add operation arguments to ordinary calls;
5. one deterministic specialization for the smallest necessary binding family;
   or
6. unsupported/manual completion.

Generic functions do not unconditionally receive zero/equality/copy/key/RTTI
dictionaries. A specialized callee keeps the source argument list:

```ts
maps.Keys$Point(points)
```

not:

```ts
maps.Keys(points, zeroPoint, equalPoint, clonePoint, keyPoint, pointRTTI);
```

Specialization identity includes declaration, type/value bindings,
representation plan, and demanded operations. It is emitted once. Large bodies
cannot be cloned indiscriminately; specialization cost is part of planning.

## Collections, Strings, And Pointers

Arrays preserve length, copying, bounds, equality, and range-copy timing.
Slices preserve nil, length, capacity, backing storage, reslicing, append,
copy, clear, and overlapping aliases. A native array is used when sufficient;
otherwise a small typed slice header owns only the missing semantics.

Maps use native `Map` only for key classes whose JavaScript identity/equality
is Go-exact. Structural keys use one canonical encoded/equality representation
owned by the map plan, not key helpers threaded through each operation. Lookup
provides one-result zero behavior and comma-ok behavior distinctly. Range
honors Go's permitted outcome set.

Strings preserve immutable Go bytes, byte indexing/slicing, UTF-8 rune range,
invalid UTF-8, and conversions. Text-only regions may use direct JavaScript
strings. Crossing into byte-sensitive behavior uses one explicit branded
boundary, not pervasive wrappers.

Pointer plans escalate from eliminated location, to direct aggregate identity,
to stable cell, to typed field/element location. Address-taking points to
storage, not a value snapshot. Pointer equality observes Go storage identity;
field-name strings and generic load/store closures are not pointer identity.

## Calls, Functions, And Closures

Calls preserve callee/argument evaluation, receiver adjustment, nil-function
panic, variadics, multi-results, generic instantiation, and deferred capture.
Multiple results use one readonly tuple ABI. A multi-result call may fill a
variadic tail only according to Go's call rules.

Function values and method values retain exact receiver capture and nil
behavior. Closures capture binding/storage identities, not source names. An
escaping captured scalar uses a shared cell only when required.

Each function literal receives a Stage-1 `DefinitionID` anchored to the
`FuncLit` root, one definition site, a header region containing its callable
shape, and a separate block execution boundary. Evidence anchors cover the
complete literal while header digests exclude executable bytes; invalid
identity/span relations fail evidence production. Later planning may derive one
or more concrete `ImplementationID`s, but never reuses the body span as either
identity.

## Control, Defer, Panic, And Concurrency

Direct TypeScript control flow is preferred. Normalization is local when Go
evaluation or branch semantics require it:

- complex `for` initialization/post clauses use the same declaration/store
  lowering as statements;
- `continue` still executes the post clause exactly once;
- expression/type switches evaluate their guard once;
- range behavior is selected by typed operand kind;
- labels and fallthrough retain exact targets; and
- return writes named results before deferred calls.

`defer` captures function, receiver, and arguments when executed and runs LIFO
on normal and panic exit. Simple cases use `try/finally`; a local stack appears
only when control requires it. `panic` uses a typed Go carrier distinct from
host exceptions. `recover` succeeds only in the permitted directly deferred
context.

Channels, send, receive, close, goroutines, and `select` are first-class
catalog operations. Until an exact scheduler/channel plan exists, they are
explicitly unsupported or manual; they are never approximated synchronously.
An accepted implementation must preserve blocking, close, panic, ordering,
and the allowed nondeterministic outcome set.

`reflect` is standard-library behavior but requires compiler-produced canonical
runtime metadata for every reachable reflected operation. It never inspects
JavaScript host shape. `unsafe`, assembly, and `cgo` operations receive explicit
typed environment/manual contracts or remain unsupported; their effects cannot
be inferred from names or approximated by ordinary JavaScript access.

## Packages And Initialization

Initialization follows Go dependency order, file/declaration dependency rules,
variable initialization, and lexical `init` ordering. Blank imports create
effect edges. A package exposes one idempotent initialization entry only when
needed. Module cycles are analyzed by value, type, heritage, initializer, and
function-reference edge class; function-reference cycles alone do not justify
merging files.

Package variables preserve live mutation and addressability across modules.
The reachability graph activates initialization only for reachable imports and
registrations.

## Whole-Program Fact And Plan Examples

Phase 2 records Go meaning; Phase 3 records only observations that require a
whole-program answer and then freezes the implementation choice. The same
semantic operation cannot be reinterpreted by a lowerer.

For direct reachable code:

```go
func add(left, right int32) int32 { return left + right }
func main() { println(add(1, 2)) }
func unused() {}
```

Phase 3 records the direct `main -> add` call edge, the exact `int32`
arithmetic requirement, and roots `main`. `unused` receives
`NoSelectedRootPath` and no plan. `add` and its call receive direct
function/call plans. The eventual lowering is source-shaped:

```ts
function add(left: number, right: number): number {
  return (left + right) | 0;
}
```

The width operation is present because the `int32` fact requires it; the
function call itself gains no hidden arguments.

For value copying:

```go
type Point struct{ X int32 }

func move(input Point) Point {
    output := input
    output.X++
    return output
}
```

Phase 3 records copy boundaries at parameter entry, assignment, and return;
the mutation makes aliasing observable. The atomic type/binding/operation plans
therefore require independent `Point` storage at those boundaries. If a
whole-region proof later establishes that one copy is unobservable, the plan
cites that exact proof; lowering cannot opportunistically omit a copy.

For interface dispatch:

```go
type Speaker interface{ Speak() string }
type Dog struct{}
func (Dog) Speak() string { return "dog" }
func say(value Speaker) string { return value.Speak() }
```

Phase 3 records the method slot, `Dog -> Speaker` conversion, observed dynamic
type, and call site. The call plan is one constant-size dynamic dispatch
record. Adding 200 implementers may enlarge the interface fact set and
conversion/adapter definitions, but it cannot add 200 arms to each call plan.
The eventual ordinary call remains:

```ts
return value.Speak();
```

or uses one separately planned constant-size adapter when direct class/interface
dispatch is not exact. A per-call implementer switch is never a candidate.

For Go embedding:

```go
type Base struct{ Position int }
type Identifier struct {
    Base
    Text string
}
```

The object-family facts record the physical embed, promoted fields/methods,
copy/zero/construction behavior, collisions, and override graph. `extends Base`
is selected only if the full single-spine proof passes. Otherwise the same
semantic input receives a composition plan. Neither source resemblance nor a
type name selects inheritance.

For package initialization:

```go
import _ "example.com/register"

var value = load()
func init() { publish(value) }
```

Typed package-import edges, initializer definitions, the blank-import effect,
and lexical `init` order produce one initialization graph. Import text is not
consulted after Stage 1. Only roots whose traversal reaches this package
activate its initialization plan.

## Complex Translation Example

```go
type Key struct {
    Namespace string
    ID int32
}

type Loader[T any] interface {
    Load(Key) (T, error)
}

func GetOrLoad[T any](cache map[Key]T, key Key, loader Loader[T])
    (value T, err error) {
    defer func() {
        if err != nil {
            err = fmt.Errorf("load %s/%d: %w", key.Namespace, key.ID, err)
        }
    }()
    if cached, ok := cache[key]; ok {
        return cached, nil
    }
    value, err = loader.Load(key)
    if err != nil {
        return value, err
    }
    cache[key] = value
    return
}
```

The semantic model records structural-key map operations, a typed interface
call, named-result storage, defer capture, parallel assignment, and bare
return. Whole-program facts prove whether `T` needs copy machinery and whether
`Loader<T>` is direct or adapted. A representative plan may emit:

```ts
export function GetOrLoad<T>(
  cache: GoMap<Key, T>,
  key: Key,
  loader: Loader<T> | undefined,
): readonly [T, GoError | undefined] {
  let value!: T;
  let err: GoError | undefined;
  try {
    $return: {
      const [cached, ok] = cache.lookup(key);
      if (ok) {
        value = cached;
        err = undefined;
        break $return;
      }
      [value, err] = goNilCheck(loader).Load(key);
      if (err !== undefined) break $return;
      cache.set(key, value);
    }
  } finally {
    if (err !== undefined) {
      err = fmt.Errorf("load %s/%d: %w", key.Namespace, key.ID, err);
    }
  }
  return [value, err] as const;
}
```

This shape is valid only if the plan proves the shown copy and interface
representations. Otherwise the necessary operation appears locally. The call
does not acquire generic dictionaries merely because `T` is generic.

## Unsupported Constructs

Failure to lower one construct produces a typed record containing project,
package, implementation, occurrence, construct kind, semantic variant, role,
type evidence, source span, and reason. The complete body remains analyzable
where safe so later unsupported occurrences are not hidden.

Manual mode may materialize a typed throwing placeholder for the owning body.
Normal publication remains blocked until every reachable unsupported operation
has automatic, manual, standard-library, or external ownership. There is no
fabricated zero return, swallowed error, textual approximation, or broad
fallback lowering.
