package tsgo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativePrintNodeRoundTrip(t *testing.T) {
	t.Setenv("GOOS", "js")
	t.Setenv("GOARCH", "wasm")
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOFLAGS", "-tags=ambientmustnotselect")
	workingDirectory := t.TempDir()
	client, err := StartClientWithTool(selectedTool(t), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})

	processID := client.command.Process.Pid
	encoded, err := EncodeSourceFile(representativeSourceFile())
	if err != nil {
		t.Fatal(err)
	}
	var printed string
	for range 2 {
		actual, err := client.PrintNode(representativeSourceFile(), PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		const expected = "let answer = 42;\nanswer = 43;\n"
		if actual != expected {
			t.Fatalf("printed TypeScript = %q, want %q", actual, expected)
		}
		if client.command.Process.Pid != processID {
			t.Fatal("PrintNode replaced the persistent TS-Go process")
		}
		printed = actual
	}
	printedEncoded, err := client.PrintEncodedSourceFile(encoded, PrintOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if printedEncoded != printed {
		t.Fatalf("encoded source printed %q, want %q", printedEncoded, printed)
	}

	outputPath := filepath.Join(workingDirectory, "answer.ts")
	if err := os.WriteFile(outputPath, []byte(printed), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := CompileWithTool(
		ctx,
		selectedTool(t),
		workingDirectory,
		[]string{"--noEmit", "--strict", outputPath},
	); err != nil {
		t.Fatalf("strict TS-Go typecheck: %v", err)
	}
}

func TestPrintEncodedSourceFileRejectsWrongProtocol(t *testing.T) {
	client := &Client{}
	for _, payload := range [][]byte{
		nil,
		make([]byte, headerSize),
	} {
		_, err := client.PrintEncodedSourceFile(payload, PrintOptions{})
		var clientError *ClientError
		if !errors.As(err, &clientError) {
			t.Fatalf("error = %v, want ClientError", err)
		}
	}
}

func TestPinnedToolIdentityRejectsWrongExecutable(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	err = verifyPinnedTool(executable)
	var toolError *ToolError
	if !errors.As(err, &toolError) {
		t.Fatalf("error = %v, want ToolError", err)
	}
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..")
}
