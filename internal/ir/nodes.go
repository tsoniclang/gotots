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
	Bind    string
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

// Expr is one Go expression in IR form with its resolved type.
type Expr interface {
	expr()
	Type() Type
}

// Const is an exact folded constant (from go/constant evidence).
type Const struct {
	T Type
	// Value is the exact decimal (integers), shortest round-trip decimal
	// (floats), Go-quoted text (strings), or "true"/"false" (bools).
	Value string
}

// VarRef reads a variable. Pkg is empty for locals and parameters; for
// package-level variables it names the declaring package, and reads
// from other unit packages go through the (live) ESM namespace binding.
type VarRef struct {
	Name string
	Pkg  string
	T    Type
}

// Binary is one binary operation with exact operand/result semantics.
type Binary struct {
	Op   token.Token
	L, R Expr
	T    Type
}

// Unary is one unary operation.
type Unary struct {
	Op token.Token // token.SUB, token.NOT, token.XOR
	X  Expr
	T  Type
}

// Convert is an explicit Go conversion between supported types.
type Convert struct {
	X  Expr
	To Type
}

// Call invokes a package-level function of the translated unit directly.
// TypeArgs, when set, are the explicit type arguments of an instantiated
// generic call.
type Call struct {
	Pkg      string // Go package path declaring the callee
	Callee   string // generated symbol name
	TypeArgs []Type
	Args     []Expr
	Results  []Type
}

// ExternalCall invokes an external package-level function through its
// generated typed stub: a runtime-fail-closed contract the emulation
// layer supplies behavior for.
type ExternalCall struct {
	Pkg      string
	Name     string
	TypeArgs []Type
	Args     []Expr
	Results  []Type
}

// MethodCall invokes a statically resolved method, generated as a
// package-level function over its receiver. A nil pointer receiver flows
// into a pointer-receiver method exactly as in Go (the body's own
// dereferences carry the panics); a value receiver dereferences a
// pointer caller at the call and clones inside the method.
type MethodCall struct {
	Pkg         string // Go package path declaring the method
	TypeName    string // receiver's named type
	PointerRecv bool   // the declared receiver is a pointer
	// TypeArgs are the receiver's type arguments for methods of generic
	// types, spelled explicitly so inference can never change types.
	TypeArgs []Type
	Recv     Expr
	Method   string
	Args     []Expr
	Results  []Type
}

// FieldLoad reads a struct field through a nil-checked pointer.
type FieldLoad struct {
	X     Expr
	Field string
	T     Type
}

// Closure is a function literal: a JS arrow function capturing enclosing
// variables by reference, exactly as Go captures them.
type Closure struct {
	Params  []Var
	Results []Var
	Body    *Block
	// UsesDeferStack marks a closure body with defers below its top
	// level (see Func.UsesDeferStack).
	UsesDeferStack bool
	T              Type // the KindFunc type
}

// FuncRef references a package-level function of the translated unit as
// a first-class value.
type FuncRef struct {
	Pkg  string
	Name string
	T    Type // the KindFunc type
}

// DynCall invokes a function value (a nil value panics at the call, as
// in Go).
type DynCall struct {
	Fun     Expr
	Args    []Expr
	Results []Type
}

// TypeAssert is x.(T) for a concrete target: identity comparison against
// the target's rtti. The panic form carries the static source interface
// spelling for Go's exact message; the comma-ok form is consumed through
// tuple slots.
type TypeAssert struct {
	X             Expr
	Target        Type
	Rtti          RttiRef
	SourceDisplay string
	TargetDisplay string
	CommaOk       bool
}

// StructNew allocates a struct on the heap (&T{...}) with every field
// value in declaration order (omitted fields are explicit zeros).
// EvalOrder, when set, lists the Args indexes of the provided values in
// their SOURCE order: a keyed literal whose source order differs from
// field order stages its values so evaluation stays exact.
type StructNew struct {
	Pkg       string // Go package path declaring the struct type
	TypeName  string
	Args      []Expr
	EvalOrder []int
	T         Type // the pointer type
}

// StructCopy is Go's value copy at a binding site: a deep clone along the
// value-struct spine of one struct instance.
type StructCopy struct{ X Expr }

// AddrOf is &x on an addressable struct value: the pointer is the very
// instance (whole-value stores are in place, so the alias stays exact).
type AddrOf struct {
	X Expr
	T Type // the pointer type
}

// Deref is *p: the nil-checked pointee instance as a struct value (the
// copy, when the context binds it, happens at the binding site).
type Deref struct {
	X Expr
	T Type // the pointee struct type
}

// FieldCell is &s.f on a struct field whose type is NOT an identity
// carrier (a map, slice, scalar, pointer, interface, or function): the
// pointer is a proxying cell that reads and writes the live field, so an
// aliasing write through the pointer (the *p = make(...) lazy-init
// idiom) mutates the very field. The struct base is evaluated exactly
// once at the address-of; the cell closes over that single reference.
type FieldCell struct {
	Base  Expr   // the addressable struct base (a value or a pointer to one)
	Field string // the Go field name accessed
	Elem  Type   // the field's type (the pointee)
	T     Type   // the pointer type (result)
}

