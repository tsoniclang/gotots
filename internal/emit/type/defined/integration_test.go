package defined_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDefinedBasicFamilyExecutesDifferentially(t *testing.T) {
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
	for _, testCase := range []struct {
		name         string
		options      emit.Options
		nativeSuffix string
	}{
		{"number", definedNumberOptions(), ""},
		{
			"bigint",
			emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderPreserveGo,
			},
			"n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission, err := emit.CompileWithOptions(
				loaded.Program(),
				roots,
				testCase.options,
			)
			if err != nil {
				t.Fatalf("defined-basic compile failed: %v", err)
			}
			workingDirectory := t.TempDir()
			artifacts := printDefined(t, workingDirectory, emission)
			for _, forbidden := range []string{
				" as any",
				" as unknown",
				".call(",
				".apply(",
				".bind(",
			} {
				if strings.Contains(artifacts.printed, forbidden) {
					t.Fatalf(
						"defined artifact contains %q:\n%s",
						forbidden,
						artifacts.printed,
					)
				}
			}
			goOutput := runDefinedGo(t, workingDirectory)
			targetOutput := runDefinedTypeScript(
				t,
				workingDirectory,
				artifacts,
				testCase.nativeSuffix,
			)
			if targetOutput != goOutput {
				t.Fatalf(
					"TypeScript output differs from Go\nTypeScript:\n%s\nGo:\n%s",
					targetOutput,
					goOutput,
				)
			}
		})
	}
}
