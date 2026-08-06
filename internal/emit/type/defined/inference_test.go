package defined_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
)

func TestGeneratedNumericConversionAnnotatesInferenceBoundary(t *testing.T) {
	workingDirectory := t.TempDir()
	artifacts := printDefined(
		t,
		workingDirectory,
		compileDefinedFixture(t, emit.DefaultOptions()),
	)
	declarations := map[string]string{
		"let shortConverted: int32 = value;":    "let shortConverted = value;",
		"let declaredConverted: int32 = value;": "let declaredConverted = value;",
	}
	for declaration := range declarations {
		if !strings.Contains(artifacts.printed, declaration) {
			t.Fatalf(
				"generated numeric conversion lacks %q:\n%s",
				declaration,
				artifacts.printed,
			)
		}
	}
	mutations := 0
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		mutated := string(content)
		for declaration, erased := range declarations {
			mutated = strings.Replace(mutated, declaration, erased, 1)
		}
		if mutated == string(content) {
			continue
		}
		mutations++
		writeDefinedFile(t, path, mutated)
	}
	if mutations != 1 {
		t.Fatalf("files with inference-annotation mutations = %d, want one", mutations)
	}
	runnerPath := filepath.Join(workingDirectory, "inference-mutation.ts")
	writeDefinedFile(t, runnerPath, "export {};\n")
	if err := typecheckDefined(
		workingDirectory,
		artifacts.paths,
		runnerPath,
	); err == nil {
		t.Fatal("erased generated-numeric inference annotation typechecked")
	}
}
