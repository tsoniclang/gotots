# Contextual Translation Contract

## Total Construct Dispatch

The emitter has one closed dispatcher over supported Go AST construct kinds.
Every dispatch request reaches exactly one handler or one typed unsupported
diagnostic. The test-only construct catalog accounts separately for syntax
consumed directly by its parent owner. It is not a per-program artifact and is
never passed through compilation.

Handlers are grouped by semantic family rather than generated as a framework
per AST type. Category dispatchers route one requested node and never recurse.
The selected handler explicitly emits meaningful direct children in Go
evaluation order and supplies each child's role.

## Closed Child Contracts

Production translation does not use the conventional automatic visitor
pattern. `ast.Walk`, `ast.Inspect`, generated visitors, and equivalent generic
recursion are not emission mechanisms. The production components are:

- the root `Emitter`;
- declaration, statement, expression, and type `Dispatcher` functions;
- semantic-owner `Handler` packages; and
- the narrow `ChildEmitter` callback implemented by the root.

Each handler has a closed child contract. It uses narrow contextual entry
points such as value expression, condition, store target, type expression,
callee, statement, and block. Those entry points are not interchangeable.

For example, the `if` owner handles:

```text
Init -> optional statement
Cond -> condition expression
Body -> block
Else -> absent, block, or nested if
```

Although `IfStmt.Else` has static type `ast.Stmt`, the handler must not route an
arbitrary statement through the general statement dispatcher. A return
statement in that field is a malformed AST, not an alternative translation.

The `for` owner similarly uses grammar-specific child entries:

```text
Init -> for initializer, not an arbitrary target statement
Cond -> condition expression
Post -> legal post statement rendered as a target expression
Body -> block entered with an explicit loop-control capability
```

An unlabeled `break` or `continue` is accepted only while that capability is
present. A function boundary clears it, so a nested function cannot branch to
an enclosing loop. The branch handler neither walks to a parent nor infers a
target from source spelling. Labeled control flow will add an identity-keyed
target capability at its own construct boundary; it must not weaken this rule.

An `if` initializer is not flattened into the surrounding target block. Its Go
bindings are scoped to the condition, both branches, and nested `else if`
chain, so the owner emits one target block containing the initializer
statements followed by the target `if`:

```go
if current := value; current < limit {
	use(current)
}
```

```ts
{
  let current: GoInt = value;
  if (current < limit) {
    use(current);
  }
}
```

An `else if` is delegated through the dedicated `if` child entry, not the
general statement dispatcher. A nested initializer is therefore wrapped at
that alternate boundary rather than leaked.

The three `for` clauses are independently optional. Their absence becomes an
absent TS-Go `ForStatement` child, so `for condition {}`, `for {}`, and partial
three-clause loops remain direct `for` statements. A present initializer or
post clause still uses its narrow grammar entry, and a child with prerequisite
statements remains unsupported until those statements can be placed at the
per-iteration boundary exactly.

An expression switch owns its tag, ordered case expressions, clause bodies,
default, implicit clause scopes, and implicit breaks as one construct family:

```go
switch current := value; current {
case 0:
	branch := 10
	result = branch
case 1, 2:
	branch := 20
	result = branch
default:
	branch := 30
	result = branch
}
```

The initializer uses the same scoped-simple-statement child contract as an
`if` initializer. The owner emits a target block around the initializer and
switch so the binding does not escape. The tag is evaluated once. Case
expressions are delegated in source order with the tag's exact Go type.
Multiple expressions become adjacent target case labels sharing one body.
Each Go clause body is wrapped in its own target block because Go clauses are
implicit lexical blocks while TypeScript case labels otherwise share one
scope. The owner appends the one target `break` needed to preserve Go's
implicit non-fallthrough behavior.

Case-expression prerequisite statements, expressionless switches,
fallthrough, type switches, and types whose target equality is not yet proved
remain distinct typed-unsupported cases. They are not approximated by
re-evaluation, source spelling, loose equality, or a generic statement walk.

