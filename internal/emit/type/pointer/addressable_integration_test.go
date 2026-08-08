package pointer_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestAddressablePointersPrintAndTypecheckWithCanonicalMarkers(t *testing.T) {
	loaded := loadAddressablePointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	target := materializedModuleSource(t, workingDirectory, artifacts, "source.ts")

	for _, required := range []string{
		`from "@tsonic/core/types.js"`,
		`from "@tsonic/core/lang.js"`,
		"addressOf<",
		"allocatePointer<",
		"loadPointer<",
		"storePointer(",
		"equalPointer<",
		"goSliceAddress",
		"export class Box",
		"static Add(box:",
		"static Nil(box:",
	} {
		if !strings.Contains(target, required) {
			t.Fatalf("addressable pointer artifact lacks %q:\n%s", required, target)
		}
	}
	for _, forbidden := range []string{
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
		"GoPointer",
		"runtime/pointer",
		"goSliceAddressView",
		"export function Box_Add",
		"export function Box_Nil",
	} {
		if strings.Contains(target, forbidden) {
			t.Fatalf("addressable pointer artifact contains %q:\n%s", forbidden, target)
		}
	}
	for _, name := range []string{"NestedField", "DeepField"} {
		body := exportedFunction(t, target, name)
		if !strings.Contains(body, "addressOf<") ||
			!strings.Contains(body, "loadPointer(") {
			t.Fatalf("%s did not preserve its marker-selected location:\n%s", name, body)
		}
	}
	typecheckMaterializedTypeScript(t, workingDirectory, artifacts)
}

func TestAddressabilityDoesNotWrapUnrelatedLocals(t *testing.T) {
	loaded := loadAddressablePointerProject(t)
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(t, loaded, workingDirectory)
	source := materializedModuleSource(t, workingDirectory, artifacts, "source.ts")
	if strings.Contains(source, "delta$storage") {
		t.Fatal("unaddressed pointer-receiver argument became a storage cell")
	}
	if strings.Contains(source, "result$storage") &&
		!strings.Contains(source, "function NamedResult") {
		t.Fatal("storage evidence was not scoped to the selected declaration")
	}
}

func TestImportedPackageStoragePrintsAndTypechecks(t *testing.T) {
	projectDirectory, err := filepath.Abs(filepath.Join(
		repositoryRoot(),
		"testdata",
		"projects",
		"pointer-addressable",
	))
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./app",
	})
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeExportedProgram(
		t,
		program.Roots()[0],
		workingDirectory,
	)
	app := materializedModuleSource(t, workingDirectory, artifacts, "app.ts")
	for _, required := range []string{"addressOf<", "loadPointer(", "equalPointer<"} {
		if !strings.Contains(app, required) {
			t.Fatalf("imported pointer artifact lacks %q:\n%s", required, app)
		}
	}
	typecheckMaterializedTypeScript(t, workingDirectory, artifacts)
}

func materializedModuleSource(
	t *testing.T,
	workingDirectory string,
	artifacts materializedProgram,
	base string,
) string {
	t.Helper()
	path := filepath.Join(
		workingDirectory,
		filepath.FromSlash(strings.TrimPrefix(artifacts.module(t, base), "./")),
	)
	content, err := os.ReadFile(strings.TrimSuffix(path, ".js") + ".ts")
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func loadAddressablePointerProject(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: addressablePointerProjectDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func addressablePointerProjectDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"value",
		"pointer",
		"addressable",
	)
}

func exportedFunction(t *testing.T, source string, name string) string {
	t.Helper()
	start := strings.Index(source, "export function "+name+"(")
	if start < 0 {
		t.Fatalf("exported function %s is absent", name)
	}
	remainder := source[start:]
	end := strings.Index(remainder[1:], "\nexport function ")
	if end < 0 {
		return remainder
	}
	return remainder[:end+1]
}
