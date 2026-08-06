package defined_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRecursiveStructWithCallableKeepsNativeClassShape(t *testing.T) {
	target := compileDefinedSource(t, `package spelling

type Node struct {
	Next *Node
	Visit func()
}

func Identity(value Node) Node { return value }
`)
	if len(target) > 10_000 {
		t.Fatalf("recursive callable wrapper expanded to %d bytes", len(target))
	}
	for _, required := range []string{
		"export class Node {",
		"public Next: Node | undefined",
		"public Visit: (() => void) | undefined",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("recursive native class lacks %q:\n%s", required, target)
		}
	}
	if strings.Contains(target, "$Value") {
		t.Fatalf("recursive native class acquired a value facet:\n%s", target)
	}
	if strings.Contains(target, "$go$recovery") {
		t.Fatalf("source callable field exposes recovery authority:\n%s", target)
	}
}

func TestGenericDefinedArityFollowsDeclarationOrigin(t *testing.T) {
	target := compileDefinedSource(t, `package spelling

type CallbackHolder struct {
	Callback func(int32) bool
}

type Cache[T any] map[string]T

func Identity(value Cache[CallbackHolder]) Cache[CallbackHolder] {
	return value
}
`)
	for _, required := range []string{
		"export class Cache<T> {",
		"Cache<CallbackHolder>",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("generic defined value lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		"Cache<T, $Value",
		"Cache<CallbackHolder, ",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf(
				"generic defined value acquired profile facet %q:\n%s",
				forbidden,
				target,
			)
		}
	}
}

func TestDefinedBasicFamilyHasMinimalNominalShape(t *testing.T) {
	emission := compileDefinedFixture(t, emit.DefaultOptions())
	source := definedSourceFile(t, emission)
	classes := make(map[string]tsgo.ClassDeclaration)
	var alias tsgo.TypeAliasDeclaration
	for _, statement := range source.Statements() {
		switch statement := statement.(type) {
		case tsgo.ClassDeclaration:
			classes[statement.Name().Text()] = statement
		case tsgo.TypeAliasDeclaration:
			if statement.Name().Text() == "Alias" {
				alias = statement
			}
		}
	}
	for _, name := range []string{
		"Count",
		"Other",
		"Label",
		"Switch",
		"Ratio",
		"Narrow",
		"Signal",
	} {
		class := classes[name]
		if class == nil {
			t.Fatalf("defined class %s is absent", name)
		}
		assertDefinedClass(t, class)
		delete(classes, name)
	}
	if len(classes) != 0 {
		t.Fatalf("unexpected source classes remain: %v", classes)
	}
	if alias == nil {
		t.Fatal("Alias type declaration is absent")
	}
	target, ok := alias.Type().(tsgo.TypeReferenceNode)
	if !ok ||
		target.TypeName().(tsgo.Identifier).Text() != "Count" ||
		len(alias.TypeParameters()) != 0 {
		t.Fatalf("Alias target = %#v, want direct Count type alias", alias.Type())
	}
}

func TestDefinedNominalityGateDetectsRemovedBrands(t *testing.T) {
	workingDirectory := t.TempDir()
	emission := compileDefinedFixture(t, emit.DefaultOptions())
	artifacts := printDefined(t, workingDirectory, emission)
	runnerPath := filepath.Join(workingDirectory, "nominality.ts")
	writeDefinedFile(t, runnerPath, `import * as values from "`+artifacts.sourceModule+`";
const count: values.Count = values.CountFromInt(1);
const alias: values.Alias = count;
const other: values.Other = count;
void alias;
void other;
`)
	if err := typecheckDefined(
		workingDirectory,
		artifacts.paths,
		runnerPath,
	); err == nil {
		t.Fatal("unrelated defined types became structurally assignable")
	}

	replacements := 0
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := strings.ReplaceAll(
			string(content),
			"declare private readonly $goType: void;\n",
			"",
		)
		if mutated == string(content) {
			continue
		}
		replacements++
		writeDefinedFile(t, path, mutated)
	}
	if replacements == 0 {
		t.Fatal("nominality mutation removed no generated brands")
	}
	if err := typecheckDefined(
		workingDirectory,
		artifacts.paths,
		runnerPath,
	); err != nil {
		t.Fatalf("brand-removal foil did not expose structural assignability: %v", err)
	}
}

func TestDefinedUnderlyingSpellingDoesNotOwnRepresentation(t *testing.T) {
	int32Target := compileDefinedSource(t, `package spelling

type Count int32
func Identity(value Count) Count { return value }
`)
	runeTarget := compileDefinedSource(t, `package spelling

type Count rune
func Identity(value Count) Count { return value }
`)
	if int32Target != runeTarget {
		t.Fatalf(
			"equivalent int32/rune checker types emitted different targets\nint32:\n%s\nrune:\n%s",
			int32Target,
			runeTarget,
		)
	}
}

func TestDefinedLiteralOperationHasNoTransientWrapper(t *testing.T) {
	workingDirectory := t.TempDir()
	artifacts := printDefined(
		t,
		workingDirectory,
		compileDefinedFixture(t, emit.DefaultOptions()),
	)
	if strings.Contains(artifacts.printed, "new Count(2).$value") ||
		!strings.Contains(
			artifacts.printed,
			"return new Count(value.$value + 2);",
		) {
		t.Fatalf(
			"defined literal operation is not direct:\n%s",
			artifacts.printed,
		)
	}
}

