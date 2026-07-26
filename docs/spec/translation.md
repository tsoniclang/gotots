# Contextual Translation Contract

## Total Construct Dispatch

The emitter has one closed dispatcher over supported Go AST construct kinds.
Every dispatch request reaches exactly one handler or one typed unsupported
diagnostic. An independent test derives and structurally fingerprints the
selected toolchain's complete AST domain. Forms that are not declarations,
expressions, or statements are parent-owned syntax; the owning handler's
focused child-contract tests account for them. There is no construct catalog
or per-program coverage artifact.

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

A local `var` declaration is owned by its enclosing declaration statement:

```go
var left, right int = first(), second()
```

The owner uses the `types.Var` identity for each name, verifies each initializer
against that object's exact Go type, delegates initializers left-to-right, and
constructs one typed target declaration list. The Go scope begins after the
`ValueSpec`, so an initializer that resolves to an outer same-named object must
retain that outer identity. The target name owner allocates a distinct inner
name when needed; it must not rely on TypeScript's temporal-dead-zone behavior.
When an explicit type has no initializer, the same value-representation owner
used by assignment supplies the exact zero value. If that type has no proved
zero representation, the declaration fails at the local-value boundary.

A standalone Go block becomes one target block and is never flattened into its
parent. This preserves local declaration scopes directly. Grouped `var`
declarations emit their `ValueSpec` records in source order, with each spec
forming its own declaration statement and scope boundary.

An explicitly typed package constant is a direct declaration owned by its
source-file module:

```go
const Base int = 40
```

```ts
export const Base: int32 = 40;
```

The owner uses the package-scope `types.Const` identity and exact constant value
while the initializer handler preserves the supported source expression shape.
Every emitted package declaration is exported for static generated-module
linking; package assembly later selects the public surface. Untyped constants,
implicit constant expressions and `iota` remain one later constant-semantics
family because their representation may depend on each use context.

Package variables are package-state fields, never source-file-local `let`
bindings. A same-package read or store uses the package's direct state import;
a qualified cross-package use routes through the dependency's passive
assembly:

```go
var Count int

func Add() { Count++ }
func Read() int { return Count }
```

```ts
import { $state } from "../../packages/<module-key>/<package>/state.js";

export function Add(): void {
  $state.Count++;
}

export function Read(): int32 {
  return $state.Count;
}
```

The package-state owner derives the field name and represented type from the
exact package-scope `types.Var`. The package-assembly owner emits one passive
`$initialize` body containing zero assignments for all reached-package
variables and then consumes the selected `go/types.Info.InitOrder` directly.
It does not recover initializers by scanning declarations or sort files.
One-to-one, multi-result, blank-target, prerequisite-producing, and
function-literal initializers are contextual cases of that same owner and are
admitted only as each complete assignment shape is proved. `init` functions
join the assembly after variable initialization in the selected toolchain
loader's file order and each file's declaration order. The emitter preserves
that order directly; it does not rescan the filesystem or sort a second file
list.

One compilation-owned `program.ts` consumes the reached `types.Package` import
graph and calls package `$initialize` functions in Go's global
import-path-sorted dependency order. Package assemblies never execute Go
initialization at module top level. Generated target import order is not a
semantic proxy for Go package order.

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
  requests: imports/declarations/helpers or declaration requirements with
            explicit placement policy

StatementEmission
  statements: ordered typed TS-Go protocol statements
  requests: placement or declaration-requirement requests

DeclarationEmission
  declarations: ordered typed TS-Go protocol declarations
  requests: placement or declaration-requirement requests
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

A declaration requirement similarly carries no mutable target node. It names
the exact declaration owner and one closed obligation. The root routes it back
to that semantic owner, which reconstructs its complete typed TS-Go declaration
assembly from the source declaration and accumulated requirement set. This
replacement occurs only in compilation-local target state before file sealing;
no handler patches a previously printed artifact.

Result composition is owner-directed:

- a block flattens each child `StatementEmission.statements` in source order;
- a parent combines child requests without rendering or string-key
  deduplication;
