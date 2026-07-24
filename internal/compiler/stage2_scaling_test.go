package compiler

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/language/frontend"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

type stage2ScalingSample struct {
	size int
	work frontend.Work
}

func TestStage2ProductionWorkIsLinearForWideAndDeepDefinitions(
	t *testing.T,
) {
	for _, fixture := range []struct {
		name   string
		sizes  []int
		source func(int) string
	}{
		{
			name: "wide-siblings", sizes: []int{16, 32, 64},
			source: stage2WideSource,
		},
		{
			name: "deep-nesting", sizes: []int{8, 16, 32},
			source: stage2DeepSource,
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			samples := make(
				[]stage2ScalingSample, 0, len(fixture.sizes),
			)
			for _, size := range fixture.sizes {
				sample := inspectStage2Scaling(
					t, size, fixture.source(size),
				)
				assertStage2WorkConservation(t, sample)
				samples = append(samples, sample)
			}
			assertStage2DoublingBound(t, samples)
		})
	}
}

func inspectStage2Scaling(
	t *testing.T,
	size int,
	sourceText string,
) stage2ScalingSample {
	t.Helper()
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/scaling\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "scaling.go", sourceText)
	inspection, err := inspectConstructsForTest(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return stage2ScalingSample{
		size: size, work: inspection.SemanticWork(),
	}
}

func assertStage2WorkConservation(
	t *testing.T,
	sample stage2ScalingSample,
) {
	t.Helper()
	work := sample.work
	occurrences := work.InputOccurrences
	if occurrences == 0 ||
		work.ContextAssignments != occurrences ||
		work.ObjectOccurrenceVisits != occurrences ||
		work.ImplicitBindingVisits != occurrences ||
		work.CaptureOccurrenceVisits != occurrences ||
		work.ResolutionVisits != occurrences ||
		work.OccurrenceResolutions != occurrences {
		t.Fatalf(
			"size %d has non-total named passes: %+v",
			sample.size, work,
		)
	}
	if work.DefinitionContainmentVisits !=
		2*work.DefinitionContainmentEntries ||
		work.DefinitionContainmentEdges >
			work.DefinitionContainmentEntries ||
		work.ContainmentProbes > work.CaptureOccurrenceVisits {
		t.Fatalf(
			"size %d has unbounded containment work: %+v",
			sample.size, work,
		)
	}
	if work.OccurrenceScopeProbes > 2*occurrences ||
		work.CheckerScopeProbes > 2*occurrences ||
		work.MemberTypeVisits > 4*occurrences+256 ||
		work.LinearOperations() > 18*occurrences+1024 ||
		work.CanonicalSortInputs > 10*occurrences+1024 {
		t.Fatalf(
			"size %d exceeds the fixed-coefficient Stage-2 bound: %+v linear=%d",
			sample.size, work, work.LinearOperations(),
		)
	}
	t.Logf(
		"size=%d occurrences=%d linear=%d sorts=%d containment=%d scope=%d/%d",
		sample.size,
		occurrences,
		work.LinearOperations(),
		work.CanonicalSortInputs,
		work.DefinitionContainmentEntries,
		work.OccurrenceScopeProbes,
		work.CheckerScopeProbes,
	)
}

func assertStage2DoublingBound(
	t *testing.T,
	samples []stage2ScalingSample,
) {
	t.Helper()
	if len(samples) != 3 {
		t.Fatalf("scaling gate requires three samples, got %d", len(samples))
	}
	linear := func(sample stage2ScalingSample) int {
		return sample.work.LinearOperations()
	}
	sorts := func(sample stage2ScalingSample) int {
		return sample.work.CanonicalSortInputs
	}
	containment := func(sample stage2ScalingSample) int {
		return sample.work.DefinitionContainmentEntries
	}
	for name, metric := range map[string]func(stage2ScalingSample) int{
		"linear-work":          linear,
		"canonical-sort-input": sorts,
		"containment-storage":  containment,
	} {
		firstDelta := metric(samples[1]) - metric(samples[0])
		secondDelta := metric(samples[2]) - metric(samples[1])
		if firstDelta <= 0 ||
			secondDelta <= 0 ||
			secondDelta > 3*firstDelta {
			t.Fatalf(
				"%s deltas %d/%d exceed the linear doubling bound: %+v",
				name, firstDelta, secondDelta, samples,
			)
		}
	}
}

func stage2WideSource(width int) string {
	var sourceText strings.Builder
	sourceText.WriteString("package scaling\n\n")
	sourceText.WriteString("func Wide(seed int) int {\n\ttotal := 0\n")
	for index := 0; index < width; index++ {
		fmt.Fprintf(
			&sourceText,
			"\tvalue%d := func() int { return seed + %d }\n",
			index,
			index,
		)
		fmt.Fprintf(
			&sourceText, "\ttotal += value%d()\n", index,
		)
	}
	sourceText.WriteString("\treturn total\n}\n")
	return sourceText.String()
}

func stage2DeepSource(depth int) string {
	var sourceText strings.Builder
	sourceText.WriteString("package scaling\n\n")
	sourceText.WriteString("func Deep(seed int) any {\n\treturn ")
	for level := 0; level < depth; level++ {
		sourceText.WriteString(
			"func() any {\n\t\t_ = seed\n\t\treturn ",
		)
	}
	sourceText.WriteString("seed\n")
	for level := 0; level < depth; level++ {
		sourceText.WriteString("\t}\n")
	}
	sourceText.WriteString("}\n")
	return sourceText.String()
}
