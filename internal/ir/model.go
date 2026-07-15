// Package ir is the typed body intermediate representation: Go semantic
// operations with explicit types and evaluation order, built from go/ast
// plus go/types evidence and consumed by representation planning and
// lowering.
//
// The IR is closed over the reviewed semantic subset. Constructs outside
// it fail with stable GOTOTS_UNSUPPORTED_* diagnostics carrying the exact
// source span — they are never approximated, skipped, or passed through.
package ir

import (
	"fmt"
	"go/token"
)

// Kind is the exact Go semantic type class of an IR value.
type Kind int

const (
	KindInvalid Kind = iota
	KindBool
	KindString
	KindInt8
	KindInt16
	KindInt32
	KindInt64
	KindInt // 64-bit under the linux-amd64 profile
	KindUint8
	KindUint16
	KindUint32
	KindUint64
	KindUint // 64-bit under the linux-amd64 profile
	KindUintptr
	KindFloat32
	KindFloat64
)

// Type is the resolved semantic type of an IR value with its canonical Go
// spelling for provenance.
type Type struct {
	Kind Kind
	Go   string // canonical Go type string
}

// Signed reports whether the kind is a signed integer.
func (k Kind) Signed() bool {
	switch k {
	case KindInt8, KindInt16, KindInt32, KindInt64, KindInt:
		return true
	}
	return false
}

// Unsigned reports whether the kind is an unsigned integer.
func (k Kind) Unsigned() bool {
	switch k {
	case KindUint8, KindUint16, KindUint32, KindUint64, KindUint, KindUintptr:
		return true
	}
	return false
}

// Integer reports whether the kind is any integer.
func (k Kind) Integer() bool { return k.Signed() || k.Unsigned() }

// Wide64 reports whether the kind needs a 64-bit exact carrier.
func (k Kind) Wide64() bool {
	switch k {
	case KindInt64, KindInt, KindUint64, KindUint, KindUintptr:
		return true
	}
	return false
}

// Float reports whether the kind is a floating-point type.
func (k Kind) Float() bool { return k == KindFloat32 || k == KindFloat64 }

// Bits returns the integer width in bits.
func (k Kind) Bits() int {
	switch k {
	case KindInt8, KindUint8:
		return 8
	case KindInt16, KindUint16:
		return 16
	case KindInt32, KindUint32:
		return 32
	case KindInt64, KindInt, KindUint64, KindUint, KindUintptr:
		return 64
	}
	return 0
}

// Span is an exact source location.
type Span struct {
	File string
	Line int
	Col  int
}

// Func is one translated function.
type Func struct {
	ID       string // census declaration identity
	Package  string
	Name     string
	Exported bool
	Span     Span
	Params   []Var
	Results  []Var
	Body     *Block
	// BodyHash matches the census body record for drift detection.
	BodyHash string
	// Operations is the sorted set of IR operation names the body uses,
	// recorded in the proof chain.
	Operations []string
}

// Var is a parameter, result, or local.
type Var struct {
	Name string
	Type Type
}

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
	Values []Expr // len == len(Names); multi-result via CallValues
	// CallValues, when set, is a single multi-result call initializing all
	// names simultaneously; Values is nil in that case.
	CallValues *Call
}

// AssignStmt stores values into existing variables. All right-hand values
// are evaluated before any store, in source order (Go simultaneous
// assignment semantics).
type AssignStmt struct {
	Targets []string // v1: plain variable targets only
	Values  []Expr
	// CallValues, when set, is a single multi-result call assigned to all
	// targets simultaneously; Values is nil in that case.
	CallValues *Call
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
	CallValue *Call
}

// ExprStmt evaluates a call for effect.
type ExprStmt struct {
	Call *Call
}

// BranchStmt is an unlabeled break or continue.
type BranchStmt struct {
	Tok token.Token // token.BREAK or token.CONTINUE
}

func (*Block) stmt()      {}
func (*DeclStmt) stmt()   {}
func (*AssignStmt) stmt() {}
func (*IfStmt) stmt()     {}
func (*ForStmt) stmt()    {}
func (*ReturnStmt) stmt() {}
func (*ExprStmt) stmt()   {}
func (*BranchStmt) stmt() {}

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

// Call invokes an owned package-level function directly.
type Call struct {
	Callee  string // generated symbol name (same package, v1)
	Args    []Expr
	Results []Type
}

func (*Const) expr()   {}
func (*VarRef) expr()  {}
func (*Binary) expr()  {}
func (*Unary) expr()   {}
func (*Convert) expr() {}
func (*Call) expr()    {}

func (c *Const) Type() Type   { return c.T }
func (v *VarRef) Type() Type  { return v.T }
func (b *Binary) Type() Type  { return b.T }
func (u *Unary) Type() Type   { return u.T }
func (c *Convert) Type() Type { return c.To }

// Type of a call is its single result; multi-result calls are consumed
// only through DeclStmt/AssignStmt/ReturnStmt CallValues.
func (c *Call) Type() Type {
	if len(c.Results) == 1 {
		return c.Results[0]
	}
	return Type{}
}

// Unsupported is the stable fail-closed diagnostic for a construct outside
// the reviewed subset.
type Unsupported struct {
	Code      string // GOTOTS_UNSUPPORTED_{STATEMENT,EXPRESSION,TYPE,DECLARATION,OPERATION}
	Construct string
	Span      Span
}

func (u *Unsupported) Error() string {
	return fmt.Sprintf("%s:\n%s at %s:%d:%d", u.Code, u.Construct, u.Span.File, u.Span.Line, u.Span.Col)
}
