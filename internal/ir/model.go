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
	"go/types"
	"sort"
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
	// KindPointer is a pointer to a named struct of the translated unit,
	// carried as direct object identity with undefined for nil.
	KindPointer
	// KindStruct is a named struct type of the translated unit; values of
	// this kind appear only behind pointers and as receivers in the
	// reviewed subset (value copies have their own future lowering).
	KindStruct
	// KindMap is a Go map, carried as Map | undefined with exact nil, zero,
	// comma-ok, and write-panic behavior through the language ABI.
	KindMap
	// KindSlice is a Go slice; its representation is selected per value
	// flow by the planner (native array or GoSliceView).
	KindSlice
	// KindFunc is a first-class function value, carried as a JS closure
	// with undefined for nil. Go and JS both capture variables by
	// reference, so closure semantics coincide.
	KindFunc
	// KindIface is an interface value, carried as a box pairing a shared
	// per-type rtti object with the concrete value; undefined is the nil
	// interface. Boxing at conversion sites preserves the nil-pointer-in-
	// interface distinction exactly.
	KindIface
	// KindArray is a Go fixed array [N]T: a value type carried as a
	// native array whose copies happen at Go copy boundaries and whose
	// whole-value stores overwrite elements in place, so element aliases
	// (including slices over the array) observe them exactly.
	KindArray
	// KindExternal is a named type declared outside the translated unit,
	// carried as an opaque handle. Every operation on it is a reviewed
	// external contract: methods, zero construction, and value copies
	// dispatch by canonical identity and fail closed at runtime unless
	// the emulation layer registered behavior. Value-receiver contracts
	// must not mutate their receiver (Go runs them on copies).
	KindExternal
	// KindUnit is the anonymous empty struct type struct{}: a unit type
	// with exactly one value, carried as the number literal 0. Copies,
	// stores, and equality are all trivially exact.
	KindUnit
	// KindTypeParam is a generic function's type parameter: an opaque
	// carrier admitted only for operations exact under every recorded
	// instantiation (the unit-wide closed-world evidence excludes struct
	// values, whose copy semantics a single body cannot express).
	KindTypeParam
)

// Type is the resolved semantic type of an IR value with its canonical Go
// spelling for provenance. Composite kinds carry their structure.
type Type struct {
	Kind Kind
	Go   string // canonical Go type string
	// Named is the struct type name for KindStruct and for the element of
	// a KindPointer.
	Named string
	// Pkg is the Go package path declaring Named. The emitter compares it
	// against the emitting module to select local or imported spelling.
	Pkg string
	// Elem is the pointee (KindPointer), value (KindMap), or element
	// (KindSlice) type.
	Elem *Type
	// Key is the key type of a KindMap.
	Key *Type
	// Sig is the signature of a KindFunc.
	Sig *FuncSig
	// ArrayLen is the fixed length of a KindArray.
	ArrayLen int64
	// TypeParamName, on a KindIface carrier, names the generic type
	// parameter this type is; signatures spell it generically.
	TypeParamName string
	// TypeArgs are the type arguments of an instantiated generic named
	// type (structs), spelling the class generically at every use.
	TypeArgs []Type
}

// FuncSig is the shape of a function value's type.
type FuncSig struct {
	Params  []Type
	Results []Type
}

// Scope is the translation unit's shared context: the set of package
// paths translated together (references outside it fail closed) plus the
// unit-wide generic instantiation evidence that admits generic
// declarations under the closed-world contract.
type Scope struct {
	packages map[string]bool
	// generics maps each generic function to the type-argument tuples of
	// every instantiation anywhere in the unit.
	generics map[*types.Func][][]types.Type
	// externals records every admitted external package-level function:
	// the typed contracts the generated stub modules cover.
	externals map[*types.Func]bool
	// typeGenerics maps each generic named type to the type-argument
	// tuples of every instantiation anywhere in the unit.
	typeGenerics map[*types.TypeName][][]types.Type
	// anonStructs collects, per package, the synthesized classes of the
	// anonymous struct shapes its bodies use, keyed by class name.
	anonStructs map[string]map[string]*Struct
	// externTypes records, per external named type ("pkg.Type"), the
	// contract members the unit references: methods by declared object,
	// plus the value-semantics trio every external value carrier needs.
	// The stub modules type and export each one statically.
	externTypes map[string]*ExternTypeObligation
	// externVars records external package variables the unit reads, by
	// canonical "pkg.Name" identity.
	externVars map[string]*types.Var
}

