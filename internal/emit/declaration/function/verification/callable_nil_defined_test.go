package function_test

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

func TestCallableNilAndDefinedTypesCreateExactTargetShapes(t *testing.T) {
	loaded := loadCallableNilProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	printed := readMaterializedSource(t, artifacts, "source.ts")

	for _, required := range []string{
		"class Transform",
		"declare private readonly $goType: void;",
		"constructor(public readonly $value:",
		"| undefined",
		"new Transform(",
		"= value.$value;",
		`?? GoPanic.raiseRuntime("call of nil function")`,
		"GoPointer.cell<",
		"return value.$value === undefined;",
		"return !(value.$value === undefined);",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("callable artifact lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{
		".call(",
		".apply(",
		".bind(",
		" as any",
		" as unknown",
		"Object.assign(",
		`"$goType:`,
	} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("callable artifact contains %q:\n%s", forbidden, printed)
		}
	}

	nilCall := printedFunction(t, printed, "NilCallOrder")
	callee := strings.Index(nilCall, "const __gotots_callee_")
	argument := strings.Index(nilCall, "const __gotots_argument_")
	guard := strings.Index(nilCall, `?? GoPanic.raiseRuntime("call of nil function")`)
	call := strings.LastIndex(nilCall, ")(__gotots_argument_")
	if callee < 0 || argument < 0 || guard < 0 || call < 0 ||
		!(callee < argument && argument < guard && guard < call) {
		t.Fatalf("nil call order is not callee -> arguments -> guard -> direct call:\n%s", nilCall)
	}
	if strings.Contains(nilCall, "function (__gotots_callee_") {
		t.Fatalf("ordinary nil call contains an IIFE wrapper:\n%s", nilCall)
	}
	nilVoidCall := printedFunction(t, printed, "NilVoidCallOrder")
	voidCallee := strings.Index(nilVoidCall, "const __gotots_callee_")
	voidArgument := strings.Index(nilVoidCall, "const __gotots_argument_")
	voidGuard := strings.Index(nilVoidCall, `?? GoPanic.raiseRuntime("call of nil function")`)
	voidCall := strings.LastIndex(nilVoidCall, ")(__gotots_argument_")
	if voidCallee < 0 || voidArgument < 0 || voidGuard < 0 || voidCall < 0 ||
		!(voidCallee < voidArgument &&
			voidArgument < voidGuard &&
			voidGuard < voidCall) ||
		strings.Contains(nilVoidCall, "function (__gotots_callee_") {
		t.Fatalf("discarded nil call is not callee -> arguments -> guard -> direct call:\n%s", nilVoidCall)
	}
	shortCircuit := printedFunction(t, printed, "ShortCircuit")
	if strings.Contains(shortCircuit, "function (__gotots_callee_") {
		t.Fatalf("short-circuit callable path contains an IIFE:\n%s", shortCircuit)
	}

	source := callableNilSourceFile(t, loaded)
	classes := make(map[string]tsgo.ClassDeclaration)
	aliases := make(map[string]tsgo.TypeAliasDeclaration)
	for _, statement := range source.Statements() {
		switch declaration := statement.(type) {
		case tsgo.ClassDeclaration:
			classes[declaration.Name().Text()] = declaration
		case tsgo.TypeAliasDeclaration:
			aliases[declaration.Name().Text()] = declaration
		}
	}
	transform := classes["Transform"]
	if transform == nil {
		t.Fatal("defined callable Transform declaration is absent")
	}
	if len(transform.Members()) != 2 {
		t.Fatalf(
			"Transform members = %d, want only brand and value constructor",
			len(transform.Members()),
		)
	}
	brand, ok := transform.Members()[0].(tsgo.PropertyDeclaration)
	if !ok ||
		brand.Name().(tsgo.Identifier).Text() != "$goType" ||
		brand.Type().Kind() != tsgo.SyntaxKindVoidKeyword {
		t.Fatalf("Transform nominal brand has the wrong target shape: %#v", transform.Members()[0])
	}
	constructor, ok := transform.Members()[1].(tsgo.ConstructorDeclaration)
	if !ok || len(constructor.Parameters()) != 1 ||
		constructor.Parameters()[0].Name().(tsgo.Identifier).Text() != "$value" {
		t.Fatalf("Transform value constructor has the wrong target shape: %#v", transform.Members()[1])
	}
	transformAlias := aliases["TransformAlias"]
	if transformAlias == nil {
		t.Fatal("TransformAlias declaration is absent")
	}
	reference, ok := transformAlias.Type().(tsgo.TypeReferenceNode)
	if !ok || reference.TypeName().(tsgo.Identifier).Text() != "Transform" {
		t.Fatalf("TransformAlias target = %#v, want Transform", transformAlias)
	}
	rawAlias := aliases["RawAlias"]
	if rawAlias == nil {
		t.Fatal("RawAlias declaration is absent")
	}
	rawUnion, ok := rawAlias.Type().(tsgo.UnionTypeNode)
	if !ok || len(rawUnion.Types()) != 2 {
		t.Fatalf("RawAlias target = %#v, want callable | undefined", rawAlias)
	}
}

