package provider_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestDisabledFmtWriterCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directfmtwriter\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directfmtwriter

import "fmt"

type writer struct{}

func (*writer) Write(source []byte) (int, error) { return len(source), nil }

func Result() (int, error) {
	target := &writer{}
	first, failure := fmt.Fprint(target, "first")
	if failure != nil {
		return 0, failure
	}
	second, failure := fmt.Fprintf(target, "%s", "second")
	if failure != nil {
		return 0, failure
	}
	third, failure := fmt.Fprintln(target, "third")
	return first + second + third, failure
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
	options := providerNumberOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Result"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	for _, expected := range []string{
		"FprintDirect",
		"FprintfDirect",
		"FprintlnDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled fmt writer profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled fmt writer profiles are not direct:\n%s", artifacts.printed)
	}
}

func TestDisabledIoFsCallablesSelectDirectProfiles(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directiofs\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directiofs

import (
	"io/fs"
	"time"
)

type information struct{}

func (information) Name() string       { return "entry" }
func (information) Size() int64        { return 0 }
func (information) Mode() fs.FileMode  { return 0 }
func (information) ModTime() time.Time { return time.Time{} }
func (information) IsDir() bool        { return false }
func (information) Sys() any           { return nil }

type entry struct{}

func (entry) Name() string               { return "entry" }
func (entry) IsDir() bool                { return false }
func (entry) Type() fs.FileMode          { return 0 }
func (entry) Info() (fs.FileInfo, error) { return information{}, nil }

type file struct{}

func (*file) Stat() (fs.FileInfo, error) { return information{}, nil }
func (*file) Read([]byte) (int, error)    { return 0, fs.ErrClosed }
func (*file) Close() error                { return nil }
func (*file) ReadDir(int) ([]fs.DirEntry, error) {
	return []fs.DirEntry{entry{}}, nil
}

type fileSystem struct{}

func (*fileSystem) Open(string) (fs.File, error) { return &file{}, nil }
func (*fileSystem) ReadFile(string) ([]byte, error) { return []byte("data"), nil }
func (*fileSystem) ReadDir(string) ([]fs.DirEntry, error) {
	return []fs.DirEntry{entry{}}, nil
}
func (*fileSystem) Stat(string) (fs.FileInfo, error) { return information{}, nil }

func Result() error {
	source := &fileSystem{}
	information, failure := fs.Stat(source, ".")
	if failure != nil {
		return failure
	}
	_ = fs.FileInfoToDirEntry(information)
	if _, failure = fs.ReadFile(source, "."); failure != nil {
		return failure
	}
	if _, failure = fs.ReadDir(source, "."); failure != nil {
		return failure
	}
	return fs.WalkDir(source, ".", func(
		_ string,
		_ fs.DirEntry,
		failure error,
	) error {
		return failure
	})
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
	options := providerNumberOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Result"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	for _, expected := range []string{
		"IoFsFileInfoToDirEntryDirect",
		"IoFsReadDirDirect",
		"IoFsReadFileDirect",
		"IoFsStatDirect",
		"IoFsWalkDirDirect",
	} {
		if !strings.Contains(artifacts.printed, expected) {
			t.Fatalf("disabled io/fs profile omits %s:\n%s", expected, artifacts.printed)
		}
	}
	if strings.Contains(artifacts.printed, "Canonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled io/fs profiles are not direct:\n%s", artifacts.printed)
	}
}

func TestDisabledIoReadFullSelectsDirectProfile(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directioread\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directioread

import "io"

type reader struct{}

func (*reader) Read(target []byte) (int, error) {
	copy(target, "source")
	return len(target), nil
}

func Result(target []byte) (int, error) {
	return io.ReadFull(&reader{}, target)
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
	options := providerNumberOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Result"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if !strings.Contains(artifacts.printed, "IoReadFullDirect") ||
		strings.Contains(artifacts.printed, "IoReadFullCanonical") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled io.ReadFull profile is not direct:\n%s", artifacts.printed)
	}
}

func TestDisabledStatefulProfileSelectsDirectCarrier(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/directstatefulprofile\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package directstatefulprofile

import "bufio"

type sink struct { data []byte }

func (target *sink) Write(source []byte) (int, error) {
	target.data = append(target.data, source...)
	return len(source), nil
}

func Render(text string) string {
	target := &sink{}
	writer := bufio.NewWriter(target)
	_, _ = writer.Write([]byte(text))
	_ = writer.WriteByte('!')
	_ = writer.Flush()
	return string(target.data)
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
	options := providerNumberOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(
		program,
		[]emit.Root{mustProviderRoot(
			t,
			program.Roots()[0].Types().Scope().Lookup("Render"),
		)},
		options,
	)
	if err != nil {
		t.Fatal(err)
	}
	workingDirectory := t.TempDir()
	artifacts := materializeArtifacts(t, emission, workingDirectory)
	waveThreeTypecheck(t, workingDirectory, artifacts.paths)
	if !strings.Contains(artifacts.printed, "DirectBufioWriter") ||
		!strings.Contains(artifacts.printed, "NewWriterDirect") ||
		strings.Contains(artifacts.printed, "CanonicalBufioWriter") ||
		strings.Contains(artifacts.printed, "async ") ||
		strings.Contains(artifacts.printed, "await ") {
		t.Fatalf("disabled stateful profile is not direct:\n%s", artifacts.printed)
	}
}
