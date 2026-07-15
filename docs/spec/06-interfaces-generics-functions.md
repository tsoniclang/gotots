# Interfaces, Generics And Functions

## Method Sets And Calls

The typed frontend provides declared methods, promoted selections, receiver
kind, implicit address or dereference adjustment, and instantiated signature.
Lowering consumes that evidence directly.

The canonical method form is an ordinary TypeScript function with an explicit
receiver. It preserves nil-receiver calls and makes method expressions and
method values explicit. A class method or direct specialized call is an
optimization candidate only when the planner proves the complete call/value/
interface region needs no adapter and preserves:

- value-receiver copy timing;
- pointer-receiver mutation;
- nil-receiver invocation;
- method expressions and method values;
- interface satisfaction; and
- promoted method ownership.

Host prototype lookup is not semantic evidence.

## Function Values

Function values retain exact parameters, results, variadic state, and nilability.
The default output is an ordinary TypeScript function type, with
`| undefined` only when nil is reachable.

Invoking a nil function evaluates the callee and arguments in Go order and then
panics. Generated code must not leak a host `TypeError` with different timing.

## Closures

Closures capture variables rather than snapshots unless Go defines a copy
boundary. Direct JavaScript closure capture is preferred. A storage cell is
introduced only when generated restructuring, address-taking, or representation
splitting would otherwise lose shared mutable storage.

Range-variable capture follows the pinned Go language version. Escape analysis
accounts for closure lifetime and calls through retained function values.

## Method Values

Evaluating a method value captures its receiver at that point. A value receiver
is copied then; a pointer receiver retains pointer identity. The resulting
function remains callable after the original variable changes.

Method expressions retain the receiver as an explicit first parameter.

## Interface Semantics

A Go interface value is nil or contains an exact dynamic type and dynamic
value. Reachable operations may observe:

- nil interface versus typed nil;
- dynamic defined type;
- method dispatch;
- assignment to another interface;
- type assertion and type switch;
- equality and map-key hashing;
- copying of a dynamic struct/array value; and
- panic for an uncomparable dynamic value.

The semantic IR always retains these facts. Output materializes only the facts
required by the complete region.

## Interface Representation Candidates

Candidate forms are:

1. direct concrete value and direct calls when dynamic behavior is unobservable;
2. a finite generated union when all reachable dynamic types are closed and
   TypeScript can express required operations directly;
3. a typed box or adapter carrying dynamic identity when typed nil, assertions,
   interface equality, open dispatch, or heterogeneous storage requires it; and
4. unimplemented when no exact accepted representation exists.

There is no universal interface hierarchy by default.

Finite unions are valid only when every branch retains any dynamic defined-type
identity observed by the region. Two defined Go types with the same primitive
storage cannot collapse into one untagged branch when assertions, switches,
equality, hashing, or dispatch distinguish them.

## Typed Nil

```go
var pointer *User
var value any = pointer
return value == nil
```

The result is false. The interface carries dynamic type `*User` and nil data.
A direct `undefined` representation would lose this observation and therefore
must escalate this region.

## Dynamic Type Identity

Dynamic identity is a generated canonical token or statically discriminated
type, never a source-name string or reflection lookup. A box retains a concrete
typed payload in a generated subtype or finite union branch; it does not expose
an untyped universal payload to generated core.

Runtime comparison of an explicitly planned semantic type token is permitted
Go behavior. Runtime guessing from `typeof`, constructor names, property
presence, emitted representation shape, or source spelling is forbidden.

When boxing is required, the product-closed interface ABI is a generated
discriminated union of statically typed branches. Each branch carries one
canonical type token and its concrete payload; method, equality, hash, copy,
and assertion dispatch is a generated exhaustive token switch. There is no
erased `any`/`unknown` payload or runtime member lookup. External dynamic types
must join the union through a versioned external ABI contract. A genuinely
open dynamic-type set is unimplemented until a static contract exists.

For a closed interface containing only `A` and `B`, the emitted shape is
conceptually `undefined | { type: typeof typeA; value: A } |
{ type: typeof typeB; value: B }`. An assertion to `B` switches on `typeB`;
it never infers the branch because both payloads happen to use `number`.

