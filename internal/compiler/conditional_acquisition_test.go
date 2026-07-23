package compiler

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func TestConditionalAcquisitionCoversTrueAndFalseCandidates(t *testing.T) {
	requireCgo(t)
	directory := t.TempDir()
	writeCompilerFile(
		t,
		directory,
		"go.mod",
		"module example.com/conditional\n\ngo 1.26.0\n",
	)
	writeCompilerFile(t, directory, "pure.go", `package conditional

func pure() int { return 1 }
`)
	writeCompilerFile(t, directory, "cgo.go", `package conditional

/*
static int value(void) { return 7; }
*/
import "C"

func usesC() int { return int(C.value()) }
`)
	request := source.Request{
		Dir:              directory,
		Patterns:         []string{"."},
		ProviderContract: contract.DefaultID,
		Env:              []string{"CGO_ENABLED=1"},
	}
	universe, err := source.ResolveUniverse(request)
	if err != nil {
		t.Fatal(err)
	}
	root := requestedRootPackage(t, universe)
	base, err := contract.Default()
	if err != nil {
		t.Fatal(err)
	}
	derived, err := deriveProviderPackage(
		request,
		universe,
		base,
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	definitions := definitionsByDiagnosticName(
		t,
		derived.graph,
		root,
	)
	derived.discard()
	pure := definitions["pure"]
	usesC := definitions["usesC"]
	if pure.IsZero() || usesC.IsZero() ||
		pure.File() == usesC.File() {
		t.Fatalf(
			"conditional candidates pure=%s usesC=%s",
			pure,
			usesC,
		)
	}

	rules := make([]contract.Rule, 0, len(base.Rules())+2)
	for _, rule := range base.Rules() {
		if rule.Selector() == contract.SelectorNamespace &&
			rule.Namespace() == identity.OwnerModule &&
			rule.Condition() == contract.ConditionAlways {
			replacement, err := contract.NewNamespaceRule(
				identity.OwnerModule,
				contract.ConditionAlways,
				contract.SelectionFactInvalid,
				contract.ProviderExternalObligation,
			)
			if err != nil {
				t.Fatal(err)
			}
			rules = append(rules, replacement)
			continue
		}
		rules = append(rules, rule)
	}
	for _, definition := range []identity.DefinitionID{pure, usesC} {
		rule, err := contract.NewDefinitionRule(
			definition,
			contract.ConditionFactTrue,
			contract.SelectionFactCDependent,
			contract.ProviderAutomaticTranslation,
		)
		if err != nil {
			t.Fatal(err)
		}
		rules = append(rules, rule)
	}
	selected, err := contract.New(
		"conditional-acquisition@v1",
		rules,
	)
	if err != nil {
		t.Fatal(err)
	}
	request.ProviderContract = selected.ID()
	request.ProviderContractDigest = selected.Fingerprint()
	request.ProviderContractArtifact =
		writeContractArtifact(t, selected)
	providerPath := filepath.Join(
		t.TempDir(),
		"conditional-provider.gotots",
	)
	provider, err := AuditCatalog(request, providerPath)
	if err != nil {
		t.Fatal(err)
	}
	request.AuditArtifact = providerPath
	request.AuditArtifactDigest = provider.Digest
	inspection, err := InspectConstructs(request)
	if err != nil {
		t.Fatal(err)
	}

	for _, definition := range []identity.DefinitionID{pure, usesC} {
		decision, present := inspection.SourcePlan().For(
			definition.File(),
		)
		if !present || decision.Kind() != sourceplan.KindLocalSyntax {
			t.Fatalf(
				"conditional candidate %s source decision=%+v present=%t",
				definition,
				decision,
				present,
			)
		}
	}
	assertConditionalOutcome(
		t,
		inspection,
		pure,
		false,
		contract.DepthExternalBoundary,
	)
	assertConditionalOutcome(
		t,
		inspection,
		usesC,
		true,
		contract.DepthFullSemantic,
	)
	if _, present := inspection.Executable().For(pure); present {
		t.Fatal("false conditional candidate owns an executable region")
	}
	if _, present := inspection.Executable().For(usesC); !present {
		t.Fatal("true conditional candidate lacks an executable region")
	}
}

func requireCgo(t *testing.T) {
	t.Helper()
	output, err := exec.Command("go", "env", "CGO_ENABLED").Output()
	if err != nil || strings.TrimSpace(string(output)) != "1" {
		t.Skip("cgo is unavailable")
	}
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("C compiler is unavailable")
	}
}

func requestedRootPackage(
	t *testing.T,
	universe *source.Universe,
) identity.PackageID {
	t.Helper()
	for _, pkg := range universe.Packages() {
		if pkg.RequestedRoot() {
			return pkg.ID()
		}
	}
	t.Fatal("fixture has no requested root")
	return identity.PackageID{}
}

func definitionsByDiagnosticName(
	t *testing.T,
	graph *structure.Graph,
	pkgID identity.PackageID,
) map[string]identity.DefinitionID {
	t.Helper()
	out := map[string]identity.DefinitionID{}
	if err := graph.VisitPackages(func(pkg structure.PackageGraph) error {
		if pkg.ID() != pkgID {
			return nil
		}
		for _, definition := range pkg.Definitions() {
			out[definition.Name()] = definition.ID()
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertConditionalOutcome(
	t *testing.T,
	inspection *Inspection,
	definition identity.DefinitionID,
	fact bool,
	depth contract.EvidenceDepth,
) {
	t.Helper()
	value, present := inspection.SelectionFacts().Value(
		definition,
		contract.SelectionFactCDependent,
	)
	if !present || value != fact {
		t.Fatalf(
			"definition %s C-dependent=%t present=%t, want %t",
			definition,
			value,
			present,
			fact,
		)
	}
	selection, present := inspection.Selections().For(definition)
	if !present || selection.Depth() != depth {
		t.Fatalf(
			"definition %s selection=%+v present=%t, want depth %s",
			definition,
			selection,
			present,
			depth,
		)
	}
	if fact &&
		(selection.Witness().Selector !=
			contract.SelectorExactDefinition ||
			selection.Witness().Condition !=
				contract.ConditionFactTrue) {
		t.Fatalf(
			"true conditional candidate has witness %+v",
			selection.Witness(),
		)
	}
}
