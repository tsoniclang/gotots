package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestStatefulProviderProfileDoesNotDependOnImplementerDiscovery(
	t *testing.T,
) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/profileorder\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package profileorder

import (
	"bufio"
	"io"
)

type holder struct { reader *bufio.Reader }

func Build(source io.ReadWriter) *bufio.Reader {
	return (&holder{reader: bufio.NewReader(source)}).reader
}

type localReadWriter struct{}

func (*localReadWriter) Read([]byte) (int, error) { return 0, io.EOF }
func (*localReadWriter) Write(source []byte) (int, error) { return len(source), nil }

func Use() *bufio.Reader { return Build(&localReadWriter{}) }

func Consume() bool {
	_, failure := Use().ReadByte()
	return failure == io.EOF
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{
			mustProviderRoot(
				t,
				program.Roots()[0].Types().Scope().Lookup("Consume"),
			),
			mustProviderRoot(
				t,
				program.Roots()[0].Types().Scope().Lookup("Use"),
			),
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
			file.PackageName() == "profileorder" {
			assemblyPath = file.OutputPath()
			break
		}
	}
	if assemblyPath == "" {
		t.Fatal("profile-order package assembly is absent")
	}
	typecheckProviderRunner(
		t,
		workingDirectory,
		artifacts.paths,
		assemblyPath,
		[]string{"Consume"},
		`console.log(await Consume());
`,
	)
	if !strings.Contains(artifacts.printed, "CanonicalBufioReader") {
		t.Fatalf("canonical cooperative provider state is absent:\n%s", artifacts.printed)
	}
	for _, forbidden := range []string{
		"CanonicalReaderSync",
		"CanonicalReaderAsync",
		"bufio__from_gostdlib.NewReader(",
	} {
		if strings.Contains(artifacts.printed, forbidden) {
			t.Fatalf("cooperative state profile contains stale path %q:\n%s", forbidden, artifacts.printed)
		}
	}
}
