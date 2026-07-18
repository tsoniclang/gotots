// A classic three-clause for-loop with a multi-result init or post
// (`for a, b := f(); …; a, b = g()`) is a generic loop-lowering case, not
// an unsupported construct: the clause destructures into simple variable
// targets and stays a native JS for-header, so continue still runs the post
// exactly once (native for-loop semantics) with one-shot source evaluation.
package emit

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestForClauseTupleDestructuring(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	p := &printer{module: m}
	src := &ir.ParamRef{Name: "src"}

	initClause, err := p.forClause(&ir.DeclStmt{
		Names: []string{"ch", "size"},
		Types: []ir.Type{{Kind: ir.KindInt64}, {Kind: ir.KindInt}},
		Tuple: src,
	}, true)
	if err != nil {
		t.Fatalf("init tuple clause: %v", err)
	}
	if initClause != "let [ch, size]: [goabi$.GoInt64, goabi$.GoInt] = src" {
		t.Fatalf("init clause = %q", initClause)
	}

	postClause, err := p.forClause(&ir.AssignStmt{
		Targets: []ir.Target{
			ir.VarTarget{Name: "ch", T: ir.Type{Kind: ir.KindInt64}},
			ir.VarTarget{Name: "size", T: ir.Type{Kind: ir.KindInt}},
		},
		Tuple: src,
	}, false)
	if err != nil {
		t.Fatalf("post tuple clause: %v", err)
	}
	if postClause != "[ch, size] = src" {
		t.Fatalf("post clause = %q", postClause)
	}
}

func TestForClauseParallelAssign(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	p := &printer{module: m}
	// a, b = c, d — the right sides destructure so every value evaluates
	// before any store (Go's two-phase rule, and a swap works).
	clause, err := p.forClause(&ir.AssignStmt{
		Targets: []ir.Target{
			ir.VarTarget{Name: "a", T: ir.Type{Kind: ir.KindInt}},
			ir.VarTarget{Name: "b", T: ir.Type{Kind: ir.KindInt}},
		},
		Values: []ir.Expr{&ir.ParamRef{Name: "c"}, &ir.ParamRef{Name: "d"}},
	}, false)
	if err != nil {
		t.Fatalf("parallel assign clause: %v", err)
	}
	if clause != "[a, b] = [c, d]" {
		t.Fatalf("parallel clause = %q", clause)
	}
}

func TestForClauseStructTargetNotDestructured(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	p := &printer{module: m}
	// A value-copy carrier target (struct) needs a copy call, not a plain
	// destructure — it must NOT be silently mis-emitted as `[x] = …`.
	_, err := p.forClause(&ir.AssignStmt{
		Targets: []ir.Target{
			ir.VarTarget{Name: "x", T: ir.Type{Kind: ir.KindStruct}},
			ir.VarTarget{Name: "y", T: ir.Type{Kind: ir.KindInt}},
		},
		Tuple: &ir.ParamRef{Name: "src"},
	}, false)
	if err == nil {
		t.Fatal("a struct destructuring target must not be accepted as a plain clause")
	}
}
