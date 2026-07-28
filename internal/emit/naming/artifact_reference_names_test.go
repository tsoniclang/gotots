package naming

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestArtifactReferencesRecordExactConsumedFacet(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/current", "current")
	consumerDeclaration := &ast.FuncDecl{
		Name: &ast.Ident{NamePos: token.Pos(10), Name: "Consumer"},
		Type: &ast.FuncType{
			Func: token.Pos(9),
			Params: &ast.FieldList{
				Opening: token.Pos(18),
				Closing: token.Pos(19),
			},
		},
	}
	sourceFile := &ast.File{
		Name:  &ast.Ident{NamePos: token.Pos(1), Name: "current"},
		Decls: []ast.Decl{consumerDeclaration},
	}
	function := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	typeName := types.NewTypeName(token.Pos(2), sourcePackage, "Record", nil)
	types.NewNamed(typeName, types.NewStruct(nil, nil), nil)
	value := types.NewConst(
		token.Pos(3),
		sourcePackage,
		"Value",
		types.Typ[types.Int],
		constant.MakeInt64(1),
	)
	registry := NewRegistry()
	for _, object := range []types.Object{function, typeName, value} {
		if err := registry.reserve(object, targetBinding{
			name:         object.Name(),
			sourceFile:   sourceFile,
			sourcePath:   "modules/current/source.ts",
			moduleExport: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	names := NewOwner(
		sourcePackage.Scope(),
		&types.Info{Defs: make(map[*ast.Ident]types.Object)},
		registry,
	).ForFile(
		sourceFile,
		sourcePackage.Scope(),
		tsgo.NewFactory(),
		"modules/current/source.ts",
		nil,
	).(*File)
	consumer := types.NewFunc(
		token.Pos(10),
		sourcePackage,
		"Consumer",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	finish, err := names.BeginArtifact(
		api.MustSourceArtifactOwner(consumer),
		consumerDeclaration,
		sourceFile,
		"modules/current/source.ts",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer finish()
	for name, test := range map[string]struct {
		reference func() (api.NameReference, error)
		provider  types.Object
		facet     api.ArtifactFacet
	}{
		"callable": {
			reference: func() (api.NameReference, error) {
				return names.Reference(function)
			},
			provider: function,
			facet:    api.ArtifactFacetCallableSignature,
		},
		"constructor": {
			reference: func() (api.NameReference, error) {
				return names.Reference(typeName)
			},
			provider: typeName,
			facet:    api.ArtifactFacetConstructorSurface,
		},
		"instance type": {
			reference: func() (api.NameReference, error) {
				return names.TypeReference(typeName)
			},
			provider: typeName,
			facet:    api.ArtifactFacetInstanceTypeSurface,
		},
		"static operation": {
			reference: func() (api.NameReference, error) {
				return names.NamedStructOperation(
					typeName,
					api.NamedStructOperationCopy,
				)
			},
			provider: typeName,
			facet:    api.ArtifactFacetStaticSurface,
		},
		"value": {
			reference: func() (api.NameReference, error) {
				return names.Reference(value)
			},
			provider: value,
			facet:    api.ArtifactFacetValueSurface,
		},
	} {
		t.Run(name, func(t *testing.T) {
			reference, err := test.reference()
			if err != nil {
				t.Fatal(err)
			}
			var dependencies []api.ArtifactDependency
			for _, request := range reference.Requests() {
				if dependency, ok := request.ArtifactDependency(); ok {
					dependencies = append(dependencies, dependency)
				}
			}
			if len(dependencies) != 1 {
				t.Fatalf("dependencies = %#v", dependencies)
			}
			sourceProvider, sourceOwned := dependencies[0].Provider().Source()
			if !sourceOwned ||
				sourceProvider != test.provider ||
				dependencies[0].Facet() != test.facet {
				t.Fatalf("dependencies = %#v", dependencies)
			}
		})
	}
}
