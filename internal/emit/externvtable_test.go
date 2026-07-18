// The external-implementer box vtable is keyed by each method's CANONICAL
// dispatch slot (MethodSlot), the SAME selector interface dispatch reads
// (IfaceMember.Slots / printIfaceCall). A same-bare-name promotion — two
// distinct methods sharing a bare name, disambiguated to name$s<digest> —
// must therefore produce two DISTINCT vtable properties, never a single
// bare-name key that collapses one adapter onto the other.
package emit

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestExternBoxVtableKeysByCanonicalSlotNotBareName(t *testing.T) {
	m := NewModule("example.com/owned", "owned", ABIImports{}, map[string]string{})
	m.ExternMethods = map[string][]ExternMethod{}
	// Two distinct methods with the SAME bare name "foo", disambiguated by
	// MethodSlot into distinct slots — the exact collision the slot exists
	// for (e.g. unexported foo promoted from two different packages).
	m.ExternMethods["ext.T"] = []ExternMethod{
		{Name: "foo", Slot: "foo$s0a", Adapter: "$adapterA", AdapterType: "$typeA"},
		{Name: "foo", Slot: "foo$s0b", Adapter: "$adapterB", AdapterType: "$typeB"},
	}
	p := &printer{module: m}

	got, err := p.boxVtable(ir.RttiRef{ExternID: "ext.T"})
	if err != nil {
		t.Fatalf("boxVtable: %v", err)
	}
	// Both distinct slots present — neither adapter is silently dropped by a
	// duplicate object-literal key.
	if !strings.Contains(got, "foo$s0a: $adapterA") || !strings.Contains(got, "foo$s0b: $adapterB") {
		t.Fatalf("box vtable dropped a collided method: %s", got)
	}
	// The bare name is NOT used as a key (it would collapse the two).
	if strings.Contains(got, "foo: ") {
		t.Fatalf("box vtable used the bare method name as a key: %s", got)
	}
}

func TestExternBoxVtableFailsClosedOnMissingSlot(t *testing.T) {
	// A method with an empty slot (an unresolved canonical identity that
	// reached emission) must fail closed at the choke point, never emit a
	// nameless or defaulted property.
	m := NewModule("example.com/owned", "owned", ABIImports{}, map[string]string{})
	m.ExternMethods = map[string][]ExternMethod{}
	m.ExternMethods["ext.T"] = []ExternMethod{{Name: "foo", Slot: "", Adapter: "$adapterA"}}
	p := &printer{module: m}
	defer func() {
		if recover() == nil {
			t.Fatal("boxVtable must fail closed on an empty slot, not emit a bare/blank key")
		}
	}()
	_, _ = p.boxVtable(ir.RttiRef{ExternID: "ext.T"})
}
