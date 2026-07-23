package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestInspectConstructsPrintsOnlyBoundedDenominators(t *testing.T) {
	project := writeCLIProject(t)
	var output bytes.Buffer
	if err := run([]string{
		"inspect", "constructs",
		"-contract", "portable@v1",
		"-dir", project,
	}, &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) < 3 || len(lines) > 22 {
		t.Fatalf("inspect output has %d lines:\n%s", len(lines), output.String())
	}
	if !strings.HasPrefix(lines[0], "toolchain ") ||
		!strings.HasPrefix(lines[1], "denominators: ") ||
		!strings.Contains(lines[1], "residentDefinitions=") ||
		!strings.Contains(lines[1], "fullSemanticDefinitions=") ||
		!strings.Contains(lines[1], "declarationContractDefinitions=") ||
		!strings.Contains(lines[1], "externalBoundaryDefinitions=") ||
		!strings.Contains(lines[1], "intrinsicDefinitions=") ||
		!strings.Contains(lines[1], "providerDefinitions=0") ||
		!strings.Contains(lines[1], "largestProviderShardBytes=0") ||
		!strings.Contains(lines[1], "providerShardLoads=0") ||
		!strings.Contains(lines[1], "maxProviderPackagesResident=0") ||
		!strings.Contains(lines[1], "unknownConstructs=0") ||
		!strings.Contains(lines[1], "unknownDirectives=0") {
		t.Fatalf("inspect output lacks bounded proof denominators:\n%s", output.String())
	}
	for _, line := range lines[2:] {
		if !strings.HasPrefix(line, "header-tail ") {
			t.Fatalf("unexpected unbounded detail line %q", line)
		}
	}
	if output.Len() > 16*1024 ||
		strings.Contains(output.String(), "func Main") ||
		strings.Contains(output.String(), project) {
		t.Fatalf("inspect output leaks unbounded source evidence:\n%s", output.String())
	}
}

func TestInspectConstructsReportsExactMultiModuleClosures(t *testing.T) {
	root, err := filepath.Abs(
		filepath.Join("..", "..", "testdata", "workspaces"),
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		directory string
		patterns  []string
		packages  int
		files     int
	}{
		{
			name:      "independent roots",
			directory: filepath.Join(root, "dual"),
			patterns:  []string{"./a/...", "./b/..."},
			packages:  2,
			files:     2,
		},
		{
			name:      "linked modules",
			directory: filepath.Join(root, "linked"),
			patterns:  []string{"./app/...", "./lib/..."},
			packages:  2,
			files:     2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			arguments := []string{
				"inspect",
				"constructs",
				"-contract",
				"portable@v1",
				"-dir",
				test.directory,
			}
			arguments = append(arguments, test.patterns...)
			var output bytes.Buffer
			if err := run(arguments, &output); err != nil {
				t.Fatal(err)
			}
			denominators := strings.Split(output.String(), "\n")[1]
			if got := reportInteger(
				t, denominators, "closurePackages",
			); got != test.packages {
				t.Errorf(
					"closure packages=%d, want %d",
					got,
					test.packages,
				)
			}
			if got := reportInteger(
				t, denominators, "structuralFiles",
			); got != test.files {
				t.Errorf(
					"structural files=%d, want %d",
					got,
					test.files,
				)
			}
			definitions := reportInteger(
				t, denominators, "definitions",
			)
			partition := 0
			for _, field := range []string{
				"fullSemanticDefinitions",
				"declarationContractDefinitions",
				"externalBoundaryDefinitions",
				"intrinsicDefinitions",
			} {
				partition += reportInteger(
					t,
					denominators,
					field,
				)
			}
			if partition != definitions {
				t.Errorf(
					"evidence-depth partition=%d, definitions=%d",
					partition,
					definitions,
				)
			}
			if reportInteger(
				t, denominators, "unknownConstructs",
			) != 0 ||
				reportInteger(
					t, denominators, "unknownDirectives",
				) != 0 {
				t.Fatal("multi-module inventory contains unknown input")
			}
		})
	}
}

func TestRunPropagatesWriterFailure(t *testing.T) {
	project := writeCLIProject(t)
	want := errors.New("writer failed")
	err := run([]string{
		"inspect", "constructs",
		"-contract", "portable@v1",
		"-dir", project,
	}, failingWriter{err: want})
	if !errors.Is(err, want) {
		t.Fatalf("writer failure = %v, want %v", err, want)
	}
}

func TestProviderArtifactSelectionFailsClosed(t *testing.T) {
	project := writeCLIProject(t)
	for name, arguments := range map[string][]string{
		"path-without-digest": {
			"inspect", "constructs", "-contract", "portable@v1",
			"-dir", project, "-provider", "provider.gotots",
		},
		"digest-without-path": {
			"inspect", "constructs", "-contract", "portable@v1",
			"-dir", project, "-provider-digest", strings.Repeat("0", 64),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(arguments, &bytes.Buffer{}); err == nil {
				t.Fatal("incomplete provider selection was accepted")
			}
		})
	}
}

func TestUnsupportedCommandIsTyped(t *testing.T) {
	for _, arguments := range [][]string{
		nil,
		{"translate"},
		{"inspect"},
		{"audit"},
		{"audit", "catalog"},
		{"audit", "verify"},
	} {
		err := run(arguments, &bytes.Buffer{})
		var unsupported *UnsupportedCommandError
		if !errors.As(err, &unsupported) {
			t.Errorf("run(%q) error = %T, want *UnsupportedCommandError", arguments, err)
		}
	}
}

func reportInteger(
	t *testing.T,
	report string,
	field string,
) int {
	t.Helper()
	match := regexp.MustCompile(
		`(?:^| )` + regexp.QuoteMeta(field) + `=([0-9]+)(?: |$)`,
	).FindStringSubmatch(report)
	if len(match) != 2 {
		t.Fatalf("report lacks %s: %s", field, report)
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatal(err)
	}
	return value
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func writeCLIProject(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/cli\n\ngo 1.26.0\n",
		"main.go": `package cli

func Main() int {
	return 1
}
`,
	} {
		if err := os.WriteFile(
			filepath.Join(directory, name), []byte(content), 0o644,
		); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}
