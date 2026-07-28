package conversion_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	conversionruntime "github.com/tsoniclang/gotots/internal/emit/runtime/conversion"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNumberToBigIntRuntimeIsOneTypedNonThrowingDefinition(t *testing.T) {
	factory := tsgo.NewFactory()
	statements, err := conversionruntime.Build(
		factory,
		[]api.RuntimeSymbol{api.RuntimeNumberToBigInt},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 {
		t.Fatalf("conversion runtime definitions = %d, want 1", len(statements))
	}
	client, err := tsgo.StartClient(repositoryRoot(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	filePath, err := tsgo.NewPath("runtime/conversion.ts")
	if err != nil {
		t.Fatal(err)
	}
	source := factory.SourceFile(
		statements,
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        filePath,
			Path:            filePath,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
	printed, err := client.PrintNode(source, tsgo.PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"export function goNumberToBigInt(value: number): bigint",
		"Number.isFinite(value)",
		"Math.trunc(value)",
		"BigInt(",
		": 0",
	} {
		if !strings.Contains(printed, required) {
			t.Fatalf("conversion runtime lacks %q:\n%s", required, printed)
		}
	}
	for _, forbidden := range []string{"any", "unknown", ".call(", ".apply(", ".bind("} {
		if strings.Contains(printed, forbidden) {
			t.Fatalf("conversion runtime contains %q:\n%s", forbidden, printed)
		}
	}
}

func repositoryRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve conversion-runtime repository root")
	}
	return filepath.Clean(
		filepath.Join(filepath.Dir(file), "..", "..", "..", ".."),
	)
}

func TestConversionRuntimeRejectsWrongDefinitionSets(t *testing.T) {
	factory := tsgo.NewFactory()
	for _, symbols := range [][]api.RuntimeSymbol{
		nil,
		{api.RuntimeNumberToBigInt, api.RuntimeNumberToBigInt},
		{api.RuntimeFloat32Round},
	} {
		if _, err := conversionruntime.Build(factory, symbols); err == nil {
			t.Fatalf("conversion runtime accepted %v", symbols)
		}
	}
}
