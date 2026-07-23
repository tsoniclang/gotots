package contract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
)

func TestDefaultContractBindsEveryOwnerAndSemanticException(t *testing.T) {
	selected, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	modulePackage := testPackage(t, testModuleOwner(t), "example.com/app")
	moduleDefinition := testDefinition(
		t, modulePackage, identity.DefinitionFuncDecl,
	)
	tests := []struct {
		name       string
		definition identity.DefinitionID
		pkg        identity.PackageID
		intrinsic  bool
		cDependent bool
		want       Provider
	}{
		{
			name: "module source", definition: moduleDefinition,
			pkg: modulePackage, want: ProviderAutomaticTranslation,
		},
		{
			name: "C-dependent module source", definition: moduleDefinition,
			pkg: modulePackage, cDependent: true,
			want: ProviderExternalObligation,
		},
		{
			name: "bodyless module source",
			definition: testDefinition(
				t, modulePackage, identity.DefinitionBodylessDecl,
			),
			pkg: modulePackage, want: ProviderExternalObligation,
		},
		{
			name:       "module synthetic",
			definition: testSynthetic(t, modulePackage),
			pkg:        modulePackage, want: ProviderExternalObligation,
		},
		{
			name: "standard library",
			definition: testDefinition(
				t,
				testPackage(t, identity.StandardLibraryOwner(), "fmt"),
				identity.DefinitionFuncDecl,
			),
			pkg:  testPackage(t, identity.StandardLibraryOwner(), "fmt"),
			want: ProviderGostdlib,
		},
		{
			name: "unsafe intrinsic",
			definition: testDefinition(
				t,
				testPackage(t, identity.StandardLibraryOwner(), "unsafe"),
				identity.DefinitionFuncDecl,
			),
			pkg:       testPackage(t, identity.StandardLibraryOwner(), "unsafe"),
			intrinsic: true, want: ProviderLanguageIntrinsic,
		},
		{
			name: "toolchain source",
			definition: testDefinition(
				t,
				testPackage(t, identity.ToolchainOwner(), "cmd/compile"),
				identity.DefinitionFuncDecl,
			),
			pkg:  testPackage(t, identity.ToolchainOwner(), "cmd/compile"),
			want: ProviderToolchainSource,
		},
		{
			name: "language pseudo",
			definition: testImplicit(
				t, testPackage(t, identity.LanguagePseudoOwner(), "builtin"),
			),
			pkg:  testPackage(t, identity.LanguagePseudoOwner(), "builtin"),
			want: ProviderLanguageIntrinsic,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			facts := exactFacts(
				selected, test.definition, test.pkg, test.cDependent,
			)
			got, witness, err := selected.Bind(Query{
				Definition: test.definition,
				Package:    test.pkg,
				Intrinsic:  test.intrinsic,
				Facts:      facts,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want || witness.RuleID == "" ||
				!witness.Selector.Valid() ||
				!witness.Condition.Valid() {
				t.Fatalf("binding = %s, %+v; want %s with complete witness", got, witness, test.want)
			}
		})
	}
}

func TestBindFailsClosedOnIncompleteOrIncoherentQuery(t *testing.T) {
	selected, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	pkg := testPackage(t, testModuleOwner(t), "example.com/app")
	definition := testDefinition(t, pkg, identity.DefinitionFuncDecl)
	other := testPackage(t, identity.StandardLibraryOwner(), "fmt")
	tests := []Query{
		{Definition: definition, Package: pkg, Facts: nil},
		{
			Definition: definition, Package: pkg,
			Facts: map[SelectionFactKind]bool{
				SelectionFactCDependent: false,
				SelectionFactKind(99):   false,
			},
		},
		{
			Definition: definition, Package: other,
			Facts: exactFacts(selected, definition, other, false),
		},
		{
			Definition: identity.DefinitionID{}, Package: pkg,
			Facts: map[SelectionFactKind]bool{},
		},
	}
	for index, query := range tests {
		if _, _, err := selected.Bind(query); err == nil {
			t.Errorf("invalid query %d was admitted", index)
		}
	}
}

func TestBindUsesSpecificityAndRejectsSameTierDisagreement(t *testing.T) {
	pkg := testPackage(t, testModuleOwner(t), "example.com/app")
	definition := testDefinition(t, pkg, identity.DefinitionFuncDecl)
	namespace, err := NewNamespaceRule(
		identity.OwnerModule,
		ConditionAlways,
		SelectionFactInvalid,
		ProviderGostdlib,
	)
	if err != nil {
		t.Fatal(err)
	}
	exactPackage, err := NewPackageRule(
		pkg,
		ConditionAlways,
		SelectionFactInvalid,
		ProviderAutomaticTranslation,
	)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := New("specificity@v1", []Rule{namespace, exactPackage})
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := selected.Bind(Query{
		Definition: definition, Package: pkg,
		Facts: map[SelectionFactKind]bool{},
	})
	if err != nil || got != ProviderAutomaticTranslation {
		t.Fatalf("specific binding = %s, %v", got, err)
	}

	bodyless, err := NewPackageRule(
		pkg,
		ConditionBodyless,
		SelectionFactInvalid,
		ProviderExternalObligation,
	)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := NewPackageRule(
		pkg,
		ConditionFactTrue,
		SelectionFactCDependent,
		ProviderAutomaticTranslation,
	)
	if err != nil {
		t.Fatal(err)
	}
	ambiguous, err := New("ambiguous@v1", []Rule{bodyless, fact})
	if err != nil {
		t.Fatal(err)
	}
	bodylessDefinition := testDefinition(
		t, pkg, identity.DefinitionBodylessDecl,
	)
	if _, _, err := ambiguous.Bind(Query{
		Definition: bodylessDefinition,
		Package:    pkg,
		Facts: map[SelectionFactKind]bool{
			SelectionFactCDependent: true,
		},
	}); err == nil || !strings.Contains(err.Error(), "same-tier") {
		t.Fatalf("ambiguous binding error = %v", err)
	}
}

func TestContractArtifactIsStrictAndIdentityBound(t *testing.T) {
	dir := t.TempDir()
	valid := `{
		"id":"custom@v1",
		"version":1,
		"rules":[{
			"bind":"namespace",
			"namespace":"module",
			"condition":"always",
			"provider":"automatic-translation"
		}]
	}`
	path := filepath.Join(dir, "contract.json")
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	selected, err := Resolve("custom@v1", "", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("other@v1", "", path); err == nil {
		t.Fatal("request/artifact identity disagreement was admitted")
	}
	if _, err := Resolve("custom@v1", strings.Repeat("0", 64), path); err == nil {
		t.Fatal("contract digest disagreement was admitted")
	}
	if _, err := Resolve("custom@v1", selected.Fingerprint(), path); err != nil {
		t.Fatalf("exact contract digest rejected: %v", err)
	}

	invalid := map[string]string{
		"unknown field": strings.Replace(
			valid, `"version":1`, `"version":1,"unknown":true`, 1,
		),
		"selector field overlap": strings.Replace(
			valid, `"namespace":"module"`,
			`"namespace":"module","package":"mod=example.com/app::example.com/app"`,
			1,
		),
		"trailing value": valid + `{}`,
	}
	for name, content := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Resolve("custom@v1", "", path); err == nil {
				t.Fatal("invalid artifact was admitted")
			}
		})
	}
}

