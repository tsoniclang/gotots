// Collection expression nodes: maps, strings, and slices with their
// exact nil/zero/comma-ok/bounds semantics.
package ir

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

// SliceAppendSlice is append(s, source...) — the spread form: source's
// current elements are copied (struct values clone) with the same
// capacity-reuse aliasing as element append.
type SliceAppendSlice struct {
	X      Expr
	Source Expr
	T      Type
}

// SliceCopy is copy(dst, src) between slices: min(len) elements with
// memmove overlap semantics; struct elements overwrite in place.
type SliceCopy struct {
	Dst Expr
	Src Expr
}

// SliceLen / SliceCap are len/cap (0 for nil).
type SliceLen struct{ X Expr }
type SliceCap struct{ X Expr }

// SliceTarget assigns one element (bounds-checked).
type SliceTarget struct {
	X     Expr
	Index Expr
}