- a direct expression normally has an empty `before` list;
- under `preserve-go`, when a later eager child has `before` statements,
  already encountered values are captured before those statements so source
  evaluation order is retained;
- under `direct`, no capture is introduced solely because target reshaping
  evaluates otherwise-direct children in a different order; and
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

For an identifier of a directly represented signed integer type, compound
addition remains one direct target operation:

```go
total += delta
```

```ts
total += delta;
```

The assignment owner proves the exact `types.Var`, represented operand type,
and assignability of the right side. It accepts the direct form only when the
right-side emission has no prerequisite statements, so reading the left value
cannot move across right-side effects. Selector, index, pointer, other
compound-operator, unsupported-width, and prerequisite-statement cases remain
separate typed failures until their single-evaluation rules are proved.

An ordinary Go function with two or more results has one direct TypeScript
tuple carrier. Result declarations use `[T0, T1, ...]`; an explicit
`return left, right` constructs `[left, right]`; and `return pair()` preserves
the tuple-valued call directly. There is no generated result class, wrapper,
out-parameter ABI, erased carrier, or per-function representation choice.

For one multi-valued right side, the assignment owner evaluates it exactly once
into a typed tuple temporary and then performs target declarations/stores from
numeric element accesses:

```go
next, ok := pair(value)
```

```ts
const __gotots_results_0: [GoInt, GoBool] = pair(value);
let next: GoInt = __gotots_results_0[0];
let ok: GoBool = __gotots_results_0[1];
```

A blank target omits only its final declaration/store; it never omits the
source evaluation or changes tuple position. When one multi-valued call is the
complete argument list of another call, the call owner uses the same
single-evaluation rule and passes the indexed values in parameter order.
Direct generated-program verification must prove that the selected source tuple
maps to the TypeScript tuple without an alternate ABI.

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

If `pair` is single-valued, `pairResult` has that direct represented type; a
multi-valued source expression instead uses the direct tuple rule above. The
important ownership is that the assignment handler—not each child—controls the
transaction.

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
that its call owner can represent. Calls returning supported single or multiple
results may also be discarded. A receive statement, `go`, or `defer` remains
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

An ordinary named Go struct is a value type. TypeScript objects and classes are
reference values, so direct target assignment cannot represent Go copying.
GoToTS makes an observably required zero, copy, or equality operation explicit
through one representation owner; it does not assume that a later compiler,
plugin, or runtime will reinterpret ordinary TypeScript assignment.

The first supported struct family is narrower and selects one nominal record
class for non-generic, non-embedded named structs whose fields recursively
contain only `bool`, profile-represented `int32` (including a selected 32-bit
`int`), and members of the same supported struct family. A generated record
class has:

- one erased private brand, which makes distinct named Go structs nominally
  distinct to strict TypeScript without adding an instance field;
- public data fields initialized by its constructor;
- no instance receiver methods, inheritance, dynamic lookup, or hidden
  semantic payload.

For example, the base shape for a supported `Pair` is:

```ts
export class Pair {
  declare private readonly $goType: void;

  constructor(public Left: int32, public Ready: bool) {}
}
```

Operations are top-level companion declarations in the same generated
source-file module and exist only when selected code requests them:

```ts
export function Pair$zero(): Pair {
  return new Pair(0, false);
}

export function Pair$copy(source: Pair): Pair {
  return new Pair(source.Left, source.Ready);
}

export function Pair$equal(left: Pair, right: Pair): bool {
  return left.Left === right.Left && left.Ready === right.Ready;
}
```

The class supplies target nominality and stable runtime constructor identity;
it does not change Go value semantics. Distinct named structs remain statically
incompatible even when their fields match. A requested nested struct operation
requests the corresponding companion from the nested type's owner. The erased
brand must produce no JavaScript instance field.

Tags, embedding, pointers, interfaces, method values/expressions, generics,
reflection entry, and fields whose complete standalone representation is not
yet exact are neighboring typed-unsupported families. They may be admitted
only by extending or replacing this representation under their own complete
proof. They must not be approximated by structural assignment or virtual
dispatch.