func TestCallableNilAndDefinedTypesExecuteDifferentially(t *testing.T) {
	loaded := loadCallableNilProject(t)
	workingDirectory := t.TempDir()
	goOutput := executeCallableNilGo(t, workingDirectory)
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	writeFile(t, runnerPath, `import "`+artifacts.initialization(t)+`";
import {
    ApplyDefined,
    ApplyRawAlias,
    AssignDefinedToRaw,
    CallPackage,
    Conditional,
    ConvertNilDefined,
    ConvertNilRaw,
    DefinedFromRaw,
    Increment,
    IsNilDefined,
    IsNilRaw,
    IsNonNilAlias,
    LocalDefined,
    LocalImplicitDefined,
    LocalNil,
    LocalVarCall,
    NewDefinedIsNil,
    NewRawIsNil,
    NilResult,
    NilCallOrder,
    NilVoidCallOrder,
	    Offset,
	    Other,
	    ImplicitOffset,
    PassRawToDefined,
    OtherFromDefined,
    PackageIsNil,
    ResetTrace,
    SetPackage,
    ShortCircuit,
    StoreThroughPointer,
    TraceValue,
	    TransformFromOther,
	    Transform,
    ReturnDefinedAsRaw,
    ReturnRawAsDefined,
} from "`+artifacts.module(t, "source.ts")+`";

const defined = DefinedFromRaw(Increment);
const other = OtherFromDefined(defined);
console.log(String(IsNilRaw(undefined)));
console.log(String(IsNilDefined(new Transform(undefined))));
console.log(String(IsNonNilAlias(defined)));
console.log(String(LocalNil()));
console.log(String(IsNilDefined(NilResult())));
console.log(String(IsNilDefined(ConvertNilRaw())));
console.log(String(IsNilRaw(ConvertNilDefined())));
console.log(String(IsNilDefined(TransformFromOther(new Other(undefined)))));
console.log(String(ApplyDefined(defined, 4)));
console.log(String(ApplyDefined(TransformFromOther(other), 5)));
console.log(String(ApplyRawAlias(Increment, 6)));
console.log(String(ApplyDefined(Offset(3), 7)));
console.log(String(PassRawToDefined(11)));
console.log(String(ApplyDefined(ReturnRawAsDefined(), 12)));
console.log(String(ApplyRawAlias(ReturnDefinedAsRaw(), 13)));
console.log(String(AssignDefinedToRaw(defined, 14)));
console.log(String(LocalVarCall(defined, 17)));
console.log(String(LocalImplicitDefined(18)));
console.log(String(ApplyDefined(ImplicitOffset(4), 15)));
console.log(String(LocalDefined(10)));
console.log(String(NewRawIsNil()));
console.log(String(NewDefinedIsNil()));
console.log(String(StoreThroughPointer(8)));
console.log(String(PackageIsNil()));
SetPackage(defined);
console.log(String(CallPackage(9)));
console.log(String(ShortCircuit(false)));
console.log(String(ShortCircuit(true)));
console.log(String(Conditional(true)));
console.log(String(Conditional(false)));
ResetTrace();
try {
    NilCallOrder();
    console.log("no-panic");
} catch {
    console.log("panic:" + String(TraceValue()));
}
ResetTrace();
try {
    NilVoidCallOrder();
    console.log("no-panic");
} catch {
    console.log("panic:" + String(TraceValue()));
}
`)
	targetOutput := executeMaterializedTypeScript(
		t,
		workingDirectory,
		artifacts,
		runnerPath,
	)
	if targetOutput != goOutput {
		t.Fatalf("TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s", targetOutput, goOutput)
	}
}

func TestDefinedCallableNominalityRejectsUnconvertedValues(t *testing.T) {
	loaded := loadCallableNilProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	runnerPath := filepath.Join(workingDirectory, "nominality.ts")
	writeFile(t, runnerPath, `import {
    DefinedFromRaw,
    Increment,
    type Other,
    type Transform,
} from "`+artifacts.module(t, "source.ts")+`";

const wrongTransform: Transform = Increment;
const wrongOther: Other = Increment;
const transform = DefinedFromRaw(Increment);
const wrongNominal: Other = transform;
void wrongTransform;
void wrongOther;
void wrongNominal;
`)
	if err := typecheckMaterializedTypeScript(
		workingDirectory,
		artifacts,
		runnerPath,
	); err == nil {
		t.Fatal("unconverted functions became assignable to defined callable types")
	}
}

