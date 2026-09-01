package arrayvalue_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
)

func compileArrayFixture(t *testing.T) emit.ProgramEmission {
	t.Helper()
	return compileArrayFixtureWithOptions(t, arrayNumberOptions())
}

func arrayNumberOptions() emit.Options {
	return emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	}
}

func compileArrayFixtureWithOptions(
	t *testing.T,
	options emit.Options,
) emit.ProgramEmission {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: arrayValuesDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

type materializedProgram struct {
	paths        []string
	sourceModule string
	programInit  string
	printed      map[string]string
}

func materializeArrayProgram(
	t *testing.T,
	directory string,
	emission emit.ProgramEmission,
) materializedProgram {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	result := materializedProgram{printed: make(map[string]string)}
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			for index, statement := range file.SourceFile().Statements() {
				if _, statementErr := tsgo.EncodeNode(statement); statementErr != nil {
					name := ""
					if function, ok := statement.(tsgo.FunctionDeclaration); ok &&
						function.Name() != nil {
						name = function.Name().Text()
					}
					t.Fatalf(
						"print %s statement %d %s (%T): %v",
						file.OutputPath(),
						index,
						name,
						statement,
						statementErr,
					)
				}
			}
			t.Fatalf("print %s: %v", file.OutputPath(), err)
		}
		targetPath := filepath.Join(
			directory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeFile(t, targetPath, printed)
		result.paths = append(result.paths, targetPath)
		result.printed[file.OutputPath()] = printed
		module := "./" + strings.TrimSuffix(file.OutputPath(), ".ts") + ".js"
		switch file.Kind() {
		case emit.TargetFileSource:
			if result.sourceModule != "" {
				t.Fatal("array fixture emitted multiple source files")
			}
			result.sourceModule = module
		case emit.TargetFileProgramInitialization:
			result.programInit = module
		}
	}
	if result.sourceModule == "" || result.programInit == "" {
		t.Fatalf(
			"array emission modules = source %q, init %q",
			result.sourceModule,
			result.programInit,
		)
	}
	return result
}

func compileTypeScript(
	t *testing.T,
	directory string,
	paths []string,
) error {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(directory); err != nil {
		return err
	}
	if err := runtimefixture.InstallResolution(directory, filepath.Join(directory, "out")); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--outDir", filepath.Join(directory, "out"),
	}
	arguments = append(arguments, paths...)
	return tsgo.Compile(ctx, repositoryRoot(), directory, arguments)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(
	t *testing.T,
	directory string,
	name string,
	arguments ...string,
) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, name, arguments...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOMEMLIMIT=1GiB")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(arguments, " "), err, output)
	}
	return string(output)
}
