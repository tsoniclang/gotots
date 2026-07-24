package compiler

import (
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

const mixedDepthSemanticSource = `package lexical

func Outer() {
	type first int
	type score struct{ value int }
	unused := 0
	base := score{value: 1}
	for index, item := range []score{{value: 2}} {
		use := func(delta score) int {
			return base.value + delta.value + item.value + index
		}
		_, _ = unused, use
	}
}
`

func TestMixedDepthSemanticClosureUsesDirectCheckerDefinitions(
	t *testing.T,
) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/lexical\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "lexical.go", mixedDepthSemanticSource)

	base := inspectDepthFixture(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	outer, child := nestedPair(t, base)
	noneContract := writeDepthContract(t, "lexical-none@v1")
	childContract := writeDepthContract(
		t, "lexical-child@v1", child,
	)

	output := t.TempDir()
	structurePath := filepath.Join(
		output, "provider.structure.gotots",
	)
	semanticPath := filepath.Join(
		output, "provider.semantic.gotots",
	)
	provider, err := AuditCatalog(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         "lexical-none@v1",
		ProviderContractArtifact: noneContract,
	}, structurePath, semanticPath)
	if err != nil {
		t.Fatal(err)
	}
	contractOnly := inspectDepthFixture(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:          "lexical-none@v1",
		ProviderContractArtifact:  noneContract,
		ProviderStructureArtifact: structurePath,
		ProviderStructureDigest:   provider.Structure.Digest,
		ProviderSemanticArtifact:  semanticPath,
		ProviderSemanticDigest:    provider.Semantic.Digest,
	})
	assertMixedDepthSemanticClosure(
		t,
		semanticPackageByImportPath(
			t, contractOnly.Semantic(), "example.com/lexical",
		),
		outer,
		child,
		false,
	)

	childFull := inspectDepthFixture(t, source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         "lexical-child@v1",
		ProviderContractArtifact: childContract,
	})
	assertMixedDepthSemanticClosure(
		t,
		semanticPackageByImportPath(
			t, childFull.Semantic(), "example.com/lexical",
		),
		outer,
		child,
		true,
	)
}

func assertMixedDepthSemanticClosure(
	t *testing.T,
	pkg semantic.Package,
	outer identity.DefinitionID,
	child identity.DefinitionID,
	childFull bool,
) {
	t.Helper()
	var score semantic.Declaration
	for _, declaration := range semanticDeclarations(pkg) {
		switch declaration.Name() {
		case "first":
			t.Error("unused excluded-parent type entered semantic closure")
		case "score":
			score = declaration
		}
	}
	if score.ID().IsZero() ||
		score.ID().Form() != identity.SemanticDeclarationOccurrence ||
		score.ID().Ordinal() != 1 {
		t.Fatalf(
			"score declaration=%s ordinal=%d, want occurrence ordinal 1",
			score.ID(), score.ID().Ordinal(),
		)
	}

	operationsByDefinition := map[identity.DefinitionID]int{}
	resolutions := map[identity.OccurrenceID]bool{}
	for _, operation := range semanticOperations(pkg) {
		operationsByDefinition[operation.Definition()]++
	}
	for _, resolution := range semanticResolutions(pkg) {
		resolutions[resolution.Occurrence()] = true
	}
	if operationsByDefinition[outer] != 0 {
		t.Fatalf(
			"excluded outer operations=%d, want 0",
			operationsByDefinition[outer],
		)
	}

	bindings := map[string]semantic.Binding{}
	for _, binding := range semanticBindings(pkg) {
		switch binding.Name() {
		case "unused":
			t.Error("unused excluded-parent binding entered semantic closure")
		case "base", "index", "item":
			bindings[binding.Name()] = binding
		}
	}
	base := bindings["base"]
	if !childFull {
		if !base.ID().IsZero() ||
			operationsByDefinition[child] != 0 {
			t.Fatalf(
				"contract child base=%s operations=%d, want none",
				base.ID(), operationsByDefinition[child],
			)
		}
		return
	}
	if base.ID().IsZero() ||
		base.Definition() != outer ||
		base.ID().Ordinal() != 1 ||
		len(base.CapturedBy()) != 1 ||
		base.CapturedBy()[0] != child {
		t.Fatalf(
			"base=%s owner=%s ordinal=%d captures=%v",
			base.ID(), base.Definition(), base.ID().Ordinal(),
			base.CapturedBy(),
		)
	}
	for _, name := range []string{"index", "item"} {
		binding := bindings[name]
		if binding.ID().IsZero() ||
			binding.Definition() != outer ||
			binding.ID().Role() != identity.SemanticBindingRange ||
			len(binding.CapturedBy()) != 1 ||
			binding.CapturedBy()[0] != child {
			t.Fatalf(
				"%s=%s role=%s owner=%s captures=%v",
				name,
				binding.ID(),
				binding.ID().Role(),
				binding.Definition(),
				binding.CapturedBy(),
			)
		}
	}
	if operationsByDefinition[child] == 0 {
		t.Fatal("full child has no semantic operations")
	}
	if resolutions[base.Source()] {
		t.Fatalf(
			"excluded-parent source %s gained an occurrence resolution",
			base.Source(),
		)
	}
}