func TestCallableNilGuardMutationsFailOwningEvidence(t *testing.T) {
	t.Run("guard before argument changes panic timing", func(t *testing.T) {
		loaded := loadCallableNilProject(t)
		workingDirectory := t.TempDir()
		artifacts := materializeExportedProgram(t, loaded, workingDirectory)
		sourcePath := materializedSourcePath(t, artifacts, "source.ts")
		printed := readFile(t, sourcePath)
		function := printedFunction(t, printed, "NilCallOrder")
		argumentStart := strings.Index(function, "const __gotots_argument_")
		returnStart := strings.LastIndex(function, "return (__gotots_callee_")
		if argumentStart < 0 || returnStart < 0 || argumentStart >= returnStart {
			t.Fatalf("nil-call mutation cannot locate owning statements:\n%s", function)
		}
		returnEnd := strings.Index(function[returnStart:], ";")
		if returnEnd < 0 {
			t.Fatalf("nil-call mutation cannot locate return terminator:\n%s", function)
		}
		returnEnd += returnStart + 1
		returnStatement := function[returnStart:returnEnd]
		callBoundary := strings.LastIndex(returnStatement, ")(__gotots_argument_")
		if callBoundary < 0 {
			t.Fatalf("nil-call mutation cannot locate guarded callee:\n%s", function)
		}
		guardedCallee := strings.TrimPrefix(
			returnStatement[:callBoundary+1],
			"return ",
		)
		mutatedFunction := function[:argumentStart] +
			"const __gotots_guarded_mutation = " + guardedCallee + ";\n    " +
			function[argumentStart:returnStart] +
			"return __gotots_guarded_mutation" +
			returnStatement[callBoundary+1:] +
			function[returnEnd:]
		writeFile(
			t,
			sourcePath,
			strings.Replace(printed, function, mutatedFunction, 1),
		)
		runnerPath := filepath.Join(workingDirectory, "runner.ts")
		writeFile(t, runnerPath, `import {
    NilCallOrder,
    ResetTrace,
    TraceValue,
} from "`+artifacts.module(t, "source.ts")+`";

ResetTrace();
try {
    NilCallOrder();
    console.log("no-panic");
} catch {
    console.log("panic:" + String(TraceValue()));
}
`)
		output := executeMaterializedTypeScript(
			t,
			workingDirectory,
			artifacts,
			runnerPath,
		)
		if output != "panic:1\n" {
			t.Fatalf("guard-order mutation output = %q, want panic before argument", output)
		}
	})

	t.Run("removed guard fails strict typecheck", func(t *testing.T) {
		loaded := loadCallableNilProject(t)
		workingDirectory := t.TempDir()
		artifacts := materializeExportedProgram(t, loaded, workingDirectory)
		sourcePath := materializedSourcePath(t, artifacts, "source.ts")
		printed := readFile(t, sourcePath)
		function := printedFunction(t, printed, "NilCallOrder")
		returnStart := strings.LastIndex(function, "return (__gotots_callee_")
		if returnStart < 0 {
			t.Fatalf("nil-call mutation cannot locate guard:\n%s", function)
		}
		returnEnd := strings.Index(function[returnStart:], ";")
		if returnEnd < 0 {
			t.Fatalf("nil-call mutation cannot locate return terminator:\n%s", function)
		}
		returnEnd += returnStart + 1
		returnStatement := function[returnStart:returnEnd]
		callBoundary := strings.LastIndex(returnStatement, ")(__gotots_argument_")
		const checkPrefix = "return ("
		const guardSuffix = ` ?? GoPanic.raiseRuntime("call of nil function")`
		guardBoundary := strings.Index(returnStatement, guardSuffix)
		if callBoundary < 0 || guardBoundary < 0 ||
			!strings.HasPrefix(returnStatement, checkPrefix) {
			t.Fatalf("nil-call mutation cannot locate guarded callee:\n%s", function)
		}
		nonNilCallee := returnStatement[len(checkPrefix):guardBoundary]
		mutatedFunction := function[:returnStart] +
			"return " + nonNilCallee + returnStatement[callBoundary+1:] +
			function[returnEnd:]
		writeFile(
			t,
			sourcePath,
			strings.Replace(printed, function, mutatedFunction, 1),
		)
		runnerPath := filepath.Join(workingDirectory, "guard.ts")
		writeFile(t, runnerPath, `import { NilCallOrder } from "`+
			artifacts.module(t, "source.ts")+`";
void NilCallOrder;
`)
		if err := typecheckMaterializedTypeScript(
			workingDirectory,
			artifacts,
			runnerPath,
		); err == nil {
			t.Fatal("removed nil guard passed strict typechecking")
		}
	})

	t.Run("removed defined projection fails strict typecheck", func(t *testing.T) {
		loaded := loadCallableNilProject(t)
		workingDirectory := t.TempDir()
		artifacts := materializeExportedProgram(t, loaded, workingDirectory)
		sourcePath := materializedSourcePath(t, artifacts, "source.ts")
		printed := readFile(t, sourcePath)
		function := printedFunction(t, printed, "ApplyDefined")
		if !strings.Contains(function, "value.$value") {
			t.Fatalf("defined-call mutation cannot locate projection:\n%s", function)
		}
		mutatedFunction := strings.Replace(function, "value.$value", "value", 1)
		writeFile(
			t,
			sourcePath,
			strings.Replace(printed, function, mutatedFunction, 1),
		)
		runnerPath := filepath.Join(workingDirectory, "projection.ts")
		writeFile(t, runnerPath, `import { ApplyDefined } from "`+
			artifacts.module(t, "source.ts")+`";
void ApplyDefined;
`)
		if err := typecheckMaterializedTypeScript(
			workingDirectory,
			artifacts,
			runnerPath,
		); err == nil {
			t.Fatal("removed defined-call projection passed strict typechecking")
		}
	})
}

