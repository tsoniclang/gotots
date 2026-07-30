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

Package clauses, import syntax, comments, semicolon insertion, and punctuation
are parent- or loader-owned source structure, not independent target
constructs. Ordinary comments—including `//go:generate`, which does not affect
compilation—produce no target node. Build selection remains owned by the
selected Go loader. A compiler directive with an emitted semantic effect must
be handled by its named language or environment owner before publication; it
must not be inferred from comment spelling by an unrelated handler.

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

Within those narrow entries, the direct simple-statement family includes a
single short declaration, a single-target assignment, `++`/`--`, and a
discarded direct call. A `for` post excludes short declarations exactly as Go
does. The owner accepts a target expression only when child emission has no
prerequisite statements; it does not drop a statement wrapper, flatten a
parallel transaction into a comma expression, or move work across the loop
boundary.

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

An expressionless switch supplies the semantic boolean tag directly:

```go
switch {
case ready:
	use()
default:
	wait()
}
```

```ts
switch (true) {
case ready: {
	use();
	break;
}
default: {
	wait();
	break;
}
}
```

Its case expressions are dispatched as conditions in source order. No source
node, spelling check, or fabricated semantic record represents the absent tag.
The switch owner selects one of two target shapes. Primitive tags whose Go
equality is exactly target strict equality and whose cases have no prerequisite
statements retain the native switch above. Every other represented comparable
tag is copied once, and ordered case expressions select one numeric clause
before one native execution switch runs the clause bodies. This keeps case
evaluation lazy, preserves custom equality, and emits each body once.
`fallthrough` removes only the implicit break from the selected clause,
including a non-final default clause. Type switches remain owned by the
interface family; they are not approximated by this expression-switch owner.

### Structured Loops, Range, And Labels

A three-clause `for` remains a direct target `for` whenever its initializer and
post each fit one target header and its condition has no prerequisite
statements. Otherwise the loop owner emits one enclosing lexical block:
initializer statements execute there once. If only the initializer requires
statements, the native target `for` remains. Otherwise the owner emits one
direct infinite target loop. A post clause executes at the loop top except on
the first iteration, then condition prerequisites and the condition execute,
then the body. Native normal and labeled `continue` therefore reach the post;
`break`, return, and panic do not. No synthetic condition/post callback,
callable facet, or extra scheduling boundary is created. A cooperative
condition or post remains directly in the enclosing cooperative source
callable.

Range is one parent-owned family selected from the checked range-expression
type. Arrays, pointers to arrays, slices, strings, maps, and integers each
construct typed TS-Go loops directly; no generic loop or operation IR exists.
The array owner applies Go's constant-length rule exactly: with at most one
iteration variable, an array or pointer-to-array operand containing no
function call or channel receive is not evaluated; otherwise it is captured
once. Conversions recurse into their operands and are not misclassified as
ordinary calls. Array values are copied where value iteration requires the
range copy, pointer arrays remain aliases, strings use UTF-8 byte indexes and
invalid-byte `RuneError`, and integer ranges preserve the selected integer
carrier.

Map range snapshots the initial key set once and performs a live lookup before
each body. A deleted unseen entry is skipped, each snapshotted key is considered
at most once, and additions may be omitted as Go permits. Keys and values enter
the same represented-value copy and range-assignment owners as ordinary
assignment. Output size depends on source syntax, never the runtime collection
length.

Per-iteration declarations use the exact `types.Var` definitions and create
fresh target bindings or addressable cells. Assignment-form range prepares
non-blank target locations once per iteration after the iteration values exist,
then stores left-to-right. Labels and labeled branches use exact
`*types.Label` definitions/uses. A loop label is attached to the actual target
loop, not an enclosing prerequisite block.

Function-exit control is assembled once by the exact callable owner. For
example:

```go
func read() (result int) {
	defer func() { result++ }()
	result = source()
	return result
}
```

The defer statement first captures its function value. The explicit return
stores a copied value into `result`, exits the lexical body, drains the
invocation-local LIFO stack, and only then reads `result` for the target return.
Ordinary functions with no `defer`, `recover`, or non-structural `goto` keep
their existing direct body byte-for-byte.

`panic(value)` converts a non-nil `value` to the represented empty interface
and throws the one `GoPanic` carrier. `panic(nil)` creates a distinct
runtime-error value with canonical `*runtime.PanicNilError` dynamic identity
and payload; ordinary generated runtime faults retain their separate runtime
dynamic identity. Both satisfy canonical `error`,
`interface { Error() string }`, and `runtime.Error` contracts. Consequently,
recovered concrete/interface assertions and `Error()` calls observe Go
behavior rather than an empty-interface-only placeholder. `recover()` reads
only an optional hidden
`GoRecovery` parameter on the directly invoked deferred callable:

```go
func direct() any { return recover() }  // non-nil only when called by defer
func indirect() any { return direct() } // nil: one call below
```

Deferred direct functions, receiver functions, function and method values,
interface method adapters, and generic functions forward that parameter
statically. Ordinary calls omit it. No global state or dynamic invocation is
permitted.

A defer-site call is not emitted as the original call expression. Its owner
evaluates the callee, receiver, and arguments in Go order, applies the ordinary
Go copy owner to value receivers and arguments, and stores one typed
zero-argument closure. Invocation happens only during LIFO unwind. A deferred
nil function therefore panics during unwind, while a nil pointer implicit
dereference needed for a value receiver panics at registration.

Exact label definitions and uses come from `go/types.Info.Defs` and `Uses`.
Labeled `break`/`continue` and switch `fallthrough` retain their direct target
forms. A forward edge representable by a target label and a backward edge
representable by one target loop remain direct. Crossing or multi-entry goto
regions select the smallest enclosing callable state machine: one numeric
state, one linear switch, and no persisted CFG. Go's type checker has already
rejected jumps into scopes; target assembly preserves source initialization
boundaries and composes with defer and range.

The same ordered-statement owner handles block, switch-clause, and
type-switch-clause lists. If its generated dispatch is nested in a source loop,
range, switch, or type switch, unlabeled source `break` and `continue` target a
generated label on that source construct rather than the nearer dispatch
loop/switch. A source `fallthrough` remains owned by the enclosing switch
clause and executes after the clause-local state machine exits.

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

An explicitly typed local constant uses the same exact `types.Const` value
contract as an explicitly typed package constant, but its declaration remains
inside the owning block:

```go
const limit int32 = 4
```

```ts
const limit: int32 = 4;
```

Constants are materialized from their checker evidence, never from a
re-evaluation of the source value expression. The value is the exact
`go/constant.Value` on the `*types.Const`; the source initializer syntax
(`iota`, shifts, inherited/omitted specs) is validated and accounted for as a
grammatical child, but never re-executed as target code. `iota` is a
declaration-time constant-folding fact only and never appears in target output.

An explicitly **typed** constant — package or local — is one direct binding at
the constant's own concrete type:

```go
const Base int = 40
```

```ts
export const Base: int32 = 40;
```

An **untyped** constant has no single runtime type: Go converts it to a concrete
type independently at each use. It therefore has no single binding. Each bare
named use requests one projection at its effective concrete type through the
demand-driven reconstructible-artifact lifecycle. The effective type is the
concrete checker type on that occurrence when one exists; otherwise it is the
concrete type supplied by the owning parent context after assignability is
validated. For example, in `return Scale`, `Scale` remains `untyped int` in
`go/types.Info.Types`, while the return owner supplies the function's `int32`
result type. The emitter must not call `types.Default`, infer a type from
spelling, or assume that a child occurrence carries its parent's conversion.
The exact `*types.Const` selected by `go/types.Info.Uses` owns the declaration
value. An occurrence's `TypeAndValue.Value` may already reflect contextual
conversion or rounding (for example, an untyped `Pi` used as `float64`), so it
proves that the occurrence is constant but is not a second declaration-value
truth and is not compared byte-for-byte with `Const.Val()`.

A non-reference constant expression is different. In
`return Scale + Scale`, the two identifiers remain untyped, while the
`*ast.BinaryExpr` carries the exact folded checker value. The binary owner emits
that enclosing value at the parent's concrete result type; it does not emit
runtime addition and does not ask either child to rediscover the context.

A package-level projection is one exported declaration imported statically. A
function-local projection is emitted at its original lexical `const`
declaration, never flattened into a function prologue. Sibling blocks may
therefore declare the same Go spelling without colliding or widening scope.
Every use — a same-package identifier, a package-qualified selector, or a
dot-imported identifier — routes through the same constant-use owner and is a
constant-size reference to its projection. Cross-package import identity is the
full `(package constant identity, target representation)` pair, so two packages
exporting `Width` cannot create duplicate local bindings. Output size is
`O(value-size + uses)`, never `O(value-size × uses)`; inlining a named value at
each use is forbidden. A projection whose value is not representable in the
selected profile fails at the typed value boundary, not at a caller invariant.

Emission roots retain their intent. Whole-file coverage and exported-Go-API
roots account for an unused untyped constant as a compile-time-only declaration;
they do not invent a runtime value. This is necessary rather than an
optimization: Go's exported untyped-constant contract admits an open set of
future contextual conversions, so no finite set of target runtime bindings can
be its exact public replacement. A translated Go consumer requests the exact
projection it uses. An explicit representation root must name its concrete
basic representation and materialize that projection or fail; a generic
`NewRoot` request for an untyped constant is rejected as ambiguous. Projection
kinds are the closed concrete constant-capable basic domain; untyped kinds,
`Invalid`, out-of-range values, and `unsafe.Pointer` fail at construction.
These typed dispositions may not be confused with failed projection or use
`types.Default`.

All bare named-use paths share this one classification: same-package
identifiers, package-qualified selectors, and dot-imported identifiers identify
the exact `*types.Const` and delegate to the single constant-use owner with the
source expression and `types.TypeAndValue`. Root policy is a separate typed
owner because coverage and public representation are different obligations.
Every materialized package declaration (typed binding or untyped projection) is
exported for static generated-module linking; package assembly later selects the
public surface. A single `go/constant.Value.Kind` dispatcher owns value
materialization; integer, string, float, and later complex materializers have no
other semantic callers. A syntax owner may validate its AST form and choose the
enclosing checker value, but delegates value materialization to this owner.

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
  requests: typed root requests for imports/declarations/helpers, declaration
            requirements, or facet-specific artifact dependencies

