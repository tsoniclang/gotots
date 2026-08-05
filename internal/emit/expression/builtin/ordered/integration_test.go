package ordered_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestOrderedBuiltinsExecuteDifferentially(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		options    emit.Options
		wideSuffix string
	}{
		{"number", emit.DefaultOptions(), ""},
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
			emission := compileOrdered(t, testCase.options)
			workingDirectory := t.TempDir()
			paths, sourceModule, printed := printOrdered(
				t,
				workingDirectory,
				emission,
			)
			for _, forbidden := range []string{
				" as any",
				" as unknown",
				".call(",
				".apply(",
				".bind(",
			} {
				if strings.Contains(printed, forbidden) {
					t.Fatalf("ordered artifact contains %q:\n%s", forbidden, printed)
				}
			}
			goOutput := runOrderedGo(t, workingDirectory)
			targetOutput := runOrderedTypeScript(
				t,
				workingDirectory,
				paths,
				sourceModule,
				testCase.wideSuffix,
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

func compileOrdered(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: orderedFixtureDirectory(),
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
		t.Fatalf("ordered compile failed: %v", err)
	}
	return emission
}