Each zero, copy, or equality capability is emitted at most once and only when a
selected occurrence requests it. Its typed request is routed to the defining
source-file module even when the first use is in another file. Companion
definitions grow linearly with fields while each use remains constant size.
The first family constructs only through `new Name(...)`. Positional composites
are direct in both profiles. For keyed composites, `direct` emits constructor
arguments directly in declaration order, while `preserve-go` captures values
in keyed source order before consuming them in declaration order. Omitted
fields request the same zero owner rather than a second default table.

Generated source modules use a caller-owns-value convention. A borrowed struct
expression (for example, a local identifier or field selection) is copied
exactly when it crosses an initialization, argument, receiver, or composite
field boundary. A fresh composite literal or function result already owns
fresh storage and transfers that ownership without another copy. A supported
single-result return transfers the function's owned local value. Because
pointers, shared containers, closures over struct storage, and package
variables are outside this first family, no admitted path can retain an alias
to that transferred storage.

Module-level `export` exists for deterministic generated-module linking; it is
not by itself a JavaScript FFI contract. A future explicitly selected external
entry adapter must copy incoming struct values once at that true external
boundary. Generated internal calls must not pay a second callee-prologue copy,
grow a wrapper per call, or carry a hidden ownership flag.

Initialization, argument passing, return, value receivers, interface boxing,
map stores, channel sends, and append/copy operations each request copying
where Go requires it. Assignment to an ordinary local or field in the initial
pointer-free family rebinds that target to a copied value:

```ts
target = Pair$copy(value);
```

No admitted construct can observe the replaced target object's identity.
Destination-preserving mutation is therefore not emitted speculatively. A
future pointer/addressable-storage family must install its exact storage owner
when such identity first becomes observable.

For example:

```go
copy := value
copy.Ready = false
```

```ts
let copy = Pair$copy(value);
copy.Ready = false;
```

Emitting `let copy = value` is forbidden because it aliases the same object.
Every representation capability requires strict typechecking and direct
Go-versus-generated-ESM differential execution. If zeroing, copying, equality,
or field representation is unproved, that struct case remains unsupported.

Receiver bodies still have one owner. A value receiver may emit as an explicit
typed receiver entry that requests the same copy owner:

```go
type Flag struct{ Ready bool }
func (flag Flag) Disable() { flag.Ready = false }
```

```ts
export function Flag_Disable(flag: Flag): void {
  flag.Ready = false;
}
```

Concrete generated calls invoke `Flag_Disable(Flag$copy(flag))` directly, so
the selected value receiver owns exactly one copy before its body runs. Pointer
receivers, method values, and interface calls select their own exact checked
entry shape from `go/types`; they do not reuse a value-receiver entry when its
copy or nil behavior differs. Generated code never uses `.call`, `.apply`, or
`.bind`.

A TypeScript class outside this bounded nominal-record family is selected only
when reference identity and all relevant Go call, nil, copy, promotion,
equality, construction, and runtime-type rules remain exact. Its constructor
and value operations are side-effect-free generated behavior rather than an
assumption that class assignment copies. A class is never selected merely to
attach receiver syntax.

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
equivalent. Otherwise structural fields, explicit copy/zero owners, composition,
and owner-specific receiver functions are used. `implements` states a target
contract; it never supplies storage or behavior.

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

Handlers preserve within the selected profile:

- exact Go evaluation order when `preserve-go` is selected; `direct` accepts
  target order only where representation reshaping reorders direct
  expressions;
- parallel assignment and other atomic multi-value operations under both
  evaluation-order selections;
- zero values and fresh mutable aggregate zeros;
- value copying at assignment, argument, result, receiver, interface, map,
  channel, and append/copy boundaries;
- nil and panic behavior;
- selected integer carriers, explicit conversions, shifts, `float32`, complex,
  and exact untyped constants within the declared integer-representation axis;
- `defer`, `panic`, `recover`, `go`, `select`, channels, and package
  initialization;
- labels, `goto`, fallthrough, range variants, and termination; and
- generic constraints, instantiation, type inference, and operations.

