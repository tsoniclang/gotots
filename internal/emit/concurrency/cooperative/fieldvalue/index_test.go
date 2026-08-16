package fieldvalue_test

import (
	"context"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative/fieldvalue"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestOpaquePackageImplementationKeepsCallableFieldOpen(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/opaque

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package opaque

type holder struct {
	callback func()
}

func target() {}

func assign(value *holder) {
	value.callback = target
}
`)
	writeFile(t, filepath.Join(project, "opaque.s"), "// opaque implementation\n")
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	if len(source.OtherFiles()) != 1 {
		t.Fatalf("selected opaque files = %v, want one", source.OtherFiles())
	}
	otherFiles := source.OtherFiles()
	otherFiles[0] = "mutated"
	if source.OtherFiles()[0] == "mutated" {
		t.Fatal("selected opaque files expose mutable backing storage")
	}
	named := source.Types().Scope().Lookup("holder")
	structure := named.Type().Underlying().(*types.Struct)
	if _, exact := fieldvalue.New(program).Assignments(structure.Field(0)); exact {
		t.Fatal("opaque package implementation produced a closed field write set")
	}
}

func TestUnsafePackageKeepsCallableFieldOpen(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/unsafeowner

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package unsafeowner

import "unsafe"

type holder struct { callback func() }

var _ unsafe.Pointer

func target() {}
func assign(value *holder) { value.callback = target }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	named := source.Types().Scope().Lookup("holder")
	structure := named.Type().Underlying().(*types.Struct)
	if _, exact := fieldvalue.New(program).Assignments(structure.Field(0)); exact {
		t.Fatal("unsafe package produced a closed callable-field write set")
	}
}

func TestCallableFieldAssignmentsExposeNoMutableBackingStorage(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/closed

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package closed

type holder struct {
	callback func()
}

func target() {}
func second() {}

func assign(value *holder) {
	value.callback = target
	value.callback = second
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	named := source.Types().Scope().Lookup("holder")
	structure := named.Type().Underlying().(*types.Struct)
	index := fieldvalue.New(program)
	assignments, exact := index.Assignments(structure.Field(0))
	if !exact || len(assignments) != 2 || assignments[0] == nil ||
		assignments[1] == nil || assignments[0].Pos() >= assignments[1].Pos() {
		t.Fatalf("assignments = (%v, %v), want two source-ordered expressions", assignments, exact)
	}
	assignments[0] = nil
	second, exact := index.Assignments(structure.Field(0))
	if !exact || len(second) != 2 || second[0] == nil {
		t.Fatal("callable field assignments expose mutable backing storage")
	}
}

func TestIndirectCallableFieldWritesStayOpen(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/indirect

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package indirect

type holder struct {
	ranged func()
	tuple func()
	addressed func()
}

func pair() (func(), bool) { return nil, false }

func assign(value *holder, callbacks []func()) {
	for _, value.ranged = range callbacks {}
	value.tuple, _ = pair()
	_ = &value.addressed
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	named := source.Types().Scope().Lookup("holder")
	structure := named.Type().Underlying().(*types.Struct)
	index := fieldvalue.New(program)
	for fieldIndex := 0; fieldIndex < structure.NumFields(); fieldIndex++ {
		field := structure.Field(fieldIndex)
		if _, exact := index.Assignments(field); exact {
			t.Fatalf("indirect field %s produced a closed write set", field.Name())
		}
	}
}

func TestInstantiatedCallableFieldWritesJoinTheirDeclaration(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/genericfield

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package genericfield

type holder[T any] struct { callback func(T) }

func target(int) {}

func assign(value *holder[int]) { value.callback = target }
func use(value *holder[int]) func(int) { return value.callback }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	origin := source.Types().Scope().Lookup("holder").Type().(*types.Named).Underlying().(*types.Struct).Field(0)
	var selected *types.Var
	for selector, selection := range source.TypesInfo().Selections {
		parent, _ := source.SyntaxParent(selector)
		if selector.Sel.Name == "callback" {
			if _, assignment := parent.(*ast.AssignStmt); !assignment {
				selected, _ = selection.Obj().(*types.Var)
			}
		}
	}
	if selected == nil {
		t.Fatal("generic field use has no exact go/types selection")
	}
	index := fieldvalue.New(program)
	for name, field := range map[string]*types.Var{
		"declaration": origin,
		"selected":    selected,
	} {
		assignments, exact := index.Assignments(field)
		if !exact || len(assignments) != 1 || assignments[0] == nil {
			t.Fatalf(
				"%s field assignments = (%v, %v), want one canonical write",
				name,
				assignments,
				exact,
			)
		}
	}
}

func TestDefinedCallableFieldRetainsNamedRepresentation(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/namedfield

go 1.26.4
`)
	writeFile(t, filepath.Join(project, "source.go"), `package namedfield

type callback func()
type holder struct { callback callback }

func target() {}
func assign(value *holder) { value.callback = callback(target) }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	field := source.Types().Scope().Lookup("holder").Type().(*types.Named).Underlying().(*types.Struct).Field(0)
	if _, exact := fieldvalue.New(program).Assignments(field); exact {
		t.Fatal("defined function type was admitted as plain callable transport")
	}
}

func TestForeignExportedFieldDoesNotOpenLocalFieldCensus(t *testing.T) {
	project := t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), `module example.com/foreignuse

go 1.26.4
`)
	if err := os.MkdirAll(filepath.Join(project, "foreign"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(project, "foreign", "foreign.go"), `package foreign

type Holder struct { Callback func() }
`)
	writeFile(t, filepath.Join(project, "source.go"), `package foreignuse

import "example.com/foreignuse/foreign"

type holder struct { callback func() }

func target() {}
func makeHolder() *holder { return &holder{callback: target} }
func foreignCallback(value foreign.Holder) func() { return value.Callback }
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory: project,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := program.Roots()[0]
	field := source.Types().Scope().Lookup("holder").Type().(*types.Named).Underlying().(*types.Struct).Field(0)
	assignments, exact := fieldvalue.New(program).Assignments(field)
	if !exact || len(assignments) != 1 || assignments[0] == nil {
		t.Fatalf("local assignments = (%v, %v), want one closed write", assignments, exact)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
