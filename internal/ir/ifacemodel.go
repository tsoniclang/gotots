// Interface-member model: the closed union members interfaces resolve
// to, including instantiated-generic members and their vtable surfaces.
package ir

import (
	"go/types"
)

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
	// ValueCarrier names an OWNED named scalar member's basic carrier
	// (string/boolean/number/bigint) for the typed key encoder.
	ValueCarrier string
	// KeyEncodable marks a value-STRUCT member whose goKey$ encoding
	// exists (map-key admission for interface keys consults it; other
	// member classes derive keyability from their carrier).
	KeyEncodable bool
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
	// Composite, when set, marks an INSTANTIATED-GENERIC member: its box
	// brand is "c:"+Composite (the interned composite rtti token), its
	// payload spells InstType, and its vtable is built inline at box
	// sites from InstSlots (the instantiated method surface).
	Composite string
	InstType  Type
	InstSlots []InstSlot
}

// InstCandidate is one concrete generic instantiation from the closed
// evidence, with its canonical identity precomputed.
type InstCandidate struct {
	Named *types.Named
	Canon string
}

// instSlotsEntry caches one instantiation's full vtable surface.
type instSlotsEntry struct {
	slots []InstSlot
	ok    bool
}

// InstSlot is one instantiated-generic member's vtable slot: the
// generated generic function it dispatches to, with the instantiation's
// exact types (factory derivations happen at emission from TypeArgs).
type InstSlot struct {
	Slot        string
	MethodName  string
	PointerRecv bool
	TypeArgs    []Type
	KeyedParams []bool
	// RttiParams marks the declaring type's rtti-required positions;
	// RttiArgs carries each required position's box operation for THIS
	// instantiation (concrete bindings only — an instantiated member is
	// never a free-parameter vector).
	RttiParams []bool
	RttiArgs   []ParamRttiArg
	Params     []Type
	Results    []Type
}
