package emit

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestTargetDeclarationOrderUsesExactEagerDependencies(t *testing.T) {
	constantDeclaration, typeDeclaration := orderTestDeclarations()
	ordered, err := orderTargetDeclarations([]targetDeclaration{
		constantDeclaration,
		typeDeclaration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0].owner != typeDeclaration.owner ||
		ordered[1].owner != constantDeclaration.owner {
		t.Fatal("provider was not ordered before its eager consumer")
	}

	constantDeclaration.eagerDependencies = nil
	foil, err := orderTargetDeclarations([]targetDeclaration{
		constantDeclaration,
		typeDeclaration,
	})
	if err != nil {
		t.Fatal(err)
	}
	if foil[0].owner != constantDeclaration.owner {
		t.Fatal("dependency-removal foil did not restore source order")
	}
}

func TestTargetDeclarationOrderRejectsEagerCycles(t *testing.T) {
	constantDeclaration, typeDeclaration := orderTestDeclarations()
	typeDeclaration.eagerDependencies = []api.ArtifactOwner{
		constantDeclaration.owner,
	}
	if _, err := orderTargetDeclarations([]targetDeclaration{
		constantDeclaration,
		typeDeclaration,
	}); err == nil {
		t.Fatal("eager declaration cycle was accepted")
	}
}

func orderTestDeclarations() (targetDeclaration, targetDeclaration) {
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
	return targetDeclaration{
			owner:    api.MustSourceArtifactOwner(constantObject),
			name:     constantObject.Name(),
			position: constantObject.Pos(),
			eagerDependencies: []api.ArtifactOwner{
				typeOwner,
			},
		}, targetDeclaration{
			owner:    typeOwner,
			name:     typeName.Name(),
			position: typeName.Pos(),
		}
}
