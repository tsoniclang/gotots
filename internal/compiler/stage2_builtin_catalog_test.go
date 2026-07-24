package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

func requireBuiltinCatalogs(
	t *testing.T,
	model *semantic.Model,
	operations []semantic.Operation,
) {
	t.Helper()
	builtin := semanticPackageByImportPath(t, model, "builtin")
	if builtin.DeclarationCount() != len(catalog.AllPredeclared()) {
		t.Fatalf(
			"predeclared declarations=%d, want %d",
			builtin.DeclarationCount(),
			len(catalog.AllPredeclared()),
		)
	}
	unsafePackage := semanticPackageByImportPath(t, model, "unsafe")
	members := map[string]semantic.Declaration{}
	for _, declaration := range semanticDeclarations(unsafePackage) {
		members[declaration.Name()] = declaration
	}
	if len(members) != len(catalog.AllUnsafeMembers()) {
		t.Fatalf(
			"unsafe declarations=%d, want %d",
			len(members),
			len(catalog.AllUnsafeMembers()),
		)
	}
	for _, member := range catalog.AllUnsafeMembers() {
		declaration, present := members[member.Name()]
		if !present {
			t.Errorf("unsafe member %s is absent", member)
			continue
		}
		switch member.Class() {
		case catalog.UnsafeMemberClassType:
			if declaration.Class() != identity.SemanticObjectType ||
				declaration.Type().IsZero() {
				t.Errorf(
					"unsafe type member %s has invalid semantics",
					member,
				)
			}
		case catalog.UnsafeMemberClassBuiltin:
			if declaration.Class() != identity.SemanticObjectBuiltin ||
				!declaration.Type().IsZero() {
				t.Errorf(
					"unsafe builtin member %s has an ordinary type",
					member,
				)
			}
		}
	}
	sizeOf := members[catalog.UnsafeMemberSizeof.Name()]
	var foundCall bool
	for _, operation := range operations {
		spec := operation.Spec()
		if operation.Kind() != semantic.OperationBuiltinCall ||
			spec.Object.Kind() !=
				semantic.ObjectReferenceDeclaration ||
			spec.Object.Declaration() != sizeOf.ID() {
			continue
		}
		foundCall = true
		if spec.ResultType.IsZero() || len(spec.Operands) != 2 {
			t.Fatalf(
				"unsafe.Sizeof call-site evidence result=%s operands=%v",
				spec.ResultType, spec.Operands,
			)
		}
	}
	if !foundCall {
		t.Fatal("unsafe.Sizeof call-site semantics are absent")
	}
}
