package api_test

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestClassMethodRequirementUsesExactReceiverIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	owner := types.NewTypeName(token.Pos(10), sourcePackage, "Record", nil)
	named := types.NewNamed(owner, types.NewStruct(nil, nil), nil)
	receiver := types.NewVar(token.Pos(20), sourcePackage, "value", named)
	method := types.NewFunc(
		token.Pos(30),
		sourcePackage,
		"Read",
		types.NewSignatureType(receiver, nil, nil, nil, nil, false),
	)

	request, err := api.NewClassMethodRequest(owner, method)
	if err != nil {
		t.Fatal(err)
	}
	requirement, ok := request.DeclarationRequirement()
	if !ok ||
		requirement.Kind() != api.DeclarationRequirementClassMethod ||
		requirement.Owner() != api.MustSourceArtifactOwner(owner) {
		t.Fatal("class-method request lost its exact owner")
	}
	selectedOwner, selectedMethod, ok := requirement.ClassMethod()
	if !ok || selectedOwner != owner || selectedMethod != method {
		t.Fatal("class-method request lost its exact method")
	}

	forged := types.NewTypeName(
		owner.Pos(),
		sourcePackage,
		owner.Name(),
		owner.Type(),
	)
	if _, err := api.NewClassMethodRequest(forged, method); err == nil {
		t.Fatal("same-spelling receiver owner was accepted")
	}
}

func TestClassMethodRequirementAcceptsPointerReceiver(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	owner := types.NewTypeName(token.Pos(10), sourcePackage, "Record", nil)
	named := types.NewNamed(owner, types.NewStruct(nil, nil), nil)
	receiver := types.NewVar(
		token.Pos(20),
		sourcePackage,
		"value",
		types.NewPointer(named),
	)
	method := types.NewFunc(
		token.Pos(30),
		sourcePackage,
		"Write",
		types.NewSignatureType(receiver, nil, nil, nil, nil, false),
	)
	requirement, err := api.NewClassMethodRequirement(owner, method)
	if err != nil {
		t.Fatal(err)
	}
	selectedOwner, selectedMethod, ok := requirement.ClassMethod()
	if !ok || selectedOwner != owner || selectedMethod != method {
		t.Fatal("pointer receiver was not joined to its named class owner")
	}
}

func TestValueReceiverCopyRequirementRejectsPointerReceiver(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	owner := types.NewTypeName(token.Pos(10), sourcePackage, "Record", nil)
	named := types.NewNamed(owner, types.NewStruct(nil, nil), nil)
	valueReceiver := types.NewVar(token.Pos(20), sourcePackage, "value", named)
	valueMethod := types.NewFunc(
		token.Pos(30),
		sourcePackage,
		"Read",
		types.NewSignatureType(valueReceiver, nil, nil, nil, nil, false),
	)
	requirement, err := api.NewValueReceiverCopyRequirement(valueMethod)
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := requirement.ValueReceiverCopy()
	if !ok || selected != valueMethod {
		t.Fatal("value-receiver copy requirement lost its method identity")
	}

	pointerReceiver := types.NewVar(
		token.Pos(40),
		sourcePackage,
		"pointer",
		types.NewPointer(named),
	)
	pointerMethod := types.NewFunc(
		token.Pos(50),
		sourcePackage,
		"Write",
		types.NewSignatureType(pointerReceiver, nil, nil, nil, nil, false),
	)
	if _, err := api.NewValueReceiverCopyRequirement(pointerMethod); err == nil {
		t.Fatal("pointer receiver admitted a value-copy requirement")
	}
}