func TestContractCollectionsAreIsolated(t *testing.T) {
	selected, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	rules := selected.Rules()
	original := rules[0]
	rules[0] = Rule{}
	if selected.Rules()[0].ID() != original.ID() {
		t.Fatal("Contract.Rules exposes backing storage")
	}
}

func exactFacts(
	selected Contract,
	definition identity.DefinitionID,
	pkg identity.PackageID,
	cDependent bool,
) map[SelectionFactKind]bool {
	out := map[SelectionFactKind]bool{}
	for _, kind := range selected.RequestedFacts(definition, pkg) {
		switch kind {
		case SelectionFactCDependent:
			out[kind] = cDependent
		}
	}
	return out
}

func testModuleOwner(t *testing.T) identity.Owner {
	t.Helper()
	module, err := identity.NewModuleID("example.com/app", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	return owner
}

func testPackage(
	t *testing.T,
	owner identity.Owner,
	path string,
) identity.PackageID {
	t.Helper()
	pkg, err := identity.NewPackageID(owner, path)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func testDefinition(
	t *testing.T,
	pkg identity.PackageID,
	kind identity.DefinitionKind,
) identity.DefinitionID {
	t.Helper()
	file, err := identity.NewFileID(pkg.Owner(), "fixture.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := identity.NewSpanID(file, int(kind), int(kind)+1)
	if err != nil {
		t.Fatal(err)
	}
	occurrence, err := identity.NewOccurrenceID(span, 1)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := identity.NewSourceDefinitionID(occurrence, kind)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func testSynthetic(
	t *testing.T,
	pkg identity.PackageID,
) identity.DefinitionID {
	t.Helper()
	definition, err := identity.NewSyntheticDefinitionID(
		pkg, identity.SyntheticDefinitionAdapter, "fixture",
	)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func testImplicit(
	t *testing.T,
	pkg identity.PackageID,
) identity.DefinitionID {
	t.Helper()
	definition, err := identity.NewImplicitDefinitionID(
		pkg, identity.ImplicitDefinitionPackageInit,
	)
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
