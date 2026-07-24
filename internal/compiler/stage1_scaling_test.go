package compiler

import (
	"fmt"
	"math/bits"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestStage1ConstructionWorkScalesWithProducedStructure(t *testing.T) {
	sizes := []int{20, 40, 80}
	var previous stage1ScalePoint
	for _, size := range sizes {
		point := inspectScaleFixture(
			t,
			fmt.Sprintf("example.com/wide-%d", size),
			wideDefinitionFixture(size),
		)
		if point.definitions != 3*size+1 {
			t.Fatalf(
				"size=%d definitions=%d, want %d",
				size,
				point.definitions,
				3*size+1,
			)
		}
		assertStage1WorkBound(t, "wide", point)
		if previous.scale != 0 {
			assertLinearGrowth(t, "wide", previous, point, 1.08)
		}
		previous = point
	}

	previous = stage1ScalePoint{}
	for _, depth := range []int{2, 4, 8} {
		point := inspectScaleFixture(
			t,
			fmt.Sprintf("example.com/deep-%d", depth),
			deepDefinitionFixture(depth),
		)
		if point.definitions != depth+2 {
			t.Fatalf(
				"depth=%d definitions=%d, want %d",
				depth,
				point.definitions,
				depth+2,
			)
		}
		assertStage1WorkBound(t, "deep", point)
		if previous.scale != 0 {
			assertLinearGrowth(t, "deep", previous, point, 1.15)
		}
		previous = point
	}

	previous = stage1ScalePoint{}
	for _, depth := range sizes {
		point := inspectScaleFixture(
			t,
			fmt.Sprintf("example.com/shared-path-%d", depth),
			sharedContainmentFixture(depth),
		)
		if point.definitions != depth+2 {
			t.Fatalf(
				"shared depth=%d definitions=%d, want %d",
				depth,
				point.definitions,
				depth+2,
			)
		}
		assertStage1WorkBound(t, "shared-path", point)
		if previous.scale != 0 {
			assertLinearGrowth(
				t,
				"shared-path",
				previous,
				point,
				1.08,
			)
		}
		previous = point
	}
}

type stage1ScalePoint struct {
	scale                 int
	definitions           int
	sites                 int
	uniqueAnchors         int
	headerOccurrences     int
	boundaryEntries       int
	structuralOccurrences int
	executableOccurrences int
	linearWork            int
	sortComparisons       int
}

func inspectScaleFixture(
	t *testing.T,
	module string,
	sourceText string,
) stage1ScalePoint {
	t.Helper()
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module "+module+"\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "fixture.go", sourceText)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	point := stage1ScalePoint{
		definitions:           len(inspection.Structure().DefinitionCensus()),
		headerOccurrences:     inspection.Structure().HeaderOccurrenceCount(),
		boundaryEntries:       inspection.Structure().BoundaryEntryCount(),
		structuralOccurrences: len(inspection.Structure().ResidentOccurrences()),
	}
	if err := inspection.Structure().VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		point.sites += len(pkg.Sites())
		for _, file := range pkg.Files() {
			point.uniqueAnchors += len(file.Containment().Anchors())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, region := range inspection.Executable().Regions() {
		point.executableOccurrences += len(region.Members())
	}
	structuralWork := inspection.Structure().Work()
	executableWork := inspection.Executable().Work()
	point.linearWork =
		linearStructureWork(structuralWork) +
			linearExecutableWork(executableWork)
	point.sortComparisons =
		structuralWork.SortComparisons +
			executableWork.SortComparisons
	point.scale = point.definitions
	return point
}

func linearStructureWork(work structure.Work) int {
	return work.CatalogEdges +
		work.BoundaryProbes +
		work.RecordAppends +
		work.IdentityProbes +
		work.JoinProbes
}

func linearExecutableWork(work executable.Work) int {
	return work.CatalogEdges +
		work.BoundaryProbes +
		work.RecordAppends +
		work.IdentityProbes +
		work.JoinProbes
}

func assertStage1WorkBound(
	t *testing.T,
	class string,
	point stage1ScalePoint,
) {
	t.Helper()
	denominator := point.definitions +
		point.sites +
		point.uniqueAnchors +
		point.headerOccurrences +
		point.boundaryEntries +
		point.structuralOccurrences +
		point.executableOccurrences
	if denominator == 0 {
		t.Fatalf("%s fixture produced no structure", class)
	}
	const maximumLinearOperationsPerRecord = 32
	if point.linearWork > maximumLinearOperationsPerRecord*denominator {
		t.Fatalf(
			"%s linear work=%d exceeds %d× structural denominator=%d: %+v",
			class,
			point.linearWork,
			maximumLinearOperationsPerRecord,
			denominator,
			point,
		)
	}
	sortBound := 12 * denominator * bits.Len(uint(denominator+1))
	if point.sortComparisons > sortBound {
		t.Fatalf(
			"%s sort comparisons=%d exceed n-log-n bound=%d: %+v",
			class,
			point.sortComparisons,
			sortBound,
			point,
		)
	}
}

func assertLinearGrowth(
	t *testing.T,
	class string,
	previous stage1ScalePoint,
	current stage1ScalePoint,
	growthAllowance float64,
) {
	t.Helper()
	structureGrowth := float64(current.scale) / float64(previous.scale)
	workGrowth := float64(current.linearWork) / float64(previous.linearWork)
	if workGrowth > structureGrowth*growthAllowance {
		t.Fatalf(
			"%s work grew %.2fx while definitions grew %.2fx (allowance %.2fx): previous=%+v current=%+v",
			class,
			workGrowth,
			structureGrowth,
			growthAllowance,
			previous,
			current,
		)
	}
}

func wideDefinitionFixture(count int) string {
	var sourceText strings.Builder
	sourceText.WriteString("package scale\n")
	for index := 0; index < count; index++ {
		fmt.Fprintf(&sourceText, "\nfunc F%d() int {\n", index)
		sourceText.WriteString("\touter := func() int {\n")
		fmt.Fprintf(
			&sourceText,
			"\t\tinner := func() int { return %d }\n",
			index,
		)
		sourceText.WriteString(
			"\t\treturn inner()\n\t}\n\treturn outer()\n}\n",
		)
	}
	return sourceText.String()
}

func deepDefinitionFixture(depth int) string {
	var sourceText strings.Builder
	sourceText.WriteString("package scale\n\nfunc Deep() int {\n")
	for index := 0; index < depth; index++ {
		fmt.Fprintf(
			&sourceText,
			"%sf%d := func() int {\n",
			strings.Repeat("\t", index+1),
			index,
		)
	}
	fmt.Fprintf(
		&sourceText,
		"%sreturn %d\n",
		strings.Repeat("\t", depth+1),
		depth,
	)
	for index := depth - 1; index >= 0; index-- {
		indent := strings.Repeat("\t", index+1)
		sourceText.WriteString(indent + "}\n")
		if index > 0 {
			fmt.Fprintf(
				&sourceText,
				"%sreturn f%d()\n",
				indent,
				index,
			)
		}
	}
	sourceText.WriteString("\treturn f0()\n}\n")
	return sourceText.String()
}

func sharedContainmentFixture(depth int) string {
	var sourceText strings.Builder
	sourceText.WriteString("package scale\n\nfunc Shared() {\n")
	for index := 0; index < depth; index++ {
		sourceText.WriteString(strings.Repeat("\t", index+1) + "{\n")
		fmt.Fprintf(
			&sourceText,
			"%s_ = func() int { return %d }\n",
			strings.Repeat("\t", index+2),
			index,
		)
	}
	for index := depth - 1; index >= 0; index-- {
		sourceText.WriteString(strings.Repeat("\t", index+1) + "}\n")
	}
	sourceText.WriteString("}\n")
	return sourceText.String()
}
