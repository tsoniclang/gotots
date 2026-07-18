// The interface-equality emitter is a TOTAL function over the closed
// EqKind set: every non-invalid variant emits a concrete operation, a
// nested uncomparable/external element fails closed at element depth
// (never a silent ===), and an EqInvalid or unhandled variant panics at
// generation rather than emitting wrong code.
package emit

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestEmitEqPlanIsTotalOverClosedKindSet(t *testing.T) {
	cases := []struct {
		name string
		plan *ir.EqPlan
		want string // substring the emitted expression must contain
	}{
		{"identity", &ir.EqPlan{Kind: ir.EqIdentity}, "A === B"},
		{"goEq", &ir.EqPlan{Kind: ir.EqGoEq}, "A.goEq$(B)"},
		{"iface", &ir.EqPlan{Kind: ir.EqIface, IfaceID: "some.Iface"}, "$eq(A, B)"},
		{"array-of-identity", &ir.EqPlan{Kind: ir.EqArray, Elem: &ir.EqPlan{Kind: ir.EqIdentity}}, "goArrayEqualWith"},
		// A nested uncomparable/external element must fail closed at depth
		// with the element type's display — NOT ===.
		{"nested-uncomparable", &ir.EqPlan{Kind: ir.EqArray, Elem: &ir.EqPlan{Kind: ir.EqUncomparable, Display: "net.IP"}}, "goPanicUncomparable(\"net.IP\")"},
		{"nested-external", &ir.EqPlan{Kind: ir.EqArray, Elem: &ir.EqPlan{Kind: ir.EqExternal, Display: "time.Time"}}, "goPanicExternalEq(\"time.Time\")"},
	}
	for _, c := range cases {
		got := emitEqPlan(c.plan, "A", "B")
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: emitEqPlan = %q, want substring %q", c.name, got, c.want)
		}
		if strings.Contains(got, "=== B") && c.name != "identity" && !strings.Contains(c.name, "identity") {
			t.Errorf("%s: emitted a JavaScript === fallback: %q", c.name, got)
		}
	}
}

func TestEmitEqPlanPanicsOnInvalidKind(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("emitEqPlan must panic on EqInvalid, not emit a === fallback")
		}
	}()
	_ = emitEqPlan(&ir.EqPlan{Kind: ir.EqInvalid}, "A", "B")
}

func TestMemberEqCasePanicsOnNilPlan(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("memberEqCase must panic on a nil plan, not default to identity")
		}
	}()
	_ = memberEqCase("pkg.T", nil)
}
