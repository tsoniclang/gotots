package emit_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func sourceModuleForExport(
	t *testing.T,
	artifacts waveFourArtifacts,
	workingDirectory string,
	name string,
) string {
	t.Helper()
	var selected string
	for _, path := range artifacts.paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		printed := string(content)
		if !strings.Contains(
			printed,
			"export function "+name+"(",
		) {
			continue
		}
		if selected != "" {
			t.Fatalf("multiple source modules export %s", name)
		}
		selected = path
	}
	if selected == "" {
		t.Fatalf("no source module exports %s", name)
	}
	relative, err := filepath.Rel(workingDirectory, selected)
	if err != nil {
		t.Fatal(err)
	}
	return "./" + strings.TrimSuffix(filepath.ToSlash(relative), ".ts") + ".js"
}

func environmentDeclarationLine(
	t *testing.T,
	printed string,
	prefix string,
) string {
	t.Helper()
	start := strings.Index(printed, prefix)
	if start < 0 {
		t.Fatalf("environment declaration lacks %q:\n%s", prefix, printed)
	}
	end := strings.IndexByte(printed[start:], '\n')
	if end < 0 {
		return printed[start:]
	}
	return printed[start : start+end]
}

func waveNineOptions() emit.Options {
	options := emit.DefaultOptions()
	return options
}

func waveNineConcurrencyDirectory() string {
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"constructs",
		"concurrency",
		"wave9",
	)
}

func packageStateExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-source.ts",
		)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return supportExpectedPath(projectDirectory, file)
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func packageInitializationExpectedPath(
	t *testing.T,
	projectDirectory string,
	file emit.TargetFile,
) string {
	t.Helper()
	switch file.Kind() {
	case emit.TargetFileSource:
		name := "expected-source.ts"
		if file.PackageName() == "sideeffect" {
			name = "expected-" + filepath.Base(file.OutputPath())
		}
		return filepath.Join(projectDirectory, file.PackageName(), name)
	case emit.TargetFilePackageState:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-state.ts",
		)
	case emit.TargetFilePackageAssembly:
		return filepath.Join(
			projectDirectory,
			file.PackageName(),
			"expected-package.ts",
		)
	case emit.TargetFileProgramInitialization:
		return filepath.Join(projectDirectory, "expected-program.ts")
	case emit.TargetFileSupport:
		return supportExpectedPath(projectDirectory, file)
	default:
		t.Fatalf("unexpected target file kind %d", file.Kind())
		return ""
	}
}

func supportExpectedPath(
	projectDirectory string,
	file emit.TargetFile,
) string {
	if file.OutputPath() == "runtime/source-fact.ts" {
		return filepath.Join(projectDirectory, "expected-source-fact.ts")
	}
	return filepath.Join(
		repositoryRoot(),
		"testdata",
		"support",
		"scalars-int32.ts",
	)
}

func TestCanonicalSourceFactsCoverRepresentativeNativeDecisions(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/sourcefacts\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package sourcefacts

type Count int32
type CountAlias = Count

type Embedded struct {
	Name string
}

type Record struct {
	Embedded
	ID Count `+"`json:\"id\"`"+`
	_ [2]byte
}

type RecordPointer *Record

type Output chan<- int32

type Counter interface {
	Next(delta int32) (int32, bool)
}

var Shared Record

func (record *Record) Next(delta int32) (int32, bool) {
	record.ID += Count(delta)
	return int32(record.ID), true
}

func AsCounter(record *Record) Counter {
	return record
}

func Launch(values Output) {
	go func() {
		values <- 1
	}()
}

func Local() int32 {
	type LocalRecord struct {
		Value int32
	}
	return LocalRecord{Value: 2}.Value
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeArtifacts(t, emission, t.TempDir())
	for _, required := range []string{
		"GoAggregateFact",
		"GoBasicFact",
		"GoCallableFact",
		"GoDeclarationFact",
		"GoInterfaceFact",
		"GoOperationFact",
		"GoStorageFact",
		"gotots-go-source-compilation-fact-v1",
		"gotots-go-source-declaration-fact-v1",
		"gotots-go-source-basic-fact-v1",
		"gotots-go-source-aggregate-fact-v1",
		"gotots-go-source-callable-fact-v1",
		"gotots-go-source-interface-fact-v1",
		"gotots-go-source-storage-fact-v1",
		"gotots-go-source-member-fact-v3",
		"gotots-go-interface-implementation-fact-v1",
		"gotots-go-struct-operation-fact-v1",
		"gotots-go-package-storage-fact-v1",
		"gotots-go-package-initialization-fact-v2",
		"gotots-go-runtime-declaration-fact-v2",
		"example.com/sourcefacts",
		`json:\"id\"`,
		"send-only",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"canonical source facts omit %q\n%s",
				required,
				factEvidenceExcerpt(artifacts.printed),
			)
		}
	}
	if !strings.Contains(artifacts.printed, "goSpawn") ||
		!strings.Contains(artifacts.printed, "1105") {
		t.Fatal("goroutine output omits the exact concurrency operation identity")
	}
	if strings.Contains(artifacts.printed, "go func") {
		t.Fatal("canonical output retained source spelling instead of typed AST evidence")
	}
}

func factEvidenceExcerpt(printed string) string {
	var selected []string
	for _, line := range strings.Split(printed, "\n") {
		if strings.Contains(line, "attribute") ||
			strings.Contains(line, "gotots-go-") {
			selected = append(selected, line)
		}
		if len(selected) == 40 {
			break
		}
	}
	return strings.Join(selected, "\n")
}
