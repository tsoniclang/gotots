package semantic

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestBindingCaptureEligibilityIsClosed(t *testing.T) {
	tests := []struct {
		role identity.SemanticBindingRole
		want bool
	}{
		{identity.SemanticBindingImport, false},
		{identity.SemanticBindingLocal, true},
		{identity.SemanticBindingReceiver, true},
		{identity.SemanticBindingParameter, true},
		{identity.SemanticBindingResult, true},
		{identity.SemanticBindingTypeParameter, false},
		{identity.SemanticBindingRange, true},
		{identity.SemanticBindingTypeSwitch, true},
		{identity.SemanticBindingLabel, false},
		{identity.SemanticBindingImplicit, false},
	}
	for _, test := range tests {
		if got := BindingRoleCanBeCaptured(test.role); got != test.want {
			t.Errorf(
				"BindingRoleCanBeCaptured(%s)=%t, want %t",
				test.role, got, test.want,
			)
		}
	}

	fixture := semanticFixture(t)
	integer, err := NewType(TypeSpec{
		Kind: TypeBasic, Basic: BasicInt,
	})
	if err != nil {
		t.Fatal(err)
	}
	bindingID, err := identity.NewSemanticBindingID(
		fixture.root,
		fixture.body,
		identity.SemanticBindingTypeParameter,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewBinding(
		bindingID,
		fixture.pkg,
		fixture.definition,
		identity.SemanticBindingTypeParameter,
		"T",
		integer.ID(),
		fixture.body,
		[]identity.DefinitionID{fixture.definition},
		fixture.authority,
	); err == nil {
		t.Fatal("type-parameter binding accepted a runtime capture")
	}
}
