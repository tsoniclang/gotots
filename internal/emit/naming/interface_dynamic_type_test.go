package naming

import (
	"errors"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestInterfaceDynamicTypeTokenIsCanonicalForLocalGoType(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/local", "local")
	functionScope := types.NewScope(
		sourcePackage.Scope(),
		token.Pos(1),
		token.Pos(100),
		"function",
	)
	typeName := types.NewTypeName(
		token.Pos(10),
		sourcePackage,
		"Local",
		nil,
	)
	localType := types.NewNamed(
		typeName,
		types.NewStruct(
			[]*types.Var{
				types.NewField(
					token.Pos(11),
					sourcePackage,
					"Value",
					types.Typ[types.Int32],
					false,
				),
			},
			nil,
		),
		nil,
	)
	if existing := functionScope.Insert(typeName); existing != nil {
		t.Fatal("local type insertion failed")
	}
	key := strings.Repeat("a", 64)
	registry := NewRegistry()
	first, err := registry.internInterfaceDynamicTypeToken(key, localType)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internInterfaceDynamicTypeToken(key, localType)
	if err != nil {
		t.Fatal(err)
	}
	selectedType, ok := first.owner.InterfaceDynamicType()
	if first.owner != second.owner ||
		first.name != second.name ||
		!ok ||
		selectedType != localType ||
		first.owner.Placement() != api.GeneratedArtifactPlacementCompilation ||
		first.owner.OutputPath() != output.InterfaceTypeSupportPath {
		t.Fatalf("canonical dynamic type = %#v / %#v", first, second)
	}
}

func TestInterfaceDynamicTypeTokenRejectsTruncatedNameCollision(t *testing.T) {
	registry := NewRegistry()
	first := types.NewNamed(
		types.NewTypeName(token.Pos(1), nil, "First", nil),
		types.Typ[types.Int32],
		nil,
	)
	second := types.NewNamed(
		types.NewTypeName(token.Pos(2), nil, "Second", nil),
		types.Typ[types.Int32],
		nil,
	)
	prefix := strings.Repeat("b", interfaceTargetNameHexLength)
	if _, err := registry.internInterfaceDynamicTypeToken(
		prefix+strings.Repeat("1", 64-interfaceTargetNameHexLength),
		first,
	); err != nil {
		t.Fatal(err)
	}
	_, err := registry.internInterfaceDynamicTypeToken(
		prefix+strings.Repeat("2", 64-interfaceTargetNameHexLength),
		second,
	)
	var nameError *api.NameError
	if !errors.As(err, &nameError) ||
		nameError.Reason !=
			"interface-dynamic-type target-name prefix collision" {
		t.Fatalf("collision error = %#v", err)
	}
}