func TestDefinedCallableErasedBrandMutationIsDetected(t *testing.T) {
	loaded := loadCallableNilProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	sourcePath := materializedSourcePath(t, artifacts, "source.ts")
	printed := readFile(t, sourcePath)
	const brand = "declare private readonly $goType: void;\n"
	if count := strings.Count(printed, brand); count < 2 {
		t.Fatalf("erased callable brands = %d, want at least Transform and Other", count)
	}
	writeFile(
		t,
		sourcePath,
		strings.ReplaceAll(printed, brand, ""),
	)
	runnerPath := filepath.Join(workingDirectory, "brand.ts")
	writeFile(t, runnerPath, `import {
    DefinedFromRaw,
    Increment,
    type Other,
} from "`+artifacts.module(t, "source.ts")+`";

const value: Other | undefined = DefinedFromRaw(Increment);
void value;
`)
	if err := typecheckMaterializedTypeScript(
		workingDirectory,
		artifacts,
		runnerPath,
	); err != nil {
		t.Fatalf("brand-removal mutation did not erase nominal distinction: %v", err)
	}
}

func loadCallableNilProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: callableNilProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func callableNilSourceFile(t *testing.T, loaded *load.Package) tsgo.SourceFile {
	t.Helper()
	emission, err := emit.Compile(
		loaded.Program(),
		mustExportedRoots(t, loaded),
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFileSource &&
			strings.HasSuffix(file.OutputPath(), "source.ts") {
			return file.SourceFile()
		}
	}
	t.Fatal("callable nil source target is absent")
	return nil
}

func mustExportedRoots(t *testing.T, loaded *load.Package) []emit.Root {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	return roots
}

func readMaterializedProgram(
	t *testing.T,
	artifacts materializedProgram,
) string {
	t.Helper()
	var content strings.Builder
	for _, path := range artifacts.targetPaths {
		value, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content.Write(value)
	}
	return content.String()
}

func materializedSourcePath(
	t *testing.T,
	artifacts materializedProgram,
	suffix string,
) string {
	t.Helper()
	for _, path := range artifacts.targetPaths {
		if strings.HasSuffix(filepath.ToSlash(path), suffix) {
			return path
		}
	}
	t.Fatalf("materialized program has no %s", suffix)
	return ""
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	value, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}

func printedFunction(t *testing.T, printed, name string) string {
	t.Helper()
	startMarker := "export function " + name
	start := strings.Index(printed, startMarker)
	if start < 0 {
		t.Fatalf("printed target has no %s", startMarker)
	}
	body := printed[start:]
	if next := strings.Index(body[len(startMarker):], "\nexport function "); next >= 0 {
		body = body[:len(startMarker)+next]
	}
	return body
}

func typecheckMaterializedTypeScript(
	workingDirectory string,
	artifacts materializedProgram,
	runnerPath string,
) error {
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, artifacts.targetPaths...)
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

func callableNilProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"expression",
		"function-value",
		"nil-defined",
	)
}
