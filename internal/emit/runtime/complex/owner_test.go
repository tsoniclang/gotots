package complex_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	runtimecomplex "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestComplexRuntimeOwnsOneDefinitionPerClosedSymbol(t *testing.T) {
	symbols := []api.RuntimeSymbol{
		api.RuntimeComplex64,
		api.RuntimeComplex128,
		api.RuntimeComplexDivide,
		api.RuntimeComplex64Add,
		api.RuntimeComplex64Sub,
		api.RuntimeComplex64Mul,
		api.RuntimeComplex64Div,
		api.RuntimeComplex64Neg,
		api.RuntimeComplex64Equal,
		api.RuntimeComplex128Add,
		api.RuntimeComplex128Sub,
		api.RuntimeComplex128Mul,
		api.RuntimeComplex128Div,
		api.RuntimeComplex128Neg,
		api.RuntimeComplex128Equal,
	}
	definitions, err := runtimeemission.Build(
		tsgo.NewFactory(),
		api.RuntimeModuleComplex,
		symbols,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != len(symbols) {
		t.Fatalf(
			"complex definitions = %d, want %d",
			len(definitions),
			len(symbols),
		)
	}
	for index, definition := range definitions {
		if definition.Symbol() != symbols[index] ||
			definition.Statement() == nil {
			t.Fatalf(
				"complex definition %d = %#v, want symbol %d",
				index,
				definition,
				symbols[index],
			)
		}
	}
}

func TestComplexRuntimeRejectsDuplicateAndForeignSymbols(t *testing.T) {
	for _, symbols := range [][]api.RuntimeSymbol{
		{api.RuntimeComplex64, api.RuntimeComplex64},
		{api.RuntimeFloat32Round},
	} {
		_, err := runtimeemission.Build(
			tsgo.NewFactory(),
			api.RuntimeModuleComplex,
			symbols,
			api.ConcurrencySemanticsDisabled,
		)
		if err == nil {
			t.Fatalf("complex symbols %v were accepted", symbols)
		}
		var buildError *runtimecomplex.BuildError
		if !errors.As(err, &buildError) {
			t.Fatalf(
				"complex symbols %v error = %#v, want BuildError",
				symbols,
				err,
			)
		}
	}
}

func TestComplexRuntimePrintsNominalConstantSizeOperations(t *testing.T) {
	factory := tsgo.NewFactory()
	symbols := []api.RuntimeSymbol{
		api.RuntimeComplex64,
		api.RuntimeComplex128,
		api.RuntimeComplexDivide,
		api.RuntimeComplex64Add,
		api.RuntimeComplex64Sub,
		api.RuntimeComplex64Mul,
		api.RuntimeComplex64Div,
		api.RuntimeComplex64Neg,
		api.RuntimeComplex64Equal,
		api.RuntimeComplex128Add,
		api.RuntimeComplex128Sub,
		api.RuntimeComplex128Mul,
		api.RuntimeComplex128Div,
		api.RuntimeComplex128Neg,
		api.RuntimeComplex128Equal,
	}
	definitions, err := runtimeemission.Build(
		factory,
		api.RuntimeModuleComplex,
		symbols,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	client, err := tsgo.StartClient(
		filepath.Join("..", "..", "..", ".."),
		t.TempDir(),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var printed strings.Builder
	for _, definition := range definitions {
		text, err := client.PrintNode(
			definition.Statement(),
			tsgo.PrintOptions{},
		)
		if err != nil {
			t.Fatal(err)
		}
		printed.WriteString(text)
	}
	artifact := printed.String()
	for _, required := range []struct {
		text  string
		count int
	}{
		{"export class GoComplex64", 1},
		{"declare private readonly goComplex64Brand: void", 1},
		{"export class GoComplex128", 1},
		{"declare private readonly goComplex128Brand: void", 1},
		{"private constructor(public readonly real: number", 2},
		{"public static make(real: number, imag: number)", 2},
		{"export function goComplexDivide(", 1},
		{"export function goComplex64Multiply(", 1},
		{"export function goComplex128Divide(", 1},
	} {
		if strings.Count(artifact, required.text) != required.count {
			t.Fatalf(
				"complex runtime count(%q) = %d, want %d:\n%s",
				required.text,
				strings.Count(artifact, required.text),
				required.count,
				artifact,
			)
		}
	}
	for _, forbidden := range []string{
		"any",
		"unknown",
		".call(",
		".apply(",
		".bind(",
		"switch (",
	} {
		if strings.Contains(artifact, forbidden) {
			t.Fatalf("complex runtime contains %q:\n%s", forbidden, artifact)
		}
	}
	if len(artifact) > 6_000 {
		t.Fatalf("complex runtime = %d bytes, want <= 6000", len(artifact))
	}
}