StatementEmission
  statements: ordered typed TS-Go protocol statements
  requests: typed root requests

DeclarationEmission
  declarations: ordered typed TS-Go protocol declarations
  requests: typed root requests
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

The request slice is a shallow immutable transport view. When multiple child
views are combined, the API retains their atomic typed requests in an opaque
persistent sequence rather than eagerly flattening and recopying all
transitive leaves. Accessors copy only the exposed top-level view. Root
consumers traverse the sequence once in exact order; sequence shape cannot
affect placement, scheduling, deduplication, or generated output. Composition
is proportional to immediate children instead of
`source depth * transitive requests`.

A declaration requirement similarly carries no mutable target node. It names
the exact declaration owner and one closed obligation. The root routes it back
to that semantic owner, which reconstructs its complete typed TS-Go declaration
assembly from the source declaration and accumulated requirement set. This
replacement occurs only in compilation-local target state before file sealing;
no handler patches a previously printed artifact.

A target reference also carries no provider node. It records a closed artifact
dependency from the currently assembled declaration to the exact provider
`types.Object` and observable facet consumed by that reference. The root
replaces the consumer's complete dependency set when its artifact transaction
commits. If a provider reconstruction changes that facet's canonical TS-Go AST
projection, the root requeues the consumer; exact unchanged projections do
nothing. Dependencies emitted outside a reconstructible artifact are rejected
until that enclosing target owner participates in the same lifecycle rather
than being silently dropped.

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

For an identifier, direct named field, or pointer-dereferenced field of a
directly represented signed integer type, compound addition remains one direct
target operation:

```go
total += delta
```

```ts
total += delta;
```

The assignment owner proves the exact store target, represented operand type,
and assignability of the right side. A direct primitive target accepts this
form only when the right-side emission has no prerequisite statements, so
reading the left value cannot move across right-side effects. The target
owner's prerequisite statements remain before the direct compound operation.

For a represented defined value behind an array, slice, or map accessor, one
accessor-store transaction captures the receiver and index/key once. Under the
`preserve-go` profile it then evaluates and captures the right side before
calling the getter, applies the exact underlying-family operation, wraps the
result, and calls the setter once:

```go
values[nextIndex()] += nextValue()
```

```ts
const receiver = values;
const index = nextIndex();
const right = nextValue();
receiver.set(index, new Count(receiver.get(index).$value + right.$value));
```

The `direct` profile retains its explicit target-order tradeoff and may keep a
direct right expression. Primitive accessor compounds and every operation not
owned by the represented value family remain typed failures; no generic host
operator fallback is inferred.

An ordinary Go function with two or more results has one direct TypeScript
tuple carrier. Result declarations use `[T0, T1, ...]`; an explicit
`return left, right` constructs `[left, right]`; and `return pair()` preserves
the tuple-valued call directly. There is no generated result class, wrapper,
out-parameter ABI, erased carrier, or per-function representation choice.
When any explicit result has prerequisite statements, the return owner
evaluates and copies every result left-to-right into one local constant before
evaluating the next result, then returns the tuple of constants. Direct
multi-result returns acquire no such captures.

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

Named results are the function owner's lexical variables, initialized to the
same exact zero representation used elsewhere:

```go
func Next(value int32) (next int32, ok bool) {
	next = value + 1
	ok = value >= 0
	return
}
```

```ts
export function Next(value: int32): [int32, bool] {
	let next: int32 = 0;
	let ok: bool = false;
	next = value + 1;
	ok = value >= 0;
	return [next, ok];
}
```

The selected signature supplies each exact result `types.Var`; the name owner
supplies its lexical binding; and the value owner supplies its zero. A bare
return is accepted only when every result is named. A nested function literal
enters a new function-result context, so it cannot return an enclosing
function's named results.

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

The initial callable-value family uses the direct form for an unnamed function
type whenever the signature's parameter and result representations are already
supported:

```go
func Apply(
	transform func(int32) int32,
	value int32,
) int32 {
	return transform(value)
}

func Offset(delta int32) func(int32) int32 {
	return func(value int32) int32 {
		return value + delta
	}
}
```

```ts
export function Apply(
	transform: (value: int32) => int32,
	value: int32,
): int32 {
	return transform(value);
}

export function Offset(delta: int32): (value: int32) => int32 {
	return function (value: int32): int32 {
		return value + delta;
	};
}
```

The one `go/types.Signature` is the callable truth. Function declarations,
function types, function literals, values, and calls consume the same signature
owner. JavaScript lexical capture directly represents admitted Go variable
capture; no environment object, callback registry, hidden receiver, or callable
side table is introduced. Copying an admitted function value is a direct
reference-value copy.

The generated callable ABI is canonical per exact signature, not per place
that stores a callable. For example, `var f func(int)`, `box.F`, `pointer.F`,
and `values[0]` all carry the same signature-owned target callable contract.
An ordinary `f(1)` omits the optional recovery-authority facet; a captured
`defer f(1)` invokes that same callable value with the authority. The field,
pointer, and slice paths do not create recovery identities or a storage-flow
model.

An unnamed callable value that may contain Go nil has the static target type
`((parameters) => results) | undefined`. Known non-nil declarations and
literals remain direct functions. Calls evaluate the callee once, evaluate and
capture arguments left-to-right, then perform the nil guard at the invocation
boundary; this preserves the fact that Go evaluates arguments before a nil
function call panics. The non-nil branch invokes the function directly, never
through `.call`, `.apply`, `.bind`, reflection, or an erased carrier.

A defined callable has one nominal class for its non-nil value and two
canonical static boundary operations:

```go
type Transform func(int32) int32
type Alias = Transform
```

```ts
export class Transform {
  declare private readonly $goType: void;
  constructor(public readonly $value: (value: int32) => int32) {}
  static $from(
    value: ((value: int32) => int32) | undefined,
  ): Transform | undefined {
    return value === undefined ? undefined : new Transform(value);
  }
  static $valueOf(
    value: Transform | undefined,
  ): ((value: int32) => int32) | undefined {
    return value === undefined ? undefined : value.$value;
  }
}
export type Alias = Transform | undefined;
```

Every Go use of `Transform` is represented as `Transform | undefined`; the
class itself represents only the non-nil member. `$from` and `$valueOf` are the
single wrap/projection owners for both nil and non-nil values. They avoid
duplicated use-site conditionals and remain valid when TypeScript flow analysis
already knows an operand is `undefined`; no property access is emitted on a
branch narrowed to `never`. Conversion from an unnamed function calls `$from`.
Conversion between distinct defined callable types calls the source `$valueOf`
and destination `$from`. Calling a defined callable checks the wrapper for
`undefined` and then invokes `wrapper.$value(arguments)` directly. The class
contains no string brand, runtime property mutation, `Object.assign`, callback
registry, or speculative method protocol. A declared alias reuses the same
representation and emits no class.

Nil equality compares only with `undefined`, as selected by the checker.
`new(func(...))` creates a fresh pointer cell whose value is `undefined`.
Generic signatures and interface callables remain at their separately owned
boundaries. Variadic method values and method expressions use the same
represented final Go-slice parameter as every other variadic callable; they
never acquire a JavaScript rest-parameter convention.

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

A represented variadic callable has one ordinary final parameter whose target
type is the represented Go slice. It is not emitted as a JavaScript rest
parameter. A non-spread call evaluates and copies its arguments in Go order,
then constructs that one slice argument. A spread call projects the exact
source slice descriptor, including a defined-slice wrapper, and passes the
descriptor directly. The multiple-result adjustment is applied before the
variadic partition, so a final call returning the remaining fixed and
variadic arguments is accepted exactly when the selected Go signature accepts
it. No valid Go call is lowered through target `...` argument spreading; this
keeps call behavior independent of host argument-count limits.

```go
type Values []int32
func Sum(prefix int32, values ...int32) int32
func Pair() (int32, int32)

func A() int32         { return Sum(Pair()) }
func B(values Values) int32 { return Sum(1, values...) }
```

```ts
export function Sum(prefix: int32, values: GoSlice<int32>): int32;
export function A(): int32 {
  const pair = Pair();
  return Sum(pair[0], GoSlice.literal<int32>(pair[1]));
}
export function B(values: Values): int32 {
  return Sum(1, values.$value);
}
```

The schematic target omits prerequisite temporaries that are unnecessary for
the shown operands; production output is constructed only as TS-Go AST.

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

For an admitted function-valued callee, the call handler emits the callee
expression once and invokes it directly:

```go
selected := choose(flag)
return selected(value)
```

```ts
let selected = choose(flag);
return selected(value);
```

The signature comes from `types.Info.TypeOf(call.Fun)`. A named function
identifier additionally requests its declaration through the ordinary
identity-keyed name owner. An arbitrary callee is delegated through the
call-callee child role. Generated code never uses `.call`, `.apply`, `.bind`,
an erased callable carrier, or source-spelling classification.

If argument translation requires statements, the arbitrary callee is captured
before those statements so Go's function-before-arguments order remains exact:

```go
return selectPair()(pair())
```

```ts
const callee = selectPair();
const results = pair();
return callee(results[0], results[1]);
```

The capture is requested only by this execution-boundary requirement; a direct
call remains a direct call and possible-function count never expands the call
site.

## Defined Types And Aliases

The exact `*types.TypeName` and its `go/types` type decide whether a declaration
is an alias or a defined type. Source spelling, underlying-type text, and
structural target assignability never make that decision.

An alias adds no runtime identity. Its exported target declaration is a
TypeScript type alias to the already selected representation:

```go
type Count int32
type Alias = Count
```

```ts
export class Count {
  declare private readonly $goType: void;
  constructor(public readonly $value: int32) {}
}
export type Alias = Count;
```

