package tsgoprinter

import (
	"bufio"
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRunPrintsEveryFramedSourceFile(t *testing.T) {
	payload, err := tsgo.EncodeSourceFile(representativeSourceFile(t))
	if err != nil {
		t.Fatal(err)
	}
	var input bytes.Buffer
	input.WriteString(inputMagic)
	if err := writeUint32(&input, 2); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := writeFrame(&input, payload); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	moduleDirectory, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(Config{
		ModuleDirectory:  moduleDirectory,
		WorkingDirectory: t.TempDir(),
	}, &input, &output); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(&output)
	magic := make([]byte, len(outputMagic))
	if _, err := io.ReadFull(reader, magic); err != nil {
		t.Fatal(err)
	}
	if string(magic) != outputMagic {
		t.Fatalf("output magic = %q, want %q", magic, outputMagic)
	}
	count, err := readUint32(reader)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("output count = %d, want 2", count)
	}
	for index := range count {
		printed, err := readFrame(reader)
		if err != nil {
			t.Fatal(err)
		}
		if string(printed) != "export const value: number = 1 + 2;\n" {
			t.Fatalf("frame %d = %q", index, printed)
		}
	}
	if extra, err := reader.ReadByte(); err != io.EOF {
		t.Fatalf("output trailing read = byte %d, error %v", extra, err)
	}
}

func TestRunRejectsInvalidFramingBeforeStartingPrinter(t *testing.T) {
	var oversizedCount bytes.Buffer
	oversizedCount.WriteString(inputMagic)
	if err := writeUint32(&oversizedCount, maximumFileCount+1); err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string][]byte{
		"magic": []byte("INVALID!\x00\x00\x00\x00"),
		"count": oversizedCount.Bytes(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := Run(Config{}, bytes.NewReader(input), io.Discard); err == nil {
				t.Fatal("Run accepted invalid framing")
			}
		})
	}
}

func representativeSourceFile(t *testing.T) tsgo.SourceFile {
	t.Helper()
	factory := tsgo.NewFactory()
	filePath, err := tsgo.NewPath("/source.ts")
	if err != nil {
		t.Fatal(err)
	}
	declaration := factory.VariableDeclaration(
		factory.Identifier("value"),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.BinaryExpression(
			nil,
			factory.NumericLiteral("1", tsgo.TokenFlagsNone),
			nil,
			factory.PlusToken(),
			factory.NumericLiteral("2", tsgo.TokenFlagsNone),
		),
	)
	return factory.SourceFile(
		[]tsgo.Statement{
			factory.VariableStatement(
				[]tsgo.ModifierLike{factory.ExportKeyword()},
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{declaration},
					tsgo.NodeFlagsConst,
				),
			),
		},
		factory.EndOfFile(),
		tsgo.SourceFileData{
			FileName:        filePath,
			Path:            filePath,
			LanguageVariant: tsgo.LanguageVariantStandard,
			ScriptKind:      tsgo.ScriptKindTS,
		},
	)
}
