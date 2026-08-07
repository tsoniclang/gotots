package naming

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGenericInterfaceCallableFamilyAndRuntimeTokenAreDistinct(t *testing.T) {
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", `package interfacecallable

type Value[T any] interface { Get() T }
type Renamed[U any] interface { Get() U }
type IntValue interface { Get() int32 }
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
	names := testFileNames(
		t,
		NewOwner(sourcePackage.Scope(), info, registry),
		source,
		sourcePackage.Scope(),
		tsgo.NewFactory(),
		"modules/interfacecallable/source.ts",
		nil,
	)
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
		selected, callable := requirement.InterfaceMethodCallable()
		if !ok || !callable || selected != valueFamily.Artifacts()[0] {
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
		!slices.Contains(concreteFamily.Artifacts(), valueFamily.Artifacts()[0]) {
		t.Fatal("closed interface callable family did not retain its origin")
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
		matched = matched || artifact.ArtifactKey() == concreteArtifact.ArtifactKey()
	}
	if !matched {
		t.Fatal("closed callable facet does not correspond to its runtime token")
	}
}

func TestPointerRepresentationCanonicalizesGenericNominalFamily(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/family", "family")
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, sourcePackage, "T", nil),
		types.NewInterfaceType(nil, nil).Complete(),
	)
	typeName := types.NewTypeName(token.NoPos, sourcePackage, "Box", nil)
	origin := types.NewNamed(
		typeName,
		types.NewStruct(
			[]*types.Var{types.NewField(
				token.NoPos,
				sourcePackage,
				"Value",
				parameter,
				false,
			)},
			nil,
		),
		nil,
	)
	origin.SetTypeParams([]*types.TypeParam{parameter})
	instantiated, err := types.Instantiate(
		types.NewContext(),
		origin,
		[]types.Type{types.Typ[types.Int32]},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	declaration := pointerRepresentationFamily(types.NewPointer(origin))
	occurrence := pointerRepresentationFamily(types.NewPointer(instantiated))
	if !types.Identical(declaration, occurrence) ||
		!types.Identical(declaration.Elem(), origin) {
		t.Fatal("generic nominal pointer family did not converge on its origin")
	}
}

func TestCrossPackagePrivateLinkageUsesDefiningModule(t *testing.T) {
	provider := types.NewPackage("example.com/provider", "provider")
	consumer := types.NewPackage("example.com/consumer", "consumer")
	private := types.NewTypeName(token.NoPos, provider, "hidden", nil)
	public := types.NewTypeName(token.NoPos, provider, "Visible", nil)
	for _, object := range []types.Object{private, public} {
		types.NewNamed(object.(*types.TypeName), types.NewStruct(nil, nil), nil)
	}
	registry := NewRegistry()
	registry.assemblyPathByPackage[provider] = "packages/provider/package.ts"
	names := &File{
		owner:        &Owner{registry: registry},
		packageScope: consumer.Scope(),
	}
	binding := targetBinding{sourcePath: "modules/provider/source.ts"}
	for object, want := range map[types.Object]string{
		private: "modules/provider/source.ts",
		public:  "packages/provider/package.ts",
	} {
		got, crossPackage, err := names.sourceReferencePath(object, binding)
		if err != nil {
			t.Fatal(err)
		}
		if got != want || !crossPackage {
			t.Fatalf(
				"%s reference path = %q/%t, want %q/true",
				object.Name(),
				got,
				crossPackage,
				want,
			)
		}
	}
}