A fully initialized local `var` declaration is owned by its enclosing
declaration statement:

```go
var left, right int = first(), second()
```

The owner uses the `types.Var` identity for each name, verifies each initializer
against that object's exact Go type, delegates initializers left-to-right, and
constructs one typed target declaration list. The Go scope begins after the
`ValueSpec`, so an initializer that resolves to an outer same-named object must
retain that outer identity. The target name owner allocates a distinct inner
name when needed; it must not rely on TypeScript's temporal-dead-zone behavior.

A standalone Go block becomes one target block and is never flattened into its
parent. This preserves local declaration scopes directly. Grouped `var`
declarations emit their `ValueSpec` records in source order, with each spec
forming its own declaration statement and scope boundary.

Zero-initialized declarations, package declarations and initialization order,
constants, multi-result initializers, and initializer prerequisite statements
remain separate typed-unsupported cases until their complete semantic owners
are installed.

Likewise, for:

```go
value, ok = values[key]
```

the assignment owner emits `value` and `ok` as store targets and dispatches the
index expression with a two-result comma-ok context. For:

```go
value = values[key]
```

it dispatches the same AST form with a one-result value context. The child does
not rediscover its parent.

Every direct field is accounted for. Inseparable syntax is consumed explicitly
by the owner; semantically independent children are delegated exactly once;
optional absence is acknowledged; metadata belongs to a named non-semantic
owner; nested boundaries are dispatched deliberately; and impossible shapes
fail. Nothing is silently skipped or visited automatically.

## Result Shapes

The small result algebra contains target nodes only:

```text
ExpressionEmission
  before: ordered typed TS-Go protocol statements constrained to this point
  value:  one typed TS-Go protocol expression
  requests: imports/declarations/helpers with explicit placement policy

StatementEmission
  statements: ordered typed TS-Go protocol statements
  requests: placement requests

DeclarationEmission
  declarations: ordered typed TS-Go protocol declarations
  requests: placement requests
```

Narrow contextual entries whose target category is not an expression,
statement, or declaration use the same immutable `value + requests` shape
specialized to that exact TS-Go protocol category (currently type, block, and
`for` initializer). They have no independent semantic payload and no
`before` list; they do not enlarge the source model.

These wrappers coordinate insertion and evaluation order. They do not encode
source semantics independently of the Go AST/type graph.

The wrappers are immutable values. Slice accessors return copies. A handler
may reserve a deterministic target name through the name owner, but it does
not install an import, declaration, helper, or statement into a mutable parent.
The corresponding typed placement request travels in the result and is
applied once by the root placement owner.

Result composition is owner-directed:

- a block flattens each child `StatementEmission.statements` in source order;
- a parent combines child requests without rendering or string-key
  deduplication;
- a direct expression normally has an empty `before` list;
- when a later eager child has `before` statements, already encountered
  side-effecting values are captured before those statements so source
  evaluation order is retained; and
- a lazy child keeps its `before` statements inside the branch, short-circuit
  arm, loop iteration, or closure that owns its execution.

No compatibility entry point may unwrap an expression and discard its
`before` statements or placement requests. Until an owner implements the
required composition rule, that contextual case remains an explicit
unsupported failure.

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

Even an identifier-only swap requires transactional ownership:

```go
left, right = right, left
```

An exact direct form captures every right side before the first store:

```ts
const __gotots_assign_0: GoInt = right;
const __gotots_assign_1: GoInt = left;
left = __gotots_assign_0;
right = __gotots_assign_1;
```

The captures are target coordination nodes, not a source IR. Their names and
types come from the lexical name owner and the existing `go/types` objects.
For a mixed short declaration, existing variables are stored and new variables
are declared only after all captures. A shadowing new variable receives a
distinct target name so an earlier right-side reference still denotes the
outer Go object.

```go
i, values[i] = i+1, pair()
```