func TestDefinedContainersKeepNominalityAtSourceBoundaries(t *testing.T) {
	workingDirectory := t.TempDir()
	artifacts := printDefined(
		t,
		workingDirectory,
		compileDefinedFixture(t, emit.DefaultOptions()),
	)
	for _, required := range []string{
		"CountArrayValues(): [\n    GoArray<Count, 2>,",
		"CountSliceValues(): RuntimeSlice<Count>",
		"CountMapLookup(values: GoMapValue<Count, Label>, key: Count)",
		"values.lookupOk(key)",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("defined container artifact lacks %q:\n%s", required, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"GoMap<Count, Label>",
		"GoMapValue<int32, Label>",
		"values.lookupOk(key.$value)",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("defined container artifact contains %q:\n%s", forbidden, artifacts.printed)
		}
	}
}

func TestLocalDefinedTypeAndAliasUseThePackageOwners(t *testing.T) {
	source := definedSourceFile(
		t,
		compileDefinedFixture(t, emit.DefaultOptions()),
	)
	var localTypes tsgo.FunctionDeclaration
	for _, statement := range source.Statements() {
		function, ok := statement.(tsgo.FunctionDeclaration)
		if ok && function.Name().Text() == "LocalTypes" {
			localTypes = function
			break
		}
	}
	if localTypes == nil {
		t.Fatal("LocalTypes target function is absent")
	}
	statements := localTypes.Body().(tsgo.Block).Statements()
	if len(statements) != 5 {
		t.Fatalf("LocalTypes statements = %d, want 5", len(statements))
	}
	class, ok := statements[0].(tsgo.ClassDeclaration)
	if !ok {
		t.Fatalf("local defined declaration = %#v", statements[0])
	}
	localName := class.Name().Text()
	if localName == "Count" {
		t.Fatal("local defined type collided with the package Count declaration")
	}
	if len(class.Modifiers()) != 0 || len(class.Members()) != 2 {
		t.Fatalf("local defined class has package-level shape: %#v", class)
	}
	alias, ok := statements[1].(tsgo.TypeAliasDeclaration)
	if !ok || alias.Name().Text() == "Alias" || len(alias.Modifiers()) != 0 {
		t.Fatalf("local alias declaration = %#v", statements[1])
	}
	target, ok := alias.Type().(tsgo.TypeReferenceNode)
	if !ok ||
		target.TypeName().(tsgo.Identifier).Text() != localName {
		t.Fatalf("local alias target = %#v", alias.Type())
	}
}

func assertDefinedClass(
	t *testing.T,
	class tsgo.ClassDeclaration,
) {
	t.Helper()
	const wantMembers = 2
	if kinds := modifierKinds(class.Modifiers()); len(kinds) != 1 ||
		kinds[0] != tsgo.SyntaxKindExportKeyword ||
		len(class.TypeParameters()) != 0 ||
		len(class.HeritageClauses()) != 0 ||
		len(class.Members()) != wantMembers {
		t.Fatalf(
			"class %s members = %d, want %d",
			class.Name().Text(),
			len(class.Members()),
			wantMembers,
		)
	}
	brand, ok := class.Members()[0].(tsgo.PropertyDeclaration)
	if !ok ||
		brand.Name().(tsgo.Identifier).Text() != "$goType" ||
		brand.Type().Kind() != tsgo.SyntaxKindVoidKeyword ||
		brand.Initializer() != nil {
		t.Fatalf("class %s brand has the wrong shape", class.Name().Text())
	}
	wantBrand := []tsgo.SyntaxKind{
		tsgo.SyntaxKindDeclareKeyword,
		tsgo.SyntaxKindPrivateKeyword,
		tsgo.SyntaxKindReadonlyKeyword,
	}
	if kinds := modifierKinds(brand.Modifiers()); !equalKinds(kinds, wantBrand) {
		t.Fatalf("class %s brand modifiers = %v", class.Name().Text(), kinds)
	}
	constructor, ok := class.Members()[1].(tsgo.ConstructorDeclaration)
	if !ok || len(constructor.Parameters()) != 1 {
		t.Fatalf("class %s constructor has the wrong shape", class.Name().Text())
	}
	parameter := constructor.Parameters()[0]
	if parameter.Name().(tsgo.Identifier).Text() != "$value" ||
		parameter.Type() == nil ||
		parameter.Initializer() != nil ||
		!equalKinds(
			modifierKinds(parameter.Modifiers()),
			[]tsgo.SyntaxKind{
				tsgo.SyntaxKindPublicKeyword,
				tsgo.SyntaxKindReadonlyKeyword,
			},
		) {
		t.Fatalf("class %s value carrier has the wrong shape", class.Name().Text())
	}
	body, ok := constructor.Body().(tsgo.Block)
	if !ok || len(body.Statements()) != 0 {
		t.Fatalf("class %s constructor is not empty", class.Name().Text())
	}
}

func modifierKinds(modifiers []tsgo.ModifierLike) []tsgo.SyntaxKind {
	result := make([]tsgo.SyntaxKind, len(modifiers))
	for index, modifier := range modifiers {
		result[index] = modifier.Kind()
	}
	return result
}

func equalKinds(left, right []tsgo.SyntaxKind) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func typecheckDefined(
	workingDirectory string,
	paths []string,
	runnerPath string,
) error {
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, paths...)
	arguments = append(arguments, runnerPath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	)
}

func compileDefinedFixture(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: definedFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(
		loaded.Program(),
		roots,
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func definedSourceFile(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			file.PackageName() == "definedbasic" {
			return file.SourceFile()
		}
	}
	t.Fatal("defined-basic target source is absent")
	return nil
}

func compileDefinedSource(t *testing.T, source string) string {
	t.Helper()
	directory := t.TempDir()
	writeDefinedFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/spelling\n\ngo 1.26.4\n",
	)
	writeDefinedFile(t, filepath.Join(directory, "source.go"), source)
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		target, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return target
	}
	t.Fatal("spelling fixture source target is absent")
	return ""
}
