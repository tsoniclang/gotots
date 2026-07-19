// Expression nodes of the IR: one Go expression form each, carrying
// its resolved type evidence.
package ir

import "go/token"

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

// ParamReprView reads a representation-uniform type-parameter value at
// its carrier: every constraint term shares the representation, so the
// view is the identity at runtime (a checked TS widening).
type ParamReprView struct {
	X Expr
	T Type // the representative basic type
}

// ParamReprCast types a representative-carrier value as the parameter
// it represents — the inverse view, admitted only for exact single-kind
// constraints (the conversion into the parameter is one function).
type ParamReprCast struct {
	X Expr
	T Type // the parameter's carrier type (TypeParamName set)
}

// Call invokes a package-level function of the translated unit directly.
// TypeArgs, when set, are the explicit type arguments of an instantiated
// generic call.
type Call struct {
	Pkg      string // Go package path declaring the callee
	Callee   string // generated symbol name
	TypeArgs []Type
	// KeyedParams marks, per type parameter, whether the callee's
	// declaration keys a map by it — exactly the parameters whose key$P
	// operation the call passes (requirement-scoped, never universal).
	KeyedParams []bool
	// HardKeyed marks the HARD positions (the callee family-splits on
	// these; a struct binding selects the "$ek" variant).
	HardKeyed []bool
	// ErasedParams marks core-typed positions dropped from the emitted
	// surface (no type argument, no factory group).
	ErasedParams []bool
	// PtrParams marks the pointer-family positions (a non-object binding
	// selects the "$pc" variant).
	PtrParams []bool
	Args      []Expr
	Results   []Type
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
	// KeyedParams marks, per receiver type parameter, whether the
	// declaring type keys a map by it (requirement-scoped key$P).
	KeyedParams []bool
	// ErasedParams marks core-typed receiver positions dropped from the
	// emitted surface.
	ErasedParams []bool
	// PtrParams marks the receiver's pointer-family positions.
	PtrParams []bool
	Recv      Expr
	Method    string
	Args      []Expr
	Results   []Type
}

// FieldLoad reads a struct field through a nil-checked pointer. Cell
// marks a field stored as a stable per-instance cell (its address is
// taken in the unit): the value read unwraps the cell.
type FieldLoad struct {
	X     Expr
	Field string
	T     Type
	Cell  bool
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

// ParamCopy is the value copy of a type-parameter binding at a bind
// site: the emitter spells it through the in-scope clone$P factory, so a
// value-copy carrier binding copies exactly and every other carrier is
// the identity. Loads wrap; already-fresh values bind directly.
type ParamCopy struct {
	X     Expr
	Param string
}

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

// FieldCellRef is &s.f on a struct field stored as a stable per-instance
// cell (a non-identity field whose address is taken in the unit): the
// pointer IS that cell, so repeated &s.f yields the same pointer and a
// store through it (the *p = make(...) lazy-init idiom) mutates the field.
type FieldCellRef struct {
	Base  Expr   // the addressable struct base (a value or a pointer to one)
	Field string // the Go field name whose cell is referenced
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
func (*ParamCopy) expr()        {}
func (*StructZero) expr()       {}
func (*AddrOf) expr()           {}
func (*Deref) expr()            {}
func (*FieldCellRef) expr()     {}
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
func (*ParamReprView) expr()    {}
func (*ParamReprCast) expr()    {}
func (*Call) expr()             {}

func (c *Const) Type() Type   { return c.T }
func (v *VarRef) Type() Type  { return v.T }
func (b *Binary) Type() Type  { return b.T }
func (u *Unary) Type() Type   { return u.T }
func (c *Convert) Type() Type       { return c.To }
func (v *ParamReprView) Type() Type { return v.T }
func (c *ParamReprCast) Type() Type { return c.T }

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

func (f *FieldLoad) Type() Type    { return f.T }
func (s *StructNew) Type() Type    { return s.T }
func (s *StructCopy) Type() Type   { return s.X.Type() }
func (s *ParamCopy) Type() Type    { return s.X.Type() }
func (s *StructZero) Type() Type   { return s.T }
func (a *AddrOf) Type() Type       { return a.T }
func (d *Deref) Type() Type        { return d.T }
func (f *FieldCellRef) Type() Type { return f.T }
func (n *NilConst) Type() Type     { return n.T }
func (i *IsNil) Type() Type        { return boolType }
func (m *MapMake) Type() Type      { return m.T }
func (m *MapFrom) Type() Type      { return m.T }
func (m *MapGet) Type() Type       { return m.T }
func (m *MapLookup) Type() Type    { return m.T }
func (m *MapLen) Type() Type       { return intType }
func (s *StringLen) Type() Type    { return intType }

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