Interface-to-interface conversion preserves the concrete dynamic identity and
Go-copy behavior while presenting the destination method set.

## Interface Equality

Equality first compares nil state, then dynamic type. Equal dynamic type must
be comparable or equality panics. Comparable values use their statically
selected Go equality. Pointer dynamic values compare storage identity; struct
and array dynamic values compare recursively.

Hashing uses the same dynamic type and value semantics. Equal interface keys
hash coherently.

## Assertions And Type Switches

Assertions use frontend-proven target type and generated dynamic-type evidence.
Comma-ok form returns the exact zero value on failure. Non-comma-ok form panics
with the accepted Go error category and information.

Type switches preserve case order, nil case behavior, interface assignability,
and the selected case-variable type.

## Interface Custom-Mechanism Gate

A box or adapter is accepted only when machine analysis identifies reachable
dynamic observations that direct values or a finite union cannot preserve. The
necessity record includes all member sites, dynamic-type closure, typed-nil
flows, assertions, equality/hash operations, and measured allocation/dispatch
cost.

## Generic Declarations

The IR retains type parameters, constraints, type sets, approximation terms,
method requirements, inference evidence, and each reachable instantiation.
Aliases and defined generic types remain distinct.

Generated TypeScript generics are preferred when they can express the same
relationships and the body uses only representation-independent operations.

## Generic Lowering Candidates

Candidate choices are:

1. an ordinary shared TypeScript generic body;
2. closed specialization for instantiations requiring different representation
   or operations;
3. a narrowly typed operation parameter for a genuinely open boundary whose
   body needs type-dependent behavior; and
4. unimplemented.

A universal capability dictionary is forbidden. A typed operation parameter
contains only operations the body actually performs, such as zero, copy,
equality, or hash.

The current selected program is analyzed as a closed product together with
manual and extension contracts. Specialization is preferred over an open
operation mechanism when the instantiation set is complete and code growth is
bounded.

The product profile commits specialization-count, generated-byte, typecheck,
and compile-time limits before planning. A specialization candidate exceeding
one is unavailable; the planner cannot raise the limit after seeing coverage.

Specialization and a typed operation parameter are incomparable strategies,
not escalation steps. Each declares the instantiation closure, supported
operations, calling convention, code-growth cost, and boundary predicates.
Deterministic cost ordering applies only after semantic satisfaction.

A typed operation parameter becomes part of the single generated static ABI
for that region. Direct calls, recursion, function values, interface methods,
manual bodies, externals, and exports either receive the same statically typed
parameter or cross one explicit generated adapter declared by the boundary
plan. There is no hidden runtime registry and no second implementation without
the parameter.

## Generic Operations

```go
func Equal[T comparable](left, right T) bool {
    return left == right
}
```

An instantiation with `int32` may use direct integer equality. An
instantiation with `any` requires interface equality and possible panic.
One shared `===` implementation is not exact.

Zero construction, copying, map keys, interface conversion, type assertions,
numeric operators, and channel/slice/map representation receive the same
instantiation-sensitive treatment.

## Specialization Identity

Every specialization records its source generic declaration, exact type
arguments, representation plan, call sites, and structural hash. Equivalent
instantiations may share a body only when all semantic operations and
representations are proven identical.

Recursive generic instantiation is discovered to a deterministic fixed point.
Unbounded or unsupported instantiation blocks the affected closure.

## Variadics And Multiple Results

Variadic calls preserve the difference between individual arguments and a
spread slice, including allocation and alias behavior. A spread slice is not
silently copied by a JavaScript rest parameter when Go observes its backing
identity.

Multiple results use one statically typed readonly tuple in source result
order and retain evaluation as one call. Another product-wide ABI requires an
explicit architecture decision and migration; per-function alternatives are
forbidden.

Discarded results do not authorize skipping evaluation or side effects.

## Reflection And Dynamic Code

Generated core does not use reflection, source-name method lookup, `eval`,
generated functions, proxies, or unchecked `any`/`unknown` to implement
interfaces, generics, or calls. A semantic class that appears to require such a
mechanism remains unimplemented until a static design exists.
