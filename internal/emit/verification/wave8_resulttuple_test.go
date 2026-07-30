package emit_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveEightControlledResultCaptureKeepsExactTupleType(t *testing.T) {
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveEightControlDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := roots[:0]
	for _, root := range roots {
		switch root.Object().Name() {
		case "DeferBuiltins", "RuntimeFaultIdentity":
			selected = append(selected, root)
		}
	}
	emission, err := emit.Compile(program, selected)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"DeferBuiltins", "RuntimeFaultIdentity"} {
		function := waveEightResultFunction(t, emission, name)
		resultType, ok := function.Type().(tsgo.TupleTypeNode)
		if !ok {
			t.Fatalf("%s result = %T, want tuple", name, function.Type())
		}
		body, ok := function.Body().(tsgo.Block)
		if !ok {
			t.Fatalf("%s body = %T, want block", name, function.Body())
		}
		captures := waveEightResultCaptures(body.Statements())
		if len(captures) != 1 {
			t.Fatalf("%s controlled result captures = %d, want 1", name, len(captures))
		}
		captureType, ok := captures[0].Type().(tsgo.TupleTypeNode)
		if !ok {
			t.Fatalf(
				"%s controlled result capture type = %T, want tuple",
				name,
				captures[0].Type(),
			)
		}
		if _, ok := captures[0].Initializer().(tsgo.ArrayLiteralExpression); !ok {
			t.Fatalf(
				"%s controlled result initializer = %T, want array literal",
				name,
				captures[0].Initializer(),
			)
		}
		got := encodeWaveEightTupleType(t, captureType)
		want := encodeWaveEightTupleType(t, resultType)
		if !bytes.Equal(got, want) {
			t.Fatalf("%s controlled result capture does not preserve its result tuple", name)
		}
	}
}

func waveEightResultFunction(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range emission.Files() {
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || function.Name() == nil {
				continue
			}
			identifier, ok := function.Name().(tsgo.Identifier)
			if ok && identifier.Text() == name {
				return function
			}
		}
	}
	t.Fatalf("generated function %s not found", name)
	return nil
}

func waveEightResultCaptures(
	statements []tsgo.Statement,
) []tsgo.VariableDeclaration {
	var result []tsgo.VariableDeclaration
	for _, statement := range statements {
		switch statement := statement.(type) {
		case tsgo.VariableStatement:
			for _, declaration := range statement.DeclarationList().Declarations() {
				name, ok := declaration.Name().(tsgo.Identifier)
				if ok &&
					strings.HasPrefix(name.Text(), "__gotots_results_") &&
					declaration.Initializer() != nil {
					if _, ok := declaration.Initializer().(tsgo.ArrayLiteralExpression); ok {
						result = append(result, declaration)
					}
				}
			}
		case tsgo.Block:
			result = append(result, waveEightResultCaptures(statement.Statements())...)
		case tsgo.LabeledStatement:
			result = append(
				result,
				waveEightResultCaptures([]tsgo.Statement{statement.Statement()})...,
			)
		case tsgo.TryStatement:
			result = append(
				result,
				waveEightResultCaptures(statement.TryBlock().Statements())...,
			)
			if statement.CatchClause() != nil {
				result = append(
					result,
					waveEightResultCaptures(
						statement.CatchClause().Block().Statements(),
					)...,
				)
			}
			if statement.FinallyBlock() != nil {
				result = append(
					result,
					waveEightResultCaptures(statement.FinallyBlock().Statements())...,
				)
			}
		}
	}
	return result
}

func encodeWaveEightTupleType(
	t *testing.T,
	target tsgo.TupleTypeNode,
) []byte {
	t.Helper()
	encoded, err := tsgo.EncodeNode(target)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}
