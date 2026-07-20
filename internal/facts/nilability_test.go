package facts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func methodBody(t *testing.T, src string) (string, bool, *ast.BlockStmt) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "fixture.go", "package p\n"+src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
			field := fn.Recv.List[0]
			name := ""
			if len(field.Names) > 0 {
				name = field.Names[0].Name
			}
			_, pointer := field.Type.(*ast.StarExpr)
			return name, pointer, fn.Body
		}
	}
	t.Fatal("no method in fixture")
	return "", false, nil
}

// The A1 shape: the first observable event is a receiver dereference —
// a call-site check is observationally equivalent, so the method emits
// as an ordinary class method.
func TestFirstDerefProves(t *testing.T) {
	cases := []string{
		`type T struct{ m map[string]int }
func (r *T) Get(k string) int { return r.m[k] }`,
		`type T struct{ m map[string]int }
func (r *T) Init() { if r.m == nil { r.m = map[string]int{} } }`,
		`type T struct{ n int }
func (r *T) Add(x int) int { v := r.n + x; return v }`,
		`type T struct{}
func (r *T) Call() string { return r.String() }
func (r *T) String() string { return "" }`,
	}
	for _, src := range cases {
		name, pointer, body := methodBody(t, src)
		fact := AnalyzeReceiverNilability(name, pointer, body)
		if !fact.EquivalentAtEntry || fact.ToleratesNil {
			t.Fatalf("must prove entry equivalence: %s -> %+v", src, fact)
		}
	}
}

// MUTATION: an observable event BEFORE the first dereference fails the
// proof — the panic would fire earlier than Go's.
func TestEffectBeforeDerefFails(t *testing.T) {
	cases := []string{
		// A call before the dereference.
		`type T struct{ n int }
func (r *T) M() int { log(); return r.n }
func log() {}`,
		// A defer boundary before the dereference.
		`type T struct{ n int }
func (r *T) M() int { defer log(); return r.n }
func log() {}`,
		// An early return: the method may never dereference.
		`type T struct{ n int }
func (r *T) M(x int) int { if x > 0 { return 0 }; return r.n }`,
	}
	for _, src := range cases {
		name, pointer, body := methodBody(t, src)
		fact := AnalyzeReceiverNilability(name, pointer, body)
		if fact.EquivalentAtEntry {
			t.Fatalf("must NOT prove entry equivalence: %s", src)
		}
	}
}

// The B7 class: a receiver-nil comparison marks the method
// nil-tolerant — it takes the free-function exception, never a check.
func TestNilComparisonMarksTolerant(t *testing.T) {
	src := `type T struct{ n int }
func (r *T) Len() int { if r == nil { return 0 }; return r.n }`
	name, pointer, body := methodBody(t, src)
	fact := AnalyzeReceiverNilability(name, pointer, body)
	if !fact.ToleratesNil || fact.EquivalentAtEntry {
		t.Fatalf("nil comparison must mark tolerant: %+v", fact)
	}
}

// A value receiver cannot be nil: the question is vacuous and the
// method is ordinary with no check.
func TestValueReceiverIsVacuouslyOrdinary(t *testing.T) {
	src := `type T struct{ n int }
func (r T) N() int { return r.n }`
	name, pointer, body := methodBody(t, src)
	fact := AnalyzeReceiverNilability(name, pointer, body)
	if !fact.EquivalentAtEntry {
		t.Fatalf("value receiver: %+v", fact)
	}
}
