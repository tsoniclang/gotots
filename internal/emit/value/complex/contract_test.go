package complex_test

import (
	"bytes"
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	runtimefixture "github.com/tsoniclang/gotots/internal/testfixture/gototsruntime"
)

func TestComplexWidthsAreNominalUnderStrictTypeScript(t *testing.T) {
	emission := compileComplex(t)
	workingDirectory := t.TempDir()
	targetPaths, _, _ := printComplex(t, workingDirectory, emission)
	source := `import { GoComplex64, GoComplex128 } from "./runtime/complex.js";
const narrow: GoComplex64 = GoComplex64.make(0, 0);
const invalid: GoComplex128 = narrow;
console.log(invalid);
`
	path := filepath.Join(workingDirectory, "nominal.ts")
	writeFile(t, path, source)
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noEmit",
	}
	arguments = append(arguments, targetPaths...)
	arguments = append(arguments, path)
	if err := runtimefixture.InstallResolution(workingDirectory, filepath.Join(workingDirectory, "out")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err == nil {
		t.Fatal("complex64 was assignable to complex128")
	}
}

func TestComplexSourceSpellingCannotOverrideCheckerValue(t *testing.T) {
	loaded := loadComplex(t)
	baseline := encodedComplex(t, compileLoadedComplex(t, loaded))
	mutated := false
	ast.Inspect(loaded.Files()[0].Syntax(), func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok && literal.Kind == token.IMAG && literal.Value == "7i" {
			literal.Value = "999i"
			mutated = true
		}
		return true
	})
	if !mutated {
		t.Fatal("imaginary literal mutation target was absent")
	}
	target := encodedComplex(t, compileLoadedComplex(t, loaded))
	if !bytes.Equal(target, baseline) {
		t.Fatal("complex source spelling overrode checker-owned value")
	}
}

func TestComplexBuiltinSpellingCannotSelectBehavior(t *testing.T) {
	loaded := loadComplex(t)
	baseline := encodedComplex(t, compileLoadedComplex(t, loaded))
	mutated := false
	ast.Inspect(loaded.Files()[0].Syntax(), func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok ||
			loaded.TypesInfo().Uses[identifier] !=
				types.Universe.Lookup("complex") {
			return true
		}
		identifier.Name = "forgedBuiltinSpelling"
		mutated = true
		return false
	})
	if !mutated {
		t.Fatal("complex builtin mutation target was absent")
	}
	target := encodedComplex(t, compileLoadedComplex(t, loaded))
	if !bytes.Equal(target, baseline) {
		t.Fatal("complex builtin behavior was selected by source spelling")
	}
}

func TestComplexGeneratedSurfaceIsBoundedAndDemandOwned(t *testing.T) {
	emission := compileComplex(t)
	workingDirectory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		switch {
		case file.OutputPath() == "runtime/complex.ts":
			executableFile := executableComplexFile(file.SourceFile())
			if len(executableFile.Statements()) != 16 {
				t.Fatalf(
					"complex runtime statements = %d, want 1 import + 15 definitions",
					len(executableFile.Statements()),
				)
			}
			executable, printErr := client.PrintNode(executableFile, tsgo.PrintOptions{})
			if printErr != nil {
				t.Fatal(printErr)
			}
			if len(executable) > 6_000 {
				t.Fatalf("complex runtime = %d executable bytes (%d canonical), want <= 6000", len(executable), len(text))
			}
		case strings.HasSuffix(file.OutputPath(), "/source.ts"):
			executable, printErr := client.PrintNode(
				executableComplexFile(file.SourceFile()),
				tsgo.PrintOptions{},
			)
			if printErr != nil {
				t.Fatal(printErr)
			}
			if len(executable) > 6_000 {
				t.Fatalf("complex source = %d executable bytes (%d canonical), want <= 6000", len(executable), len(text))
			}
			for _, operation := range []string{
				"goComplex64Multiply(left, right)",
				"goComplex128Divide(left, right)",
			} {
				if strings.Count(executable, operation) != 1 {
					t.Fatalf(
						"source count(%q) = %d:\n%s",
						operation,
						strings.Count(executable, operation),
						executable,
					)
				}
			}
		}
	}
}

func executableComplexFile(source tsgo.SourceFile) tsgo.SourceFile {
	statements := make([]tsgo.Statement, 0, len(source.Statements()))
	for _, statement := range source.Statements() {
		if declaration, ok := statement.(tsgo.ImportDeclaration); ok {
			module, moduleOK := declaration.ModuleSpecifier().(tsgo.StringLiteral)
			if moduleOK && (strings.HasSuffix(module.Text(), "/source-fact.js") ||
				module.Text() == "@tsonic/core/lang.js") {
				continue
			}
		}
		if _, fact := statement.(tsgo.ExpressionStatement); fact {
			continue
		}
		statements = append(statements, statement)
	}
	factory := tsgo.NewFactory()
	return factory.SourceFile(statements, source.EndOfFileToken(), source.SourceData())
}

func loadComplex(t *testing.T) *load.Package {
	t.Helper()
	loaded, err := load.One(context.Background(), load.Request{
		Directory: complexFixtureDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func compileLoadedComplex(
	t *testing.T,
	loaded *load.Package,
) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func encodedComplex(
	t *testing.T,
	emission emit.ProgramEmission,
) []byte {
	t.Helper()
	var encoded bytes.Buffer
	for _, file := range emission.Files() {
		encoded.WriteString(file.OutputPath())
		encoded.WriteByte(0)
		value, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		encoded.Write(value)
		encoded.WriteByte(0)
	}
	return encoded.Bytes()
}