A non-struct defined type has one minimal nominal wrapper class keyed by its
exact `*types.TypeName`. The erased private brand prevents unrelated defined
types with the same underlying type from becoming assignable. The public
readonly `$value` carries the exact represented underlying value. The class
contains no operator table, callback, dynamic tag, source node, or speculative
zero/copy/equality method.

The containing semantic owner composes the underlying value operation directly:

```go
type Count int32
func Add(left, right Count) Count { return left + right }
```

```ts
export function Add(left: Count, right: Count): Count {
  return new Count(left.$value + right.$value);
}
```

Zero, copy, equality, hashing, addressing, conversion, and container operations
follow the same rule: unwrap once, invoke the exact existing underlying-family
operation, and wrap the result only when the Go result has the defined type.
Immutable underlying values may share their immutable wrapper on copy.
Aggregate underlying values request their recursive copy operation before
wrapping. Use sites remain O(1), and each defined type contributes exactly one
class definition.

Generic non-struct defined types use the same owner and wrapper, parameterized
by the declaration's exact Go type parameters. Every instantiated reference
projects all `go/types.Named.TypeArgs` through the ordinary represented-type
owner in order; none are dropped or reconstructed from spelling. For example,
`iter.Seq[int32]` is `Seq<int32> | undefined`, and range first projects its
callable payload through `Seq.$valueOf` before invocation. It never calls the
wrapper object itself. The generic declaration remains one class rather than a
class per instantiation, and static wrap/project operations carry or infer the
same parameters.

Named struct definitions keep the direct nominal record class described below;
they do not acquire a redundant wrapper around that class. Nil-capable defined
types use `Defined | undefined`; `undefined` remains the sole nil value and only
a non-nil underlying value is wrapped. Aliases reuse that same union without a
new class.

Type conversions are the only bridges between a defined type and its
underlying or another defined type. The conversion owner strips only the exact
source wrapper, applies the existing source-to-destination representation
conversion, then constructs only the exact destination wrapper. Constants are
materialized from the checker value at the destination underlying type before
wrapping. No conversion falls through to an ordinary call.

## Structs, Receivers, And Classes

An ordinary named Go struct is a value type. TypeScript objects and classes are
reference values, so direct target assignment cannot represent Go copying.
GoToTS makes an observably required zero, copy, or equality operation explicit
through one representation owner; it does not assume that a later compiler,
plugin, or runtime will reinterpret ordinary TypeScript assignment.

The supported struct family selects one nominal record class for non-generic
named structs whose fields have admitted representations. An embedded field is
ordinary owned storage in that class; it does not imply target inheritance.
A generated record class has:

- one erased private brand, which makes distinct named Go structs nominally
  distinct to strict TypeScript without adding an instance field;
- public data fields initialized by its constructor;
- no instance receiver methods, inheritance, dynamic lookup, or hidden
  semantic payload;
- use-demanded static zero, copy, and equality operations incorporated into
  the class by its one declaration owner.

For example, when selected uses require all three operations, a supported
`Pair` is:

```ts
export class Pair {
  declare private readonly $goType: void;

  constructor(public Left: int32, public Ready: bool) {}

  static $zero(): Pair {
    return new Pair(0, false);
  }

  static $copy(source: Pair): Pair {
    return new Pair(source.Left, source.Ready);
  }

  static $equal(left: Pair, right: Pair): bool {
    return left.Left === right.Left && left.Ready === right.Ready;
  }
}
```

The class supplies target nominality and stable runtime constructor identity;
it does not change Go value semantics. Distinct named structs remain statically
incompatible even when their fields match. A requested nested struct operation
requests the corresponding static operation from the nested type's owner. The
erased brand must produce no JavaScript instance field.

Go permits a receiver declaration to omit its source name:

```go
func (Token) Kind() int32 { return 1 }
```

The receiver function still requires its explicit target receiver parameter.
The declaration owner derives that parameter from the exact
`go/types.Signature.Recv()` object and assigns the deterministic target-only
positional name (`$0` when the method has no source parameters). It does not
invent a Go binding, scan the body, or infer a name from source spelling:

```ts
export function Token_Kind($0: Token): int32 {
  return 1;
}
```

Interface behavior, generics, reflection entry, pointers to unrepresented
targets, and fields whose complete standalone representation is not yet exact
are neighboring typed-unsupported families. Tags and blank fields participate
in exact type identity without implying reflection metadata or ordinary
mutable storage. Embedded storage, promoted field selection, and promoted
concrete methods use the exact `go/types.Selection` path. No neighboring family
may be approximated by structural assignment or virtual dispatch.

The recursive-value extension admits represented fields through arrays,
slices, maps, pointers, callables, and other structs. A direct infinitely sized
cycle is invalid Go and never reaches emission; every legal recursive cycle
crosses a reference-capable representation. Static zero, copy, equality, hash,
and address operations therefore request one another by exact type identity
and converge instead of recursively expanding source at each use.

An anonymous struct receives one canonical nominal target declaration because
plain TypeScript structural typing cannot preserve Go field tags, blank
fields, unexported-field package identity, or exact anonymous-type identity.
`go/types` and `types.Identical` are the sole identity authority. A stable
structural fingerprint may select a candidate bucket and target artifact name,
but every reuse decision exact-checks `types.Identical`; a fingerprint collision
must retain two declarations. A named component's stable key is derived from
its authoritative declaration object's semantic package and deterministic
lexical declaration identity, while the selected object remains the in-memory
truth for the decision. Package path plus source spelling alone is never
identity.

The canonical declaration is placed at the highest preferred lexical scope
that can name every component representation. Shapes whose components are all
module-nameable use one compilation-owned shared artifact, so identical
anonymous structs in different Go packages remain one target type. A shape
depending on a local named type remains in the owning lexical scope; it is
never hoisted to a module that cannot name that type. The first encounter does
not decide placement. A root request or digest is only an artifact key and
must carry or validate the exact Go type; it is not a second type-identity
model.

Each canonical anonymous declaration owns its field storage and demanded
static zero, copy, equality, hash, and structural-conversion members once.
Ordinary construction may remain direct, but operation use sites are
constant-size calls. Doubling uses must not duplicate field walks. Field tags
participate in identity but emit no runtime metadata unless a later reflection
consumer requests it. Blank fields participate in identity and required
evaluation order but expose no mutable target property. Embedded fields remain
storage composition. Their later read, address, store, and concrete-method
consumers all use the same exact selection-path owner.

Each zero, copy, or equality capability is emitted at most once and only when a
selected occurrence requests it. Its typed request is routed to the defining
source-file module even when the first use is in another file. The class is
reconstructed from its complete accumulated operation set, so exactly one
class definition remains and no top-level operation helper exists. Static
operation definitions grow linearly with fields while each use remains
constant size. Calls use the exact statically selected class
(`Pair.$copy(value)`), never `value.$copy()`, so TypeScript virtual dispatch
cannot replace Go's static method selection. The first family constructs only
through `new Name(...)`. Positional composites are direct in both profiles.
For keyed composites, `direct` emits constructor arguments directly in
declaration order, while `preserve-go` captures values in keyed source order
before consuming them in declaration order. Omitted fields request the same
zero owner rather than a second default table.

Generated source modules use a caller-owns-value convention. A borrowed struct
expression (for example, a local identifier or field selection) is copied
exactly when it crosses an initialization, argument, receiver, or composite
field boundary. A fresh composite literal or function result already owns
fresh storage and transfers that ownership without another copy. A supported
single-result return transfers the function's owned local value. When no
address is selected, no admitted path can retain an alias to that transferred
storage. When an address is selected, the exact variable becomes a mutable
cell and projections read through that cell, so replacing the whole struct
updates what an existing field pointer observes without destination-preserving
mutation of every ordinary struct.

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
target = Pair.$copy(value);
```

No unaddressed construct can observe the replaced target object's identity.
Destination-preserving mutation is therefore not emitted speculatively.
Addressability installs storage only at the exact `*types.Var` whose identity
becomes observable.

For example:

```go
copy := value
copy.Ready = false
```

```ts
let copy = Pair.$copy(value);
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

Concrete generated calls invoke `Flag_Disable(Flag.$copy(flag))` directly, so
the selected value receiver owns exactly one copy before its body runs. Pointer
receivers, method values, and interface calls select their own exact checked
entry shape from `go/types`; they do not reuse a value-receiver entry when its
copy or nil behavior differs. Generated code never uses `.call`, `.apply`, or
`.bind`.

Every concrete field or method selector is validated against the exact
`*types.Selection`: kind, selected object identity, receiver type, result type,
and every `Index()` component must agree with the selected checker graph.
Source spelling is not a selection key. The one selection-path owner projects
embedded fields for reads, stores, addresses, direct calls, method values, and
method expressions.

For example:

```go
type Base struct{ Value int32 }
func (base Base) Read() int32 { return base.Value }
func (base *Base) Add(delta int32) { base.Value += delta }

type Derived struct{ Base }

read := value.Read
add := value.Add
method := Derived.Read
```

The semantic decisions are:

- `value.Read` captures one copied `Base` value at method-value formation;
- `value.Add` captures the canonical address of the promoted `Base` storage;
- `Derived.Read` is a typed adapter whose first explicit argument is
  `Derived`, projects its selected `Base`, copies it once, and calls
  `Base_Read`; and
- a direct unpromoted method expression is the existing receiver-function
  reference, with no adapter.

A representative target shape is:

```ts
const readReceiver = Base.$copy(value.Base);
const read = (): int32 => Base_Read(readReceiver);

const addReceiver = GoPointer.field(valueAddress, "Base");
const add = (delta: int32): void => Base_Add(addReceiver, delta);

const method = (receiver: Derived): int32 =>
  Base_Read(Base.$copy(receiver.Base));
```

Each receiver expression is evaluated once. A nil pointer passed to a
pointer-receiver method may be captured and used by a nil-safe body. Forming a
value-receiver method value through a nil pointer performs the required
dereference and panics at formation. Promoted pointer fields are dereferenced
only at the selected path boundary, preserving the corresponding Go panic
timing.