// ExternTypeObligation is one external named type's referenced contract
// surface.
type ExternTypeObligation struct {
	Pkg     string
	Name    string
	Methods map[string]*types.Func
}

// NewScope builds a unit scope over the given package paths.
func NewScope(paths ...string) Scope {
	packages := make(map[string]bool, len(paths))
	for _, path := range paths {
		packages[path] = true
	}
	return Scope{
		packages:     packages,
		generics:     map[*types.Func][][]types.Type{},
		externals:    map[*types.Func]bool{},
		typeGenerics: map[*types.TypeName][][]types.Type{},
		anonStructs:  map[string]map[string]*Struct{},
		externTypes:  map[string]*ExternTypeObligation{},
		externVars:   map[string]*types.Var{},
	}
}

// Owns reports whether the package path is part of the unit.
func (s Scope) Owns(path string) bool { return s.packages[path] }

// Paths returns every unit package path (unordered).
func (s Scope) Paths() []string {
	paths := make([]string, 0, len(s.packages))
	for path := range s.packages {
		paths = append(paths, path)
	}
	return paths
}

// AddGenericInstance records one instantiation of a generic function.
func (s Scope) AddGenericInstance(fn *types.Func, typeArgs []types.Type) {
	s.generics[fn] = append(s.generics[fn], typeArgs)
}

// GenericInstances returns every recorded instantiation of fn.
func (s Scope) GenericInstances(fn *types.Func) [][]types.Type { return s.generics[fn] }

// AddGenericTypeInstance records one instantiation of a generic named
// type.
func (s Scope) AddGenericTypeInstance(name *types.TypeName, typeArgs []types.Type) {
	s.typeGenerics[name] = append(s.typeGenerics[name], typeArgs)
}

// GenericTypeInstances returns every recorded instantiation of a
// generic named type.
func (s Scope) GenericTypeInstances(name *types.TypeName) [][]types.Type { return s.typeGenerics[name] }

// RegisterAnonStruct records one synthesized anonymous-struct class for
// the package's module (idempotent per shape).
func (s Scope) RegisterAnonStruct(pkg string, decl *Struct) {
	if s.anonStructs[pkg] == nil {
		s.anonStructs[pkg] = map[string]*Struct{}
	}
	s.anonStructs[pkg][decl.Name] = decl
}

// AnonStructs returns the synthesized anonymous-struct classes of one
// package in sorted name order.
func (s Scope) AnonStructs(pkg string) []*Struct {
	names := make([]string, 0, len(s.anonStructs[pkg]))
	for name := range s.anonStructs[pkg] {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*Struct, 0, len(names))
	for _, name := range names {
		out = append(out, s.anonStructs[pkg][name])
	}
	return out
}

// AddExternalFunc records one admitted external function contract.
func (s Scope) AddExternalFunc(fn *types.Func) { s.externals[fn] = true }

// AddExternalType records one external named type the unit carries; the
// stub module exports its value-semantics contract.
func (s Scope) AddExternalType(pkg, name string) *ExternTypeObligation {
	id := pkg + "." + name
	obligation, has := s.externTypes[id]
	if !has {
		obligation = &ExternTypeObligation{Pkg: pkg, Name: name, Methods: map[string]*types.Func{}}
		s.externTypes[id] = obligation
	}
	return obligation
}

// AddExternalMethod records one referenced method of an external type.
func (s Scope) AddExternalMethod(pkg, name string, method *types.Func) {
	s.AddExternalType(pkg, name).Methods[method.Name()] = method
}

// ExternalTypes returns every referenced external type obligation in
// sorted identity order.
func (s Scope) ExternalTypes() []*ExternTypeObligation {
	ids := make([]string, 0, len(s.externTypes))
	for id := range s.externTypes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*ExternTypeObligation, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.externTypes[id])
	}
	return out
}

