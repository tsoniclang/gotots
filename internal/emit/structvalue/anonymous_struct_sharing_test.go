package structvalue_test

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestAnonymousStructSharedArtifactIsExactAndRootOrderIndependent(
	t *testing.T,
) {
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/shared\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(directory, "one", "one.go"), `package one

func Make(value int32) struct{ Value int32 } {
	return struct{ Value int32 }{Value: value}
}
`)
	writeProgramFile(t, filepath.Join(directory, "two", "two.go"), `package two

func Make(value int32) struct{ Value int32 } {
	return struct{ Value int32 }{Value: value}
}
`)
	writeProgramFile(t, filepath.Join(directory, "source.go"), `package shared

import (
	"example.com/shared/one"
	"example.com/shared/two"
)

func FromOne(value int32) struct{ Value int32 } { return one.Make(value) }
func FromTwo(value int32) struct{ Value int32 } { return two.Make(value) }
func Copy(value struct{ Value int32 }) struct{ Value int32 } {
	copy := value
	return copy
}
func Equal(left, right struct{ Value int32 }) bool { return left == right }
func Zero() struct{ Value int32 } {
	var value struct{ Value int32 }
	return value
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	roots, err := emit.ExportedAPIRoots(program.Roots()[0])
	if err != nil {
		t.Fatal(err)
	}
	first, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(roots)
	second, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	assertEncodedProgramsEqual(t, first, second)
	typecheckSharedAnonymousProgram(t, first)

	support := anonymousStructSupport(t, first)
	var classes []tsgo.ClassDeclaration
	for _, statement := range support.Statements() {
		if class, ok := statement.(tsgo.ClassDeclaration); ok {
			classes = append(classes, class)
		}
	}
	if len(classes) != 1 {
		t.Fatalf("shared anonymous classes = %d, want one", len(classes))
	}
	class := classes[0]
	if !strings.HasPrefix(class.Name().Text(), "$goStruct$Struct_") {
		t.Fatalf("anonymous class name is not semantic: %q", class.Name().Text())
	}
	assertStaticOperationSequence(
		t,
		support,
		class.Name().Text(),
		[]string{"$zero", "$copy", "$equal"},
	)
}

func TestAnonymousStructSupportIgnoresUnrelatedSourceNames(t *testing.T) {
	ordinary := compileAnonymousStructNamedField(t, false)
	withCollision := compileAnonymousStructNamedField(t, true)
	ordinarySupport, err := tsgo.EncodeSourceFile(
		anonymousStructSupport(t, ordinary),
	)
	if err != nil {
		t.Fatal(err)
	}
	collisionSupport, err := tsgo.EncodeSourceFile(
		anonymousStructSupport(t, withCollision),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ordinarySupport, collisionSupport) {
		t.Fatal("unrelated source name changed compilation-shared support")
	}
}

func compileAnonymousStructNamedField(
	t *testing.T,
	withCollision bool,
) emit.ProgramEmission {
	t.Helper()
	directory := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/supportcontext\n\ngo 1.26.4\n",
	)
	writeProgramFile(
		t,
		filepath.Join(directory, "model", "model.go"),
		"package model\n\ntype Record struct{ Value int32 }\n",
	)
	collision := ""
	if withCollision {
		collision = `
func unrelated() {
	Record__from_model := int32(0)
	_ = Record__from_model
}
`
	}
	writeProgramFile(
		t,
		filepath.Join(directory, "source.go"),
		`package supportcontext

import "example.com/supportcontext/model"
`+collision+`
func Value(value model.Record) struct{ Value model.Record } {
	return struct{ Value model.Record }{Value: value}
}
`,
	)
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
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
	return emission
}

func typecheckSharedAnonymousProgram(
	t *testing.T,
	emission emit.ProgramEmission,
) {
	t.Helper()
	directory := t.TempDir()
	client, err := tsgo.StartClient(repositoryRoot(), directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	var paths []string
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, filepath.FromSlash(file.OutputPath()))
		writeProgramFile(t, path, printed)
		paths = append(paths, path)
	}
	writeProgramFile(
		t,
		filepath.Join(directory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	if err := typecheckStructuralFiles(directory, paths); err != nil {
		t.Fatal(err)
	}
}

func assertEncodedProgramsEqual(
	t *testing.T,
	left emit.ProgramEmission,
	right emit.ProgramEmission,
) {
	t.Helper()
	leftFiles := left.Files()
	rightFiles := right.Files()
	if len(leftFiles) != len(rightFiles) {
		t.Fatalf("target files = %d/%d", len(leftFiles), len(rightFiles))
	}
	for index := range leftFiles {
		if leftFiles[index].OutputPath() != rightFiles[index].OutputPath() ||
			leftFiles[index].Kind() != rightFiles[index].Kind() {
			t.Fatalf("target file %d identity changed", index)
		}
		leftEncoded, err := tsgo.EncodeSourceFile(leftFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		rightEncoded, err := tsgo.EncodeSourceFile(rightFiles[index].SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(leftEncoded, rightEncoded) {
			t.Fatalf(
				"root order changed target artifact %s",
				leftFiles[index].OutputPath(),
			)
		}
	}
}
