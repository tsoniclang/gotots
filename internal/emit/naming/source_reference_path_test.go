package naming

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGeneratedArtifactSourceReferencesUseDefiningModulesAndAliases(
	t *testing.T,
) {
	firstPackage := types.NewPackage("example.com/first", "model")
	secondPackage := types.NewPackage("example.com/second", "model")
	first := types.NewTypeName(token.Pos(1), firstPackage, "record", nil)
	second := types.NewTypeName(token.Pos(2), secondPackage, "record", nil)
	types.NewNamed(first, types.NewStruct(nil, nil), nil)
	types.NewNamed(second, types.NewStruct(nil, nil), nil)
	firstPackage.Scope().Insert(first)
	secondPackage.Scope().Insert(second)

	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{
		firstPackage,
		secondPackage,
	}); err != nil {
		t.Fatal(err)
	}
	for object, path := range map[types.Object]string{
		first:  "modules/first/source.ts",
		second: "modules/second/source.ts",
	} {
		if err := registry.reserve(object, targetBinding{
			name:       "record",
			sourceFile: &ast.File{},
			sourcePath: path,
		}); err != nil {
			t.Fatal(err)
		}
	}
	names := NewOwner(nil, nil, registry).ForFile(
		nil,
		nil,
		tsgo.NewFactory(),
		"support/anonymous-structs.ts",
		nil,
	)
	firstReference, err := names.TypeReference(first)
	if err != nil {
		t.Fatal(err)
	}
	secondReference, err := names.TypeReference(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstReference.Name() == secondReference.Name() {
		t.Fatalf(
			"generated references share local name %q",
			firstReference.Name(),
		)
	}
	assertGeneratedSourceImport(
		t,
		firstReference,
		"../modules/first/source.js",
	)
	assertGeneratedSourceImport(
		t,
		secondReference,
		"../modules/second/source.js",
	)
}

func assertGeneratedSourceImport(
	t *testing.T,
	reference api.NameReference,
	modulePath string,
) {
	t.Helper()
	requests := reference.Requests()
	if len(requests) != 1 ||
		requests[0].ExportedName() != "record" ||
		requests[0].LocalName() != reference.Name() ||
		requests[0].ModulePath() != modulePath {
		t.Fatalf(
			"generated source reference %q requests = %#v",
			reference.Name(),
			requests,
		)
	}
}
