package emit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	anonymousstruct "github.com/tsoniclang/gotots/internal/emit/type/anonymousstruct"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAnonymousStructCanonicalizationUsesExactGoTypeIdentity(t *testing.T) {
	sourceFile, sourcePackage, info, structTypes := checkAnonymousStructs(t, `package identity

func Identical() {
	var _ struct{ Value int32 }
	var _ struct{ Value int32 }
}

func Distinct() {
	var _ struct{ Value int32 `+"`json:\"a\"`"+` }
	var _ struct{ Value int32 `+"`json:\"b\"`"+` }
	var _ struct{ _ int32 }
	var _ struct{ _ int64 }
	var _ struct{ Left int32 }
	var _ struct{ Right int32 }
}
`)
	if len(structTypes) != 8 {
		t.Fatalf("anonymous struct types = %d, want 8", len(structTypes))
	}
	registry := newDeclarationRegistry()
	names := newNameOwnerWithRegistry(
		sourcePackage.Scope(),
		info,
		registry,
	).ForFile(
		sourceFile,
		sourcePackage.Scope(),
		tsgo.NewFactory(),
		"modules/identity/source.ts",
		nil,
	).(*fileNames)
	references := make([]api.NameReference, len(structTypes))
	for index, structType := range structTypes {
		var err error
		references[index], err = names.AnonymousStruct(
			structType.target,
			api.AnonymousStructDemandDefinition,
		)
		if err != nil {
			t.Fatal(err)
		}
	}
	if references[0].Name() != references[1].Name() {
		t.Fatal("types.Identical anonymous shapes did not canonicalize")
	}
	for index := 2; index < len(references); index += 2 {
		if references[index].Name() == references[index+1].Name() {
			t.Fatalf("non-identical anonymous structs %d/%d unified", index, index+1)
		}
	}
	for _, binding := range registry.anonymousStructs {
		if len(binding.name) > len("$goStruct_")+20 {
			t.Fatalf("anonymous struct target name is unbounded: %q", binding.name)
		}
	}
}

