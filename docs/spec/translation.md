# Contextual Translation Contract

## Total Construct Dispatch

The emitter has one closed dispatcher over supported Go AST construct kinds.
Every selected node reaches exactly one handler or one typed unsupported
diagnostic. The construct inventory is a coverage table for handlers and tests;
it is not a per-program artifact and is never passed through compilation.

Handlers are grouped by semantic family rather than generated as a framework
per AST type. A handler must still explicitly visit every source child in Go
evaluation order and pass the child's role.

## Result Shapes

The small result algebra contains target nodes only:

```text
ExpressionEmission
  before: ordered TS-Go statements constrained to this execution point
  value:  one TS-Go expression
  requests: imports/declarations/helpers with explicit placement policy

StatementEmission
  statements: ordered TS-Go statements
  requests: placement requests

DeclarationEmission
  declarations: ordered TS-Go declarations
  requests: placement requests
```

These wrappers coordinate insertion and evaluation order. They do not encode
source semantics independently of the Go AST/type graph.

The parent consumes a child result. It may place `before` statements only where
the child is guaranteed to execute. For lazy or conditional children, the
parent creates the necessary TS-Go branch/block structure instead of moving
work outside that condition.

## Context-Dependent Meaning

The same syntax node can mean different things because of its parent and type
evidence.

### Map Index

```go
value := values[key]
value, ok := values[key]
```

The `IndexExpr` node is similar in both lines. The assignment parent supplies
expected result arity. `go/types` proves that `values` is a map. The handler
therefore emits either one value lookup or a two-result lookup:

```ts
const value = values.get(key);
const [value, ok] = values.getOk(key);
```

No child scans its parent, and no inventory record is needed.

### Multiple Results And Assignment Order

```go
i, values[i] = i+1, pair()
```

The assignment handler sees two target places and the complete right side. It
queries result arity, evaluates target addresses and right values in Go order,
then emits TS-Go temporary declarations and stores:

```ts
const oldIndex = i;
const nextI = i + 1;
const pairResult = pair();
i = nextI;
values[oldIndex] = pairResult;
```

The exact representation of `pairResult` follows the selected multi-result
rule. The important ownership is that the assignment handler—not each child—
controls the transaction.

### Short-Circuit Placement

```go
if ready() && consume(next()) {
    report()
}
```

If translating `next()` needs target statements, those statements cannot be
hoisted before `ready()`: Go executes them only when `ready()` is true. The
logical-expression handler emits a TS-Go conditional/block shape that places
the statements inside the right-hand execution boundary.

The same rule applies to call arguments. For `f(first(), second())`, if
translating `second()` requires prerequisite statements, the call handler first
captures the result of `first()` and only then emits those prerequisites. It
does not reorder both argument translations ahead of execution.

### Function Literal Hoisting

```go
register(func(value int) int { return value + offset })
```

The literal may remain an inline TS-Go function expression. If a representation
rule requires a reusable static declaration, the handler requests file scope
and emits a reference at the call site. The placement service chooses the
preferred legal scope; the child does not mutate an arbitrary ancestor.

## Calls

The call handler uses `go/types` to distinguish:

- ordinary function calls;
- built-ins such as `append`, `make`, `new`, `len`, and `close`;
- conversions;
- generic instantiations;
- direct receiver calls;
- interface dispatch;
- method expressions and method values; and
- variadic expansion.

Ordinary calls remain source-shaped whenever exact:

```go
result := add(left, right)
```

```ts
const result = add(left, right);
```

Hidden operation arguments are not a default protocol. A runtime or generated
helper is allowed only for a Go behavior that direct TypeScript cannot express,
and its necessity and cost must be proved for the whole semantic class.

Generated code must not use `.call`, `.apply`, or `.bind`. If an explicit
receiver entry is required, emit a named typed function:

```ts
function Counter_Add(receiver: Counter, delta: GoInt): void {
  receiver.Value += delta;
}
```

Calls then invoke `Counter_Add(counter, delta)` directly.

## Structs, Receivers, And Classes