The basic-type owner selects integer width from the loaded `types.Sizes`
evidence and requests the corresponding alias from the generated
`support/scalars.ts` module. One immutable compilation-wide profile chooses
their carrier. The default is:

```ts
export type bool = boolean;
export type int32 = number;
export type int64 = number;
```

The CLI override emits:

```ts
export type int32 = bigint;
export type int64 = bigint;
```

Each alias is emitted once as a typed TS-Go `TypeAliasDeclaration`; generated
package modules use canonical relative type-only imports. The aliases retain
the selected Go representation name in generated source. A handler does not
infer width from `GOARCH` spelling or independently inspect configuration.

Ordinary integer syntax is source-shaped under both initial profiles:

| Go source | TS-Go AST decision | Printed TypeScript |
|---|---|---|
| `left + right` | direct addition | `left + right` |
| `left - right` | direct subtraction | `left - right` |
| `left * right` | direct multiplication | `left * right` |
| `value += delta` | direct compound assignment | `value += delta` |
| `value++` / `value--` | direct update | `value++` / `value--` |
| ordered/equality comparison or expression switch | direct comparison/switch | `left < right`, `switch (value)` |

The `number` profile prints ordinary numeric literals such as `1`; the
`bigint` profile prints `1n`. Contextual parameter, result, field, and
package-boundary binding types carry aliases. A literal is not routinely
wrapped in `as int32`, `as int64`, or `as bool`, and an initialized local
binding omits a type annotation when its initializer already makes the target
type exact.

Neither initial profile reproduces implicit fixed-width overflow. The default
also accepts JavaScript-number precision as its declared integer contract.
The BigInt override removes that precision limitation but still does not
implicitly narrow after arithmetic. Evidence names the profile and never
claims these deferred semantics. Explicit narrowing conversions and any future
fixed-width profile are separate construct families rather than baggage in
ordinary multiplication.

Boolean `&&` and `||` emit direct binary expressions and retain native
short-circuit evaluation. Neither operand may carry prerequisite statements:
moving such work before a short-circuit operator would change behavior.

An untyped Go boolean constant emits as the direct TypeScript boolean literal.
The parent still supplies the expected Go `bool`, the predeclared-constant
identifier handler verifies semantic object identity and assignability through
`go/types`, and the basic-type owner supplies the one generated `bool` alias
where a declaration boundary needs it. No operator handler guesses a carrier
from spelling.

An explicit Go parenthesized expression becomes one TS-Go
`ParenthesizedExpression` around the directly emitted child. It preserves
source grouping without creating a source-side wrapper or intermediate
expression model.

Division, remainder, shifts, bitwise operators, and explicit conversions remain
separate construct families until their profile-specific behavior is admitted.

Literal and operator handlers must also preserve exact source evidence. A
large Go integer constant cannot be passed through a JavaScript numeric-value
round trip if that changes its source digits. The handler either constructs an
exact standalone target expression with bounded size or fails typed until a
GoToTS-owned exact representation exists. Rounded numeric literals and
ordinary-value-only claims are forbidden.

The signed-integer literal owner reads the exact `go/constant.Value` and the
expected Go type supplied by its parent. It emits a decimal TS-Go numeric
literal in the `number` profile or a decimal TS-Go BigInt literal in the
`bigint` profile. A target-width mismatch remains unsupported. A number-profile
constant outside JavaScript's exact integer range also remains unsupported;
the user selects the BigInt profile rather than receiving a decomposed
approximation.

An implicit Go operation has no separate source IR node. The handler that owns
the containing construct queries type evidence and creates the required typed
TS-Go protocol AST directly. Shared runtime calls are permitted only when they
are the smallest exact reusable behavior.

Package initialization is emitted from the checker graph's authoritative
initializer order. Immutable declarations and executable bodies remain owned
by their source modules. Mutable package variables are owned once as fields of
the package state module. One passive package-assembly builder receives a
typed `$initialize` body containing zeroing, ordered initializer statements,
and admitted `init` calls. One static program-initialization builder consumes
the selected package graph and invokes those bodies in exact Go package order;
it does not build a second semantic graph.

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