Method calls and method values remain constant-size as embedding depth grows:
only the selected field path grows. They never expand with method count or
possible receiver count. Anonymous and named embedded fields share this same
path owner and their exact member names come from field-object identity.

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

### Exact Value Shape

The one interface representation is:

```text
nil interface        -> undefined
non-nil interface    -> one immutable adapter instance
dynamic Go type      -> one canonical non-string token per exact Go type
dynamic Go value     -> exact statically typed readonly adapter payload
```

There is one adapter class per reached exact concrete dynamic type, not per
interface and not per call. Its callable surface is the deterministic union of
the completed interface contracts actually required by the selected
compilation closure. A concrete-to-interface conversion requests its exact
target contract and copies a Go value once before storing it in the adapter.
Pointer, slice, map, channel, and function payloads retain their Go reference
identity. An interface copy shares the immutable adapter. A typed nil pointer
is therefore a non-nil adapter whose payload is the represented nil pointer; it
is never collapsed into `undefined`.

Adapter constructor identity is not Go dynamic-type identity. A local Go type
declared inside a function has one compile-time identity but its lexical
TypeScript adapter class declaration executes once per function invocation.
Two values boxed by two invocations must still have the same Go dynamic type.
The target-name owner therefore interns one compilation-scope frozen object
token by the exact `go/types.Type` identity. Every adapter instance carries
that token in the readonly `$go$type` member. No string, constructor,
`Symbol.for`, source spelling, or truncated hash is semantic identity.

Each adapter contains only methods selected by demanded completed interface
contracts. For every demanded interface method, `go/types` selects the exact
concrete or promoted method; the adapter emits one native TypeScript method
that calls the already-owned top-level receiver function directly. Multiple
contracts selecting the same method emit it once. The payload, parameter, and
result types remain exact; no method body uses `any`, `unknown`, a cast,
`.call`, `.apply`, or `.bind`. Ordinary concrete calls continue to call
receiver functions and never become virtual.

For example, if `*Checker` has 1,976 receiver methods but is converted only to
`interface { GetSymbolAtLocation(*Node) *Symbol; GetAliasedSymbol(*Symbol)
*Symbol }`, its adapter emits exactly those two forwards. Boxing `*Checker` as
`any` alone emits no receiver forwards. An unrelated private checker method can
never enlarge either artifact.

For example:

```go
type Reader interface{ Read(int32) int32 }
type Counter int32
func (counter Counter) Read(delta int32) int32 { return int32(counter) + delta }

var reader Reader = Counter(4)
```

has the conceptual target shape:

```ts
export interface Reader extends GoInterfaceValue {
    Read(delta: int32): int32;
}

export const Counter$DynamicType: object = Object.freeze({});

export class Counter$InterfaceAdapter implements GoInterfaceValue {
    constructor(public readonly $go$value: Counter) {}
    readonly $go$type: object = Counter$DynamicType;

    static $is(
        value: GoInterfaceValue | undefined,
    ): value is Counter$InterfaceAdapter {
        return value !== undefined &&
            value.$go$type === Counter$DynamicType;
    }

    Read(delta: int32): int32 {
        return Counter_Read(this.$go$value, delta);
    }
}

const reader: Reader | undefined =
    new Counter$InterfaceAdapter(Counter.$copy(new Counter(4)));
```

The actual declarations are constructed as TS-Go AST and printed by pinned
TS-Go; this source is explanatory, not a text template.

### Method Contracts And Open-World Assertions

Interface satisfaction is not encoded as an implementer union. One canonical
generated token object represents one exact Go method contract:

- exported identity is method name plus exact receiver-free signature;
- unexported identity additionally includes declaring package identity;
- parameter names do not affect identity;
- aliases and named types follow `go/types` identity.

Tokens are interned by the deterministic target-name owner and compared only by
object identity at runtime. They are not strings, source spellings, property
probes, or hashes used without collision validation.

Each completed interface owns an immutable contract containing its required
method tokens. Each adapter class owns one module-level `ReadonlySet<object>`
containing exactly the tokens selected by its demanded interface contracts.
The shared runtime tests whether every target contract token is present in
that set. A generated interface-specific function exposes the result as a
TypeScript type predicate:

```ts
export function Reader$is(
    value: GoInterfaceValue,
): value is Reader {
    return value.$go$implements(Reader$contract);
}
```

Concrete conversions request their target contract directly and seed that
contract on the concrete adapter. For an interface-to-interface conversion or
assertion, the compiler records the exact static source and target contracts.
It applies the target only to an adapter whose reachable-contract set already
contains the source and whose concrete type `go/types` proves implements the
target. The requirements are retained and closed transitively, so transitions
and adapters may be discovered in any order. A concrete type that happens to
implement the source is not widened unless a selected conversion made that
source contract reachable for its adapter. Therefore the selected closure
needs no implementer switch or value-flow analysis. Each reachable
adapter/contract pair is materialized once; repeated uses of the same
transition are constant-time deduplication rather than repeated whole-program
closure,
while an unrelated adapter receives no methods. Assertion cost is O(target
interface method count), and an ordinary interface call is one direct native
method call, O(1).

### Calls, Assertions, Equality, And Hashing

The interface-call owner guards `undefined` at the Go call boundary and then
calls the native member. Forming an interface method value evaluates, guards,
and captures the adapter exactly once. A method expression takes the interface
receiver explicitly and performs the guard when invoked.

Concrete assertions call the adapter's statically typed `$is` predicate, which
compares the canonical dynamic-type token and narrows the payload without a
cast. Interface assertions request the target contract for every statically
possible reached dynamic type, call the target interface's typed predicate,
and retain the same adapter. Interface-to-interface conversions make the same
request even though their runtime value is unchanged. Comma-ok and panicking
forms share these primitives. Type switches evaluate the source once and test
cases in source order; each case variable receives the exact Go case type.

Interface equality is:

1. both `undefined`: true;
2. exactly one `undefined`: false;
3. different canonical dynamic-type tokens: false;
4. the same token with a non-comparable dynamic Go type: panic;
5. otherwise: call that payload type's exact equality owner.

Interface hashing mixes the runtime object-identity hash of the same canonical
token with the payload hash owner. A non-comparable dynamic type panics when
used as an interface map key. Adapter classes implement equality and hashing
without erased payload recovery: the token-backed type predicate narrows the
other adapter to the exact class before its payload is read.

Pointer payload hashing requests one standalone `goPointerHash` runtime
definition, which hashes the pointer's readonly canonical address. The base
`GoPointer` declaration does not depend on map hashing; a program that only
creates, compares, or dereferences pointers emits neither `goPointerHash` nor
`GoMapHash`. This demand boundary prevents interface support from enlarging
ordinary pointer artifacts.

Named interface declarations remain source-owned. Anonymous interface
contracts, concrete adapters, canonical method tokens, and canonical
dynamic-type tokens are typed generated artifacts with deterministic
compilation or lexical placement. They use the existing transactional
reconstruction graph and observable facets; no side-table representation, text
patch, or post-seal mutation is introduced.

## Generics And Iterator Functions

### Generic Declarations And Types

The declaration owner emits one TypeScript generic declaration for each exact
Go generic function, alias, or named type. Type-parameter names come from their
`*types.TypeName` identities. A use of a type parameter emits that target type
reference; an instantiated type emits the same declaration with explicit
target type arguments.

Go constraints are not translated into an approximate TypeScript `extends`
clause. The selected `go/types` graph has already checked type sets, inference,
method sets, and operation legality. The target declaration instead receives
one exact typed function for each distinct operation signature required by its
emitted body:

```go
func Clone[T any](value T) T {
	return value
}

func Equal[T comparable](left, right T) bool {
	return left == right
}
```

```ts
export function Clone<T>(
  $go$copy: (value: T) => T,
  value: T,
): T {
  return $go$copy(value);
}

export function Equal<T>(
  $go$equal: (left: T, right: T) => boolean,
  left: T,
  right: T,
): boolean {
  return $go$equal(left, right);
}
```

The snippets are schematic printed output; production creates every node
through the pinned TS-Go factories. Operation functions use closed enum-owned
identifier stems, not diagnostic/source token spellings. They preserve Go
result identity: `Add[T ~int32]` calls `$go$binary_add` returning `T` rather
than using target `+`, whose result would be `number` and would lose a defined
actual type.

A local named type inside an operation signature is keyed by the canonical
enclosing source `ArtifactOwner` followed by its exact subordinate scope-child
path. Thus identical lexical shapes in two functions remain distinct, moving
one declaration across unrelated source lines is identity-neutral, and a
scope rooted in another source artifact is rejected.

Generic aliases create no runtime identity. Generic named values use one
generic representation declaration. Their demanded zero/copy/equality/hash
functions accept exact operations for constituent type parameters rather than
storing descriptors in every value. Methods on instantiated generic receiver
types receive and forward the same hidden function ABI; ordinary values remain
free of operation metadata.

Generic methods use one exact ABI derived from the method origin and its
canonical operation contracts: hidden operation functions first, then the
receiver, then source parameters. Declarations, direct calls, method values,
and method expressions all exact-join those identities; no caller separately
constructs or orders the same list.

### Instantiation And Calls

Every explicit or inferred instantiation is selected from the exact
`types.Info.Instances` entry on its identifier. `IndexExpr` and
`IndexListExpr` are generic instantiations only when that evidence exists;
otherwise their existing index-expression owners remain authoritative.
Emission exact-joins:

1. the selected generic object;
2. its ordered declared type parameters;
3. the instance's ordered concrete type arguments; and
4. the instantiated signature or named type reported by `go/types`.

The target call writes explicit TypeScript type arguments even when Go inferred
them and passes the declaration's ordered hidden operation functions before
source arguments:

```go
result := Add(int32(2), int32(3))
```

```ts
const result = Add<int32>($goCapability_binary_add_int32, 2, 3);
```

