package naming

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
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
			kind:         targetBindingSource,
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
			kind:       targetBindingSource,
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
