package conversion_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestConversionsExecuteDifferentially(t *testing.T) {
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
				EvaluationOrder:       emit.EvaluationOrderDirect,
			},
			"n",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileConversions(t, testCase.options)
			workingDirectory := t.TempDir()
			targetPaths, sourceModule, printed := printConversions(
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
				"unsafe.Pointer conversion requires an environment implementation",
			} {
				if strings.Contains(printed, forbidden) {
					t.Fatalf("conversion artifact contains %q:\n%s", forbidden, printed)
				}
			}
			for _, required := range []string{
				"export function goSliceArrayPointer<T, N extends number>",
				"public static $view<T, N extends number>",
				"private readonly $offset: number",
				"projectPointer<T | undefined, GoArray<T, N>>",
				"Pointer<Pair>",
				"export function String(value: gostring): gostring",
				"globalThis.String.fromCharCode",
				"export function GenericNilPointer<T>(): Pointer<T> | undefined {\n" +
					"    return void 0;\n}",
			} {
				if !strings.Contains(printed, required) {
					t.Fatalf(
						"slice-array pointer artifact lacks %q:\n%s",
						required,
						printed,
					)
				}
			}
			if strings.Contains(printed, "+= String.fromCharCode") {
				t.Fatalf("string conversion uses a shadowable target intrinsic:\n%s", printed)
			}
			if got := strings.Count(
				printed,
				"export function goSliceArrayPointer<",
			); got != 1 {
				t.Fatalf(
					"slice-array pointer helpers = %d, want one:\n%s",
					got,
					printed,
				)
			}
			if got := strings.Count(printed, "static $convert("); got != 0 {
				t.Fatalf(
					"superseded field-projection conversion definitions = %d, want 0:\n%s",
					got,
					printed,
				)
			}
			if got := strings.Count(
				printed,
				"TaggedRight.$fromStorage(",
			); got != 3 {
				t.Fatalf(
					"TaggedRight storage conversion uses = %d, want 3:\n%s",
					got,
					printed,
				)
			}
			goOutput := runConversionGo(t, workingDirectory)
			targetOutput := runConversionTypeScript(
				t,
				workingDirectory,
				targetPaths,
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

func compileConversions(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: conversionFixtureDirectory(),
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
		t.Fatalf("conversion compile failed: %v", err)
	}
	return emission
}
