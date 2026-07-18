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
	// Canon is the canonical semantic identity from internal/typeid: the
	// only string used as an evidence-ledger key for this type.
	Canon string
	// IfaceEmpty marks the zero-method interface: universal membership.
	IfaceEmpty bool
	// IfaceID is the interface's canonical path-qualified identity — the
	// union alias digests THIS, never the name-qualified spelling.
	IfaceID string
	// IfaceMembers is the closed implementer union of an interface type:
	// one member per concrete implementer (value or pointer flavor),
	// resolved from the whole-unit type universe. Only TYPE identities —
	// spelling uses erased type-only imports, never runtime edges.
	IfaceMembers []IfaceMember
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
	// concreteTypes is the whole-unit universe of concrete named types
	// (owned and referenced external), the closed-world set from which
	// each interface method call's dispatch branches are resolved.
	concreteTypes *[]*types.TypeName
	// externConcrete is the parallel universe of REFERENCED external
	// named types: external implementers join interface unions through
	// stub-adapter vtables.
	externConcrete *[]*types.TypeName
	// boxedComposites records every composite type actually boxed
	// anywhere in the unit (canonical id -> resolved type): the closed
	// enumeration the empty-interface union spells as EXACT members, so
	// composite assertions narrow without any cast. This is a BUILD-PHASE
	// observation log, distinct from the sealed named-type universe: it is
	// appended only while bodies build (after the universe seals) and read
	// only at emit (after every body is built), so it feeds no per-body
	// cached union and can never stale one.
	boxedComposites map[string]*boxedComposite
	// ifaceMembers caches each interface identity's resolved closed
	// implementer union (typeOf recursion makes this hot).
	ifaceMembers map[string][]IfaceMember
	// addressTakenFields is the whole-unit set of struct fields whose
	// address is taken anywhere (&x.f). A non-identity field in this set
	// is represented as one stable per-instance cell so &x.f is exact
	// pointer identity — keyed by the field's storage identity (the
	// DECLARING struct's origin plus the field name), which is stable
	// across generic instantiations (the address scan sees an instantiated
	// field object while class emission sees the generic declaration's).
	addressTakenFields map[string]bool
	// universeSealed is the closed-world dynamic-type universe's lifecycle
	// flag (pointer-shared across Scope value copies). It is false while the
	// pre-pass COLLECTS the named-type universe (concreteTypes,
	// externConcrete) and true once SealUniverse finalizes it. After
	// sealing, adding a named universe type is a construction defect and
	// PANICS: interface unions resolve and cache only over the sealed
	// universe, so a late named implementer could otherwise stale a cached
	// union. Box-site composite observations (AddBoxedComposite) require the
	// sealed phase, never mutate the sealed named universe, and are read only
	// at emit.
	universeSealed *bool
}

// SealUniverse finalizes the closed-world named-type universe: after this
// the dynamic-type set is immutable and any AddConcreteType/AddExternConcrete
// panics. It is called once, at the end of the whole-unit pre-pass, before
// any body builds and any interface union resolves.
func (s Scope) SealUniverse() { *s.universeSealed = true }

// UniverseSealed reports whether the named-type universe is finalized.
func (s Scope) UniverseSealed() bool { return *s.universeSealed }

// ExternTypeObligation is one external named type's referenced contract
// surface. Methods are held by canonical identity, never by bare name.
type ExternTypeObligation struct {
	Pkg  string
	Name string
	// methods maps each referenced method's canonical MethodKey to its
	// object: distinct methods (including same-spelled unexported methods
	// from different packages) are distinct keys, so none overwrites
	// another. The emitted symbol derives from the display name; a genuine
	// display collision fails closed at emission (SortedMethods reports
	// them so the stub builder can detect it).
	methods map[string]*types.Func
	// slots maps the same MethodKey to the method's canonical dispatch
	// SLOT within the external type's method set (MethodSlot). The box
	// vtable is keyed by this slot — the SAME selector interface dispatch
	// uses (IfaceMember.Slots) — so a same-bare-name promotion never
	// collapses two adapter entries onto one property or misses at dispatch.
	slots map[string]string
}

// NewScope builds a unit scope over the given package paths.
func NewScope(paths ...string) Scope {
	packages := make(map[string]bool, len(paths))
	for _, path := range paths {
		packages[path] = true
	}
	return Scope{
		packages:           packages,
		generics:           map[*types.Func][][]types.Type{},
		externals:          map[*types.Func]bool{},
		typeGenerics:       map[*types.TypeName][][]types.Type{},
		anonStructs:        map[string]map[string]*Struct{},
		externTypes:        map[string]*ExternTypeObligation{},
		externVars:         map[string]*types.Var{},
		concreteTypes:      &[]*types.TypeName{},
		externConcrete:     &[]*types.TypeName{},
		boxedComposites:    map[string]*boxedComposite{},
		ifaceMembers:       map[string][]IfaceMember{},
		addressTakenFields: map[string]bool{},
		universeSealed:     new(bool),
	}
}

// MarkFieldAddressTaken records that the address of the field with the
// given storage key is taken somewhere in the unit (the pre-pass calls
// this with FieldStorageKeyOfSelection).
func (s Scope) MarkFieldAddressTaken(key string) {
	if key != "" {
		s.addressTakenFields[key] = true
	}
}