A generic caller forwards its own same-signature operation function. A
cross-parameter operation remains one function such as `(T, U) => T`; there is
no artificial choice of which parameter “owns” it. If the callee needs an
operation absent from the caller, the caller's declaration contract grows
through the facet-specific reconstruction graph and its callers are revisited
only when the callable surface changes. An instantiated function value is one
typed arrow that captures the functions once; `.bind`, `.call`, `.apply`,
runtime inference, and monomorphized duplicate bodies are forbidden.

### Generic Operations

Zero, copy, equality, hash, unary/binary operators, conversions, indexing,
range, built-ins, selected methods, interface adaptation, channels, and
iterator calls enter the same closed capability mechanism only when their
source occurrence uses a type parameter. A concrete occurrence continues to
use the existing direct owner. Concrete operation-function artifacts delegate
back to those owners, so there is one semantic implementation for each
operation.

An operation selection is typed, not merely a broad enum value. Most selections
need only their closed operation kind and exact function signature. A
constraint-method selection additionally carries the selected `*types.Func`;
its identity uses Go method identity (name, unexported package when applicable)
plus the exact receiver-free instantiated signature. Thus `Read() int32` and
`Write() int32` never collapse merely because their signatures match, while
structurally equivalent exported interface methods exact-join.

For example, `var zero T` demands `$zero`; `items[index]` demands the exact
index operation proved by the checked core type; and `value.Method()` demands
the exact selected constraint method identity and receiver-free signature.
No handler switches on `~int`, `comparable`, method spelling, or a runtime type
tag. If `go/types` evidence does not prove the operation and its exact result,
translation fails at that occurrence.

### Range Over Iterator Functions

The range owner accepts only the three checker-approved iterator shapes:

```go
func(func() bool)
func(func(V) bool)
func(func(K, V) bool)
```

The range expression is evaluated once. Its yield callback is a typed arrow;
each invocation copies yielded values into the exact per-iteration declaration
or assignment targets, executes the body, returns `true` to continue, and
returns `false` for a source `break`. A source `continue` returns `true`.
Invocation reuses the ordinary callable nil guard. One callback-local state
tracks ready, body-panic, body-false, and whole-loop-exit conditions; it
reproduces the selected toolchain's distinct misuse panics when an iterator
calls yield in any forbidden state or suppresses a body panic.
Returning from the enclosing function, panic/recover, defer, labels, and
cooperative calls compose through their owning later control capabilities; the
iterator owner never approximates them with an illegal target branch.

Under the cooperative profile, the callback's exact callable ABI owns whether
its result is `boolean` or `Promise<boolean>`. For example:

```go
for value := range numbers {
    total += value + <-closed
}
```

The receive selects the generated yield callback ABI cooperative. The emitted
callback is therefore `async (value): Promise<boolean>`, the iterator provider
awaits each yield, the range invocation awaits the iterator, and that
invocation selects the enclosing source callable cooperative. A nonblocking
range remains a synchronous callback and call. Marking only the enclosing
function `async`, or adding `async` only to the callback, is invalid because
either leaves an illegal `await` or leaves the provider and invocation on a
synchronous callable contract.

The iterator function is invoked exactly once. Calling yield after it returned
`false` must reproduce the selected Go runtime panic. Output size depends on
the source range body, not the number of yields or generic instantiations.

### Callable ABIs Across Generic Instantiation

One generic source declaration retains one ordinary reconstruction path even
when distinct instantiations supply distinct concrete function types.
Demand-created static variants reuse that path with only an exact callable-ABI
profile override:

```go
func Apply[T any](value T, predicate func(T) bool) bool {
	return predicate(value)
}

func Blocking(value int32, closed <-chan int32) bool {
	return Apply(value, func(value int32) bool {
		_, ok := <-closed
		return !ok && value > 0
	})
}
```

The call owner reads the declaration signature `func(T) bool` and the selected
instantiated signature `func(int32) bool` from the same `go/types.Instance`.
It recursively pairs only structurally corresponding callable leaves. The
declaration remains the synchronous baseline. When the concrete literal
selects `func(int32) bool` cooperative, that use requests one statically named
variant:

```ts
export function Apply<T>(
    value: T,
    predicate: (($0: T) => boolean) | undefined,
): boolean {
    if (predicate === undefined) goPanicNil();
    return predicate(value);
}

export async function Apply$cooperative_c17eeb048799ded8a96d<T>(
    value: T,
    predicate: (($0: T) => Promise<boolean>) | undefined,
): Promise<boolean> {
    if (predicate === undefined) goPanicNil();
    return await predicate(value);
}
```

Synchronous calls name `Apply`; blocking calls name the profile variant.
Repeated calls with the same exact profile share that variant. The same
correspondence applies to callable leaves nested in represented containers and
results, and to generic receiver calls, deferred calls, function values, and
method values/expressions. It never treats an opaque type-parameter
substitution as evidence that the substituted type is a callable, and it never
recovers the relation from target spelling.

When `Apply` is exported and the caller is in another Go package, its generated
package assembly re-exports both reached bindings from the source module:

```ts
export {
    Apply,
    Apply$cooperative_c17eeb048799ded8a96d,
} from "./source.js";
```

The declaration artifact remains the sole producer of both definitions.
Package assembly projects the reached exported surface; it does not copy a
body, manufacture another profile identity, or make consumers import an
implementation file directly.

Direction matters when cooperation originates in the declaration:

```go
func MakeReceiver[T any](values <-chan T) func() T {
	return func() T { return <-values }
}
```

The declaration's returned `func() T` ABI is cooperative because that literal
always receives from a channel. The corresponding concrete `func() int32` ABI
is therefore cooperative and `MakeReceiver[int32]` uses the ordinary
`MakeReceiver` declaration. No duplicate profile variant is emitted. The
reverse is forbidden: a cooperative concrete callback passed to `Apply`
selects an `Apply` variant but never changes `Apply`'s baseline ABI.

When the generic owner is an environment contract rather than translated
source, the same use-site profile selects an ambient declaration instead of a
body variant. For example:

```go
func Sum(values []int32, input <-chan int32) int32 {
	var total int32
	for value := range slices.Values(values) {
		total += value + <-input
	}
	return total
}
```

The generated range callback is cooperative. Its exact instantiated yield ABI
selects a declaration shaped like:

```ts
export declare function Values$cooperative_c17eeb048799ded8a96d<Slice, E>(
    values: Slice,
): ((yield: (value: E) => Promise<boolean>) => Promise<void>) | undefined;
```

The actual generic parameter list is taken from the selected `GOROOT`
signature; the sketch abbreviates it. `Values` itself remains a synchronous
call because its result carries the selected callable ABI. The declaration has
no body. A future `gostdlib` implementation must export the same selected
profile name and satisfy that exact contract. An external function that invokes
a cooperative callback cannot acquire an async outer call by inference from
its type alone; its explicit provider implementation contract must select and
implement that effect.

The enclosing callable may be a package initializer rather than a declared
function:

```go
var Names = Map(values, func(value any) string { return value.(string) })
```

This use selects the synchronous `Map` profile and remains synchronous even
when another `Map` use supplies a cooperative callback. The other use selects
a separate, deterministic source-owned callable-profile variant. Only an
initializer whose own call selects such a cooperative variant receives the
reverse requirement: that call becomes `await`, its package `$initialize`
becomes `async`, and `program.ts` awaits that package before starting the next
Go-ordered package. The initializer is never assigned to a fabricated
function, and unrelated calls and package initializers remain byte-identical
and synchronous.

## Channels, Goroutines, And `select`

Channel syntax is projected from the exact `*types.Chan`:

```go
values := make(chan Box, 2)
values <- Box{Count: 1}
value, ok := <-values
```

The type owner first requires the explicitly selected `cooperative`
concurrency profile, then emits `GoChannel<Box> | undefined`; directional
source types emit the corresponding typed send or receive view. `make`
requests the channel-value owner and supplies capacity plus exact `Box`
zero/copy functions.
Send evaluates the channel and value in Go order, copies `Box` once, then
awaits the O(1) runtime send. Receive awaits one typed `[Box, boolean]` result;
single-result receive selects element zero, comma-ok consumes both elements,
and a discarded receive still performs the communication. `close` selects only
the predeclared builtin object and calls the runtime close owner. Channel
equality is direct canonical-identity equality, including nil.

```go
go worker(next(), value())
```

The statement owner emits prerequisites that evaluate and copy `worker`,
`next()`, and `value()` immediately in source order. It then passes one typed
closure to the scheduler. The closure invokes the captured callable directly
and awaits it only when its concrete source facet or exact-signature generated
callable ABI is cooperative. The ABI does not depend on the value's storage
location. No
`.call`, `.apply`, `.bind`, erased argument array, or runtime signature lookup
is emitted.

An immediately invoked function literal is not a transported function value.
The call owner selects the literal's exact callable facet, emits the literal
without a callable-ABI adapter, and awaits only when that literal facet is
cooperative. Restoring ABI routing at this syntactically static call is a
semantic and source-size regression.

```go
select {
case output() <- value():
    sent()
case item, ok := <-input():
    use(item, ok)
default:
    idle()
}
```

The select owner first captures `output()`, `value()`, and `input()` in source
order. It constructs typed send/receive alternatives. Because this example has
a default, it calls the channel owner's synchronous fair ready-choice/commit
operation: a ready communication returns its clause index, while no ready
communication selects the default. This selection adds no `Promise`, `async`,
`await`, scheduler request, or cooperative callable facet. If `output()`,
`value()`, or `input()` independently blocks, that operand's existing
cooperative requirement still propagates normally. A select without a default
uses the blocking Promise-returning selection operation and awaits it.
Receive assignment locations are not evaluated until the receive alternative
wins. One target switch dispatches the chosen clause. Every registered
alternative is canceled after one atomic commit, and nil alternatives cannot
become ready.

