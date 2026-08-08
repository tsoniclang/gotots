package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestStatefulWriterProfilePreservesRetainedInterfaceABI(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/statefulwriterprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package statefulwriterprofile

import (
	"bufio"
	"io"
	"sync"
)

type blockingFailure struct { mutex *sync.Mutex }

func (failure *blockingFailure) Error() string {
	failure.mutex.Lock()
	failure.mutex.Unlock()
	return "write failed"
}

type blockingWriter struct {
	mutex *sync.Mutex
	data []byte
	short bool
	fail bool
}

func (writer *blockingWriter) Write(source []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.fail {
		return 0, &blockingFailure{mutex: writer.mutex}
	}
	if writer.short {
		return len(source) - 1, nil
	}
	writer.data = append(writer.data, source...)
	return len(source), nil
}

type holder struct { writer *bufio.Writer }

func CompositeOnly(target io.Writer) *bufio.Writer {
	return (&holder{writer: bufio.NewWriter(target)}).writer
}

func Run(text string) (string, string) {
	target := &blockingWriter{mutex: &sync.Mutex{}}
	selected := holder{writer: bufio.NewWriter(target)}
	var asWriter io.Writer = selected.writer
	if _, failure := asWriter.Write([]byte(text)); failure != nil {
		return "", failure.Error()
	}
	if failure := selected.writer.WriteByte('!'); failure != nil {
		return "", failure.Error()
	}
	if failure := selected.writer.Flush(); failure != nil {
		return "", failure.Error()
	}
	return string(target.data), ""
}

func ShortWrite() bool {
	target := &blockingWriter{mutex: &sync.Mutex{}, short: true}
	writer := bufio.NewWriter(target)
	_, _ = writer.Write([]byte("abc"))
	return writer.Flush() == io.ErrShortWrite
}

func StickyFailure() bool {
	target := &blockingWriter{mutex: &sync.Mutex{}, fail: true}
	writer := bufio.NewWriter(target)
	_, _ = writer.Write([]byte("abc"))
	first := writer.Flush()
	second := writer.WriteByte('!')
	return first != nil && first == second
}

func NilConstructed() bool { return bufio.NewWriter(nil) != nil }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(t, scope.Lookup("CompositeOnly")),
			mustProviderRoot(t, scope.Lookup("Run")),
			mustProviderRoot(t, scope.Lookup("ShortWrite")),
			mustProviderRoot(t, scope.Lookup("StickyFailure")),
			mustProviderRoot(t, scope.Lookup("NilConstructed")),
		},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	assemblyPath := ""
	for _, file := range emission.Files() {
		if file.Kind() == emit.TargetFilePackageAssembly &&
			file.PackageName() == "statefulwriterprofile" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("stateful writer package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"NilConstructed", "Run", "ShortWrite", "StickyFailure"},
		`const [text, failure] = await Run("alpha");
console.log(JSON.stringify(text) + " " + JSON.stringify(failure));
console.log(await ShortWrite());
console.log(await StickyFailure());
console.log(await NilConstructed());
`,
	)
	for _, required := range []string{
		"CanonicalBufioWriter",
		"bindPointer<",
		"await $goProviderState",
		"WriteByte",
		".Flush(",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("stateful writer output lacks %q:\n%s", required, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "BufioWriterWrite") {
		t.Fatal("stateful writer adapter retained the ordinary recovery target")
	}
}
