package compiler

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestExactRuleTargetMustExistEvenWhenRuleDoesNotWin(t *testing.T) {
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/exact-rule\n\ngo 1.26.0\n",
	)
	writeCompilerFile(
		t,
		directory,
		"main.go",
		"package exactrule\n\nfunc Main() int { return 1 }\n",
	)
	baseInspection, err := InspectConstructs(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract: contract.DefaultID,
	})
	if err != nil {
		t.Fatal(err)
	}
	definitions := collectStage1Definitions(t, baseInspection)
	mainDefinitions := findDefinitionsByName(definitions, "Main")
	if len(mainDefinitions) != 1 {
		t.Fatalf("Main definitions = %v", mainDefinitions)
	}
	mainDefinition := mainDefinitions[0]
	span, err := identity.NewSpanID(
		mainDefinition.File(),
		mainDefinition.Root().Span().Start()+1,
		mainDefinition.Root().Span().End(),
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := identity.NewOccurrenceID(
		span,
		mainDefinition.Root().KindID(),
	)
	if err != nil {
		t.Fatal(err)
	}
	missing, err := identity.NewSourceDefinitionID(
		root,
		mainDefinition.Kind(),
	)
	if err != nil {
		t.Fatal(err)
	}
	base, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	rule, err := contract.NewDefinitionRule(
		missing,
		contract.ConditionFactTrue,
		contract.SelectionFactCDependent,
		contract.ProviderExternalObligation,
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := contract.New(
		"missing-exact-target@v1",
		append(base.Rules(), rule),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = InspectConstructs(source.Request{
		Dir: directory, Patterns: []string{"."},
		ProviderContract:         selected.ID(),
		ProviderContractDigest:   selected.Fingerprint(),
		ProviderContractArtifact: writeContractArtifact(t, selected),
	})
	if err == nil ||
		!strings.Contains(
			err.Error(),
			"exact-rule target join failed",
		) ||
		!strings.Contains(err.Error(), missing.String()) {
		t.Fatalf("missing exact-rule target error = %v", err)
	}
}
