package compiler

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// wideNestedLiteralFixture generates n functions, each carrying a two-level
// nested literal chain — a wide fan of nested literals. Unit count is exactly
// 3n (one func body plus two literals each), so the traversal's work is a
// deterministic function of n.
func wideNestedLiteralFixture(n int) string {
	var b strings.Builder
	b.WriteString("package m\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "\nfunc F%d() int {\n", i)
		b.WriteString("\touter := func() int {\n")
		fmt.Fprintf(&b, "\t\tinner := func() int { return %d }\n", i)
		b.WriteString("\t\treturn inner()\n\t}\n\treturn outer()\n}\n")
	}
	return b.String()
}

// TestWideNestedLiteralScaling proves the reference-model traversal is linear in
// program size, using the traversal's ACTUAL construction-operation count (one
// per visited node, accumulated in production) as the complexity proof — not
// output cardinality and not wall time. Across a doubling fixture the operations
// per unit stay constant (no per-implementer growth) and the total operations
// double when the program doubles. A quadratic control demonstrates the metric
// distinguishes O(n) construction from O(n^2). Output cardinality and isolated
// wall/heap are reported as corroboration only.
func TestWideNestedLiteralScaling(t *testing.T) {
	sizes := []int{40, 80, 160}
	type point struct {
		n, ops, occ, refs, units int
	}
	var pts []point

	for _, n := range sizes {
		dir := writeFixture(t, map[string]string{
			"go.mod":  "module scale.example/m\n\ngo 1.26\n",
			"main.go": wideNestedLiteralFixture(n),
		})
		req := source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}
		insp, err := InspectConstructs(req)
		if err != nil {
			t.Fatalf("n=%d inspect: %v", n, err)
		}
		ops := insp.Inventory().TraversalOperations()
		occ, refs, units := workCardinality(t, insp)
		if units != 3*n {
			t.Fatalf("n=%d: unit count = %d, want exactly %d (1 body + 2 literals per function)", n, units, 3*n)
		}

		// Determinism: a second run yields identical construction work.
		insp2, err := InspectConstructs(req)
		if err != nil {
			t.Fatalf("n=%d re-inspect: %v", n, err)
		}
		if got := insp2.Inventory().TraversalOperations(); got != ops {
			t.Fatalf("n=%d construction work is non-deterministic: %d vs %d ops", n, ops, got)
		}
		pts = append(pts, point{n: n, ops: ops, occ: occ, refs: refs, units: units})
	}

	for _, p := range pts {
		t.Logf("n=%-4d units=%-5d ops(construction)=%-7d ops/unit=%.2f  occ(output)=%-6d refs=%-5d",
			p.n, p.units, p.ops, float64(p.ops)/float64(p.units), p.occ, p.refs)
	}

	// Primary proof: construction operations per unit are constant across sizes
	// (no per-implementer growth). If any per-unit cost grew with program size,
	// ops/unit would rise with n.
	base := float64(pts[0].ops) / float64(pts[0].units)
	for _, p := range pts[1:] {
		ratio := (float64(p.ops) / float64(p.units)) / base
		if ratio < 0.98 || ratio > 1.02 {
			t.Errorf("n=%d construction ops/unit drifted to %.2fx baseline — traversal is not O(1) per unit", p.n, ratio)
		}
	}
	// Doubling n doubles the construction operations (linear); a quadratic would
	// ~quadruple them.
	for i := 1; i < len(pts); i++ {
		grow := float64(pts[i].ops) / float64(pts[i-1].ops)
		want := float64(pts[i].n) / float64(pts[i-1].n)
		if grow < want*0.9 || grow > want*1.1 {
			t.Errorf("construction ops grew %.2fx from n=%d to n=%d, want ~%.2fx (linear)", grow, pts[i-1].n, pts[i].n, want)
		}
	}

	// Quadratic control: a scanner that, for each unit, rescans the whole file
	// (the classic non-cubic-but-quadratic regression the linear traversal must
	// not become). Its operation count grows super-linearly, and the linear
	// traversal's real ops must NOT match this profile — proving the metric
	// catches a quadratic implementation, not merely reproduces output.
	linGrow := float64(pts[len(pts)-1].ops) / float64(pts[0].ops)
	var quad []int
	for _, n := range sizes {
		src := wideNestedLiteralFixture(n)
		quad = append(quad, quadraticScanOps(t, src))
	}
	quadGrow := float64(quad[len(quad)-1]) / float64(quad[0])
	t.Logf("n %d->%d: linear-traversal ops grew %.2fx; quadratic-scan control grew %.2fx", sizes[0], sizes[len(sizes)-1], linGrow, quadGrow)
	if quadGrow < linGrow*1.5 {
		t.Errorf("quadratic control grew only %.2fx vs linear %.2fx — the control is not actually quadratic", quadGrow, linGrow)
	}
	// The real traversal must track the linear (n) growth, not the quadratic one.
	nGrow := float64(sizes[len(sizes)-1]) / float64(sizes[0])
	if linGrow > nGrow*1.1 {
		t.Errorf("real traversal grew %.2fx, exceeding linear %.2fx — it exhibits the quadratic profile", linGrow, nGrow)
	}
}

// workCardinality reports OUTPUT cardinality (occurrences, references, units) —
// corroborating evidence only, never named as construction work.
func workCardinality(t *testing.T, insp *Inspection) (occ, refs, units int) {
	t.Helper()
	for _, pkg := range insp.Inventory().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		refs += len(pkg.References())
		for _, region := range pkg.Files() {
			occ += len(region.Occurrences())
		}
	}
	for _, pkg := range insp.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, file := range pkg.Files() {
			units += len(file.Units())
		}
	}
	return occ, refs, units
}

// quadraticScanOps counts the operations of a deliberately quadratic derivation:
// for every unit-bearing node it rescans the whole file to "find its parent".
// This is the regression the linear single-pass traversal must not become; the
// operation metric distinguishes the two.
func quadraticScanOps(t *testing.T, src string) int {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "m.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var all []ast.Node
	ast.Inspect(file, func(n ast.Node) bool {
		if n != nil {
			all = append(all, n)
		}
		return true
	})
	ops := 0
	for _, n := range all {
		if _, ok := n.(*ast.FuncLit); !ok {
			continue
		}
		// Rescan every node to locate the enclosing unit — O(nodes) per unit.
		for range all {
			ops++
		}
	}
	return ops
}
