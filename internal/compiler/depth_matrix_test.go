package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestNestedDefinitionDepthCrossProduct(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/depth-matrix\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "matrix.go", `package matrix

func Outer() func() int {
	base := 1
	return func() int { return base }
}
`)
	base := inspectDepthFixture(
		t,
		source.Request{
			Dir: directory, Patterns: []string{"."},
			ProviderContract: contract.DefaultID,
		},
	)
	outer, child := nestedPair(t, base)

	outerContract := writeDepthContract(
		t, "outer-full@v1", outer,
	)
	childContract := writeDepthContract(
		t, "child-full@v1", child,
	)
	noneContract := writeDepthContract(t, "none-full@v1")
	output := t.TempDir()
	structurePath := filepath.Join(output, "provider.structure.gotots")
	semanticPath := filepath.Join(output, "provider.semantic.gotots")
	provider, err := AuditCatalog(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         "none-full@v1",
		ProviderContractArtifact: noneContract,
	}, structurePath, semanticPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		request   source.Request
		outerFull bool
		childFull bool
	}{
		{
			name: "both-full",
			request: source.Request{
				Dir: directory, Patterns: []string{"."},
				ProviderContract: contract.DefaultID,
			},
			outerFull: true, childFull: true,
		},
		{
			name: "outer-full-child-contract",
			request: source.Request{
				Dir: directory, Patterns: []string{"."},
				ProviderContract:         "outer-full@v1",
				ProviderContractArtifact: outerContract,
			},
			outerFull: true,
		},
		{
			name: "outer-contract-child-full",
			request: source.Request{
				Dir: directory, Patterns: []string{"."},
				ProviderContract:         "child-full@v1",
				ProviderContractArtifact: childContract,
			},
			childFull: true,
		},
		{
			name: "both-contract-certified",
			request: source.Request{
				Dir: directory, Patterns: []string{"."},
				ProviderContract:          "none-full@v1",
				ProviderContractArtifact:  noneContract,
				ProviderStructureArtifact: structurePath,
				ProviderStructureDigest:   provider.Structure.Digest,
				ProviderSemanticArtifact:  semanticPath,
				ProviderSemanticDigest:    provider.Semantic.Digest,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inspection := inspectDepthFixture(t, test.request)
			assertNestedDepthShape(
				t,
				inspection,
				outer,
				child,
				test.outerFull,
				test.childFull,
			)
		})
	}
}

func inspectDepthFixture(
	t *testing.T,
	request source.Request,
) *Inspection {
	t.Helper()
	inspection, err := inspectConstructsForTest(t, request)
	if err != nil {
		t.Fatal(err)
	}
	return inspection
}

func nestedPair(
	t *testing.T,
	inspection *Inspection,
) (identity.DefinitionID, identity.DefinitionID) {
	t.Helper()
	records := collectStage1Definitions(t, inspection)
	outerDefinitions := findDefinitionsByName(records, "Outer")
	childDefinitions := findDefinitionsByName(records, "func literal")
	if len(outerDefinitions) != 1 || len(childDefinitions) != 1 {
		t.Fatalf(
			"Outer/literal definitions=%d/%d, want 1/1",
			len(outerDefinitions),
			len(childDefinitions),
		)
	}
	return outerDefinitions[0], childDefinitions[0]
}

func assertNestedDepthShape(
	t *testing.T,
	inspection *Inspection,
	outer identity.DefinitionID,
	child identity.DefinitionID,
	outerFull bool,
	childFull bool,
) {
	t.Helper()
	records := collectStage1Definitions(t, inspection)
	if records[child].site.ParentDefinition() != outer {
		t.Fatalf(
			"child parent=%s, want %s",
			records[child].site.ParentDefinition(),
			outer,
		)
	}
	for definition, full := range map[identity.DefinitionID]bool{
		outer: outerFull,
		child: childFull,
	} {
		selection, present := inspection.Selections().For(definition)
		if !present {
			t.Fatalf("definition %s has no selection", definition)
		}
		if got := selection.Depth() == contract.DepthFullSemantic; got != full {
			t.Fatalf(
				"definition %s full=%t, want %t (%s)",
				definition,
				got,
				full,
				selection.Depth(),
			)
		}
		_, hasRegion := inspection.Executable().For(definition)
		if hasRegion != full {
			t.Fatalf(
				"definition %s executable region=%t, want %t",
				definition,
				hasRegion,
				full,
			)
		}
	}
	outerRegion, outerPresent := inspection.Executable().For(outer)
	if outerFull {
		if !outerPresent ||
			len(outerRegion.References()) != 1 ||
			outerRegion.References()[0].Child() != child {
			t.Fatalf(
				"outer references=%+v present=%t, want one child reference",
				outerRegion.References(),
				outerPresent,
			)
		}
	}
	if !childFull {
		return
	}
	childRegion, present := inspection.Executable().For(child)
	if !present || len(childRegion.Members()) == 0 {
		t.Fatal("full child has no executable occurrences")
	}
	span := child.Root().Span()
	for _, member := range childRegion.Members() {
		occurrence, found := stage1Occurrence(
			inspection,
			member,
		)
		if !found {
			t.Fatalf("child member %s has no canonical payload", member)
		}
		memberSpan := occurrence.ID().Span()
		if memberSpan.Start() < span.Start() ||
			memberSpan.End() > span.End() {
			t.Errorf(
				"child occurrence %s escapes child span %s",
				member,
				span,
			)
		}
	}
}

func stage1Occurrence(
	inspection *Inspection,
	id identity.OccurrenceID,
) (structure.Occurrence, bool) {
	if occurrence, present := inspection.Structure().ResidentOccurrence(id); present {
		return occurrence, true
	}
	return inspection.Executable().AdditionalOccurrence(id)
}