func TestAnonymousStructLocalComponentsOwnDistinctLexicalArtifacts(t *testing.T) {
	sourceFile, sourcePackage, info, structTypes := checkAnonymousStructs(t, `package identity

func First() {
	type Local int32
	var _ struct{ Value Local }
}

func Second() {
	type Local int32
	var _ struct{ Value Local }
}
`)
	if len(structTypes) != 2 ||
		types.Identical(structTypes[0].target, structTypes[1].target) {
		t.Fatal("same-spelled local named components did not retain exact Go identity")
	}
	qualifier := func(sourcePackage *types.Package) string {
		return sourcePackage.Name()
	}
	if types.TypeString(structTypes[0].target, qualifier) !=
		types.TypeString(structTypes[1].target, qualifier) {
		t.Fatal("spelling-only identity foil is not actually identical")
	}
	registry := newDeclarationRegistry()
	var functions []*types.Func
	var functionDeclarations []*ast.FuncDecl
	for _, declaration := range sourceFile.Decls {
		function := declaration.(*ast.FuncDecl)
		owner := info.Defs[function.Name].(*types.Func)
		functions = append(functions, owner)
		functionDeclarations = append(functionDeclarations, function)
		if err := registry.reserve(owner, targetBinding{
			name:         owner.Name(),
			sourceFile:   sourceFile,
			sourcePath:   "modules/identity/source.ts",
			moduleExport: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	names := newNameOwnerWithRegistry(
		sourcePackage.Scope(),
		info,
		registry,
	).ForFile(
		sourceFile,
		sourcePackage.Scope(),
		tsgo.NewFactory(),
		"modules/identity/source.ts",
		nil,
	).(*fileNames)
	var references []api.NameReference
	for index, structType := range structTypes {
		finish, err := names.beginArtifact(
			sourceArtifactOwner(functions[index]),
			functionDeclarations[index],
			sourceFile,
			"modules/identity/source.ts",
		)
		if err != nil {
			t.Fatal(err)
		}
		reference, err := names.AnonymousStruct(
			structType.target,
			api.AnonymousStructDemandDefinition,
		)
		finish()
		if err != nil {
			t.Fatal(err)
		}
		references = append(references, reference)
	}
	if references[0].Name() == references[1].Name() {
		t.Fatal("same-spelled local named types unified lexical artifacts")
	}
	for _, binding := range registry.anonymousStructs {
		if binding.owner.Placement() !=
			api.GeneratedArtifactPlacementLexical ||
			binding.owner.OutputPath() != "" ||
			!binding.owner.LexicalOwner().Valid() ||
			binding.owner.LexicalAnchor() == nil {
			t.Fatalf("lexical binding = %#v", binding)
		}
	}
}

func TestAnonymousStructFingerprintCollisionStillExactJoins(t *testing.T) {
	_, _, _, structTypes := checkAnonymousStructs(t, `package collision

var First struct{ Value int32 }
var Second struct{ Value int64 }
`)
	if len(structTypes) != 2 {
		t.Fatalf("anonymous struct types = %d, want 2", len(structTypes))
	}
	objectIdentity := func(object *types.TypeName) (string, error) {
		return object.Name(), nil
	}
	firstKeys, err := anonymousstruct.BuildKeys(
		structTypes[0].target,
		objectIdentity,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondKeys, err := anonymousstruct.BuildKeys(
		structTypes[1].target,
		objectIdentity,
	)
	if err != nil {
		t.Fatal(err)
	}
	secondKeys.Fingerprint = firstKeys.Fingerprint
	registry := newDeclarationRegistry()
	placement := moduleAnonymousStructPlacement()
	first, err := registry.internAnonymousStruct(
		firstKeys,
		structTypes[0].target,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internAnonymousStruct(
		secondKeys,
		structTypes[1].target,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first.owner.ArtifactKey() == second.owner.ArtifactKey() ||
		first.name == second.name ||
		len(registry.anonymousStructBuckets[firstKeys.Fingerprint]) != 2 {
		t.Fatal("forced fingerprint collision unified non-identical Go types")
	}

	t.Run("artifact key", func(t *testing.T) {
		registry := newDeclarationRegistry()
		if _, err := registry.internAnonymousStruct(
			firstKeys,
			structTypes[0].target,
			placement,
		); err != nil {
			t.Fatal(err)
		}
		collision := secondKeys
		collision.Artifact = firstKeys.Artifact
		if _, err := registry.internAnonymousStruct(
			collision,
			structTypes[1].target,
			placement,
		); err == nil {
			t.Fatal("artifact-key collision unified non-identical Go types")
		}
	})

	t.Run("target prefix", func(t *testing.T) {
		registry := newDeclarationRegistry()
		if _, err := registry.internAnonymousStruct(
			firstKeys,
			structTypes[0].target,
			placement,
		); err != nil {
			t.Fatal(err)
		}
		collision := secondKeys
		collision.Artifact =
			firstKeys.Artifact[:20] + secondKeys.Artifact[20:]
		if collision.Artifact == firstKeys.Artifact {
			t.Fatal("target-prefix collision fixture is identical")
		}
		if _, err := registry.internAnonymousStruct(
			collision,
			structTypes[1].target,
			placement,
		); err == nil {
			t.Fatal("target-name prefix collision was accepted")
		}
	})
}

func TestAnonymousStructCrossPackageOwnershipIgnoresFirstEncounter(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	firstType := types.NewStruct(
		[]*types.Var{
			types.NewField(0, firstPackage, "Value", types.Typ[types.Int32], false),
		},
		nil,
	)
	secondType := types.NewStruct(
		[]*types.Var{
			types.NewField(0, secondPackage, "Value", types.Typ[types.Int32], false),
		},
		nil,
	)
	if !types.Identical(firstType, secondType) {
		t.Fatal("cross-package exported-field shapes are not identical")
	}
	identity := func(object *types.TypeName) (string, error) {
		return object.Name(), nil
	}
	firstKeys, err := anonymousstruct.BuildKeys(firstType, identity)
	if err != nil {
		t.Fatal(err)
	}
	secondKeys, err := anonymousstruct.BuildKeys(secondType, identity)
	if err != nil {
		t.Fatal(err)
	}
	if firstKeys != secondKeys {
		t.Fatalf("identical cross-package keys = %#v/%#v", firstKeys, secondKeys)
	}
	firstOwner, err := newDeclarationRegistry().internAnonymousStruct(
		firstKeys,
		firstType,
		moduleAnonymousStructPlacement(),
	)
	if err != nil {
		t.Fatal(err)
	}
	secondOwner, err := newDeclarationRegistry().internAnonymousStruct(
		secondKeys,
		secondType,
		moduleAnonymousStructPlacement(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if firstOwner.owner.ArtifactKey() != secondOwner.owner.ArtifactKey() ||
		firstOwner.owner.TargetName() != secondOwner.owner.TargetName() ||
		firstOwner.owner.OutputPath() != secondOwner.owner.OutputPath() {
		t.Fatal("first encounter changed cross-package anonymous-struct ownership")
	}
}

func moduleAnonymousStructPlacement() anonymousStructPlacement {
	return anonymousStructPlacement{
		kind:       api.GeneratedArtifactPlacementCompilation,
		outputPath: "support/anonymous-structs.ts",
	}
}

type checkedAnonymousStruct struct {
	source *ast.StructType
	target *types.Struct
}

func checkAnonymousStructs(
	t *testing.T,
	source string,
) (*ast.File, *types.Package, *types.Info, []checkedAnonymousStruct) {
	t.Helper()
	fileSet := token.NewFileSet()
	sourceFile, err := parser.ParseFile(
		fileSet,
		"source.go",
		source,
		parser.SkipObjectResolution,
	)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
		Types: make(map[ast.Expr]types.TypeAndValue),
	}
	sourcePackage, err := new(types.Config).Check(
		"example.com/identity",
		fileSet,
		[]*ast.File{sourceFile},
		info,
	)
	if err != nil {
		t.Fatal(err)
	}
	var result []checkedAnonymousStruct
	ast.Inspect(sourceFile, func(node ast.Node) bool {
		sourceStruct, ok := node.(*ast.StructType)
		if !ok {
			return true
		}
		target, ok := types.Unalias(info.TypeOf(sourceStruct)).(*types.Struct)
		if !ok {
			t.Fatalf("struct type evidence = %T", info.TypeOf(sourceStruct))
		}
		result = append(result, checkedAnonymousStruct{
			source: sourceStruct,
			target: target,
		})
		return true
	})
	return sourceFile, sourcePackage, info, result
}
