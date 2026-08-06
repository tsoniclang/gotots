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
				"export function goSliceArrayPointer<",
				"public static $view<T, N extends number>",
				"private readonly $offset: number",
				"static arrayRegion<L, T, S extends",
				"GoPointer<Pair, GoArray<int32, 2>>",
				"GoUnsafePointer.from<int32, int32>(value, $goUnsafeCodec_",
				"GoUnsafePointer.to<int32, int32>(",
				"GoUnsafePointer.to<UnsafeBox, UnsafeBox$Storage>(",
				"GoUnsafePointer.toInteger(",
				"GoUnsafePointer.fromInteger(",
				"GoUnsafePointer.fromRelative(",
				"export const $goUnsafeCodec_",
				"new GoUnsafeCodec<",
				"export function String(value: gostring): gostring",
				"globalThis.String.fromCharCode",
				"export function GenericNilPointer<T>(): GoPointerType<T> | undefined {\n" +
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
			if got := strings.Count(
				printed,
				"GoUnsafePointer.fromRelative(",
			); got != 3 {
				t.Fatalf(
					"closed unsafe offsets = %d, want 3:\n%s",
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

func TestUnsafePointerIntegerConversionsEnforceLiveAddressBoundary(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		options       emit.Options
		uintptrOne    string
		pointerScalar string
	}{
		{
			name:          "number",
			options:       emit.DefaultOptions(),
			uintptrOne:    "1",
			pointerScalar: "1",
		},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderDirect,
			},
			uintptrOne:    "1n",
			pointerScalar: "1",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileConversions(t, testCase.options)
			workingDirectory := t.TempDir()
			targetPaths, sourceModule, _ := printConversions(
				t,
				workingDirectory,
				emission,
			)
			runner := `import * as values from "` + sourceModule + `";
import { GoPanic } from "./runtime/panic.js";
import { GoPointer } from "./runtime/pointer.js";

function fails(action: () => void): boolean {
    try {
        action();
        return false;
    } catch (failure) {
        if (failure instanceof GoPanic) return true;
        throw failure;
    }
}

console.log(fails(() => { values.IntegerToUnsafePointer(` + testCase.uintptrOne + `); }));
console.log(fails(() => {
    values.UnsafePointerToInteger(
        GoPointer.cell<number, number>(` + testCase.pointerScalar + `),
    );
}));
`
			if output := executeConversionTypeScript(
				t,
				workingDirectory,
				targetPaths,
				runner,
			); output != "true\nfalse\n" {
				t.Fatalf("unsafe integer boundary output = %q", output)
			}
		})
	}
}

func TestUnsafePointerMemoryAliasesAndOffsetsDifferentially(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		options emit.Options
	}{
		{
			name:    "number",
			options: emit.DefaultOptions(),
		},
		{
			name: "bigint",
			options: emit.Options{
				IntegerRepresentation: emit.IntegerRepresentationBigInt,
				EvaluationOrder:       emit.EvaluationOrderDirect,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			emission := compileConversions(t, testCase.options)
			workingDirectory := t.TempDir()
			targetPaths, sourceModule, _ := printConversions(
				t,
				workingDirectory,
				emission,
			)
			runner := `import { GoArray } from "./runtime/array.js";
import { GoPointer } from "./runtime/pointer.js";
import * as values from "` + sourceModule + `";

const scalar = GoPointer.cell<number, number>(16909060);
const bytes = GoPointer.cell<GoArray<number, 8>, GoArray<number, 8>>(
    GoArray.literal<number, 8>(8, 0, [0, 1, 2, 3, 4, 5, 6, 7], [1, 2, 3, 4, 5, 6, 7, 8]),
);
console.log(String(values.UnsafePointerAliases(scalar)));
console.log(String(values.UnsafePointerOffset(bytes)));
console.log(String(values.UnsafePointerInlineOffset(bytes)));
console.log(String(values.UnsafePointerInlineOffsetOrdered(bytes)), String(values.UnsafePointerOffsetTrace()));
console.log(String(values.UnsafePointerInlineNilOffset()));
console.log(String(values.UnsafePointerSafeThenUnsafe(scalar)));
console.log(String(values.UnsafeStructLayout()));
console.log(String(values.UnsafeStringHeaderLength()));
console.log(String(values.UnsafeSliceHeaderMutation()));
`
			targetOutput := executeConversionTypeScript(
				t,
				workingDirectory,
				targetPaths,
				runner,
			)
			goOutput := runUnsafePointerMemoryGo(t, workingDirectory)
			if targetOutput != goOutput {
				t.Fatalf(
					"unsafe-pointer output differs\nTypeScript:\n%s\nGo:\n%s",
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