Classes are preferred when they preserve Go behavior directly, but Go
embedding is not TypeScript subtyping.

```go
type Counter struct { Value int }
func (counter *Counter) Add(delta int) { counter.Value += delta }
```

may emit:

```ts
class Counter {
  Value: GoInt = 0n;
  Add(delta: GoInt): void {
    this.Value += delta;
  }
}
```

The emitter chooses among a native member, a named checked entry, and a named
explicit-receiver body based on nil behavior, value-receiver copying, method
selection, method values, and interface use. The body has one owner.

This Go program demonstrates why embedding cannot blindly become `extends`:

```go
type Base struct{}
func (*Base) Name() string { return "base" }
func (base *Base) Call() string { return base.Name() }

type Derived struct{ Base }
func (*Derived) Name() string { return "derived" }
```

Calling the promoted `Derived.Call` executes `Base.Name`, because the call in
`Base.Call` is statically selected. Naive TypeScript virtual overriding would
execute `Derived.Name` and is wrong. An exact direct form is:

```ts
function Base_Name(_base: Base): string { return "base"; }
function Base_Call(base: Base): string { return Base_Name(base); }
```

`extends` is selected only when every relevant field, construction, copy,
promotion, method-selection, equality, nil, and runtime-type rule remains
equivalent. Otherwise composition and owner-specific receiver functions are
used. `implements` states a target contract; it never supplies storage or
behavior.

## Interfaces

Interface representation must preserve:

- nil interface versus interface containing a typed nil;
- dynamic type identity;
- exact method-set membership and selection;
- type assertions and type switches;
- equality and comparability; and
- boxing/copy behavior.

Dispatch is O(1) per call. Emitting one switch arm per known implementer at
every call is forbidden, even if it typechecks. A large interface must not turn
one Go call into hundreds of TypeScript lines.

Native class dispatch is allowed only for genuine virtual interface methods.
Ordinary concrete method selection remains owner-specific. Representation
decisions query the authoritative Go type graph directly and create target
declarations immediately; they are not stored in a later-consumed plan.

## Values, Control Flow, And Implicit Semantics

Handlers preserve:

- Go evaluation order and parallel assignment;
- zero values and fresh mutable aggregate zeros;
- value copying at assignment, argument, result, receiver, interface, map,
  channel, and append/copy boundaries;
- nil and panic behavior;
- integer width, overflow, shifts, conversion, `float32`, complex, and exact
  untyped constants;
- `defer`, `panic`, `recover`, `go`, `select`, channels, and package
  initialization;
- labels, `goto`, fallthrough, range variants, and termination; and
- generic constraints, instantiation, type inference, and operations.

An implicit Go operation has no separate source IR node. The handler that owns
the containing construct queries type evidence and emits the required TS-Go
AST directly. Shared runtime calls are permitted only when they are the
smallest exact reusable behavior.

Package initialization is emitted from the checker graph's authoritative
initializer order. Declarations still belong to their source modules; one
package-initialization target builder receives the ordered initializer
statements. The emitter does not rebuild an initialization graph.

## Packages, Standard Library, And Externals

Go does not assign different language semantics to standard and third-party
imports. The loader records resolved provenance separately from translation.

```go
fmt.Println("hello")
```

The call handler resolves the exact selected `fmt.Println` object. Output
routing then binds that declaration to the reusable `gostdlib/fmt` contract.
The call itself is emitted like any other typed call; no source spelling
special-case exists.

For unavailable behavior, output contains an exact declaration and a throwing
placeholder at the declaration/body owner. Reachable placeholders block
publication. Manual completion replaces bodies or declarations through
structural TS-Go AST ownership, never textual patches or per-file ownership.

## Failure

The emitter fails with typed diagnostics for:

- an unhandled Go construct or contextual variant;
- absent/incoherent type evidence;
- an illegal or unsatisfied placement request;
- a representation with no exact static TypeScript form;
- target-schema drift;
- duplicate declaration/helper ownership; or
- an unresolved reachable manual/external obligation.

It never retries through a weaker path.
