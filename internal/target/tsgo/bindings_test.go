package tsgo_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestGeneratedFactoriesBuildConstDeclaration(t *testing.T) {
	factory := tsgo.NewFactory()
	filePath, err := tsgo.NewPath("/answer.ts")
	if err != nil {
		t.Fatal(err)
	}
	name := factory.Identifier("answer")
	value := factory.NumericLiteral("42", tsgo.TokenFlagsNone)
	declaration := factory.VariableDeclaration(name, nil, nil, value)
	declarations := factory.VariableDeclarationList(
		[]tsgo.VariableDeclaration{declaration},
		tsgo.NodeFlagsConst,
	)
	statement := factory.VariableStatement(nil, declarations)
	sourceFile := factory.SourceFile(
		[]tsgo.Statement{statement},
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        filePath,
			Path:            filePath,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)

	if sourceFile.Kind() != tsgo.SyntaxKindSourceFile {
		t.Fatalf("kind = %v", sourceFile.Kind())
	}
	if len(sourceFile.Statements()) != 1 {
		t.Fatalf("statements = %d", len(sourceFile.Statements()))
	}
}
