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
	"go/types"
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
	// KindChan is a channel TYPE position: an opaque, never-constructible
	// nilable carrier so declarations carrying channel fields complete and
	// their packages materialize. Every channel OPERATION (make, send,
	// receive, close, select, range) stays typed-unimplemented — nothing
	// can create a non-nil value of this type in emitted code.
	KindChan
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
	// Uncomparable marks a struct type Go's == rejects (slice, map, or
	// function fields): its goEq$ is never generated, and an equality
	// factory over such a binding is provably unreachable (the type
	// system rejects == wherever it could run), spelled fail-closed.
	Uncomparable bool
	// ErasedParamName names a CORE-TYPED parameter this carrier erased
	// (S ~[]E): the emitted surface drops it from the generic list and
	// its factory groups.
	ErasedParamName string
	// HardKeyedParams marks, per type parameter of a generic struct's
	// declaration, the HARD map-key requirement (family variants split on
	// these positions).
	HardKeyedParams []bool
	// PtrParams marks the pointer-family positions of the declaration.
	PtrParams []bool
	// MapFamilyPtrCell marks a generic-struct INSTANCE whose
	// pointer-required binding is a NON-object carrier: its symbols take
	// the "$pc" variant.
	MapFamilyPtrCell bool
	// MapFamilyEnc marks a generic-struct INSTANCE whose hard map-keyed
	// binding is a struct: its class/method symbols take the "$ek"
	// (encoded-family) variant.
	MapFamilyEnc bool
	// EncodedParamKey marks a PARAMETER-keyed map type whose enclosing
	// declaration's closed evidence includes a struct binding: the map
	// takes the encoded carrier with the key$P factory. SVZ-only evidence
	// keeps the direct Map carrier (so map values crossing the generic
	// boundary into concrete SVZ contexts stay representation-identical).
	EncodedParamKey bool
	// ClassKeyParams marks, per type parameter of a generic struct
	// instance's DECLARATION, whether its class captures key$P at that
	// position (the requirement store's per-param verdict) — every
	// construction site passes the per-binding key operation for exactly
	// these positions.
	ClassKeyParams []bool
	// KeyEncodable marks a struct whose generated class carries goKey$
	// (the canonical key encoding) — consumed by the per-binding key
	// operation derivation.
	KeyEncodable bool
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
	// ParamRepr, on a type-parameter carrier, is the representative
	// basic type when every constraint term shares one representation
	// carrier — kind-driven conversions OUT of the parameter are exact.
	// ParamReprExact additionally means all terms share the exact KIND,
	// admitting conversions INTO the parameter and compound operations.
	ParamRepr      *Type
	ParamReprExact bool
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
	// valueBoxedNamed logs every NAMED type whose VALUE form is boxed
	// into an interface anywhere in the unit (pointer boxes excluded):
	// the reachability evidence behind goKeyUnreachable claims in union
	// $key encoders — verified corpus-complete at finalize.
	valueBoxedNamed map[string]bool
	// ifaceMembers caches each interface identity's resolved closed
	// implementer union (typeOf recursion makes this hot).
	ifaceMembers map[string][]IfaceMember
	// instCandidates caches the corpus-wide concrete-instantiation
	// candidate list (named instance + canonical id), computed once: the
	// per-interface enumeration then only runs Implements over it.
	instCandidates *[]InstCandidate
	// instSlotsCache caches each instantiation's full vtable surface by
	// canon+pointerness ("*"-prefixed) — typeOf over method signatures is
	// the expensive part of member construction.
	instSlotsCache map[string]instSlotsEntry
	// addressTakenFields is the whole-unit set of struct fields whose
	// address is taken anywhere (&x.f). A non-identity field in this set
	// is represented as one stable per-instance cell so &x.f is exact
	// pointer identity — keyed by the field's storage identity (the
	// DECLARING struct's origin plus the field name), which is stable
	// across generic instantiations (the address scan sees an instantiated
	// field object while class emission sees the generic declaration's).
	addressTakenFields map[string]bool
	// genericEdges are free-parameter instantiation edges (an
	// instantiation inside a generic declaration whose arguments mention
	// the outer parameters), closed by CloseGenericEvidence.
	genericEdges *[]genericEdge
	// paramKeyReqs records, per generic declaration (by canonical key),
	// which type parameters must bind SameValueZero key carriers.
	paramKeyReqs map[string][]bool
	// paramPtrReqs marks parameters whose declarations take addresses
	// into or read through *P — the pointer-family split axis.
	paramPtrReqs map[string][]bool
	// paramCaptureReqs is the SOFT level: the parameter's key$P is
	// forwarded (a key-encodable class origin captures it) but no map is
	// keyed by it — any Go-legal binding admits, the derivation is total.
	paramCaptureReqs map[string][]bool
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

