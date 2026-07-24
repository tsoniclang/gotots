package semantic

import "testing"

func TestTypeMemberValidationIsClosedAndAllocationFree(t *testing.T) {
	for kind := TypeKind(1); kind <= typeKindCount; kind++ {
		if allowedTypeSpecFields(kind) == 0 {
			t.Fatalf("%s has no closed member mask", kind)
		}
	}
	if allowedTypeSpecFields(TypeInvalid) != 0 ||
		allowedTypeSpecFields(typeKindCount+1) != 0 {
		t.Fatal("invalid type kind has an allowed member mask")
	}
	spec := TypeSpec{Kind: TypeBasic, Basic: BasicInt}
	if err := validateTypeMembers(spec); err != nil {
		t.Fatal(err)
	}
	if allocations := testing.AllocsPerRun(1_000, func() {
		_ = validateTypeMembers(spec)
	}); allocations != 0 {
		t.Fatalf("type member validation allocates %.2f times", allocations)
	}
	spec.Element = testSemanticTypeID(t, "ef")
	if err := validateTypeMembers(spec); err == nil {
		t.Fatal("forbidden type member survived the closed mask")
	}
}
