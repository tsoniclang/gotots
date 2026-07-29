package naming

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestCallableABICanonicalizesExactSignatureAcrossStorageForms(
	t *testing.T,
) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package callableabi

type Holder struct {
	Field func(chan int32) int32
}

type Box[T any] struct {
	Value T
}

var Package func(chan int32) int32
var Array [1]func(chan int32) int32
var Slice []func(chan int32) int32
var Mapping map[int32]func(chan int32) int32
var Generic Box[func(chan int32) int32]
var Other func(chan int64) int32

func Parameter(value func(chan int32) int32) {}
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
	}
	sourcePackage, err := (&types.Config{
		Importer: importer.Default(),
	}).Check("example.com/callableabi", fileSet, []*ast.File{source}, info)
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	names := NewOwner(
		sourcePackage.Scope(),
		info,
		registry,
	).ForFile(
		source,
		sourcePackage.Scope(),
		tsgo.NewFactory(),
		"modules/callableabi/source.ts",
		nil,
	).(*File)

	signatures := []*types.Signature{
		callableSignature(t, sourcePackage.Scope().Lookup("Package").Type()),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Holder").Type().
				Underlying().(*types.Struct).Field(0).Type(),
		),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Parameter").Type().(*types.Signature).Params().At(0).Type(),
		),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Array").Type().(*types.Array).Elem(),
		),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Slice").Type().(*types.Slice).Elem(),
		),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Mapping").Type().(*types.Map).Elem(),
		),
		callableSignature(
			t,
			sourcePackage.Scope().Lookup("Generic").Type().(*types.Named).Underlying().(*types.Struct).Field(0).Type(),
		),
	}
	var canonicalName string
	var canonicalArtifact *api.GeneratedArtifact
	for index, signature := range signatures {
		reference, referenceErr := names.CallableABI(signature)
		if referenceErr != nil {
			t.Fatal(referenceErr)
		}
		if index == 0 {
			canonicalName = reference.Artifact().TargetName()
			canonicalArtifact = reference.Artifact()
			continue
		}
		if reference.Artifact() != canonicalArtifact ||
			reference.Artifact().TargetName() != canonicalName {
			t.Fatalf(
				"storage form %d selected a distinct callable ABI",
				index,
			)
		}
	}
	distinct, err := names.CallableABI(callableSignature(
		t,
		sourcePackage.Scope().Lookup("Other").Type(),
	))
	if err != nil {
		t.Fatal(err)
	}
	if distinct.Artifact() == canonicalArtifact ||
		len(registry.GeneratedArtifacts(
			api.GeneratedArtifactCallableABI,
		)) != 2 {
		t.Fatal("non-identical callable signatures shared one ABI")
	}
}

func callableSignature(t *testing.T, sourceType types.Type) *types.Signature {
	t.Helper()
	signature, ok := types.Unalias(sourceType).(*types.Signature)
	if !ok {
		t.Fatalf("source type %T is not a callable signature", sourceType)
	}
	return signature
}
