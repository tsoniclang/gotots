# Collections And Strings

## Shared Planning Rule

Arrays, slices, maps, and strings begin as semantic values with complete typed
operation records. The planner chooses ordinary TypeScript storage whenever it
can prove exact behavior over the complete value-flow region. A custom runtime
structure is a final representation level, not the starting point.

## Slice Semantics

A Go slice contains:

- nil or nonnil state;
- backing storage identity;
- offset;
- length;
- capacity; and
- element type, including defined-slice identity where relevant.

These properties always exist semantically. Output materializes only the
properties selected execution can observe.

## Slice Representation Candidates

Slice requirements are independent dimensions: readonly/mutable, nil
observable/unobservable, owner/view, capacity observable/unobservable, and
local/escaping. The planner's monotonic product lattice contains those
requirements; these emitted candidates are not a mandatory linear sequence:

- `readonly T[]`, optionally `| undefined`, for read-only direct regions;
- `T[]`, optionally `| undefined`, for simple mutable direct regions;
- scalar-expanded backing/offset/length/capacity/nil state for local views;
  and
- an escaping typed view containing exactly the required header state.

Each candidate declares the requirement sets it satisfies. The deterministic
planner chooses the least-cost satisfying candidate and records tie-breaking.
An emitted region has one representation.

### Direct Array Requirements

A direct array region requires proof that omitted slice properties cannot be
observed through:

- nil comparison or interface conversion;
- `cap` or full-slice syntax;
- reslicing with nonzero offset;
- aliases observing append reuse or reallocation;
- stores through another view;
- escaping element addresses;
- return, package-global, callback, concurrency, or unknown boundaries; and
- defined-slice methods or identity.

### Reslicing

Two-index and three-index slicing are distinct typed IR operations. Bounds are
checked before constructing the result. A local view can lower to scalar state
without allocating an object:

```ts
const viewBacking = values;
const viewOffset = low;
const viewLength = high - low;
const viewCapacity = values.length - low;
```

The capacity expression is valid only when `values` represents the complete
backing allocation. For `make([]int, 2, 8)`, local lowering allocates backing
capacity eight, separately records visible length two, and never treats the
backing array's host length as the Go slice length. An object is required only
when that header must flow as a first-class escaping value.

Conceptually, a local `make([]int, 2, 8)` region may emit one backing allocation
of eight zeroed elements plus scalar `offset = 0`, `length = 2`, and
`capacity = 8`. Source indexing checks against scalar length; backing host
length is not the Go length.

The escaping view ABI is statically typed backing storage plus offset, length,
capacity, and canonical nil state. Direct/local/view conversions are explicit
plan edges. Append reuse/reallocation, element address identity, and bounds use
that backing identity; no conversion may copy merely to simplify the ABI.

### Append

Append evaluates arguments before mutation, may reuse backing capacity, and
returns a slice header. Existing aliases retain their own length and capacity
while observing shared backing writes.

For a proven simple owner, `push` or array concatenation may be exact. For a
view-sensitive region, lowering explicitly performs capacity choice, copy, and
returned-header construction. The selected growth policy is deterministic and
recorded wherever resulting capacity is observed.

Header copies are independent even when backing storage is shared. In
`other := values; values = append(values, item)`, changing `values` length must
not change `len(other)`. Therefore `push` is eligible only when analysis proves
there is no separately observable header alias. Growth/reuse policy is fixed
whenever any alias can observe backing reuse, not merely when source calls
`cap`.

### Copy, Clear And Range

`copy` handles overlap and returns the copied count. `clear` writes element
zero values across the selected range. Slice range fixes iteration length at
entry and loads elements from backing storage during iteration.

Indexing checks bounds unless range analysis proves safety. Out-of-range host
`undefined` never substitutes for a Go panic.

## Map Semantics

A Go map has nil state, key equality/hash behavior, value zero/copy behavior,
mutation, lookup presence, and an implementation-permitted iteration order.
Map elements are not addressable.

Nil lookup, delete, clear, len, and range follow Go rules. Assignment to a nil
map panics.

## Map Representation Candidates

Candidate strategies are:

1. native `Map<K,V> | undefined` when host key identity is exact;
2. native `Map` with a statically generated exact key normalization;
3. a specialized bucket or hash representation when collisions and Go
   equality require stored original keys; and
4. unimplemented when no accepted exact strategy exists.

Candidates are selected by explicit key equality/hash/copy, nil, iteration,
and boundary predicates. Native, normalized, and specialized forms may be
incomparable; deterministic cost ordering chooses only among candidates that
satisfy the complete requirement set.

The translator does not build a universal Go map merely because Go maps have
more semantics than JavaScript maps.

### Native Key Classes

Native `Map` can be exact for proven classes such as strings, booleans,
exactly represented integers, and direct pointer identities. Values still
receive Go copy and zero handling at insertion and lookup.

Floating keys require special treatment because JavaScript considers NaN equal
to itself for map lookup while Go does not. Positive and negative zero must
remain equal and hash coherently.