// AddExternalVar records one external package variable the unit reads.
func (s Scope) AddExternalVar(id string, variable *types.Var) { s.externVars[id] = variable }

// ExternalVars returns the referenced external variables in sorted
// identity order.
func (s Scope) ExternalVars() []struct {
	ID       string
	Variable *types.Var
} {
	ids := make([]string, 0, len(s.externVars))
	for id := range s.externVars {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]struct {
		ID       string
		Variable *types.Var
	}, 0, len(ids))
	for _, id := range ids {
		out = append(out, struct {
			ID       string
			Variable *types.Var
		}{ID: id, Variable: s.externVars[id]})
	}
	return out
}

// ExternalFuncs returns every admitted external function contract,
// sorted by package path then name.
func (s Scope) ExternalFuncs() []*types.Func {
	out := make([]*types.Func, 0, len(s.externals))
	for fn := range s.externals {
		out = append(out, fn)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pkg().Path() != out[j].Pkg().Path() {
			return out[i].Pkg().Path() < out[j].Pkg().Path()
		}
		return out[i].Name() < out[j].Name()
	})
	return out
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

// Nilable reports whether the kind's zero value is Go nil (carried as
// undefined).
func (k Kind) Nilable() bool {
	return k == KindPointer || k == KindMap || k == KindSlice || k == KindFunc || k == KindIface
}

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

// Func is one translated function or method.
type Func struct {
	ID       string // census declaration identity
	Package  string
	Name     string
	Exported bool
	Span     Span
	// TypeParams are the generic type parameter names, admitted under
	// the unit's closed-world instantiation evidence.
	TypeParams []string
	// Receiver is set for methods: a pointer-to-struct parameter bound to
	// the generated class instance.
	Receiver *Var
	Params   []Var
	Results  []Var
	Body     *Block
	// UsesDeferStack marks a body with defers below the top level: the
	// whole body wraps in one try/finally draining a per-function defer
	// stack in LIFO order.
	UsesDeferStack bool
	// SlicePlans maps each slice-typed local variable to its selected
	// representation candidate (the planner's fixed point).
	SlicePlans map[string]string
	// DispatchKey is the method's canonical dynamic-dispatch identity
	// (methods only): name, unexported package qualifier, and signature
	// digest — the rtti table key.
	DispatchKey string
	// PointerReceiver marks a pointer-receiver method: it belongs to the
	// pointer method set only, never the value method set.
	PointerReceiver bool
	// BodyHash matches the census body record for drift detection.
	BodyHash string
	// Support is the implementation support state; Sites records every
	// unsupported operation when the state is unimplemented.
	Support SupportState
	Sites   []UnsupportedSite
	// Operations is the sorted set of IR operation names the body uses,
	// recorded in the proof chain.
	Operations []string
}

// Var is a parameter, result, or local.
type Var struct {
	Name string
	Type Type
}

// Struct is one named struct type of the translated package, generated as
// a class with ordered typed fields.
type Struct struct {
	ID       string
	Name     string
	Exported bool
	Span     Span
	// TypeParams are the generic type parameter names, admitted under
	// the unit's closed-world instantiation evidence.
	TypeParams []string
	Fields     []Var
	Methods    []*Func
	// Promoted lists the embedded-field method promotions the rtti
	// method table delegates through value-field chains.
	Promoted []PromotedDelegate
	// Comparable marks a struct whose fields all support exact generated
	// equality: it carries goEq$, and interface equality over it never
	// panics.
	Comparable bool
}

// PromotedDelegate is one promoted method in a struct's method set: the
// rtti table entry delegates through the embedded value fields to the
// declaring type's generated method function.
type PromotedDelegate struct {
	Name     string
	Path     []string // embedded field names, outermost first
	Pkg      string
	TypeName string
	// DispatchKey is the promoted method's canonical dynamic identity;
	// ValueReceiver marks it part of the value method set.
	DispatchKey   string
	ValueReceiver bool
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
