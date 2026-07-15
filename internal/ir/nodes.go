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
}

// Target is one assignable location.
type Target interface{ target() }

// VarTarget assigns a plain variable. A struct-valued variable is stored
// in place (field overwrite), preserving every alias of the value.
type VarTarget struct {
	Name string
	T    Type
}

// BlankTarget discards a value (the blank identifier); the value is still
// evaluated.
type BlankTarget struct{}

// FieldTarget assigns a field through a nil-checked pointer or an
// addressable struct-value chain. A struct-valued field is stored in
// place.
type FieldTarget struct {
	X     Expr
	Field string
	T     Type // field type
}

// MapTarget assigns a map entry (nil-map assignment panics).
type MapTarget struct {
	Map Expr
	Key Expr
}

// PointeeTarget assigns through a pointer (*p = v): the nil-checked
// pointee's fields are overwritten in place.
type PointeeTarget struct {
	X Expr // the pointer
}

func (VarTarget) target()      {}
func (BlankTarget) target()    {}
func (*SliceTarget) target()   {}
func (*FieldTarget) target()   {}
func (*MapTarget) target()     {}
func (*PointeeTarget) target() {}

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

// BranchStmt is an unlabeled break or continue.
type BranchStmt struct {
	Tok token.Token // token.BREAK or token.CONTINUE
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

func (*Block) stmt()         {}
func (*SwitchStmt) stmt()    {}
func (*RangeSlice) stmt()    {}
func (*TryFinally) stmt()    {}
func (*MapDeleteStmt) stmt() {}
func (*DeclStmt) stmt()      {}
func (*AssignStmt) stmt()    {}
func (*IfStmt) stmt()        {}
func (*ForStmt) stmt()       {}
func (*ReturnStmt) stmt()    {}
func (*ExprStmt) stmt()      {}
func (*BranchStmt) stmt()    {}

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

// VarRef reads a variable.
type VarRef struct {
	Name string
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
type Call struct {
	Pkg     string // Go package path declaring the callee
	Callee  string // generated symbol name
	Args    []Expr
	Results []Type
}

// MethodCall invokes a method through a nil-checked pointer receiver.
type MethodCall struct {
	Recv    Expr
	Method  string
	Args    []Expr
	Results []Type
}

// FieldLoad reads a struct field through a nil-checked pointer.
type FieldLoad struct {
	X     Expr
	Field string
	T     Type
}

// StructNew allocates a struct on the heap (&T{...}) with every field
// value in declaration order (omitted fields are explicit zeros).
type StructNew struct {
	Pkg      string // Go package path declaring the struct type
	TypeName string
	Args     []Expr
	T        Type // the pointer type
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

// MapMake allocates an empty map (make or an empty literal).
type MapMake struct{ T Type }

// MapFrom builds a map from ordered key/value pairs (composite literal).
type MapFrom struct {
	T      Type
	Keys   []Expr
	Values []Expr
}

// MapGet is the single-value map read: zero value when missing or nil.
type MapGet struct {
	Map Expr
	Key Expr
	T   Type // value type
}

// MapLookup is the comma-ok form, consumed through DeclStmt/AssignStmt
// Tuple slots.
type MapLookup struct {
	Map Expr
	Key Expr
	T   Type // value type
}

// MapLen is len(m) (0 for nil).
type MapLen struct{ X Expr }

// StringLen is len(s): the UTF-8 byte length.
type StringLen struct{ X Expr }

// SliceLit builds a slice from ordered element values.
type SliceLit struct {
	T      Type
	Values []Expr
}

// SliceMake is make([]T, len[, cap]) with zero-filled elements.
type SliceMake struct {
	T        Type
	Length   Expr
	Capacity Expr // nil means Length
}

// SliceGet loads one element with Go's exact bounds panic.
type SliceGet struct {
	X     Expr
	Index Expr
	T     Type // element type
}

// SliceReslice is s[low:high] sharing backing storage.
type SliceReslice struct {
	X    Expr
	Low  Expr // nil means 0
	High Expr // nil means len(s)
	T    Type
}

// SliceAppend is append(s, values...) with capacity-reuse aliasing.
type SliceAppend struct {
	X      Expr
	Values []Expr
	T      Type
}

// SliceLen / SliceCap are len/cap (0 for nil).
type SliceLen struct{ X Expr }
type SliceCap struct{ X Expr }

// SliceTarget assigns one element (bounds-checked).
type SliceTarget struct {
	X     Expr
	Index Expr
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

func (*Const) expr()        {}
func (*MethodCall) expr()   {}
func (*FieldLoad) expr()    {}
func (*StructNew) expr()    {}
func (*StructCopy) expr()   {}
func (*StructZero) expr()   {}
func (*AddrOf) expr()       {}
func (*Deref) expr()        {}
func (*NilConst) expr()     {}
func (*IsNil) expr()        {}
func (*MapMake) expr()      {}
func (*MapFrom) expr()      {}
func (*MapGet) expr()       {}
func (*MapLookup) expr()    {}
func (*MapLen) expr()       {}
func (*StringLen) expr()    {}
func (*SliceLit) expr()     {}
func (*SliceMake) expr()    {}
func (*SliceGet) expr()     {}
func (*SliceReslice) expr() {}
func (*SliceAppend) expr()  {}
func (*SliceLen) expr()     {}
func (*SliceCap) expr()     {}
func (*VarRef) expr()       {}
func (*Binary) expr()       {}
func (*Unary) expr()        {}
func (*Convert) expr()      {}
func (*Call) expr()         {}

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

var boolType = Type{Kind: KindBool, Go: "bool"}
var intType = Type{Kind: KindInt, Go: "int"}

func (f *FieldLoad) Type() Type  { return f.T }
func (s *StructNew) Type() Type  { return s.T }
func (s *StructCopy) Type() Type { return s.X.Type() }
func (s *StructZero) Type() Type { return s.T }
func (a *AddrOf) Type() Type     { return a.T }
func (d *Deref) Type() Type      { return d.T }
func (n *NilConst) Type() Type   { return n.T }
func (i *IsNil) Type() Type      { return boolType }
func (m *MapMake) Type() Type    { return m.T }
func (m *MapFrom) Type() Type    { return m.T }
func (m *MapGet) Type() Type     { return m.T }
func (m *MapLookup) Type() Type  { return m.T }
func (m *MapLen) Type() Type     { return intType }
func (s *StringLen) Type() Type  { return intType }

func (s *SliceLit) Type() Type     { return s.T }
func (s *SliceMake) Type() Type    { return s.T }
func (s *SliceGet) Type() Type     { return s.T }
func (s *SliceReslice) Type() Type { return s.T }
func (s *SliceAppend) Type() Type  { return s.T }
func (s *SliceLen) Type() Type     { return intType }
func (s *SliceCap) Type() Type     { return intType }
