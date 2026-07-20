// Neutral semantic control: a nil interface is distinct from an
// interface holding a typed nil pointer.
package neutral

type node struct{ value int }

type describer interface{ describe() string }

func (n *node) describe() string {
	if n == nil {
		return "typed-nil"
	}
	return "node"
}

func Describe(useTypedNil bool) (string, bool) {
	var d describer
	if useTypedNil {
		var n *node
		d = n
	}
	if d == nil {
		return "nil-interface", false
	}
	return d.describe(), true
}
