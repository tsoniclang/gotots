# Types, Values And Pointers

## Semantic Type And Runtime Storage

Go semantic type and emitted storage are separate planner inputs. The semantic
descriptor always retains defined/alias identity, underlying type, width,
nilability, copy class, comparability, method set, generic origin, and container
or pointer behavior.

Generated TypeScript carries only static distinctions required to preserve
selected execution and safe assembly boundaries. The proof manifest retains the
complete descriptor even when emitted storage is an ordinary primitive or
object.

## Defined Types And Aliases

```go
type Token int32
type Count = int32
```

`Count` is identical to `int32`. `Token` has distinct assignment,
conversion, method-set, and interface dynamic-type semantics.

The simplest generated form is the underlying TypeScript type plus canonical
semantic identity in IR. A static brand is generated only when a TypeScript
boundary must reject an assignment that Go rejects. Runtime identity is
generated only when selected interface or reflection-like language behavior
observes the dynamic defined type.

Brand introduction is permitted only at a frontend-proven Go conversion or
construction. It cannot be an unchecked type recovery mechanism.

## Numeric Types

The pinned build profile determines the width of `int`, `uint`, and
`uintptr`. Semantic types preserve width and signedness regardless of storage.

Preferred storage:

- booleans use `boolean`;
- 8-, 16-, and 32-bit integers use `number`;
- `float64` uses `number`;
- `float32` uses `number` with `Math.fround` at Go-observable rounding
  boundaries;
- exact 64-bit integers use `bigint`; and
- complex numbers use the smallest static pair representation required by
  reachable operations.

Generated scalar names such as `int32` are static aliases or brands over these
storage types, not runtime wrapper classes. Contextual TypeScript typing is
used for literals: `let count: int32 = 0` and `takeCount(0)` need no
`goInt32(0)` value wrapper.

A region of a 64-bit or profile-width integer may use `number` only after
range analysis proves exact integer representation and correct overflow at
every observable boundary. Conversion between number-backed and bigint-backed
regions is explicit and statically planned.

Integer division, remainder, shifts, wrapping, signedness, and conversion use
typed operations. JavaScript bitwise operators are used only where their
32-bit behavior exactly matches the selected Go operation.

Every nonconstant numeric operation class specifies result normalization and
panic behavior:

- signed and unsigned 8/16/32-bit results wrap to their declared width;
- 64-bit and width-selected integer results use exact-width bigint
  normalization unless range proof permits number storage;
- integer division truncates toward zero, divide-by-zero panics, and the
  signed minimum divided by `-1` follows Go overflow behavior;
- shifts use Go's unsigned shift count rules and do not inherit JavaScript's
  masked 32-bit shift count;
- conversions implement Go truncation, sign extension, and modulo width;
- each Go-observable `float32` result is rounded with `Math.fround`; and
- complex operations preserve component precision and Go division behavior.

Untyped constants remain exact frontend constants until context selects a Go
type. A direct host operator is emitted only when its operation-class proof
covers all width, overflow, NaN, infinity, negative-zero, and evaluation-order
requirements.

For example, 32-bit integer multiplication uses an exact low-32-bit operation
such as `Math.imul` before signed/unsigned normalization; ordinary JavaScript
multiplication is not exact for every operand. Out-of-range float/integer
conversion and any bit-level NaN observation follow the pinned toolchain oracle
or remain unimplemented.

For example:

```go
var value int8 = 127
value++
```

may emit direct numeric storage with width normalization:

```ts
let value: int8 = 127;
value = ((value + 1) << 24) >> 24;
```

## Numeric Callable ABI

Each generated callable boundary selects one static storage ABI per Go numeric
type. On a 64-bit profile, an exported or shared `int`/`uint` boundary defaults
to exact-width bigint unless whole-boundary range proof permits number.
Internal number-backed regions cross that boundary through an explicit proven
conversion or a statically specialized callable. They do not create a
`number | bigint` ABI with runtime representation detection.

The manifest declares the storage behind generated static aliases such as
`int`, `uint`, and `int32`. Every call, function value, interface method,
manual body, and external adapter consumes the same declared ABI.

## Zero Values

Zero construction is type-directed:

- scalar zero is the direct typed scalar;
- string zero is empty;
- pointer, function, slice, map, channel, and interface zero is typed nil;
- arrays and structs recursively contain fresh zero values;
- defined types retain semantic identity;
- type parameters use a proven specialization or narrowly typed zero operation.

No universal zero object or helper exists. A helper is generated only when the
same nontrivial zero construction is required and direct emission would be
duplicative or incorrect.

Mutable zero values are fresh where Go requires independent storage.

## Nil And Empty

Nil is a semantic state of pointers, functions, slices, maps, channels, and
interfaces. Generated core uses `undefined` as the canonical nil storage for
each statically typed nilable representation. This is not a universal nil
value: operations remain selected by semantic kind, and nonnil payload shapes
remain distinct.

Nil tests are typed operations. A nil interface is `undefined`; a nonnil
interface containing a typed nil carries its dynamic type plus an
`undefined` payload. External, manual, and extension ABIs either use this
encoding or declare one explicit generated boundary adapter. `null`, empty
containers, zero numeric values, and host absence never silently become nil.

Nil and nonnil empty containers may be normalized only when fixed-point
analysis proves every distinguishing observation absent, including:

