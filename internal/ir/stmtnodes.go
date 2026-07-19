// Statement nodes of the IR: one Go statement form each, built
// fail-closed by the body builders and printed by the emitter.
package ir

import "go/token"

// Stmt is one Go statement in IR form.
type Stmt interface{ stmt() }

// Block is an ordered statement list with its own lexical scope.
type Block struct {
	Stmts []Stmt
}

// DeclStmt declares one or more variables with explicit initial values
// (zero values are materialized during build, never implied).
type DeclStmt struct {
	Names  []string
	Types  []Type
	Values []Expr // len == len(Names); multi-value via Tuple
	// Tuple, when set, is a single multi-valued expression (call or
	// comma-ok lookup) initializing all names simultaneously; Values is
	// nil in that case.
	Tuple Expr
	// Reused marks names a := statement reassigns instead of declaring
	// (Go permits existing names alongside at least one new one); nil
	// means every name is new.
	Reused []bool
	// Boxed marks tuple-bound names whose address is taken: the emitter
	// declares their stable cells directly off the tuple slot.
	Boxed []bool
}

// AssignStmt stores values into existing locations. Go's two-phase rule is
// preserved: target operands and right-hand values are evaluated in source
// order first, then every store happens.
type AssignStmt struct {
	Targets []Target
	Values  []Expr
	// Tuple, when set, is a single multi-valued expression assigned to all
	// targets simultaneously; Values is nil in that case.
	Tuple Expr
}

// TryFinally is the lowered form of function-top-level defers: Finally
// runs on every exit of Body, in Go's LIFO order via nesting.
type TryFinally struct {
	Body    *Block
	Finally *Block
}

// IfStmt is a Go if with optional init statement.
type IfStmt struct {
	Init Stmt // nil or DeclStmt/AssignStmt
	Cond Expr
	Then *Block
	Else Stmt // nil, *Block, or *IfStmt
}

// ForStmt is the classic three-clause loop (all clauses optional).
type ForStmt struct {
	Init Stmt
	Cond Expr // nil means forever
	Post Stmt
	Body *Block
}

// ReturnStmt returns the complete result list (bare returns are resolved
// or rejected during build).
type ReturnStmt struct {
	Values []Expr
	// CallValue, when set, forwards a single multi-result call.
	CallValue Expr
}

// ExprStmt evaluates a call for effect.
type ExprStmt struct {
	Call Expr // *Call or *MethodCall
}

// MapDeleteStmt is the delete builtin (a no-op on nil maps).
type MapDeleteStmt struct {
	Map Expr
	Key Expr
}

// MapClearStmt is the clear builtin on a map (a no-op on nil maps).
type MapClearStmt struct {
	Map Expr
}

// PanicStmt is the panic builtin: the value formats as Go's %v (the
// reviewed argument kinds format identically on both sides) and unwinds
// as the GoPanic carrier.
type PanicStmt struct {
	Value Expr
	// ErrorFormat, when set, is the closed token-switch dispatch of the
	// value's dynamic Error method (the %v of an error), evaluated
	// lazily so a deferred mutation is reflected. The Recv of this
	// IfaceCall reads the panic value.
	ErrorFormat *IfaceCall
}

// ParamRef is a reference to a generated binding parameter (an adapter
// arrow's own parameter) with a known type. Its name must be a bare
// generated identifier — it can never carry arbitrary expression text,
// so no unverified TypeScript enters the IR.
type ParamRef struct {
	Name string
	T    Type
}

func (*ParamRef) expr()        {}
func (e *ParamRef) Type() Type { return e.T }

// TupleSpread forwards a multi-result call's values as the complete
// argument list of an enclosing call (Go's f(g()) form): the results
// spread positionally in evaluation order.
type TupleSpread struct {
	X Expr // the multi-result call
	T Type
}

// TupleVariadicSpread is Go's f(g()) form where f ends in a variadic
// parameter: the inner call's fixed-count leading results bind the
// regular parameters and every remaining result is packed into the final
// slice (the spec's "the return values that remain"). The inner value
// evaluates exactly once, then the fixed slots forward positionally and
// the rest form the variadic slice.
type TupleVariadicSpread struct {
	X         Expr   // the multi-result call, already per-slot adapted
	SlotTypes []Type // the adapted result's slot types, in order
	Fixed     int    // number of regular (non-variadic) parameters
	Elem      Type   // the variadic element type
	SliceType Type   // the variadic parameter's slice type
	T         Type
}

