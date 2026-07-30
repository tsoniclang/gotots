package order

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestOrderUsesExactEagerDependencies(t *testing.T) {
	constantDeclaration, typeDeclaration := testDeclarations()
	ordered, err := Indices([]Declaration{
		constantDeclaration,
		typeDeclaration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0] != 1 || ordered[1] != 0 {
		t.Fatalf("ordered declaration indices = %v, want [1 0]", ordered)
	}

	constantDeclaration.EagerDependencies = nil
	foil, err := Indices([]Declaration{
		constantDeclaration,
		typeDeclaration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if foil[0] != 0 {
		t.Fatal("dependency-removal foil did not restore source order")
	}
}

func TestOrderRejectsEagerCycles(t *testing.T) {
	constantDeclaration, typeDeclaration := testDeclarations()
	typeDeclaration.EagerDependencies = []api.ArtifactOwner{
		constantDeclaration.Owner,
	}
	if _, err := Indices([]Declaration{
		constantDeclaration,
		typeDeclaration,
	}); err == nil {
		t.Fatal("eager declaration cycle was accepted")
	}
}

func testDeclarations() (Declaration, Declaration) {
	sourcePackage := types.NewPackage("example.com/order", "order")
	typeName := types.NewTypeName(
		token.Pos(20),
		sourcePackage,
		"Number",
		nil,
	)
	named := types.NewNamed(typeName, types.Typ[types.Float64], nil)
	constantObject := types.NewConst(
		token.Pos(10),
		sourcePackage,
		"Before",
		named,
		constant.MakeInt64(7),
	)
	typeOwner := api.MustSourceArtifactOwner(typeName)
	return Declaration{
			Owner:    api.MustSourceArtifactOwner(constantObject),
			Name:     constantObject.Name(),
			Position: constantObject.Pos(),
			EagerDependencies: []api.ArtifactOwner{
				typeOwner,
			},
		}, Declaration{
			Owner:    typeOwner,
			Name:     typeName.Name(),
			Position: typeName.Pos(),
		}
}