- comparison to nil;
- interface conversion;
- mutation behavior;
- capacity and backing identity;
- external/manual/extension boundaries; and
- returned or retained aliases.

The normalization proof covers the complete value-flow region.

## Value Copy

Go structs and fixed arrays are values. Copies occur at:

- assignment and initialization;
- parameter and value-receiver passing;
- result return;
- interface boxing;
- map key/value insertion and lookup;
- channel send and receive;
- append and copy operations;
- deferred value-receiver capture; and
- external/manual/extension boundaries whose contract requires values.

Copying is recursive: nested structs and arrays copy, while pointers, slices,
maps, channels, and functions retain their respective identity/header behavior.

The planner may elide a copy only when no later observation can distinguish the
two storages. It records the eliminated copy and proof dependencies.

## Struct Representation

The default representation is an ordinary statically typed object with direct
fields. A generated class is used only when a constructor, stable prototype
method contract, or runtime type identity is required. Value semantics come
from copy lowering, not from the host representation.

```go
type Outer struct {
    Inner Point
    Shared *Point
}
```

Copying `Outer` copies `Inner` and preserves the pointer in `Shared`.
A shallow spread is valid only when recursive field analysis proves that exact
result.

Embedded fields retain declaration ownership and selection paths in IR. Output
may flatten access only when field identity, addressability, ambiguity, and
method promotion remain exact.

## Fixed Arrays

Fixed arrays retain element type and length in IR. Their simplest storage is a
native array, tuple, or typed numeric array. Construction validates length.
Copies, bounds checks, equality, range-copy timing, and addressable elements are
lowered explicitly.

A container class is not required merely to remember length; the manifest and
generated static shape may carry that evidence. A stronger representation is
introduced only for a reachable observation that direct storage cannot
preserve.

## Comparability And Equality

The frontend determines comparability:

- booleans, numeric values, strings, pointers, and channels are comparable;
- arrays are comparable when their element type is comparable;
- structs are comparable when every field type, including blank fields for
  eligibility, is comparable;
- interfaces are comparable but may panic for an uncomparable dynamic value;
- slices, maps, and functions compare only to nil; and
- type-parameter comparison requires its type set.

Generated equality is statically selected. Struct comparison follows source
field order and skips blank field values after eligibility is established.
Array comparison follows index order. Both short-circuit, which matters when a
later interface comparison could panic.

Floating NaN is unequal to itself; positive and negative zero compare equal.
Hashing used by maps is coherent with generated equality.

## Addressability

IR distinguishes values from storage locations. Addressable locations include
variables, eligible fields, array and slice elements, and pointer
indirections. Map elements and temporary values are not made addressable merely
because emitted JavaScript uses objects.

Address-of captures the storage selected at that evaluation point. A load of
that storage is not an equivalent pointer.

## Pointer Planning

Pointer representation escalates through these semantic needs:

1. **eliminated location** — a nonescaping pointer whose reads and writes can
   be rewritten directly;
2. **direct aggregate identity** — a pointer to an object-backed struct or
   array where object identity is exact;
3. **storage cell** — an escaping scalar or replaceable whole-value variable;
4. **typed location** — an escaping field or element location requiring owner
   or backing identity plus a statically selected slot/index.

The planner chooses the first sufficient form. It does not create a general
pointer wrapper for direct object pointers.

Direct aggregate identity is eligible only when whole-value stores preserve
the addressed storage object, for example by statically generated copy-in
stores, or when analysis proves no replacement occurs. If `pointer := &value`
and later assignment would rebind the emitted object, `value` requires stable
cell/storage-root lowering instead.

An escaping location is rooted in the storage that Go addresses, not in a
snapshot of its current value. If `pointer := &value.Field` and a later
whole-value assignment replaces `value`, the location remains attached to the
cell for `value` and therefore reads the replacement field. A location plan
contains a statically typed storage-root identity, specialized load/store
operations, and a canonical slot or absolute element index.

Typed field locations use generated slot identities or specialized accessors,
never field-name strings. Repeated evaluations of `&value.field` compare as
the same storage location even if temporary location records differ.

Location equality and hashing use `(storage-root identity, canonical slot)`;
they never use temporary wrapper identity. Specialized closure accessors may
implement load/store, but a universal reflective path object is forbidden.

For an escaping `&value.Field` where `value` is replaceable, a specialized
location may close over `valueStorage` and generated slot `0`; whole-value
assignment updates `valueStorage.current`, and the location loads
`valueStorage.current.Field`. It must not capture the old field value or use
the text `"Field"` as semantic identity.

Slice element pointers use backing identity plus absolute index. Array element
pointers use array storage plus index. Pointer equality and pointer map keys use
storage identity, not wrapper-object allocation identity.

## Nil Pointer Behavior

Nil dereference panics at the Go operation point after required operand
evaluation. Pointer-receiver method calls preserve Go's distinction between
calling a method with a nil receiver and dereferencing the receiver inside the
method. A JavaScript property-access failure is not an acceptable substitute.

## Custom Pointer Necessity

A first-class typed-location mechanism is accepted only when reachable analysis
finds an escaping non-object storage address whose identity, mutation, equality,
or hashing is observed. The necessity record lists every mechanically assigned
site and demonstrates why elimination, direct object identity, or a storage
cell is insufficient.
