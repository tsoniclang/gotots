package naming

import (
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
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

func TestGenericInterfaceCallableFamilyAndRuntimeTokenAreDistinct(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package interfacecallable

type Value[T any] interface {
	Get() T
}

type Renamed[U any] interface {
	Get() U
}

type IntValue interface {
	Get() int32
}
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
	sourcePackage, err := new(types.Config).Check(
		"example.com/interfacecallable",
		fileSet,
		[]*ast.File{source},
		info,
	)
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
		"modules/interfacecallable/source.ts",
		nil,
	).(*File)
	interfaceMethod := func(typeName string) *types.Func {
		t.Helper()
		named := sourcePackage.Scope().Lookup(typeName).Type().(*types.Named)
		return named.Underlying().(*types.Interface).Method(0)
	}
	value := sourcePackage.Scope().Lookup("Value").Type().(*types.Named)
	instantiated, err := types.Instantiate(
		types.NewContext(),
		value,
		[]types.Type{types.Typ[types.Int32]},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	concrete := instantiated.Underlying().(*types.Interface).Method(0)

	valueFamily, err := names.InterfaceMethodCallable(interfaceMethod("Value"))
	if err != nil {
		t.Fatal(err)
	}
	renamedFamily, err := names.InterfaceMethodCallable(interfaceMethod("Renamed"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valueFamily.Artifacts()) != 1 ||
		len(renamedFamily.Artifacts()) != 1 ||
		valueFamily.Artifacts()[0] != renamedFamily.Artifacts()[0] {
		t.Fatal("alpha-equivalent generic interface methods did not converge")
	}
	for _, request := range valueFamily.Requests() {
		requirement, ok := request.DeclarationRequirement()
		selected, callable :=
			requirement.InterfaceMethodCallable()
		if !ok ||
			!callable ||
			selected != valueFamily.Artifacts()[0] {
			t.Fatal("generic callable family requested a runtime token")
		}
	}
	if _, err := names.InterfaceMethodToken(interfaceMethod("Value")); err == nil {
		t.Fatal("open generic interface method received a runtime token")
	}

	concreteFamily, err := names.InterfaceMethodCallable(concrete)
	if err != nil {
		t.Fatal(err)
	}
	if len(concreteFamily.Artifacts()) != 2 ||
		!slices.Contains(
			concreteFamily.Artifacts(),
			valueFamily.Artifacts()[0],
		) {
		t.Fatalf(
			"closed interface method facets = %d, family joined = %t",
			len(concreteFamily.Artifacts()),
			slices.Contains(
				concreteFamily.Artifacts(),
				valueFamily.Artifacts()[0],
			),
		)
	}
	concreteToken, err := names.InterfaceMethodToken(concrete)
	if err != nil {
		t.Fatal(err)
	}
	intToken, err := names.InterfaceMethodToken(interfaceMethod("IntValue"))
	if err != nil {
		t.Fatal(err)
	}
	if concreteToken.Name() != intToken.Name() {
		t.Fatal("closed generic instance and concrete interface use distinct runtime tokens")
	}
	var concreteArtifact *api.GeneratedArtifact
	for _, request := range concreteToken.Requests() {
		requirement, ok := request.DeclarationRequirement()
		if !ok {
			continue
		}
		if artifact, ok := requirement.InterfaceMethodToken(); ok {
			concreteArtifact = artifact
		}
	}
	if concreteArtifact == nil ||
		slices.Contains(concreteFamily.Artifacts(), concreteArtifact) {
		t.Fatal("closed callable facet and runtime token were not separated")
	}
	matched := false
	for _, artifact := range concreteFamily.Artifacts() {
		matched = matched ||
			artifact.ArtifactKey() == concreteArtifact.ArtifactKey()
	}
	if !matched {
		t.Fatal("closed callable facet does not correspond to its runtime token")
	}
}
