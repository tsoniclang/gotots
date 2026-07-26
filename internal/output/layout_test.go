package output

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
)

func TestLayoutOwnsCheckoutIndependentModulePackageAndImportPaths(t *testing.T) {
	projectDirectory := filepath.Join("..", "..", "testdata", "projects", "demand-program")
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	apiPackage := program.PackageByPath("example.com/demand/api")
	servicePackage := program.PackageByPath("example.com/demand/service")
	apiPath, err := SourcePath(apiPackage, apiPackage.Files()[0])
	if err != nil {
		t.Fatal(err)
	}
	servicePath, err := SourcePath(servicePackage, servicePackage.Files()[0])
	if err != nil {
		t.Fatal(err)
	}
	const moduleKey = "9dca3bbab95799300693177714a7b8334cb6334dc73709afd3f788dfcab93b6b"
	if apiPath != "modules/"+moduleKey+"/api/api.ts" {
		t.Fatalf("api path = %q", apiPath)
	}
	if servicePath != "modules/"+moduleKey+"/service/service.ts" {
		t.Fatalf("service path = %q", servicePath)
	}
	assemblyPath, err := PackageAssemblyPath(apiPackage)
	if err != nil {
		t.Fatal(err)
	}
	if assemblyPath != "packages/"+moduleKey+"/api/package.ts" {
		t.Fatalf("assembly path = %q", assemblyPath)
	}
	statePath, err := PackageStatePath(apiPackage)
	if err != nil {
		t.Fatal(err)
	}
	if statePath != "packages/"+moduleKey+"/api/state.ts" {
		t.Fatalf("state path = %q", statePath)
	}
	specifer, err := ModuleSpecifier(apiPath, servicePath)
	if err != nil {
		t.Fatal(err)
	}
	if specifer != "../service/service.js" {
		t.Fatalf("module specifier = %q, want ../service/service.js", specifer)
	}
}

func TestLayoutRejectsForeignFilesAndSameModuleImports(t *testing.T) {
	projectDirectory := filepath.Join("..", "..", "testdata", "projects", "demand-program")
	program, err := load.Load(context.Background(), load.Request{
		Directory: projectDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	apiPackage := program.PackageByPath("example.com/demand/api")
	servicePackage := program.PackageByPath("example.com/demand/service")
	if _, err := SourcePath(apiPackage, servicePackage.Files()[0]); err == nil {
		t.Fatal("foreign source file was accepted")
	}
	apiPath, err := SourcePath(apiPackage, apiPackage.Files()[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ModuleSpecifier(apiPath, apiPath); err == nil {
		t.Fatal("same-module import was accepted")
	}
}

func TestScalarSupportPathProducesCanonicalRelativeSpecifier(t *testing.T) {
	const sourcePath = "modules/example/package/source.ts"
	if ScalarSupportPath != "support/scalars.ts" {
		t.Fatalf("scalar support path = %q, want support/scalars.ts", ScalarSupportPath)
	}
	specifier, err := ModuleSpecifier(sourcePath, ScalarSupportPath)
	if err != nil {
		t.Fatal(err)
	}
	if specifier != "../../../support/scalars.js" {
		t.Fatalf(
			"scalar support specifier = %q, want ../../../support/scalars.js",
			specifier,
		)
	}
}

func TestProgramInitializationPathProducesCanonicalPackageSpecifier(t *testing.T) {
	if ProgramInitializationPath != "program.ts" {
		t.Fatalf(
			"program initialization path = %q, want program.ts",
			ProgramInitializationPath,
		)
	}
	const assemblyPath = "packages/example/api/package.ts"
	specifier, err := ModuleSpecifier(ProgramInitializationPath, assemblyPath)
	if err != nil {
		t.Fatal(err)
	}
	if specifier != "./packages/example/api/package.js" {
		t.Fatalf(
			"program package specifier = %q, want ./packages/example/api/package.js",
			specifier,
		)
	}
}

func TestLayoutIsStableAcrossCheckoutRelocationAndSeparatesModuleVersions(t *testing.T) {
	sourceDirectory := filepath.Join(
		"..",
		"..",
		"testdata",
		"projects",
		"demand-program",
	)
	relocatedDirectory := filepath.Join(t.TempDir(), "relocated")
	for _, relativePath := range []string{
		"go.mod",
		"api/api.go",
		"mathx/math.go",
		"service/service.go",
	} {
		source, err := os.ReadFile(filepath.Join(sourceDirectory, relativePath))
		if err != nil {
			t.Fatal(err)
		}
		targetPath := filepath.Join(relocatedDirectory, relativePath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetPath, source, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	original, err := load.Load(context.Background(), load.Request{
		Directory: sourceDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	relocated, err := load.Load(context.Background(), load.Request{
		Directory: relocatedDirectory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	originalAPI := original.PackageByPath("example.com/demand/api")
	relocatedAPI := relocated.PackageByPath("example.com/demand/api")
	originalPath, err := SourcePath(originalAPI, originalAPI.Files()[0])
	if err != nil {
		t.Fatal(err)
	}
	relocatedPath, err := SourcePath(relocatedAPI, relocatedAPI.Files()[0])
	if err != nil {
		t.Fatal(err)
	}
	if relocatedPath != originalPath {
		t.Fatalf("relocated path = %q, want %q", relocatedPath, originalPath)
	}
	if moduleKey("example.com/dependency", "v1.0.0") ==
		moduleKey("example.com/dependency", "v1.0.1") {
		t.Fatal("distinct semantic module versions share one output owner")
	}
}
