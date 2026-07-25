package catalog

import (
	"go/types"
	"testing"
)

func TestUnsafeMemberCatalogMatchesToolchain(t *testing.T) {
	scope := types.Unsafe.Scope()
	names := scope.Names()
	if len(names) != len(AllUnsafeMembers()) {
		t.Fatalf(
			"unsafe catalog=%d, toolchain=%d",
			len(AllUnsafeMembers()), len(names),
		)
	}
	for _, member := range AllUnsafeMembers() {
		object := scope.Lookup(member.Name())
		if object == nil {
			t.Fatalf("unsafe member %s is absent from toolchain", member)
		}
		_, builtin := object.(*types.Builtin)
		if builtin !=
			(member.Class() == UnsafeMemberClassBuiltin) {
			t.Fatalf(
				"unsafe member %s class=%d, checker=%T",
				member, member.Class(), object,
			)
		}
	}
	for _, name := range names {
		if !UnsafeMemberByName(name).Valid() {
			t.Fatalf(
				"toolchain unsafe member %s is absent from catalog",
				name,
			)
		}
	}
}

func TestUnsafeMemberIDsArePinned(t *testing.T) {
	want := map[UnsafeMemberKind]uint8{
		UnsafeMemberPointer: 1, UnsafeMemberAlignof: 2,
		UnsafeMemberOffsetof: 3, UnsafeMemberSizeof: 4,
		UnsafeMemberAdd: 5, UnsafeMemberSlice: 6,
		UnsafeMemberSliceData: 7, UnsafeMemberString: 8,
		UnsafeMemberStringData: 9,
	}
	for member, id := range want {
		if uint8(member) != id {
			t.Fatalf("%s id=%d, want %d", member, member, id)
		}
	}
}