Blocking alternatives enter one fair registration permutation and then the
channel's ordinary live sender/receiver queues. A direct receive queued before
a selected receive therefore receives first, and the inverse order stays
inverse; senders obey the same rule. Two blocking selects on opposite sides of
one unbuffered channel rendezvous through those typed offers without polling.
Multiple alternatives of one select on the same channel are not registered in
source order. Closing a channel completes each selected send with the
send-on-closed panic at the selecting goroutine's Promise boundary, while the
goroutine executing `close` returns normally. Cancellation removes the exact
offer from its insertion-ordered set; no listener side table or historical
queue entry remains.

Channel range is a receive loop. Each iteration performs one receive;
`ok == false` exits before assignment and body execution. The received value is
already the channel owner's exact copy. Existing break, continue, label,
return, panic, method, function-value, interface, and generic owners remain
authoritative.

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
The canonical constant-value owner validates a checker-provided integer
against its Go carrier width and sign, then emits that decimal through the
selected target syntax. The `number` profile does not reject an otherwise
valid `int64` or `uint64` merely because its magnitude exceeds JavaScript's
safe-integer range; that profile emits a direct `NumericLiteral` and explicitly
accepts approximate wide-number behavior. The `bigint` profile emits the same
checker value as an exact `BigIntLiteral`. Neither route re-evaluates source
spelling or inserts routine casts, wrappers, or runtime helpers.

`float32` and `float64` both carry as the `number` alias regardless of the
integer-representation profile — TypeScript has one binary64 number type and no
separate float axis. A floating-point constant materializes its exact
`go/constant` value through the one constant-value owner, never the source
spelling: a `float64` constant emits the shortest decimal that round-trips to
its binary64 value; a `float32` constant is first rounded to its nearest
binary32 (as Go does at compile time) and then emitted as the binary64 that
equals that rounded value, so the emitted literal is bit-identical to Go's
`float32` result and never the shorter binary32 spelling, which would denote a
different `number`. A runtime `float32` operation rounds through a generated
helper at each Go-required boundary; a `float64` operation is direct.

`complex64` and `complex128` use distinct immutable GoToTS-owned nominal
classes. A bare object or tuple is insufficient because TypeScript would make
the two widths structurally interchangeable; an assertion-based brand is
forbidden. Each class therefore has a distinct private, declared-only brand,
readonly `real` and `imag` number components, a private constructor, and a
closed static construction surface. The brand emits no JavaScript. Construction
is the sole component-rounding boundary: `complex64` rounds both components
through `goFloat32`, while `complex128` preserves binary64 values. The runtime
module owns demand-driven, statically typed standalone operation functions;
unused operations are absent rather than widening each class or each call site.

```go
func Product(left, right complex64) complex64 {
    return left * right
}
```

```ts
export function Product(
    left: GoComplex64,
    right: GoComplex64,
): GoComplex64 {
    return goComplex64Multiply(left, right);
}
```

The classes own construction and readonly component access. Typed standalone
functions own unary negation, addition, subtraction, multiplication, equality,
and division. Complex division is not expanded at use sites: one shared runtime
operation ports the selected Go toolchain's
robust `runtime.complex128div` algorithm, including zero, infinity, and NaN
corrections. `complex64` division widens through that operation and rounds the
two result components on construction, matching the selected toolchain's
complex-division lowering. Every source operation is therefore one
constant-size typed call, and a complex runtime definition is emitted only
when its width is reached. Immutable complex objects need no copy operation:
sharing the object preserves Go value behavior because no component can be
mutated.

Imaginary and complex constants are read only from `go/constant`. Their real
and imaginary parts are materialized independently at the selected complex
width and passed through the same constructor as runtime values. Source token
spelling never determines either component. The predeclared `complex`, `real`,
and `imag` functions are selected only from their exact `*types.Builtin`
identity; `real` and `imag` become readonly component access, while `complex`
evaluates its two arguments once in order and calls the width-owned
constructor. A defined complex operand is unwrapped only at this operation
boundary; the raw complex carrier never absorbs nominal-type semantics.

### Explicit conversions

A `CallExpr` is a conversion only when the selected `go/types.TypeAndValue`
for its callee reports `IsType()`. The conversion owner reads the checker's
source type, destination type, and converted constant value. It never
recognizes a conversion from callee spelling and never lets an unimplemented
conversion fall through to ordinary-call emission.

The conversion owner delegates by exact source and destination representation.
Integer, floating-point, complex, byte-string, represented struct,
pointer-reinterpretation, and slice-to-array value/pointer conversions are
admitted here. Interface, channel, and generic conversions retain separate
semantic owners and fail typed until those owners land. Defined types project
only through their declared representation owner before conversion and wrap
only after the destination conversion succeeds.

Constant conversions materialize the converted checker value directly:

```go
func F() float32 { return float32(16777217) }
```

```ts
export function F(): float32 {
    return 16777216;
}
```

Non-constant conversions have one closed destination-family decision:

| Go conversion | TS-Go AST/output decision |
|---|---|
| lossless integer widening | the emitted operand directly |
| integer narrowing or sign change, `bigint` profile | `BigInt.asIntN` or `BigInt.asUintN` at the destination width |
| integer narrowing to 8/16/32 bits, `number` profile | one width-owned bitwise normalization |
| integer conversion requiring a 64-bit sign boundary, `number` profile | a demand-only number-to-BigInt boundary, static width normalization, then `Number` |
| integer to float | `Number` only from a BigInt carrier; `float32` additionally uses the shared binary32 rounder |
| float to integer | truncate once, select deterministic non-finite result zero, then apply the destination integer normalization |
| `float32` to `float64` | the already-rounded operand directly |
| `float64` to `float32` | the shared binary32 rounder |
| `complex64` to `complex128` or the reverse | construct the destination nominal value from its two components; the `complex64` constructor rounds both |
| integer to string | encode one Go rune as UTF-8 bytes, replacing an invalid scalar value with `RuneError` |
| `[]byte` to string or string to `[]byte` | copy the exact Go string bytes; nil slices produce the empty string and destination slices never alias |
| `[]rune` to string or string to `[]rune` | encode or decode UTF-8 with Go's invalid-sequence width and `RuneError` rules |
| represented struct to represented struct | call the destination class's one demand-owned structural conversion member; tags may differ, fields are copied recursively, and each use is O(1) |
| slice to array value | evaluate the slice once, panic when its length is short, allocate a fresh array, and copy the first destination-length elements |
| slice to pointer-to-array | evaluate the slice once, panic when its length is short, then return a canonical pointer to an offset-aware fixed-array view of the existing backing; writes alias both ways, defined arrays keep logical nominality over underlying array storage, and a zero-length nil slice alone yields nil |

The selected GoToTS implementation-defined result for a non-finite
floating-to-integer conversion is zero before width normalization. Finite
out-of-range values are truncated and reduced to the destination width. This
choice is explicit, deterministic, and non-panicking as required by Go; tests
do not misreport implementation-defined cases as a differential match to a
particular host Go backend.

Every conversion evaluates its operand once. Direct cases add no helper.
Only a conversion that must cross from a `number` carrier into BigInt
normalization requests the one standalone number-to-BigInt operation needed
to avoid duplicated evaluation and a host exception. This includes
floating-to-integer conversion in the `bigint` profile and 64-bit sign/width
boundaries in the `number` profile.
Conversion output remains O(1) per occurrence and no generic conversion
registry, erased carrier, cast, or source-text route exists.

String conversion treats a target JavaScript string as a sequence of Go bytes,
not UTF-16 text. The generated UTF-8 encoder/decoder is one typed runtime
owner; it uses neither host text codecs nor source spelling. For example,
`string([]byte{0xff, 'A'})` retains byte values `255, 65`, and
`[]rune("\xff")` yields one `RuneError`.

Represented struct conversion is definition-owned rather than expanded at the
use:

```go
type Left struct{ Pair [2]int32 `json:"left"` }
type Right struct{ Pair [2]int32 `json:"right"` }
right := Right(left)
```

```ts
class Right {
  static $convert(source: { Pair: GoArray<int32, 2> }): Right {
    return new Right(source.Pair.copy());
  }
}
const right = Right.$convert(left);
```

The structural parameter intentionally excludes the source's private nominal
brand. `go/types.ConvertibleTo` remains the admission authority; TypeScript
structural assignability does not decide whether conversion is legal.
Anonymous destinations receive the same demand on their canonical class.
Exactly one `$convert` definition exists per reached destination class,
regardless of source-use count.

For a discarded assignment, the assignment owner preserves Go's evaluation
rule. `_ = runtimeCall()` emits the call and discards its result, while
`_ = len([4]int{})` emits nothing because the checker proves the complete
expression constant and Go does not evaluate its operand. No fictitious blank
storage location is created.

### Ordered `max` and `min`

The predeclared `max` and `min` functions are selected only from their exact
`*types.Builtin` objects. Checker-folded calls materialize the result constant
directly. Runtime calls admit the currently represented predeclared integer,
floating-point, and byte-preserving string types; defined ordered types remain
with the defined-type family.

Number-carrier integers and floats use one `Math.max` or `Math.min` call.
Those host operations have the required NaN propagation, infinity behavior,
and `-0`/`+0` ordering. BigInt integers and byte strings use demand-only typed
pair operations folded left-to-right; the string operation uses `>=` for
`max` and `<=` for `min`, preserving the first equal argument and Go's
byte-wise order. A one-argument call is the argument directly.

All arguments evaluate once in Go order. If a child has prerequisite
statements, the owner captures the argument list at the call boundary before
forming the target operation. Use-site size is O(source arguments), each
runtime pair definition is emitted once, and neither implementation depends
on source spelling, arity-specific helpers, erased values, or runtime type
dispatch.

Ordinary integer syntax is source-shaped under both initial profiles:

| Go source | TS-Go AST decision | Printed TypeScript |
|---|---|---|
| `left + right` | direct addition | `left + right` |
| `left - right` | direct subtraction | `left - right` |
| `left * right` | direct multiplication | `left * right` |
| `value += delta` | direct compound assignment | `value += delta` |
| `value++` / `value--` | direct update | `value++` / `value--` |
| ordered/equality comparison or expression switch | direct comparison/switch | `left < right`, `switch (value)` |

