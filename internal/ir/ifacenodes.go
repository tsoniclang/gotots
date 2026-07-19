// Interface and dynamic-dispatch expression nodes: the rtti token
// reference, the interface box, and the closed-token method call with
// its exhaustive dispatch branches.
package ir

// RttiRef names the shared rtti object of one concrete type. Identity
// tests compare the rtti object itself — a single ESM export per type —
// never a spelling.
type RttiRef struct {
	// Predeclared names an ABI rtti (int, string, bool, ...) when set.
	Predeclared string
	// Pkg/TypeName locate a generated named type's rtti; Pointer selects
	// the pointer-type rtti (*T) over the value rtti (T).
	Pkg      string
	TypeName string
	Pointer  bool
	// InstType, when set, is the instantiated-generic member's exact
	// payload type: box sites build the inline vtable from it.
	InstType *Type
	// InstSlots is the instantiation's FULL method-set vtable surface
	// (matching what union member types spell).
	InstSlots []InstSlot
	// Composite is the canonical (path-qualified) type identity of a
	// composite or external type, interned to one rtti object at
	// runtime; Display is its runtime-message spelling; ExternID, when
	// set, names the external contract the static method table covers.
	Composite string
	Display   string
	ExternID  string
	// CompositeEq states the composite's equality class:
	// "uncomparable" (slices, maps, functions — Go panics),
	// "identity" (pointers to unnamed types), "array-prim" (fixed
	// arrays of === carriers), or "unknown" (fails closed).
	CompositeEq string
}

// IfaceBox converts a concrete value into an interface value (struct
// values are copied into the box, as Go copies them into the interface).
type IfaceBox struct {
	X    Expr
	Rtti RttiRef
	T    Type // the interface type
}

// IfaceCall invokes an interface method through the box's method table;
// a nil interface panics with Go's exact message.
type IfaceCall struct {
	Recv Expr
	// Display is the method's source name (for readable diagnostics only).
	Display string
	// MethodKey is the interface method's canonical dispatch identity. The
	// dispatch selector is looked up per union member by THIS key (each
	// member's IfaceMember.Slots is keyed by the interface method's
	// canonical identity, not its bare name), so a member that disambiguates
	// two same-spelled methods dispatches to exactly its own slot.
	MethodKey string
	Args      []Expr
	Results   []Type
}

// PromotionStep is one embedded-field hop in a promoted dispatch chain.
type PromotionStep struct {
	Field   string
	Pointer bool // an embedded pointer field: deref with a nil check
	// FieldType is the embedded field's type (the nilable pointer type
	// when Pointer), for exact deref typing.
	FieldType Type
}
