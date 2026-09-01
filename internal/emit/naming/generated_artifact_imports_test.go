package naming

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func TestGeneratedArtifactNamesUseTheUniqueReadablePackageQualifier(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	object := types.NewTypeName(token.NoPos, sourcePackage, "Item", nil)
	if existing := sourcePackage.Scope().Insert(object); existing != nil {
		t.Fatal("type name was already present")
	}
	item := types.NewNamed(object, types.NewStruct(nil, nil), nil)
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}

	name, err := file.semanticGeneratedTypeName(
		"$goReflectType$",
		types.NewSlice(item),
	)
	if err != nil {
		t.Fatal(err)
	}
	if name != "$goReflectType$SliceOf_Named_model$Item" {
		t.Fatalf("generated name = %q", name)
	}
	if strings.Contains(name, "example") {
		t.Fatalf("generated name repeated the full import path: %q", name)
	}
}

func TestPrivateMethodNamesUseTheUniqueReadablePackageQualifier(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first/model", "model")
	secondPackage := types.NewPackage("example.com/second/model", "model")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	firstMethod := installPrivateInterfaceMethod(t, firstPackage, "First", "visit", signature)
	secondMethod := installPrivateInterfaceMethod(t, secondPackage, "Second", "visit", signature)
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers(
		[]*types.Package{firstPackage, secondPackage},
	); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	first, err := file.InterfaceMethodName(firstMethod)
	if err != nil {
		t.Fatal(err)
	}
	second, err := file.InterfaceMethodName(secondMethod)
	if err != nil {
		t.Fatal(err)
	}
	if first != "model$visit" ||
		second != "model__package_1$visit" {
		t.Fatalf("private method names = %q / %q", first, second)
	}
	if strings.Contains(first+second, "example") {
		t.Fatalf("private method names repeat full paths: %q / %q", first, second)
	}
	firstToken, err := file.semanticGeneratedMethodName(
		"$goInterfaceMethod$",
		firstMethod,
		signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondToken, err := file.semanticGeneratedMethodName(
		"$goInterfaceMethod$",
		secondMethod,
		signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstToken != "$goInterfaceMethod$model$visit$void_to_void" ||
		secondToken != "$goInterfaceMethod$model__package_1$visit$void_to_void" {
		t.Fatalf("private method tokens = %q / %q", firstToken, secondToken)
	}
}

func TestPrivateMethodNameStaysSourceLikeWithoutSemanticCollision(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	method := installPrivateInterfaceMethod(t, sourcePackage, "Contract", "visit", signature)
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	name, err := file.InterfaceMethodName(method)
	if err != nil || name != "visit" {
		t.Fatalf("private method name = %q, %v; want visit", name, err)
	}
}

func TestPrivateMethodNameIndexesAnonymousCallableContract(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	method := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"visit",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	contract := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	parameter := types.NewVar(token.NoPos, sourcePackage, "value", contract)
	callable := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"Use",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			types.NewTuple(parameter),
			nil,
			false,
		),
	)
	if previous := sourcePackage.Scope().Insert(callable); previous != nil {
		t.Fatal("callable was already present")
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	name, err := file.InterfaceMethodName(method)
	if err != nil || name != "visit" {
		t.Fatalf("anonymous contract method name = %q, %v; want visit", name, err)
	}
}

func TestPrivateMethodNameIndexesFunctionLocalInterface(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	method := types.NewFunc(
		token.NoPos,
		sourcePackage,
		"visit",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	contract := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	object := types.NewTypeName(token.Pos(3), sourcePackage, "Local", nil)
	types.NewNamed(object, contract, nil)
	information := &types.Info{
		Defs: map[*ast.Ident]types.Object{
			{Name: "Local", NamePos: token.Pos(3)}: object,
		},
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers(
		[]*types.Package{sourcePackage},
		information,
	); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	name, err := file.InterfaceMethodName(method)
	if err != nil || name != "visit" {
		t.Fatalf("local contract method name = %q, %v; want visit", name, err)
	}
}

func TestPrivateMethodNamesQualifyTargetLanguageHazards(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	hazards := []string{
		"constructor",
		"then",
		"name",
		"length",
		"caller",
		"arguments",
		"prototype",
	}
	methods := make([]*types.Func, 0, len(hazards))
	for index, hazard := range hazards {
		methods = append(methods, installPrivateInterfaceMethod(
			t,
			sourcePackage,
			"HazardContract"+strconv.Itoa(index),
			hazard,
			signature,
		))
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	for index, method := range methods {
		name, err := file.InterfaceMethodName(method)
		if err != nil {
			t.Fatal(err)
		}
		if expected := "model$" + hazards[index]; name != expected {
			t.Fatalf("hazard method name = %q, want %q", name, expected)
		}
	}
}

func TestPrivateMethodNamesDisambiguatePortableIdentifierCollisions(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	escaped := installPrivateInterfaceMethod(
		t,
		sourcePackage,
		"EscapedContract",
		"π",
		signature,
	)
	literal := installPrivateInterfaceMethod(
		t,
		sourcePackage,
		"LiteralContract",
		"__u3c0_",
		signature,
	)
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	escapedName, err := file.InterfaceMethodName(escaped)
	if err != nil {
		t.Fatal(err)
	}
	literalName, err := file.InterfaceMethodName(literal)
	if err != nil {
		t.Fatal(err)
	}
	if escapedName == literalName ||
		escapedName != "model$__u3c0___method_2" ||
		literalName != "model$__u3c0_" {
		t.Fatalf("colliding private method names = %q / %q", escapedName, literalName)
	}
}

func TestPrivateMethodNamesDoNotCollideWithExportedPortableIdentifier(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	exported := installPrivateInterfaceMethod(
		t,
		sourcePackage,
		"ExportedContract",
		"Π",
		signature,
	)
	private := installPrivateInterfaceMethod(
		t,
		sourcePackage,
		"PrivateContract",
		"__u3a0_",
		signature,
	)
	if !exported.Exported() || private.Exported() {
		t.Fatal("portable collision foil export status is invalid")
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{sourcePackage}); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	exportedName, err := file.InterfaceMethodName(exported)
	if err != nil {
		t.Fatal(err)
	}
	privateName, err := file.InterfaceMethodName(private)
	if err != nil {
		t.Fatal(err)
	}
	if exportedName != "__u3a0_" ||
		privateName != "model$__u3a0_" {
		t.Fatalf(
			"exported/private portable collision = %q / %q",
			exportedName,
			privateName,
		)
	}
}

func TestInterfaceAdapterSupportModulesFollowSemanticOwnership(t *testing.T) {
	modelPackage := types.NewPackage("example.com/model", "model")
	dependencyPackage := types.NewPackage("example.com/dependency", "dependency")
	modelType := installNamedStructType(t, modelPackage, "Item")
	dependencyType := installNamedStructType(t, dependencyPackage, "Value")
	genericModelType := installGenericNamedStructType(t, modelPackage, "Box")
	instantiatedModelType, err := types.Instantiate(
		nil,
		genericModelType,
		[]types.Type{dependencyType},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers([]*types.Package{
		modelPackage,
		dependencyPackage,
	}); err != nil {
		t.Fatal(err)
	}
	composite := types.NewStruct(
		[]*types.Var{
			types.NewVar(token.NoPos, nil, "Item", modelType),
			types.NewVar(token.NoPos, nil, "Value", dependencyType),
		},
		nil,
	)
	for _, test := range []struct {
		name   string
		source types.Type
		want   string
	}{
		{name: "language scalar", source: types.Typ[types.Int32], want: "language/scalars"},
		{name: "package-associated named type", source: modelType, want: "packages/model"},
		{name: "cross-package generic instance", source: instantiatedModelType, want: "composites/structs"},
		{name: "cross-package composite", source: composite, want: "composites/structs"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := registry.interfaceAdapterSupportModule(test.source)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("support module = %q, want %q", got, test.want)
			}
		})
	}
}

func installNamedStructType(
	t *testing.T,
	sourcePackage *types.Package,
	name string,
) *types.Named {
	t.Helper()
	object := types.NewTypeName(token.NoPos, sourcePackage, name, nil)
	named := types.NewNamed(object, types.NewStruct(nil, nil), nil)
	if previous := sourcePackage.Scope().Insert(object); previous != nil {
		t.Fatalf("type %s was already present", name)
	}
	return named
}

func installGenericNamedStructType(
	t *testing.T,
	sourcePackage *types.Package,
	name string,
) *types.Named {
	t.Helper()
	parameterName := types.NewTypeName(token.NoPos, sourcePackage, "T", nil)
	parameter := types.NewTypeParam(
		parameterName,
		types.NewInterfaceType(nil, nil).Complete(),
	)
	object := types.NewTypeName(token.NoPos, sourcePackage, name, nil)
	named := types.NewNamed(object, types.NewStruct(nil, nil), nil)
	named.SetTypeParams([]*types.TypeParam{parameter})
	if previous := sourcePackage.Scope().Insert(object); previous != nil {
		t.Fatalf("type %s was already present", name)
	}
	return named
}

func installPrivateInterfaceMethod(
	t *testing.T,
	sourcePackage *types.Package,
	typeName string,
	methodName string,
	signature *types.Signature,
) *types.Func {
	t.Helper()
	method := types.NewFunc(token.NoPos, sourcePackage, methodName, signature)
	contract := types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	object := types.NewTypeName(token.NoPos, sourcePackage, typeName, nil)
	types.NewNamed(object, contract, nil)
	if previous := sourcePackage.Scope().Insert(object); previous != nil {
		t.Fatalf("type %s was already present", typeName)
	}
	return method
}

func TestGeneratedArtifactLocalTokensRespectVisibleShadowing(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/model", "model")
	outerScope := types.NewScope(sourcePackage.Scope(), 1, 20, "outer")
	innerScope := types.NewScope(outerScope, 2, 10, "inner")
	outer := types.NewTypeName(3, sourcePackage, "Local", nil)
	inner := types.NewTypeName(4, sourcePackage, "Local", nil)
	outerScope.Insert(outer)
	innerScope.Insert(inner)
	owner := NewOwner(sourcePackage.Scope(), &types.Info{
		Defs: map[*ast.Ident]types.Object{
			{Name: "outer"}: outer,
			{Name: "inner"}: inner,
		},
	}, NewRegistry())
	file := &File{owner: owner}
	outerName, err := file.generatedNamedObjectToken(outer)
	if err != nil {
		t.Fatal(err)
	}
	innerName, err := file.generatedNamedObjectToken(inner)
	if err != nil {
		t.Fatal(err)
	}
	if outerName == innerName || !strings.Contains(innerName, "__shadow_") {
		t.Fatalf("visible local tokens = %q / %q", outerName, innerName)
	}
}

func TestGeneratedArtifactImportsUseShortestCollisionFreeFamilyName(t *testing.T) {
	firstName := "$goMap$MapOf_int32_To_string"
	secondName := "$goMap$MapOf_int64_To_string"
	first := generatedImportMapArtifact(t, strings.Repeat("1", 64), firstName)
	second := generatedImportMapArtifact(t, strings.Repeat("2", 64), secondName)
	file := &File{
		owner: &Owner{
			sourceNameBases: map[string]struct{}{},
		},
		importNames:     make(map[string]struct{}),
		generatedNames:  make(map[string]struct{}),
		artifactImports: make(map[generatedArtifactImport]string),
	}

	if got := file.generatedArtifactLocalName(first, firstName); got != "GoMap" {
		t.Fatalf("first map local name = %q, want GoMap", got)
	}
	if got := file.generatedArtifactLocalName(second, secondName); got != secondName {
		t.Fatalf("colliding map local name = %q, want full semantic name %q", got, secondName)
	}
	if got := file.generatedArtifactLocalName(first, firstName); got != "GoMap" {
		t.Fatalf("cached first map local name = %q, want GoMap", got)
	}
}

func TestGeneratedArtifactImportAvoidsAuthoredAndSemanticNameCollisions(t *testing.T) {
	name := "$goMap$MapOf_int32_To_string"
	artifact := generatedImportMapArtifact(t, strings.Repeat("3", 64), name)
	file := &File{
		owner: &Owner{
			sourceNameBases: map[string]struct{}{
				"GoMap": {},
				name:    {},
			},
		},
		importNames:     make(map[string]struct{}),
		generatedNames:  make(map[string]struct{}),
		artifactImports: make(map[generatedArtifactImport]string),
	}

	got := file.generatedArtifactLocalName(artifact, name)
	if got == "GoMap" || got == name || file.sourceNameExists(got) {
		t.Fatalf("collision-qualified map local name = %q", got)
	}
	if !strings.HasPrefix(got, "GoMap__from_") {
		t.Fatalf("collision-qualified map local name = %q, want semantic qualifier", got)
	}
}

func generatedImportMapArtifact(
	t *testing.T,
	key string,
	name string,
) *api.GeneratedArtifact {
	t.Helper()
	artifact, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactMapSpecialization,
		types.NewMap(types.Typ[types.Int32], types.Typ[types.String]),
		key,
		name,
		output.MapSpecializationSupportPath,
	)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