Constant and variable shifts are selected by the exact left carrier and
checker-typed count. A negative variable count enters `GoPanic.raise`; a count
at least the left width yields zero, except signed right shift yields the sign
fill. The `number` profile emits direct JavaScript bitwise and shift operators.
Its 8/16/32-bit carriers normalize at the selected width; wider carriers
intentionally inherit JavaScript's 32-bit bitwise coercion as part of the
declared approximate number contract. The `bigint` profile uses
`BigInt.asIntN`/`asUintN` and never emits unsigned BigInt `>>>`. Generic shift
capabilities delegate this same integer owner.

A defined integer shift count remains nominal everywhere except the operation
boundary. The count expression is emitted once at its declared type and the
shift owner projects its existing readonly payload before the direct operator:

```go
type ShiftCount uint8
func Shift(value int64, count ShiftCount) int64 { return value << count }
```

```ts
export function Shift(value: int64, count: ShiftCount): int64 {
    return value << count.$value;
}
```

The owner neither rejects the legal Go count nor inserts a target cast,
spelling check, or second conversion path.

Integer division and remainder are the bounded exception to direct operators.
JavaScript BigInt already truncates toward zero, while JavaScript `number`
division does not; both host carriers also throw or produce a host value rather
than the selected Go panic on division by zero. The integer-expression owner
therefore requests one of two profile-specific, constant-size runtime
operations:

```go
quotient := left / right
remainder := left % right
```

```ts
const quotient = goIntegerDivide(left, right);
const remainder = goIntegerRemainder(left, right);
```

BigInt helpers check `right === 0n`, enter the shared
`GoPanic.raiseRuntime` ABI, and otherwise perform exactly one `/` or `%`.
Number helpers check `right === 0`, enter the same panic ABI, then use
`Math.trunc(left / right)` or direct `left % right`, and normalize a `-0`
result to integer zero. Expressions and compound assignments request the same
runtime symbols; no per-site overflow framework or alternate assignment path
exists.

The `number` profile prints ordinary numeric literals such as `1`; the
`bigint` profile prints `1n`. Contextual parameter, result, field, and
package-boundary binding types carry aliases. A literal is not routinely
wrapped in `as int32`, `as int64`, or `as bool`, and an initialized local
binding omits a type annotation when its initializer already makes the target
type exact.

Neither initial profile reproduces implicit fixed-width overflow. The default
also accepts JavaScript-number precision and 32-bit host coercion for wider
bitwise and shift operations as its declared integer contract. The BigInt
override removes those limitations but still does not implicitly narrow after
arithmetic. Evidence names the profile and never claims these deferred
semantics. Explicit narrowing conversions and any future fixed-width profile
are separate construct families rather than baggage in ordinary
multiplication.

Boolean `&&` and `||` emit direct binary expressions when the right operand has
no prerequisite statements. Prerequisites of the always-evaluated left operand
remain immediately before the expression. If the right operand has
prerequisites, the binary owner emits one boolean temporary and places the
right prerequisites inside the selected branch:

```go
left() && ([2]int32{1, 2} == right())
```

```ts
let result = left();
if (result) {
  const rightArray = right();
  // bounded generated array comparison
  result = arraysEqual;
}
```

The `||` branch executes only when the left result is false. Moving right-side
work before the branch or evaluating either operand twice is forbidden.

The predeclared `true` and `false` are literal booleans, not source-declared
constants, so they materialize in place through the one constant-value owner.
The parent supplies the expected checker boolean type, including a defined type
such as `type Flag bool`; the identifier handler verifies semantic object
identity, assignability, and the checker-provided underlying `bool`. The value
owner emits `false` for `bool` and the exact nominal construction for `Flag`.
The basic-type owner supplies the one generated `bool` alias where a declaration
boundary needs it. A source-declared untyped boolean constant is projected like
any other untyped constant, not inlined. No operator handler guesses a carrier
from spelling.

An explicit Go parenthesized expression becomes one TS-Go
`ParenthesizedExpression` around the directly emitted child. It preserves
source grouping without creating a source-side wrapper or intermediate
expression model.

Shifts, bitwise operators, and explicit conversions remain separate
profile-specific construct families. Their admission does not authorize a
fallback operator in the parent binary dispatcher.

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

### Value-Foundation Representations

Milestone 3A uses bounded representations rather than pretending native
JavaScript values already implement Go.

Go string values are stored as TypeScript strings whose code units are Go
bytes. For example, Go `"é"` contains bytes `c3 a9`, so its target semantic
literal contains code units `0x00c3, 0x00a9`; target `.length` is therefore the
Go byte length. Concatenation and equality are direct. Indexing and slicing use
one demanded string runtime owner for bounds behavior:

```go
value := "é"
return value[1], value[:1]
```

```ts
const value: gostring = "\u00c3\u00a9";
return [goStringIndex(value, 1), goStringSlice(value, 0, 1)];
```

An admitted fixed array retains its length in the target type. A `[2]int32`
cannot become interchangeable with `[3]int32`. Array zero and copy create
fresh storage; an indexed store mutates only that storage. Equality is lowered
at the source expression into one fixed-bound loop whose element comparison is
selected statically from the exact element type. Both operands are evaluated
once, nested comparisons compose recursively, and the loop stops at the first
unequal element. The generic array runtime carries no equality callback,
semantic tag, or implementer switch; comparison source size is independent of
the array length. Aggregate zero, literal, and copy follow the same rule: the
array operation owner emits a bounded loop that directly invokes the selected
element zero or copy structure. The runtime array exposes only typed storage
allocation and access. It never receives or stores an element-zero or
element-copy function.

Array and pointer-to-array slicing evaluates the operand and bounds once,
obtains the array's canonical backing plus offset, and creates one slice
descriptor over that same region. For:

```go
array := [4]int32{1, 2, 3, 4}
view := (&array)[1:3:4]
view[0] = 9
```

the target decision is a demand-only `goArrayLocation(array)` followed by one
typed slice view and the ordinary slice bounds operation; `array[1]` becomes
`9`. No element is copied and no second bounds implementation exists.

Composite-literal element handlers use the checker-selected expression type,
not the presence of an explicit child type node. Thus an elided nested literal
such as `[][]Box{{{Value: 1}}}` is dispatched as a `[]Box` literal because
`go/types.Info.TypeOf` supplies that contextual type. The handler must not
fabricate source syntax, rediscover the parent by scanning, or reject a valid
literal merely because `CompositeLit.Type` is absent.

Composite family selection likewise uses the checker type's underlying family.
For `type Table map[int32]int32`, `Table{0: 5}` enters the one map-literal owner,
constructs the canonical map storage once, and applies the nominal `Table`
wrapper. It does not fall through to struct classification or acquire a
call-argument-specific path.

An admitted slice is a descriptor over backing storage:

```text
backing identity + offset + length + capacity + nil state
```

Copying a slice copies the descriptor and aliases the backing storage.
Slicing changes descriptor bounds. Append reuses capacity or allocates and
copies according to Go behavior. Representing a slice as a bare `T[]` is
forbidden because it cannot distinguish nil, preserve capacity, or model
subslice append aliasing.
Runtime methods narrow nullable backing storage through explicit branches.
They do not emit TypeScript non-null assertions to recover a storage invariant.
The descriptor stores no zero/copy/equality/hash strategy and no function
whose behavior depends on the element type. For aggregate elements, the
source `make`, literal, `append`, `copy`, and `clear` owners emit typed loops
that directly construct or copy each element. Runtime slice members are
limited to descriptor validation, allocation, bounds, backing access, and
capacity growth. This preserves recursive fresh-zero and deep-copy behavior
without semantic callbacks or per-element-type runtime specialization.

`append(left, right...)` consumes the right descriptor through indexed reads;
it never expands the slice into a JavaScript argument list. Distinct named
slice types are accepted when Go accepts their identical element types.
`append([]byte, string...)` is the one language-defined string expansion and
copies the string's Go bytes. The string-special `append` and `copy` owners
supply the concrete predeclared `string` expectation for an untyped string
operand and preserve the exact defined string type for a defined operand. The
constant-use owner never defaults an untyped operand independently. Reallocation
copies existing aggregate elements
with their selected copy owner; reuse preserves the existing backing identity.
`clear(slice)` writes a fresh zero at every live aggregate element, while
`clear(map)` removes entries. The slice clear and slice-spread helpers are
demanded only by source uses. Every admitted map representation implements one
closed `GoMapValue<K, V>` source contract containing lookup, store, delete,
length, nil, clear, and key iteration. Source map operations call those typed
members directly; no separate `goMapClear`/`goMapKeys` helper or
operation-specific map specialization exists. This complete contract is
necessary because a generic map value must remain statically substitutable
across scalar and generated aggregate-key implementations.

An admitted map is a reference value with an explicit nil state. Map assignment
aliases the same map. Lookup of a missing key returns the element zero;
comma-ok additionally returns `false`; storing through nil fails. A plain
object literal is forbidden because it changes key identity and prototype
behavior.

The current native-`Map` key family is exactly `bool`, represented integer, or
string, plus defined-basic wrappers whose exact checker type unwraps to one of
those primitives before every literal/store/lookup/delete operation. The map's
target key type is the underlying primitive while its Go signature retains the
defined wrapper at source boundaries:

```go
type Count int32
func Lookup(values map[Count]string, key Count) string {
	return values[key]
}
```

```ts
export function Lookup(values: GoMap<int32, gostring>, key: Count): gostring {
  return values.lookup(key.$value);
}
```

Floating keys remain a typed boundary: JavaScript `Map` uses SameValueZero,
while Go permits distinct stored NaN keys that cannot be retrieved by another
NaN comparison. Interface keys remain a typed boundary until interface dynamic
value identity and hashing exist. Admitted aggregate comparable keys and
admitted non-basic values use the exact specialized owner below; neither class
may silently use object identity or serialization.

