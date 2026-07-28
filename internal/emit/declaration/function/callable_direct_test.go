package function_test

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestKnownDirectCallableArtifactIsExact(t *testing.T) {
	loaded := loadCallableValuesProject(t)
	workingDirectory := t.TempDir()
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	printed := printTargetFile(t, targetFile, workingDirectory)
	const expected = `export function UseNamed(value: int32): int32 {
    return Apply(Double, value);
}`
	if actual := strings.TrimSpace(
		printedFunction(t, printed, "UseNamed"),
	); actual != expected {
		t.Fatalf("known direct call changed\nactual:\n%s\nwant:\n%s", actual, expected)
	}
}

func TestDirectNilCallIsSmallerThanIIFEForm(t *testing.T) {
	loaded := loadCallableValuesProject(t)
	targetFile := compileSourceFile(t, loaded, loaded.Files()[0].Syntax())
	direct := targetFunction(t, targetFile, "Apply")
	statements := direct.Body().(tsgo.Block).Statements()
	if len(statements) != 4 {
		t.Fatalf("direct Apply statements = %d, want four", len(statements))
	}
	capture := statements[0].(tsgo.VariableStatement).
		DeclarationList().
		Declarations()[0]
	calleeName := capture.Name().(tsgo.Identifier)
	factory := tsgo.NewFactory()
	foilBody := factory.Block(statements[1:], true)
	foilFunction := factory.FunctionExpression(
		nil,
		nil,
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			factory.ParameterDeclaration(
				nil,
				nil,
				calleeName,
				nil,
				direct.Parameters()[0].Type(),
				nil,
			),
		},
		direct.Type(),
		foilBody,
	)
	foilCall := factory.CallExpression(
		foilFunction,
		nil,
		nil,
		[]tsgo.Expression{capture.Initializer()},
		tsgo.NodeFlagsNone,
	)
	foil := factory.FunctionDeclaration(
		direct.Modifiers(),
		direct.AsteriskToken(),
		direct.Name(),
		direct.TypeParameters(),
		direct.Parameters(),
		direct.Type(),
		factory.Block(
			[]tsgo.Statement{factory.ReturnStatement(foilCall)},
			true,
		),
	)

	directEncoded, err := tsgo.EncodeNode(direct)
	if err != nil {
		t.Fatal(err)
	}
	foilEncoded, err := tsgo.EncodeNode(foil)
	if err != nil {
		t.Fatal(err)
	}
	directNodes := callableEncodedNodeCount(t, directEncoded)
	foilNodes := callableEncodedNodeCount(t, foilEncoded)
	if directNodes >= foilNodes || len(directEncoded) >= len(foilEncoded) {
		t.Fatalf(
			"direct nil call nodes/bytes = %d/%d, IIFE foil = %d/%d",
			directNodes,
			len(directEncoded),
			foilNodes,
			len(foilEncoded),
		)
	}
	t.Logf(
		"nil-call direct nodes/encoded-bytes=%d/%d; IIFE foil=%d/%d; delta=%d/%d",
		directNodes,
		len(directEncoded),
		foilNodes,
		len(foilEncoded),
		directNodes-foilNodes,
		len(directEncoded)-len(foilEncoded),
	)
}

func callableEncodedNodeCount(t *testing.T, encoded []byte) int {
	t.Helper()
	const (
		headerSize       = 44
		nodesOffsetField = 40
		nodeWidth        = 28
	)
	if len(encoded) < headerSize {
		t.Fatalf("encoded AST is %d bytes, shorter than header", len(encoded))
	}
	nodesOffset := int(binary.LittleEndian.Uint32(
		encoded[nodesOffsetField:headerSize],
	))
	if nodesOffset < headerSize ||
		nodesOffset > len(encoded) ||
		(len(encoded)-nodesOffset)%nodeWidth != 0 {
		t.Fatalf("encoded AST node offset = %d", nodesOffset)
	}
	return (len(encoded) - nodesOffset) / nodeWidth
}
