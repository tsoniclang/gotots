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

func TestOrderUsesCanonicalSourcePathAcrossFiles(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/order", "order")
	earlierFileObject := types.NewVar(
		token.Pos(200),
		sourcePackage,
		"EarlierFile",
		types.Typ[types.Int],
	)
	laterFileObject := types.NewVar(
		token.Pos(10),
		sourcePackage,
		"LaterFile",
		types.Typ[types.Int],
	)
	declarations := []Declaration{
		{
			Owner:      api.MustSourceArtifactOwner(earlierFileObject),
			Name:       earlierFileObject.Name(),
			Position:   earlierFileObject.Pos(),
			SourcePath: "module/earlier.ts",
		},
		{
			Owner:      api.MustSourceArtifactOwner(laterFileObject),
			Name:       laterFileObject.Name(),
			Position:   laterFileObject.Pos(),
			SourcePath: "module/later.ts",
		},
	}

	ordered, err := Indices(declarations)
	if err != nil {
		t.Fatal(err)
	}
	if ordered[0] != 0 || ordered[1] != 1 {
		t.Fatalf("ordered declaration indices = %v, want [0 1]", ordered)
	}
}

func TestOrderRejectsSourceDeclarationWithoutCanonicalPosition(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/order", "order")
	object := types.NewVar(
		token.Pos(10),
		sourcePackage,
		"MissingPosition",
		types.Typ[types.Int],
	)
	_, err := Indices([]Declaration{{
		Owner:    api.MustSourceArtifactOwner(object),
		Name:     object.Name(),
		Position: object.Pos(),
	}})
	if _, ok := err.(*SourcePositionError); !ok {
		t.Fatalf("missing source position error = %T %v", err, err)
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
			Owner:      api.MustSourceArtifactOwner(constantObject),
			Name:       constantObject.Name(),
			SourcePath: "module/source.ts",
			Position:   constantObject.Pos(),
			EagerDependencies: []api.ArtifactOwner{
				typeOwner,
			},
		}, Declaration{
			Owner:      typeOwner,
			Name:       typeName.Name(),
			SourcePath: "module/source.ts",
			Position:   typeName.Pos(),
		}
}
