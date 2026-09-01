package defined_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedNumericConversionNeedsNoInferenceAnnotation(t *testing.T) {
	workingDirectory := t.TempDir()
	artifacts := printDefined(
		t,
		workingDirectory,
		compileDefinedFixture(t, definedNumberOptions()),
	)
	for _, declaration := range []string{
		"let shortConverted = value;",
		"let declaredConverted = value;",
	} {
		if !strings.Contains(artifacts.printed, declaration) {
			t.Fatalf("generated numeric conversion lacks %q:\n%s", declaration, artifacts.printed)
		}
	}
	for _, forbidden := range []string{
		"let shortConverted: int32",
		"let declaredConverted: int32",
		"Count.$goType",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("generated numeric conversion retains %q:\n%s", forbidden, artifacts.printed)
		}
	}
	runnerPath := filepath.Join(workingDirectory, "inference.ts")
	writeDefinedFile(t, runnerPath, "export {};\n")
	if err := typecheckDefined(
		workingDirectory,
		artifacts.paths,
		runnerPath,
	); err != nil {
		t.Fatalf("direct generated-numeric inference failed: %v", err)
	}
}
