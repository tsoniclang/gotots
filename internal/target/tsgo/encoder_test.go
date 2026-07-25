package tsgo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEncoderMatchesPinnedUpstreamEncoder(t *testing.T) {
	sourceFile := representativeSourceFile()
	actual, err := EncodeSourceFile(sourceFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := encodeWithPinnedUpstream(t)
	if !bytes.Equal(actual, expected) {
		offset := firstDifference(actual, expected)
		t.Fatalf(
			"encoded bytes differ at %d (Go=%d bytes, upstream=%d bytes)\nGo: % x\nupstream: % x",
			offset,
			len(actual),
			len(expected),
			window(actual, offset),
			window(expected, offset),
		)
	}
}

func TestEncoderRejectsAbsentRequiredChild(t *testing.T) {
	factory := NewFactory()
	declaration := factory.VariableDeclaration(nil, nil, nil, factory.NumericLiteral("1", TokenFlagsNone))
	_, err := EncodeNode(declaration)
	var encodeError *EncodeError
	if !errors.As(err, &encodeError) {
		t.Fatalf("error = %v, want EncodeError", err)
	}
	if encodeError.Field != "name" && encodeError.Field != "Name" {
		t.Fatalf("field = %q, want Name", encodeError.Field)
	}
}

func representativeSourceFile() SourceFile {
	factory := NewFactory()
	name := factory.Identifier("answer")
	declaration := factory.VariableDeclaration(
		name,
		nil,
		nil,
		factory.NumericLiteral("42", TokenFlagsNone),
	)
	declarations := factory.VariableDeclarationList(
		[]VariableDeclaration{declaration},
		NodeFlagsConst,
	)
	assignment := factory.BinaryExpression(
		nil,
		factory.Identifier("answer"),
		nil,
		factory.EqualsToken(),
		factory.NumericLiteral("43", TokenFlagsNone),
	)
	return factory.SourceFile(
		[]Statement{
			factory.VariableStatement(nil, declarations),
			factory.ExpressionStatement(assignment),
		},
		factory.EndOfFile(),
		SourceFileData{
			FileName:        "answer.ts",
			Path:            "answer.ts",
			LanguageVariant: LanguageVariantStandard,
			ScriptKind:      ScriptKindTS,
		},
	)
}

func encodeWithPinnedUpstream(t *testing.T) []byte {
	t.Helper()
	return encodeWithPinnedScript(t, `
const declaration = factory.createVariableDeclaration(
    factory.createIdentifier("answer"),
    undefined,
    undefined,
    factory.createNumericLiteral("42", 0),
);
const declarations = factory.createVariableDeclarationList([declaration], 2);
const assignment = factory.createBinaryExpression(
    undefined,
    factory.createIdentifier("answer"),
    undefined,
    factory.createToken(SyntaxKind.EqualsToken),
    factory.createNumericLiteral("43", 0),
);
const sourceFile = factory.createSourceFile(
    [
        factory.createVariableStatement(undefined, declarations),
        factory.createExpressionStatement(assignment),
    ],
    factory.createToken(SyntaxKind.EndOfFile),
    "",
    "answer.ts",
    "answer.ts",
);
Object.defineProperty(sourceFile, "languageVariant", { value: 0 });
Object.defineProperty(sourceFile, "scriptKind", { value: 3 });
process.stdout.write(Buffer.from(encodeSourceFile(sourceFile)).toString("base64"));
`)
}

func encodeWithPinnedScript(t *testing.T, body string) []byte {
	t.Helper()
	contract, err := VerifyPinnedContract(schemaDirectory())
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(
		"go",
		"list",
		"-m",
		"-f",
		"{{.Dir}}",
		contract.Module()+"@"+contract.ToolVersion(),
	)
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	moduleDirectory := strings.TrimSpace(string(output))
	moduleURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(moduleDirectory)}).String()
	factoryURL := strconv.Quote(moduleURL + "/_packages/native-preview/src/ast/factory.generated.ts")
	astURL := strconv.Quote(moduleURL + "/_packages/native-preview/src/ast/index.ts")
	encoderURL := strconv.Quote(moduleURL + "/_packages/native-preview/src/api/node/encoder.ts")
	script := fmt.Sprintf(`
import * as factory from %s;
import { SyntaxKind } from %s;
import { encodeNode, encodeSourceFile } from %s;
%s
`, factoryURL, astURL, encoderURL, body)
	node := exec.Command(
		"node",
		"--experimental-strip-types",
		"--no-warnings",
		"--conditions",
		"@typescript/source",
		"--input-type=module",
		"-e",
		script,
	)
	encoded, err := node.Output()
	if err != nil {
		t.Fatal(err)
	}
	result, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func firstDifference(left []byte, right []byte) int {
	length := min(len(left), len(right))
	for index := range length {
		if left[index] != right[index] {
			return index
		}
	}
	return length
}

func window(data []byte, offset int) []byte {
	start := max(0, offset-16)
	end := min(len(data), offset+32)
	return data[start:end]
}
