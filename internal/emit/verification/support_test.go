package emit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func sourceModuleForExport(
	t *testing.T,
	artifacts waveFourArtifacts,
	workingDirectory string,
	name string,
) string {
	t.Helper()
	var selected string
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		printed := string(content)
		if !strings.Contains(
			printed,
			"export function "+name+"(",
		) {
			continue
		}
		if selected != "" {
			t.Fatalf("multiple source modules export %s", name)
		}
		selected = path
	}
	if selected == "" {
		t.Fatalf("no source module exports %s", name)
	}
	relative, err := filepath.Rel(workingDirectory, selected)
	if err != nil {
		t.Fatal(err)
	}
	return "./" + strings.TrimSuffix(filepath.ToSlash(relative), ".ts") + ".js"
}

func environmentDeclarationLine(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, prefix)
	if start < 0 {
		t.Fatalf("environment declaration lacks %q:\n%s", prefix, printed)
	}
	end := strings.IndexByte(printed[start:], '\n')
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+end]
}

func waveNineOptions() emit.Options {
	options := emit.DefaultOptions()
	return options
}

func waveNineConcurrencyDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"wave9",
	)
}

func packageStateExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-source.ts",
		)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return scalarSupportExpectedPath()
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func packageInitializationExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		name := "expected-source.ts"
		if file.PackageName() == "sideeffect" {
			name = "expected-" + filepath.Base(file.OutputPath())
		}
		return filepath.Join(projectDirectory, file.PackageName(), name)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return scalarSupportExpectedPath()
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func scalarSupportExpectedPath() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"support",
		"scalars-int32.ts",
	)
}
