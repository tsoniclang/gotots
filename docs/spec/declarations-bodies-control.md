# Declarations, Bodies And Control

## Declaration Translation

GoToTS translates selected packages, constants, variables, functions, methods,
defined types, aliases, structs, interfaces, arrays, maps, slices, channels,
function types, and generic declarations from typed frontend evidence.

Generated declarations preserve:

- exported and package-local visibility needed by the assembled program;
- canonical ownership and source maps;
- receiver and method-set semantics;
- parameter and result order, names where semantically useful, and variadic
  state;
- generic parameters, constraints, and instantiations;
- field order, embedding, tags, and blank-field semantics;
- alias versus defined-type identity; and
- initializer and package-order dependencies.

Generated names are deterministic valid TypeScript identifiers. Source names
are preserved when valid and collision-free. Mangling is deterministic and
recorded. Source names never select runtime behavior.

## Signatures

Function, method, class-like, interface, constraint, constant, variable, and
type shapes are captured completely. Multiple results use the product-wide
readonly tuple ABI defined in `06-interfaces-generics-functions.md`. Variadic
parameters remain distinguishable from ordinary slices.

An external or manual implementation never changes a generated signature.
Signature disagreement blocks assembly.

## Constants And Literals

Untyped Go constants remain arbitrary-precision semantic values until a typed
context or required operation selects representation. The emitter uses direct
literals when context preserves the selected type:

```ts
let count: int32 = 0;
consumeInt32(0);
return 0;
```

It does not wrap literals merely to restate context. Conversion or normalization
appears only when Go performs a real conversion or an observable width,
rounding, or representation boundary requires it.

## Variable Storage

Short declarations, ordinary declarations, package variables, named results,
range variables, and closure captures create explicit semantic storage.
Storage may lower to a direct TypeScript variable unless address-taking,
capture, aliasing, initialization, or boundary behavior requires a stronger
form.

Package variables exposed across generated modules use live bindings or
generated accessors that preserve Go mutation and address semantics.

## Calls

Calls preserve:

- callee and argument evaluation order;
- method receiver adjustment and copying;
- variadic construction and spread behavior;
- generic instantiation;
- multiple results;
- nil-function panic timing;
- deferred receiver and argument capture; and
- external/manual/extension effects.

Typed call identity comes from the frontend. There is no emitted-name lookup,
arity guess, or argument-shape dispatch.

## Assignment

Assignment evaluates left storage locations and right values according to Go
rules before committing stores. Parallel and compound assignments use
temporaries where necessary.

```go
i, values[i] = i+1, 7
```

The original `i` selects the element storage before either store. Generated
code must not serialize the two assignments in a way that changes that
selection.

Struct and array values copy at assignment boundaries. Slice, map, pointer,
channel, function, and interface values follow their semantic copy rules.

## Branches And Loops

Ordinary `if`, `for`, `switch`, `break`, and `continue` use direct
TypeScript control flow when exact. Labels, labeled branches, fallthrough, and
control crossing transformed regions use generated labeled blocks or a local
state machine.

State-machine lowering is scoped to the affected function. It is not a
program-wide runtime.

## Switches

Expression switches evaluate the switch expression once. Case expressions
retain Go evaluation order and short-circuiting. Fallthrough transfers only to
the next case body and does not re-evaluate case expressions.

Type switches evaluate the interface expression once, retain typed-nil
semantics, perform exact assignability tests, and bind the case variable with
the type selected by Go.

## Range

Range lowering is selected by typed operand class:

- array and pointer-to-array;
- slice;
- string;
- map;
- integer;
- channel; and
- range-over-function.

The pinned Go language version controls variable creation and closure-capture
semantics. Array range observes required array-copy timing. Slice range fixes
the iteration length while element loads observe backing mutations. String
range decodes Go UTF-8 semantics. Map order follows the accepted implementation
choice and Go's permitted outcome set.

Range-over-function preserves the yield callback protocol, early termination,
panic behavior, and invalid-yield diagnostics. Unsupported iterator signatures
form an explicit unimplemented class.

## Defer

The IR captures the deferred function value, receiver, and arguments when the
defer statement executes. Deferred calls run in last-in-first-out order during
every normal return and panic exit.

Structured cases use nested `try/finally`. A local defer stack is generated
only when conditional, repeated, or interacting control flow cannot be
represented directly. Functions without defer have no defer machinery.

Named result values are assigned before deferred calls run. A deferred closure
may observe and mutate them.

## Panic And Recover

Go panic may carry any Go value. Lowering preserves:

- operand evaluation before panic;
- stack unwinding through deferred calls;
- replacement of an active panic by a deferred panic;
- recover succeeding only in the permitted deferred-call context;
- nil and typed-nil panic values under the pinned Go version; and
- distinction between Go panic and external host failure.

A small boundary adapter may classify external failures according to an
explicit external contract. Generated core never catches arbitrary failures to
continue speculatively.

Generated Go panic uses one statically branded `GoPanic<P>` carrier whose
payload type is the closed selected union of panicable Go values for the
affected boundary. Host exceptions are not that carrier. Deferred lowering
tracks the active carrier and the exact direct-deferred-call recover context;
`recover` cannot succeed merely because code runs inside a generic catch.
`panic(nil)` follows the pinned Go-version oracle and never collapses into
absence of a panic.

## Function Exit And Temporaries

Generated temporaries are scoped by the semantic operation that requires them.
They use canonical IDs, cannot collide with source bindings, and do not survive
outside their necessary region.

Return lowering evaluates result expressions, performs required copies and
conversions, writes named-result storage, runs deferred calls, and returns final
values in Go order.

## Unimplemented Bodies

If one semantic statement or expression class lacks lowering, the enclosing
implementation is classified unimplemented with the exact operation record.
The compiler continues analyzing independent implementations and may emit an
exact typed throwing placeholder into the editable workspace, but emits no
retained runnable body for the affected dependency closure.

Missing automatic lowering defaults to `unimplemented`. Editing the placeholder
or another generated body is detected automatically from its generated
baseline and post-format body hash. A current complete body that passes every
applicable gate may separately change that unit's product ownership to
`accepted-manual` under `08-externals-manual-extensions.md`. No user-authored
manifest or promotion record is involved. This does not mark the automatic
semantic class implemented or inflate generated-class coverage. Automatic
class support and complete-body ownership are separate machine facts.

The body has one support state but retains one unsupported-operation record for
every unsupported site and semantic class. Finding one unsupported operation
does not stop IR construction or hide later unsupported operations when the
frontend can continue safely.

Difficulty never authorizes a textual approximation, host-language exception,
or broad custom runtime.
