package environment

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestEquivalentMethodsNormalizesGenericReceiverParameters(t *testing.T) {
	methods := checkedMethods(t, `package generic
type First[T, U any] struct{}
func (First[T, U]) Read(left T, right U) T { return left }
type Renamed[A, B any] struct{}
func (Renamed[A, B]) Read(left A, right B) A { return left }
type Swapped[A, B any] struct{}
func (Swapped[A, B]) Read(left B, right A) A { return right }
`)
	if !EquivalentMethods(methods("First"), methods("Renamed")) {
		t.Fatal("alpha-equivalent receiver methods do not match")
	}
	if EquivalentMethods(methods("First"), methods("Swapped")) {
		t.Fatal("receiver-parameter ordinal change was ignored")
	}
}

func TestEquivalentMethodsNormalizesGenericInterfaceParameters(t *testing.T) {
	methods := checkedMethods(t, `package generic
type First[T any] interface { Read(T) T }
type Renamed[U any] interface { Read(U) U }
type Different[T any] interface { Read(T) string }
`)
	if !EquivalentMethods(methods("First"), methods("Renamed")) {
		t.Fatal("alpha-equivalent generic interface methods do not match")
	}
	if EquivalentMethods(methods("First"), methods("Different")) {
		t.Fatal("different generic interface method results match")
	}
}

func checkedMethods(t *testing.T, sourceText string) func(string) *types.Func {
	t.Helper()
	fileSet := token.NewFileSet()
	source, err := parser.ParseFile(fileSet, "source.go", sourceText, 0)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := new(types.Config).Check(
		"example.com/generic",
		fileSet,
		[]*ast.File{source},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return func(name string) *types.Func {
		t.Helper()
		named := checked.Scope().Lookup(name).Type().(*types.Named)
		if contract, ok := named.Underlying().(*types.Interface); ok {
			return contract.Complete().Method(0)
		}
		return named.Method(0)
	}
}