// FieldAddressTaken reports whether the field with the given storage key
// has its address taken anywhere in the unit — the condition (with a
// non-identity field type) for the stable-cell field representation.
func (s Scope) FieldAddressTaken(key string) bool {
	return key != "" && s.addressTakenFields[key]
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
func (s Scope) RegisterAnonStruct(pkg string, decl *Struct) error {
	if s.anonStructs[pkg] == nil {
		s.anonStructs[pkg] = map[string]*Struct{}
	}
	if existing, ok := s.anonStructs[pkg][decl.Name]; ok && existing.Identity != decl.Identity {
		// A digest collision between distinct anonymous shapes must never
		// silently overwrite: the identity is not injective, so fail
		// closed rather than mis-dispatch one shape as the other.
		return fmt.Errorf("anonymous struct identity collision on %s: %q vs %q", decl.Name, existing.Identity, decl.Identity)
	}
	s.anonStructs[pkg][decl.Name] = decl
	return nil
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
	// MethodIdent is the canonical dispatch identity (MethodKey) of a
	// method — name, unexported package, and signature digest. It is empty
	// for free functions. The interface-assertion diagnostic compares these
	// so a wrong-signature method is reported "missing" exactly as Go does.
	MethodIdent string
	// Slot is the method's vtable property name (ir.MethodSlot): the bare
	// name when unique in the receiver's method set, disambiguated by
	// canonical identity otherwise. It is empty for free functions. The
	// vtable emits by Slot — never the bare Name — so dispatch and the
	// vtable index the same canonical slot.
	Slot string
	Span Span
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
	// Cell marks a struct field whose address is taken somewhere in the
	// unit: it is stored as one stable per-instance GoCell<Type>, so &x.f
	// is exact pointer identity. Only set on non-identity struct fields.
	Cell bool
}

// Struct is one named struct type of the translated package, generated as
// a class with ordered typed fields.
type Struct struct {
	ID       string
	Name     string
	Exported bool
	Span     Span
	// Identity is the package-path-qualified structural identity string
	// an anonymous struct's name digests; the collision check compares
	// it (empty for named structs).
	Identity string
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
	// KeyEncodable marks a struct whose fields all encode injectively onto
	// a deterministic string: it carries goKey$, so it can key a map and
	// compose as a nested field of another key struct.
	KeyEncodable bool
}

// PromotedDelegate is one promoted method in a struct's method set: the
// rtti table entry delegates through the embedded value fields to the
// declaring type's generated method function.
// IfaceMember is one branch of a closed interface union: the literal
// discriminant, the payload's concrete type, and the vtable const's
// location (the concrete type's own package).
type IfaceMember struct {
	K       string // "pkg.Type" or "*pkg.Type"
	Pkg     string
	Type    string
	Pointer bool
	// Struct distinguishes class payloads (identity carriers: the
	// pointer IS the instance) from named value carriers (pointer = a
	// cell). Payloads spell by NAME ONLY — no eager type resolution, so
	// interface membership can never recurse through generic arguments.
	Struct bool
	// Extern marks an external implementer whose vtable is built inline
	// over stub exports at box sites. ExternCarrier spells its payload
	// class: "" (branded handle, struct-underlying) or the exact value
	// carrier for basic-underlying external named types.
	Extern        bool
	ExternCarrier string
	// Eq is the recursive typed equality plan for THIS member's exact
	// payload under Go's interface equality — the single operation the
	// generated union equality narrows to, so no payload is ever erased.
	Eq *EqPlan
	// Slots maps each of the dispatching interface's method names to THIS
	// member type's vtable selector for the method that implements it (see
	// ir.MethodSlot). It is the bare name except where the member promotes
	// two same-bare-name methods from different packages, so dispatch and
	// the member's vtable always index the same canonical slot.
	Slots map[string]string
}

// EqKind is the CLOSED set of interface-equality plan variants. The zero
// value EqInvalid is never a valid plan: plan construction returns an
// explicit error rather than a fabricated fallback, and every emitter
// handles every non-zero variant exhaustively — an EqInvalid or unhandled
// variant fails closed (a compiler panic), never a silent JavaScript ===
// or identity default.
type EqKind int

const (
	EqInvalid      EqKind = iota // not a plan — a construction error was swallowed
	EqIdentity                   // === on the exact primitive/pointer/channel carrier
	EqGoEq                       // the comparable value struct's own goEq$ method
	EqArray                      // element-wise over a comparable value array (Elem)
	EqIface                      // the element interface's own union equality (IfaceID)
	EqUncomparable               // Go PANICS comparing this dynamic type (Display)
	EqExternal                   // comparable external struct/array, no stub equality yet (Display)
)

// EqPlan is one recursive typed equality operation over an exact dynamic
// type. It descends only into arrays (a struct compares through its own
// goEq$, an interface through its own union equality — neither re-enters
// interface resolution), so it is a finite tree with no cycle. Construction
// is total: `eqPlan` returns a valid plan or an explicit error.
type EqPlan struct {
	Kind    EqKind
	Elem    *EqPlan // EqArray element plan
	IfaceID string  // EqIface: the element interface's canonical union identity
	Display string  // EqUncomparable / EqExternal: the Go type display for the panic
}

type PromotedDelegate struct {
	Name string
	// Slot is the vtable property name for this promoted method: the bare
	// name, unless the promoting type carries two same-bare-name methods
	// from different packages, in which case it is disambiguated by
	// canonical identity (see ir.MethodSlot).
	Slot string
	// MethodIdent is the promoted method's canonical dispatch identity
	// (MethodKey), for the assertion diagnostic's signature-aware match.
	MethodIdent   string
	Path          []string // embedded field names, outermost first
	Pkg           string
	TypeName      string
	ValueReceiver bool
}
