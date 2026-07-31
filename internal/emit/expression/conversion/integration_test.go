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
		name    string
		options emit.Options
		suffix  string
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
			} {
				if strings.Contains(printed, forbidden) {
					t.Fatalf("conversion artifact contains %q:\n%s", forbidden, printed)
				}
			}
			for _, required := range []string{
				"export function goSliceArrayPointer<",
				"public static $view<T, N extends number>",
				"private readonly $offset: number",
				"static arrayRegion<L, T, S extends",
				"GoPointer<Pair, GoArray<int32, 2>>",
				"GoUnsafePointer.from(value)",
				"GoUnsafePointer.to<GoPointer<int32, int32>>(",
				"GoUnsafePointer.to<UnsafeBox>(",
				"unsafe.Pointer conversion requires an environment implementation",
				"export function GenericNilPointer<T, T$Pointer>(): T$Pointer | undefined {\n" +
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
				testCase.suffix,
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
