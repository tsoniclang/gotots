package structvalue_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAnonymousStructUsesDoNotDuplicateShapeOperations(t *testing.T) {
	first := compileAnonymousStructUses(t, 4)
	second := compileAnonymousStructUses(t, 8)
	firstSupport := anonymousStructSupport(t, first)
	secondSupport := anonymousStructSupport(t, second)
	firstEncoded, err := tsgo.EncodeSourceFile(firstSupport)
	if err != nil {
		t.Fatal(err)
	}
	secondEncoded, err := tsgo.EncodeSourceFile(secondSupport)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstEncoded, secondEncoded) {
		t.Fatal("doubling anonymous-struct uses duplicated shape-owned source")
	}
	for _, emission := range []emit.ProgramEmission{first, second} {
		source := structTargetSource(t, emission)
		for _, statement := range source.Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if !ok || !strings.HasPrefix(function.Name().Text(), "Use") {
				continue
			}
			body := function.Body().(tsgo.Block).Statements()
			if len(body) != 2 {
				t.Fatalf(
					"%s statements = %d, want declaration and return",
					function.Name().Text(),
					len(body),
				)
			}
			copyCall := body[0].(tsgo.VariableStatement).
				DeclarationList().Declarations()[0].
				Initializer().(tsgo.CallExpression)
			_, copyMember := targetProperty(copyCall.Expression())
			equalCall := body[1].(tsgo.ReturnStatement).
				Expression().(tsgo.CallExpression)
			_, equalMember := targetProperty(equalCall.Expression())
			if copyMember != "$copy" || equalMember != "$equal" {
				t.Fatalf(
					"%s operations = %s/%s",
					function.Name().Text(),
					copyMember,
					equalMember,
				)
			}
		}
	}
}

func compileAnonymousStructUses(
	t *testing.T,
	uses int,
) emit.ProgramEmission {
	t.Helper()
	var source strings.Builder
	source.WriteString("package boundary\n\n")
	for index := range uses {
		fmt.Fprintf(&source, `func Use%d(
	left, right struct {
		Value int32
		Ready bool
	},
) bool {
	copy := left
	return copy == right
}

`, index)
	}
	emission, err := compileTemporaryStructProgram(t, source.String())
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func anonymousStructSupport(
	t *testing.T,
	emission emit.ProgramEmission,
) tsgo.SourceFile {
	t.Helper()
	for _, file := range emission.Files() {
		if file.OutputPath() == output.AnonymousStructSupportPath &&
			file.Kind() == emit.TargetFileSupport {
			return file.SourceFile()
		}
	}
	t.Fatal("anonymous-struct support artifact is absent")
	return nil
}
