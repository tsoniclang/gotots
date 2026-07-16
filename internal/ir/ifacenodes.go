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
	// Method is the canonical dispatch identity (MethodKey); Display is
	// the source spelling for diagnostics.
	Method  string
	Display string
	Args    []Expr
	Results []Type
	// Branches is the closed set of reachable dynamic types for this
	// call, each pairing the concrete type's rtti token with the direct
	// generated method to invoke. Dispatch is an exhaustive token switch
	// over these branches — no name-selected member lookup.
	Branches []IfaceBranch
}

// IfaceBranch is one closed dispatch case: a concrete dynamic type's
// rtti token and the direct method to call for it.
type IfaceBranch struct {
	// Rtti is the branch's dynamic-type token (the switch discriminant).
	Rtti RttiRef
	// Payload is the concrete type the token narrows the box value to.
	Payload Type
	// DeclPkg/DeclType name the type that DECLARES the method (the
	// concrete type itself, or an embedded type for a promoted method);
	// External routes to its stub module. The generated symbol is
	// DeclType$Method.
	DeclPkg  string
	DeclType string
	External bool
	Method   string
	// FieldPath chains through embedded value fields from the concrete
	// payload to the promoted method's receiver (empty for a direct
	// method).
	FieldPath []string
	// ValueReceiver marks a value-receiver method: the narrowed payload
	// clones at the call (a pointer receiver takes it as is).
	ValueReceiver bool
}