// StructZero is the zero value of a named struct type: a fresh instance
// with every field zeroed.
type StructZero struct{ T Type }

// NilConst is a typed nil (pointer or map zero value).
type NilConst struct{ T Type }

// IsNil compares a pointer or map against nil.
type IsNil struct {
	X      Expr
	Negate bool
}

// RangeSlice is `for index, value := range s` over a slice: the range
// expression and its length are evaluated once, and each element is
// loaded per iteration. Index or Value may be empty (discarded).
type RangeSlice struct {
	Index string
	Value string
	VarT  Type // element type
	X     Expr
	Body  *Block
}

func (*Const) expr()            {}
func (*MethodCall) expr()       {}
func (*FieldLoad) expr()        {}
func (*StructNew) expr()        {}
func (*StructCopy) expr()       {}
func (*StructZero) expr()       {}
func (*AddrOf) expr()           {}
func (*Deref) expr()            {}
func (*FieldCell) expr()        {}
func (*Closure) expr()          {}
func (*FuncRef) expr()          {}
func (*DynCall) expr()          {}
func (*ExternalCall) expr()     {}
func (*IfaceBox) expr()         {}
func (*IfaceCall) expr()        {}
func (*TypeAssert) expr()       {}
func (*NilConst) expr()         {}
func (*IsNil) expr()            {}
func (*MapMake) expr()          {}
func (*MapFrom) expr()          {}
func (*MapGet) expr()           {}
func (*MapLookup) expr()        {}
func (*MapLen) expr()           {}
func (*StringLen) expr()        {}
func (*SliceLit) expr()         {}
func (*SliceMake) expr()        {}
func (*SliceGet) expr()         {}
func (*SliceReslice) expr()     {}
func (*SliceAppend) expr()      {}
func (*SliceAppendSlice) expr() {}
func (*SliceCopy) expr()        {}
func (*SliceLen) expr()         {}
func (*SliceCap) expr()         {}
func (*VarRef) expr()           {}
func (*Binary) expr()           {}
func (*Unary) expr()            {}
func (*Convert) expr()          {}
func (*Call) expr()             {}

func (c *Const) Type() Type   { return c.T }
func (v *VarRef) Type() Type  { return v.T }
func (b *Binary) Type() Type  { return b.T }
func (u *Unary) Type() Type   { return u.T }
func (c *Convert) Type() Type { return c.To }

// Type of a call is its single result; multi-result calls are consumed
// only through DeclStmt/AssignStmt/ReturnStmt tuple slots.
func (c *Call) Type() Type {
	if len(c.Results) == 1 {
		return c.Results[0]
	}
	return Type{}
}

func (m *MethodCall) Type() Type {
	if len(m.Results) == 1 {
		return m.Results[0]
	}
	return Type{}
}

func (c *Closure) Type() Type  { return c.T }
func (f *FuncRef) Type() Type  { return f.T }
func (b *IfaceBox) Type() Type { return b.T }

func (d *DynCall) Type() Type {
	if len(d.Results) == 1 {
		return d.Results[0]
	}
	return Type{}
}

func (c *IfaceCall) Type() Type {
	if len(c.Results) == 1 {
		return c.Results[0]
	}
	return Type{}
}

func (c *ExternalCall) Type() Type {
	if len(c.Results) == 1 {
		return c.Results[0]
	}
	return Type{}
}

// Type of a TypeAssert is the asserted concrete type; the comma-ok form
// is consumed through tuple slots.
func (a *TypeAssert) Type() Type { return a.Target }

var boolType = Type{Kind: KindBool, Go: "bool"}
var intType = Type{Kind: KindInt, Go: "int"}

func (f *FieldLoad) Type() Type  { return f.T }
func (s *StructNew) Type() Type  { return s.T }
func (s *StructCopy) Type() Type { return s.X.Type() }
func (s *StructZero) Type() Type { return s.T }
func (a *AddrOf) Type() Type     { return a.T }
func (d *Deref) Type() Type      { return d.T }
func (f *FieldCell) Type() Type  { return f.T }
func (n *NilConst) Type() Type   { return n.T }
func (i *IsNil) Type() Type      { return boolType }
func (m *MapMake) Type() Type    { return m.T }
func (m *MapFrom) Type() Type    { return m.T }
func (m *MapGet) Type() Type     { return m.T }
func (m *MapLookup) Type() Type  { return m.T }
func (m *MapLen) Type() Type     { return intType }
func (s *StringLen) Type() Type  { return intType }

func (s *SliceLit) Type() Type         { return s.T }
func (s *SliceMake) Type() Type        { return s.T }
func (s *SliceGet) Type() Type         { return s.T }
func (s *SliceReslice) Type() Type     { return s.T }
func (s *SliceAppend) Type() Type      { return s.T }
func (s *SliceAppendSlice) Type() Type { return s.T }
func (s *SliceCopy) Type() Type        { return intType }
func (s *SliceLen) Type() Type         { return intType }
func (s *SliceCap) Type() Type         { return intType }

func (t *TupleSpread) Type() Type         { return t.T }
func (t *TupleVariadicSpread) Type() Type { return t.T }
