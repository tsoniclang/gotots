package compiler

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

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

// workCount is the traversal's deterministic work: total occurrences across
// every body region plus total references. It is independent of wall time.
func workCount(t *testing.T, insp *Inspection) (occ, refs, units int) {
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

// TestWideNestedLiteralScaling proves the reference-model traversal is linear in
// program size, using a DETERMINISTIC work-count as the complexity proof, not
// wall time. Across a doubling fixture the per-unit work stays constant (no
// per-implementer growth), and the total work doubles when the program doubles.
// Isolated wall and retained-heap figures are reported as corroboration only.
func TestWideNestedLiteralScaling(t *testing.T) {
	sizes := []int{40, 80, 160}
	type point struct {
		n, occ, refs, units int
		wall                time.Duration
		heapBytes           uint64
	}
	var pts []point

	for _, n := range sizes {
		dir := writeFixture(t, map[string]string{
			"go.mod":  "module scale.example/m\n\ngo 1.26\n",
			"main.go": wideNestedLiteralFixture(n),
		})
		req := source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}

		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		start := time.Now()
		insp, err := InspectConstructs(req)
		wall := time.Since(start)
		if err != nil {
			t.Fatalf("n=%d inspect: %v", n, err)
		}
		runtime.ReadMemStats(&m1)

		occ, refs, units := workCount(t, insp)
		if units != 3*n {
			t.Fatalf("n=%d: unit count = %d, want exactly %d (1 body + 2 literals per function)", n, units, 3*n)
		}

		// Determinism: a second run yields identical work.
		insp2, err := InspectConstructs(req)
		if err != nil {
			t.Fatalf("n=%d re-inspect: %v", n, err)
		}
		occ2, refs2, _ := workCount(t, insp2)
		if occ2 != occ || refs2 != refs {
			t.Fatalf("n=%d work is non-deterministic: occ %d vs %d, refs %d vs %d", n, occ, occ2, refs, refs2)
		}

		pts = append(pts, point{n: n, occ: occ, refs: refs, units: units, wall: wall, heapBytes: m1.TotalAlloc - m0.TotalAlloc})
	}

	for _, p := range pts {
		t.Logf("n=%-4d units=%-5d occ=%-6d refs=%-5d occ/unit=%.2f  wall=%-10s heapDelta=%dKiB",
			p.n, p.units, p.occ, p.refs, float64(p.occ)/float64(p.units), p.wall.Round(time.Microsecond), p.heapBytes/1024)
	}

	// Primary proof: per-unit work is constant across sizes (no per-implementer
	// growth). If any per-call cost grew with the number of sibling/nested
	// literals, occ/unit would rise with n.
	base := float64(pts[0].occ) / float64(pts[0].units)
	for _, p := range pts[1:] {
		ratio := (float64(p.occ) / float64(p.units)) / base
		if ratio < 0.98 || ratio > 1.02 {
			t.Errorf("n=%d occ/unit drifted to %.2fx the baseline — traversal work is not O(1) per unit", p.n, ratio)
		}
	}
	// Corroborating proof: total work scales with program size. Doubling n
	// doubles the deterministic work-count (a quadratic would ~quadruple it).
	for i := 1; i < len(pts); i++ {
		grow := float64(pts[i].occ) / float64(pts[i-1].occ)
		want := float64(pts[i].n) / float64(pts[i-1].n)
		if grow < want*0.9 || grow > want*1.1 {
			t.Errorf("work grew %.2fx from n=%d to n=%d, want ~%.2fx (linear)", grow, pts[i-1].n, pts[i].n, want)
		}
	}
}