// CompoundStmt is x op= y (and x++/x--): the target's operands stage
// exactly once, the right-hand side evaluates, the staged location
// loads, the operation applies, and the staged location stores — Go's
// single-evaluation rule for every admissible target shape.
type CompoundStmt struct {
	Target   Target
	Op       token.Token
	Rhs      Expr
	OperandT Type
}

// StmtSeq is a flat statement sequence spliced into the surrounding
// block without introducing a lexical scope (used by lowerings that
// expand one source statement into several).
type StmtSeq struct {
	Stmts []Stmt
}

// DeferPush pushes one deferred call — its receiver and arguments
// already captured at the defer site — onto the function's defer stack;
// the stack drains LIFO at every function exit.
type DeferPush struct {
	Call Expr
}

// RangeInt is `for i := range n`: n is evaluated once and i counts from
// zero to n-1 in n's own carrier. Index may be empty (discarded).
type RangeInt struct {
	Index string
	N     Expr
	Body  *Block
}

// BranchStmt is a break or continue, optionally labeled: Go's labeled
// loop/switch branches coincide exactly with JS labeled statements.
type BranchStmt struct {
	Tok   token.Token // token.BREAK or token.CONTINUE
	Label string      // "" when unlabeled
}

// LabeledStmt labels a loop or switch as a branch target.
type LabeledStmt struct {
	Label string
	Stmt  Stmt
}

// SwitchStmt is a Go expression switch. The tag is evaluated once (bool
// true for tagless switches); clauses match in source order by the tag
// carrier's exact equality; at most one clause without values is the
// default. A clause with Fallthrough transfers into the next clause's
// body unconditionally.
type SwitchStmt struct {
	Init    Stmt // nil or DeclStmt/AssignStmt
	Tag     Expr
	Clauses []SwitchClause
}

// SwitchClause is one case (or default) clause with its own scope.
type SwitchClause struct {
	Values      []Expr // nil for the default clause
	Body        *Block
	Fallthrough bool
}

// TypeSwitchStmt is a Go type switch, lowered onto rtti identity tests
// in clause order. Bind, when set, names the per-clause variable: a
// single concrete clause binds the unboxed value in that type; every
// other clause binds the interface value itself.
type TypeSwitchStmt struct {
	Init    Stmt
	X       Expr
	Clauses []TypeSwitchClause
	// BreakLabel, when set, names the labeled block a direct break
	// inside a clause exits (the if/else lowering has no native break).
	BreakLabel string
}

// TypeSwitchClause is one clause of a type switch. Targets is nil for
// the default clause; a target with Nil set matches the nil interface.
type TypeSwitchClause struct {
	Targets []TypeSwitchTarget
	// Bind is this clause's own binding name ("" when the switch has no
	// guard). Each clause's implicit variable is a DISTINCT Go object in a
	// disjoint scope, so it carries its own canonical identity and unique
	// name — the binding never conflates several clause variables.
	Bind string
	// BindType is the clause variable's type: the concrete target for a
	// single-type clause, the switch operand's interface type otherwise.
	BindType Type
	Body     *Block
}

// TypeSwitchTarget is one matched type (or nil) in a type-switch clause.
type TypeSwitchTarget struct {
	Nil    bool
	Rtti   RttiRef
	Target Type
}

func (*Block) stmt()               {}
func (*SwitchStmt) stmt()          {}
func (*TypeSwitchStmt) stmt()      {}
func (*RangeSlice) stmt()          {}
func (*RangeInt) stmt()            {}
func (*TryFinally) stmt()          {}
func (*MapDeleteStmt) stmt()       {}
func (*MapClearStmt) stmt()        {}
func (*PanicStmt) stmt()           {}
func (*DeclStmt) stmt()            {}
func (*AssignStmt) stmt()          {}
func (*IfStmt) stmt()              {}
func (*ForStmt) stmt()             {}
func (*ReturnStmt) stmt()          {}
func (*ExprStmt) stmt()            {}
func (*BranchStmt) stmt()          {}
func (*LabeledStmt) stmt()         {}
func (*CompoundStmt) stmt()        {}
func (*DeferPush) stmt()           {}
func (*StmtSeq) stmt()             {}
func (*TupleSpread) expr()         {}
func (*TupleVariadicSpread) expr() {}
