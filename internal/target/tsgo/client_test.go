package tsgo

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativePrintNodeRoundTrip(t *testing.T) {
	workingDirectory := t.TempDir()
	client, err := StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close client: %v", err)
		}
	})

	processID := client.command.Process.Pid
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

	outputPath := filepath.Join(workingDirectory, "answer.ts")
	if err := os.WriteFile(outputPath, []byte(printed), 0o644); err != nil {
		t.Fatal(err)
	}
	typecheck := exec.Command(client.command.Path, "--noEmit", "--strict", outputPath)
	typecheck.Dir = workingDirectory
	if output, err := typecheck.CombinedOutput(); err != nil {
		t.Fatalf("strict TS-Go typecheck: %v\n%s", err, output)
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