// Signed reports whether the kind is a signed integer.
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
	// KeyedParams marks, per type parameter, whether this declaration
	// keys a map by it: the signature takes key$P exactly for these.
	KeyedParams []bool
	// HardKeyed marks, per type parameter, the HARD map-key requirement
	// (family variants split on these positions).
	HardKeyed []bool
	// ErasedParams marks CORE-TYPED parameters erased to their carriers
	// (dropped from the emitted generic list and factory groups).
	ErasedParams []bool
	// PtrParams marks the pointer-family positions; FamilyPtrCell marks
	// the CELL-family ("$pc") emission variant.
	PtrParams     []bool
	FamilyPtrCell bool
	// ReprParams holds, per type parameter, the representation-uniform
	// basic type of its constraint (nil when not uniform): the emitted
	// TS parameter is bounded by the carrier so representation views
	// typecheck.
	ReprParams []*Type
	// FamilyEnc marks this Func/Struct emission as the ENCODED key-family
	// variant ("$ek"-suffixed symbols; parameter-keyed maps spell the
	// encoded carrier).
	FamilyEnc bool
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
	// Placeholder marks a body materialized as a typed throwing stub: its
	// signature is exact and typechecks, but the body is a fail-closed
	// goBodyUnimplemented call because at least one operation is outside the
	// reviewed subset. The emitter renders the signature and the throw, not
	// the (absent or partial) real body. A placeholder is materialized for
	// ANALYSIS; its package stays publication-withheld.
	Placeholder bool
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
	// KeyedParams marks, per type parameter, whether the class captures
	// key$P at that position — the requirement store's per-param verdict
	// (a key-encodable origin requires every position).
	KeyedParams []bool
	// HardKeyed marks, per type parameter, the HARD map-key requirement
	// (family variants split on these). FamilyEnc marks the encoded
	// emission variant.
	HardKeyed []bool
	FamilyEnc bool
	// PtrParams / FamilyPtrCell: the pointer-family axis ("$pc").
	PtrParams     []bool
	FamilyPtrCell bool
	// Comparable marks a struct whose fields all support exact generated
	// equality: it carries goEq$, and interface equality over it never
	// panics.
	Comparable bool
	// KeyEncodable marks a struct whose fields all encode injectively onto
	// a deterministic string: it carries goKey$, so it can key a map and
	// compose as a nested field of another key struct.
	KeyEncodable bool
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
	MethodIdent string
	Path        []string // embedded field names, outermost first
	// PathPointer marks, per Path step, an embedded POINTER field: the
	// delegation dereferences it with Go's nil panic before the next step
	// (calling a promoted method through a nil embedded pointer panics at
	// the implicit dereference).
	PathPointer   []bool
	Pkg           string
	TypeName      string
	ValueReceiver bool
	// IfaceField marks a method promoted from an embedded INTERFACE
	// field: no static Type$Method function exists — the delegate emits
	// as a generated function on the EMBEDDING type whose body dispatches
	// through the field's interface value (the closed union switch).
	IfaceField bool
	// IfaceType / Params / Results carry the dispatch data for IfaceField
	// delegates: the embedded interface's resolved type and the promoted
	// method's exact signature.
	IfaceType Type
	Params    []Var
	Results   []Type
}