Struct, array, and interface keys use recursively selected equality and hashing.
A generated collision-free key encoding is preferred when it is exact and
cheaper than a map implementation. A specialized hash table is justified only
when native identity or encoding cannot preserve reachable semantics.

The key-classifier is a checked-in recursive semantic rule, not a list of
source names. A normalized encoding is valid only when it is injective with
respect to Go equality and retains a copied original key for range. A
specialized table stores original keys in collision buckets, checks Go
equality, and uses a reviewed deterministic hash. Interface-key operations
perform the required dynamic comparability check and panic before hashing an
uncomparable payload. Float keys never use native SameValueZero lookup.

### Lookup

```go
value, ok := values[key]
```

The key is evaluated once. Presence is checked separately from the zero value.
Struct and array values are copied at lookup. A direct `get` returning host
`undefined` is insufficient when the zero value can equal or contain
`undefined`.

### Range

Go permits implementation variation in map iteration. The selected generated
policy is deterministic for reproducibility and must remain inside Go's allowed
outcome set, including deletion and insertion during range. Tests compare
required outcomes rather than an upstream runtime's incidental order unless
the selected product explicitly pins that order.

The default deterministic policy snapshots entry identities present at range
entry in map insertion order, excludes entries inserted later, and checks that
each snapshotted entry remains live before yielding it. Deletion before visit
therefore suppresses the entry; value updates are observed at visit; an entry
is yielded at most once. A representation that cannot implement this policy
within Go's permitted outcomes is not a valid candidate.

## String Semantics

A Go string is an immutable byte sequence. Source literals encode bytes,
`len` counts bytes, indexing returns a byte, slicing selects byte ranges,
comparison is byte lexicographic, and range decodes UTF-8 with Go replacement
rules.

Ordinary JavaScript string storage is preferred when the complete region can
preserve those observations through direct operations or explicit UTF-8
lowering.

## String Representation Candidates

Analysis tracks `ascii-text`, `valid-utf8-text`, and `arbitrary-bytes` as
monotonic content requirements, plus required byte/text operations and
complexity. Candidate forms are:

1. ordinary JavaScript `string` for proven text regions whose operations are
   exact directly;
2. ordinary string with statically selected UTF-8 operations and optional
   immutable cached bytes when repeated byte observation requires it;
3. owned immutable byte storage for arbitrary bytes, with explicit text
   conversion only at proven boundaries; and
4. unimplemented when an exact boundary has no accepted encoding.

GoToTS does not encode every string as bytes merely because arbitrary Go
strings may contain invalid UTF-8.

The byte-owned candidate is a localized branded readonly byte-string type over
privately owned `Uint8Array` storage. Construction copies when an input may
remain mutable; no mutable typed-array alias escapes. The brand is static
boundary identity, not a wrapper applied to ordinary text regions.

### Byte-Sensitive Example

```go
text := "💚"
size := len(text)
```

Storage may remain the ordinary JavaScript text `"💚"`; `len` lowers to a
typed UTF-8 byte-length operation and returns four. A byte-preserving
representation becomes necessary only when values such as
`string([]byte{0xff})` survive and their exact bytes are later indexed,
sliced, compared, converted, or passed through an exact boundary.

For example, slicing `"é"` at byte index one produces an invalid one-byte Go
string and therefore forces the result region to byte-owned storage. Go string
ordering is UTF-8 byte lexicographic; JavaScript UTF-16 comparison is used only
for a proven class such as ASCII where the order is identical.

### Indexing, Slicing And Range

String index and slice use byte offsets. Bounds are checked in byte space.
Slicing may produce invalid UTF-8 and therefore can escalate the result region
to byte-preserving storage.

Range yields byte indexes and Unicode code points, replacing invalid sequences
exactly as Go specifies. An ASCII fast path requires a proven region or runtime
condition whose alternate path has identical semantics; representation itself
is still statically selected.

### Conversions

- string to byte slice copies exact bytes;
- byte slice to string copies exact bytes;
- string to rune slice decodes Go UTF-8;
- rune slice to string encodes code points with Go replacement behavior;
- integer/rune to string performs Go scalar conversion; and
- external text conversion follows an explicit external contract.

Repeated conversions may be optimized only with immutability and alias proofs.
Repeated `len`, index, slice, compare, or range operations cannot introduce an
unproven linear re-encoding inside a loop. Complexity analysis either selects
ASCII/direct operations, immutable cached bytes, byte-owned storage, or marks
the class unimplemented.

## Custom Collection Necessity

Every escaping slice view, specialized hash map, or byte-preserving string
mechanism has one class-level necessity record containing:

- exact semantic requirement;
- mechanically generated member sites;
- ordinary TypeScript counterexample;
- rejected local lowering or encoding;
- oracle and mutation tests;
- allocation/code-size/runtime measurements; and
- invalidation dependencies.

Without this record the class remains direct, uses local lowering, or is marked
unimplemented.