Specialized maps select key and value semantics once from the authoritative
`go/types` map type. One canonical target artifact per exact represented map
shape directly references the key family's static hash, equality, and copy
operations and the value family's zero and copy operations. This includes an
aggregate key or any value whose Go copy/zero behavior cannot be represented by
the scalar native-`Map` owner. The map value does not store function-valued
strategies, closures, operation objects, tags, or dynamic member names. A
shared runtime primitive may own only non-semantic bucket, count, and nil
storage; the canonical map-shape artifact owns typed lookup, store, and delete
behavior and invokes the selected operations directly.

```go
type Key struct{ X int32 }
var values map[Key]string
```

```ts
// Schematic shape; production output is constructed as TS-Go AST.
class map$Key$string {
  static lookup(storage: GoMapBuckets<Key, gostring>, key: Key): gostring {
    const bucket = storage.bucket(Key.$hash(key));
    // The generated owner calls Key.$equal directly while resolving collisions.
    return map$Key$string.lookupBucket(bucket, key);
  }
}
```

The declaration cost is `O(map shape)` once, while every map operation site is
constant-size. Two map values of the same exact semantic shape reuse the same
owner. A full deterministic canonical-type digest indexes one artifact, but
`go/types` identity remains authoritative and a digest collision must never
unify non-identical map types. The digest uses semantic package/declaration
identity; a local named component additionally carries its canonical enclosing
`ArtifactOwner` and exact lexical scope-child path. Source positions, source
paths, and generated TypeScript names never feed back into identity.
Generated-artifact tests reject function-valued key-operation fields and
mutations that restore callback storage.

A specialization whose exact type contains no function-local named component
is one compilation support module under `support/maps/`. A specialization that
mentions a local named key or value cannot escape that name's scope: it is
inserted as typed TS-Go class AST immediately after the deepest local type
declaration required by the shape. This rule applies inside nested blocks,
function literals, and package initializer expressions. Package initializer
placement is owned by the exact `types.Initializer`, not by an arbitrarily
chosen LHS variable. Unreachable shapes create no target name, file, or class.

An admitted pointer is a typed reference to one canonical storage location.
`new(T)` creates a fresh cell when `T` has a complete admitted value
representation including its Go zero value; assignment copies the pointer;
dereference reads or writes the selected location; nil is distinct; and
equality compares canonical address identity. The admitted matrix currently
includes primitives, pointers, scalar slices, scalar maps, fixed arrays, and
named structs. Callable values do not yet admit their nil state, so
`new(func(...))` fails at the callable-zero boundary rather than injecting
`undefined` into an otherwise non-nil function contract.

Address formation is context-directed:

```go
value := Box{Count: 1}
pointer := &value.Count
value = Box{Count: 7}
*pointer++
```

The enclosing reconstructible artifact first emits an addressable-storage
requirement for the exact `value` object. For an ordinary declaration that
artifact is the source function; for a literal in a package initializer it is
the exact checker-produced initializer. Its reconstructed TS-Go body has one
cell:

```ts
const value$storage = GoPointer.cell<Box, Box$Storage>(
  Box.$storageOf(Box.$make(1)),
);
const pointer = GoPointer.field<int32, Box, Box$Storage, "Count">(
  value$storage,
  "Count",
);
value$storage.value = Box.$storageOf(Box.$make(7));
GoPointer.dereference(pointer).value++;
```

The field pointer follows the variable's storage after whole-value assignment,
as Go requires. Unrelated locals remain ordinary variables. Address requests
inside nested function literals belong to the enclosing top-level function
artifact but the selected variable's lexical cell remains in its own scope, so
closures capture the same cell.

If the outer boundary is a package initializer, an address request inside its
function literal belongs to that package-initializer artifact instead. It
never fabricates a `*types.Func` for the literal or attaches the requirement to
the package variable.

The same address owner handles:

- locals, parameters, named results, receivers, and package-state fields;
- direct and pointer-indirect named-struct fields;
- fixed-array elements by projecting from the addressable array root;
- slice elements by canonical backing identity plus absolute index, so
  `&slice[0] == &slice[:][0]`; and
- `&*pointer`, which evaluates the pointer once, performs Go's required nil
  dereference check, and returns the same canonical location without reading
  the stored value.

Map and string indexes remain non-addressable because Go declares them so.
Unsupported aggregate representations fail before requesting storage.
Address/index operands are evaluated once in Go order. Repeated formation of
the same live location yields equal pointers; different cells with equal
values do not.

Slice-to-array pointers use the same location rule:

```go
type Pair [2]int32

values := []int32{1, 2, 3}
pair := (*Pair)(values)
pair[0] = 7
*pair = Pair{8, 9}
```

```ts
const pair: GoPointer<Pair, GoArray<int32, 2>> | undefined =
  goSliceArrayPointer<Pair, int32, 2>(values, 2);
GoPointer.index(GoPointer.dereference(pair), 0).value = 7;
GoPointer.dereference(pair).value = new Pair(
  GoArray.literal(2, 0, [0, 1], [8, 9]),
).$value;
```

`pair` and `values` share the same backing. Repeating the conversion at the
same slice offset compares equal; converting a subslice at another offset does
not. The `Pair` wrapper gains no forwarding methods: pointer storage is the
underlying `GoArray<int32, 2>`.

Each runtime call remains constant-size. Element/key/value representations are
selected from the same `go/types` evidence. Pointer projection factories may
retain only typed location accessors plus opaque canonical address identity;
runtime objects never carry erased `any`/`unknown` payloads or callbacks that
rediscover Go semantics.

Pointer representation has separate logical and storage type arguments:

```go
type Left struct {
    Value int32 `json:"left"`
}
type Right struct {
    Value int32 `json:"right"`
}

func Convert(value *Left) *Right {
    return (*Right)(value)
}
```

The selected checker proves that `Left` and `Right` have identical underlying
base types when tags are ignored. Their logical classes remain distinct, but
both storage facets are the same `{ Value: int32 }` shape:

```ts
class Left {
  private constructor(private readonly $storage: Left$Storage) {}
  static $make(Value: int32): Left {
    return new Left({ Value });
  }
  static $storageOf(value: Left): Left$Storage {
    return value.$storage;
  }
  static $fromStorage(storage: Left$Storage): Left {
    return new Left(storage);
  }
  get Value(): int32 { return this.$storage.Value; }
  set Value(value: int32) { this.$storage.Value = value; }
}

function Convert(
  value: GoPointer<Left, Left$Storage> | undefined,
): GoPointer<Right, Right$Storage> | undefined {
  return GoPointer.view<Left, Right, Left$Storage>(value);
}
```

`Right.$fromStorage(GoPointer.dereference(result).value)` therefore observes
and mutates the original `Left` storage. No pointee copy, cast, object-shape
test, semantic callback, or source-name lookup is involved. A pointer
conversion whose canonical storage facets cannot yet be represented fails at
the conversion owner rather than falling back to logical read/write adapters.

Pointer receiver declarations remain named typed receiver functions rather
than class members:

```go
func (box *Box) Add(delta int32) { box.Count += delta }
```

```ts
export function Box_Add(
  box: GoPointer<Box, Box$Storage> | undefined,
  delta: int32,
): void {
  const value = Box.$fromStorage(GoPointer.dereference(box).value);
  value.Count += delta;
}
```

`pointer.Add(1)` passes the pointer directly, including nil; the body panics
only if it dereferences nil. `value.Add(1)` is admitted only when `value` is
addressable and passes its selected cell. Calling a value-receiver method
through a pointer nil-checks, dereferences, and copies once. Selection uses the
exact `go/types.Selection`; target virtual dispatch is never substituted.

Bounds failures, nil pointer dereference, nil map store, and BigInt
divide-by-zero all enter the one generated non-generic `GoPanic` carrier. Family
runtime modules import that carrier through the closed runtime dependency
graph; they never throw `Error`/`RangeError` independently.
The function-control family adds source `panic`/`recover` and preserves each
runtime fault's represented dynamic value. A recovered `panic(nil)` therefore
asserts to `*runtime.PanicNilError`; a generic runtime fault does not. Both
retain the exact canonical error-method contracts.

Array, slice, and map indexed stores use the same accessor-store transaction.
For:

```go
values[nextIndex()] = chooseSecond(resultPair())
```

the store-target owner returns the receiver, getter, setter, and index/key
expressions. The assignment owner evaluates and captures only operands that
would otherwise move across prerequisite statements from `resultPair`, then
emits one `.set(index, value)` or `.store(key, value)` call. A compound defined
value uses the same captured location for one later getter and one setter. The
ordinary store remains a direct call with no temporary. A map-specific
assignment route, write-only target model, or unconditional capture policy is
forbidden.

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

The language stage does not guess an implementation for a bodyless source
function. Its callable owner records the exact selected `*types.Func`,
`*types.Signature`, source role, and position in a typed unresolved-obligation
diagnostic. The environment stage later converts that identity into an exact
declaration and throwing placeholder or a selected provider implementation.
Reachable placeholders block publication. Manual completion replaces bodies
or declarations through structural typed TS-Go protocol ownership, never
textual patches or per-file ownership.

The predeclared `print` and `println` functions are implementation-defined by
Go. Their call owner selects them only by exact `*types.Builtin` identity and
returns a typed environment-boundary diagnostic in ordinary and deferred call
contexts. The language stage does not silently map them to `console`, stdout,
or stderr. A later selected environment contract may install one behavior and
its differential proof.

`unsafe.Pointer` is a nullable nominal environment-boundary type, not a
primitive alias or an erased payload. Pointer-to-unsafe and unsafe-to-pointer
conversions preserve `nil` exactly. A compile-only environment profile emits
typed static conversion calls whose non-nil paths throw one explicit unresolved
placeholder because ordinary TypeScript cannot reinterpret Go storage. The
placeholder never uses `any`, `unknown`, a cast, source spelling, or a fabricated
memory model. `uintptr` conversions and other `unsafe` operations remain
separate closed dispositions until their own exact environment contracts are
selected. A reachable non-nil unsafe placeholder blocks publication.

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
