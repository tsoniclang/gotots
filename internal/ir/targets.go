package ir

// Target is one assignable location.
type Target interface{ target() }

// VarTarget assigns a plain variable. A struct-valued variable is stored
// in place (field overwrite), preserving every alias of the value.
type VarTarget struct {
	Name string
	// Pkg names the declaring package for a package-level variable
	// (empty for locals and parameters); package variables are live ESM
	// bindings and never body-local, so the representation planner never
	// treats them as owner-local regions.
	Pkg string
	T   Type
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
