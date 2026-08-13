package naming

import (
	"go/ast"
	"go/token"
	"go/types"
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
	registry := NewRegistry()
	if err := registry.indexPackageQualifiers(
		[]*types.Package{firstPackage, secondPackage},
	); err != nil {
		t.Fatal(err)
	}
	file := &File{owner: &Owner{registry: registry}}
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	first, err := file.InterfaceMethodName(types.NewFunc(
		token.NoPos,
		firstPackage,
		"visit",
		signature,
	))
	if err != nil {
		t.Fatal(err)
	}
	second, err := file.InterfaceMethodName(types.NewFunc(
		token.NoPos,
		secondPackage,
		"visit",
		signature,
	))
	if err != nil {
		t.Fatal(err)
	}
	if first != "$go$private$model$visit" ||
		second != "$go$private$model__package_1$visit" {
		t.Fatalf("private method names = %q / %q", first, second)
	}
	if strings.Contains(first+second, "example") {
		t.Fatalf("private method names repeat full paths: %q / %q", first, second)
	}
	firstToken, err := file.semanticGeneratedMethodName(
		"$goInterfaceMethod$",
		types.NewFunc(token.NoPos, firstPackage, "visit", signature),
		signature,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondToken, err := file.semanticGeneratedMethodName(
		"$goInterfaceMethod$",
		types.NewFunc(token.NoPos, secondPackage, "visit", signature),
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