The assignment handler sees two target places and the complete right side. It
queries result arity, evaluates target addresses and right values in Go order,
then creates typed TS-Go protocol temporary declarations and stores:

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
logical-expression handler creates a TS-Go conditional/block shape that places
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

A zero-result Go function emits an explicit `void` result type, and a call used
as a statement remains a direct call expression statement:

```go
func Touch(value int) {
	if value > 0 {
		return
	}
}

Touch(value)
```

```ts
function Touch(value: GoInt): void {
  if (value > 0) {
    return;
  }
}

Touch(value);
```

The function owner derives zero results from the selected `types.Signature`;
the return owner accepts a bare return only in that function context; and the
expression-statement owner admits only a toolchain-valid discarded call case
that its call owner can represent. Calls returning a supported value may also
be discarded. A receive statement, multi-result call, `go`, or `defer` remains
with its own semantic owner until that complete case is implemented.

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

The basic-type owner selects integer width from the loaded `types.Sizes`
evidence and requests the corresponding type-only Tsonic primitive import:
`int32` or `int64` from `@tsonic/core/types.js`. The file placement owner
deduplicates and renders that request before declarations. A handler does not
infer width from `GOARCH` spelling, emit `number`, or call the JavaScript
`BigInt` global. Integer operations remain direct only where the selected
primitive/target contract preserves the Go operation; otherwise their shared
semantic owner must request an explicit typed runtime operation.

The imported Tsonic primitive and its TypeScript checker carrier are different
facts. For example, a selected Tsonic consumer may expose `int64` through a
virtual declaration whose checker shape is `number` while attaching the target
fact that makes C# emission use `long`. GoToTS emits and retains the canonical
`int64` reference. It must not infer semantics from the virtual declaration's
structural carrier, replace the import with `number`, or introduce `bigint`
syntax that the selected Tsonic contract does not admit.

Literal and operator handlers must also preserve exact target evidence. A
large Go integer constant cannot be passed through a JavaScript numeric-value
round trip if that changes its source digits. The handler either constructs an
exact Tsonic-admitted target expression with bounded size or fails typed until
the selected consumer contract supplies one. Rounded numeric literals,
ordinary-value-only claims, and test declarations that pretend `int64` is
`bigint` are forbidden.

For the selected Tsonic contract, the signed-integer literal owner reads the
exact `go/constant.Value` and the expected Go type supplied by its parent. A
small value is explicitly attributed to the canonical target primitive:

```ts
42 as int64
```

A value outside JavaScript's exact integer range is split into at most two
base-`2^32` chunks, each still exactly representable:

```ts
(2147483647 as int64) * (4294967296 as int64)
    + (4294967295 as int64)
```

That expression denotes Go's `9223372036854775807` without ever materializing
an inexact JavaScript numeric literal. Negative constants use typed subtraction
rather than a target-ambiguous unary minus; `MinInt64` is:

```ts
((0 as int64) - (2147483647 as int64) - (1 as int64))
    * (4294967296 as int64)
```

These `as int64` nodes are canonical Tsonic source-primitive evidence selected
from the authoritative expected Go type. They are not unchecked casts,
structural recovery, or substitutes for a missing semantic fact. The
representation is constant-size for every signed 64-bit value, introduces no
helper or runtime call, and must be compiled and executed through the selected
target because direct Node execution cannot verify its result.

An implicit Go operation has no separate source IR node. The handler that owns
the containing construct queries type evidence and creates the required typed
TS-Go protocol AST directly. Shared runtime calls are permitted only when they
are the smallest exact reusable behavior.

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
structural typed TS-Go protocol ownership, never textual patches or per-file
ownership.

## Failure

The emitter fails with typed diagnostics for:

- an unhandled Go construct or contextual variant;
- a child shape forbidden by its owner's closed child contract;
- absent/incoherent type evidence;
- an illegal or unsatisfied placement request;
- a representation with no exact static TypeScript form;
- target-schema drift;
- duplicate declaration/helper ownership; or
- an unresolved reachable manual/external obligation.

It never retries through a weaker path.
