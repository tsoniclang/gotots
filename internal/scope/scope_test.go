package scope_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

func loadFixture(t *testing.T, contract scope.ProviderContract, files map[string]string) *source.Universe {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	policy, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	universe, err := source.LoadUniverse(source.Request{Dir: dir}, policy, source.UnitManifest{})
	if err != nil {
		t.Fatalf("LoadUniverse: %v", err)
	}
	return universe
}

var witnessFixture = map[string]string{
	"go.mod":  "module witness.example/m\n\ngo 1.26\n",
	"main.go": "package m\n\nimport \"fmt\"\n\nfunc F() { fmt.Println(1) }\n\nfunc Bodyless() int\n",
	"asm.s":   "// stub\n",
}

// TestRequestSelectedContractAndWitnesses proves contract selection is a
// request fact with exact rule-identity witnesses: empty selection fails
// typed, digest mismatch fails, and every unit selection records the exact
// rule that bound it.
func TestRequestSelectedContractAndWitnesses(t *testing.T) {
	if _, err := scope.ResolveContract("", ""); err == nil {
		t.Error("empty contract selection resolved")
	}
	if _, err := scope.ResolveContract("nonexistent@v9", ""); err == nil {
		t.Error("unknown contract resolved")
	}
	contract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		t.Fatalf("default contract: %v", err)
	}
	if _, err := scope.ResolveContract(scope.DefaultContractID, "deadbeef"); err == nil {
		t.Error("digest mismatch resolved")
	}
	if _, err := scope.ResolveContract(scope.DefaultContractID, contract.Fingerprint()); err != nil {
		t.Errorf("matching digest rejected: %v", err)
	}
	universe := loadFixture(t, contract, witnessFixture)
	selection, err := scope.Select(universe, contract)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if selection.ContractID() != scope.DefaultContractID || selection.ContractFingerprint() != contract.Fingerprint() {
		t.Error("selection does not bind its contract identity/fingerprint")
	}
	witnesses := map[string]scope.BindingWitness{}
	for _, unit := range selection.Units() {
		if unit.Witness.RuleID == "" || !unit.Witness.Selector.Valid() || !unit.Witness.Condition.Valid() {
			t.Errorf("unit %s has no exact rule witness", unit.Unit)
		}
		witnesses[unit.Unit.Kind().String()+"|"+unit.Provider.String()] = unit.Witness
	}
	// F: module namespace always-rule -> automatic. Bodyless: the
	// evidence-conditioned namespace rule wins the tier.
	if w := witnesses["func-body|automatic-translation"]; w.RuleID != "namespace:module|always->automatic-translation" {
		t.Errorf("func body witness = %+v, want the module namespace always-rule", w)
	}
	if w := witnesses["bodyless-decl|external-obligation"]; w.RuleID != "namespace:module|bodyless->external-obligation" {
		t.Errorf("bodyless witness = %+v, want the module bodyless rule", w)
	}
	stdSeen := false
	for _, unit := range selection.Units() {
		if unit.Provider == scope.ProviderGostdlib {
			stdSeen = true
		}
	}
	if !stdSeen {
		t.Error("no gostdlib-bound std units in closure")
	}
	// Implicit units are first-class ledger rows with witnesses and depths.
	if len(selection.ImplicitUnits()) == 0 {
		t.Fatal("selection carries no implicit units")
	}
	for _, implicit := range selection.ImplicitUnits() {
		if implicit.Witness.RuleID == "" || !implicit.Depth.Valid() {
			t.Errorf("implicit unit %s lacks witness or depth", implicit.Unit)
		}
	}
}

// TestContractsAreDataNotProvenance proves different contracts assign
// different providers and depths to units of identical provenance, and that
// exact-package and exact-unit rules take precedence over namespace rules
// independent of declaration order.
func TestContractsAreDataNotProvenance(t *testing.T) {
	defaultContract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		t.Fatal(err)
	}
	universe := loadFixture(t, defaultContract, witnessFixture)
	baseline, err := scope.Select(universe, defaultContract)
	if err != nil {
		t.Fatal(err)
	}
	var moduleUnit identity.SourceUnitID
	for _, unit := range baseline.Units() {
		if unit.Provider == scope.ProviderAutomaticTranslation && unit.Unit.Kind() == identity.UnitFuncBody {
			moduleUnit = unit.Unit
		}
	}
	if moduleUnit.IsZero() {
		t.Fatal("no automatic module unit in baseline")
	}
	// A contract binding the module namespace to gostdlib plus one exact-unit
	// override back to automatic: same provenance, different outcome; the
	// exact rule wins over its namespace regardless of rule order.
	mustRule := func(r scope.ContractRule, err error) scope.ContractRule {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	base := []scope.ContractRule{
		mustRule(scope.NewNamespaceRule(identity.OwnerModule, scope.ConditionAlways, scope.ProviderGostdlib)),
		mustRule(scope.NewNamespaceRule(identity.OwnerStandardLibrary, scope.ConditionAlways, scope.ProviderGostdlib)),
		mustRule(scope.NewNamespaceRule(identity.OwnerToolchain, scope.ConditionAlways, scope.ProviderToolchainSource)),
		mustRule(scope.NewNamespaceRule(identity.OwnerLanguagePseudo, scope.ConditionAlways, scope.ProviderLanguageIntrinsic)),
	}
	exact := mustRule(scope.NewExactUnitRule(moduleUnit.String(), scope.ConditionAlways, scope.ProviderAutomaticTranslation))
	for _, order := range [][]scope.ContractRule{
		append(append([]scope.ContractRule(nil), base...), exact),
		append([]scope.ContractRule{exact}, base...),
	} {
		custom, err := scope.NewProviderContract("custom@v1", order)
		if err != nil {
			t.Fatal(err)
		}
		if custom.Fingerprint() == defaultContract.Fingerprint() {
			t.Error("distinct contracts share a fingerprint")
		}
		selection, err := scope.Select(universe, custom)
		if err != nil {
			t.Fatalf("custom selection: %v", err)
		}
		for _, unit := range selection.Units() {
			owner := unit.Unit.Span().File().Owner().Class()
			switch {
			case unit.Unit == moduleUnit:
				if unit.Provider != scope.ProviderAutomaticTranslation || unit.Witness.Selector != scope.SelectorExactUnit {
					t.Errorf("exact-unit rule lost: %+v", unit)
				}
			case owner == identity.OwnerModule:
				if unit.Provider != scope.ProviderGostdlib {
					t.Errorf("module unit %s provider %s under custom contract", unit.Unit, unit.Provider)
				}
			}
		}
	}
}

// TestAmbiguityAndStaleRulesFailClosed proves same-tier provider disagreement
// is a typed ambiguity failure, duplicate match targets are rejected
// statically, and an exact rule binding nothing in the universe fails
// selection.
func TestAmbiguityAndStaleRulesFailClosed(t *testing.T) {
	mustRule := func(r scope.ContractRule, err error) scope.ContractRule {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// Static: two rules sharing selector+target+condition.
	_, err := scope.NewProviderContract("dup@v1", []scope.ContractRule{
		mustRule(scope.NewNamespaceRule(identity.OwnerModule, scope.ConditionAlways, scope.ProviderGostdlib)),
		mustRule(scope.NewNamespaceRule(identity.OwnerModule, scope.ConditionAlways, scope.ProviderAutomaticTranslation)),
	})
	if err == nil {
		t.Error("duplicate match target accepted")
	}
	// Runtime: a bodyless C-dependent unit matched by two same-tier
	// conditioned rules that disagree.
	ambiguous, err := scope.NewProviderContract("ambig@v1", []scope.ContractRule{
		mustRule(scope.NewNamespaceRule(identity.OwnerModule, scope.ConditionCDependent, scope.ProviderExternalObligation)),
		mustRule(scope.NewNamespaceRule(identity.OwnerModule, scope.ConditionBodyless, scope.ProviderGostdlib)),
	})
	if err != nil {
		t.Fatal(err)
	}
	q := scope.UnitQuery{
		Unit: mustUnit(t), Package: mustPackage(t), OwnerClass: identity.OwnerModule,
		Disposition: source.DispositionOrdinarySource, Kind: identity.UnitBodylessDecl, CDependent: true,
	}
	if _, _, err := ambiguous.Bind(q); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("same-tier disagreement did not fail as ambiguity: %v", err)
	}
	// Runtime: an exact rule naming a unit outside the universe.
	defaultContract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		t.Fatal(err)
	}
	universe := loadFixture(t, defaultContract, witnessFixture)
	stale := mustRule(scope.NewExactUnitRule("mod=ghost.example::ghost.example#0-1/func-body", scope.ConditionAlways, scope.ProviderGostdlib))
	staleContract, err := scope.NewProviderContract("stale@v1", append(defaultRules(t), stale))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Select(universe, staleContract); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Errorf("stale exact rule did not fail selection: %v", err)
	}
}

func defaultRules(t *testing.T) []scope.ContractRule {
	t.Helper()
	contract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		t.Fatal(err)
	}
	return contract.Rules()
}

func mustUnit(t *testing.T) identity.SourceUnitID {
	t.Helper()
	pkg := mustPackage(t)
	file, err := identity.NewFileID(pkg.Owner(), "main.go")
	if err != nil {
		t.Fatal(err)
	}
	span, err := identity.NewSpanID(file, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := identity.NewSourceUnitID(span, identity.UnitBodylessDecl)
	if err != nil {
		t.Fatal(err)
	}
	return unit
}

func mustPackage(t *testing.T) identity.PackageID {
	t.Helper()
	module, err := identity.NewModuleID("witness.example/m", "")
	if err != nil {
		t.Fatal(err)
	}
	owner, err := identity.NewModuleOwner(module)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := identity.NewPackageID(owner, "witness.example/m")
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

// TestContractArtifactFile proves a request can select a versioned contract
// ARTIFACT by path: the file decodes through the validating constructors, its
// fingerprint binds, and unknown spellings fail closed.
func TestContractArtifactFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "contract.json")
	artifact := `{
 "id": "filecontract@v1",
 "version": 2,
 "rules": [
  {"bind": "namespace", "namespace": "module", "condition": "always", "provider": "automatic-translation"},
  {"bind": "namespace", "namespace": "standard-library", "condition": "always", "provider": "gostdlib"},
  {"bind": "package", "package": "std::errors", "condition": "always", "provider": "toolchain-source"}
 ]
}`
	if err := os.WriteFile(path, []byte(artifact), 0o644); err != nil {
		t.Fatal(err)
	}
	contract, err := scope.ResolveContract(path, "")
	if err != nil {
		t.Fatalf("artifact contract: %v", err)
	}
	if contract.ID() != "filecontract@v1" || len(contract.Rules()) != 3 {
		t.Errorf("artifact decoded to %s with %d rules", contract.ID(), len(contract.Rules()))
	}
	if _, err := scope.ResolveContract(path, contract.Fingerprint()); err != nil {
		t.Errorf("matching artifact digest rejected: %v", err)
	}
	if _, err := scope.ResolveContract(path, "beef"); err == nil {
		t.Error("artifact digest mismatch resolved")
	}
	bad := strings.Replace(artifact, "gostdlib", "stdlib-magic", 1)
	badPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.ResolveContract(badPath, ""); err == nil {
		t.Error("unknown provider spelling decoded")
	}
}

// TestAcquisitionPolicyFromContract proves acquisition is contract-derived
// data: automatic namespaces census recursively, provider namespaces consume
// the manifest, exact-package rules override, the audit policy is recursive
// everywhere, and an uncovered class fails closed.
func TestAcquisitionPolicyFromContract(t *testing.T) {
	contract, err := scope.ResolveContract(scope.DefaultContractID, "")
	if err != nil {
		t.Fatal(err)
	}
	ordinary, err := contract.AcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	modPkg := mustPackage(t)
	stdOwner := identity.StandardLibraryOwner()
	stdPkg, err := identity.NewPackageID(stdOwner, "errors")
	if err != nil {
		t.Fatal(err)
	}
	if mode, err := ordinary.ModeFor(modPkg); err != nil || mode != source.CensusRecursive {
		t.Errorf("module mode = %v/%v, want recursive", mode, err)
	}
	if mode, err := ordinary.ModeFor(stdPkg); err != nil || mode != source.CensusManifest {
		t.Errorf("std mode = %v/%v, want manifest", mode, err)
	}
	audit, err := contract.AuditAcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	if mode, err := audit.ModeFor(stdPkg); err != nil || mode != source.CensusRecursive {
		t.Errorf("audit std mode = %v/%v, want recursive", mode, err)
	}
	partial, err := scope.NewProviderContract("partial@v1", defaultRules(t)[:1])
	if err != nil {
		t.Fatal(err)
	}
	partialPolicy, err := partial.AcquisitionPolicy()
	if err != nil {
		t.Fatal(err)
	}
	uncoveredFailed := false
	if _, err := partialPolicy.ModeFor(stdPkg); err != nil {
		uncoveredFailed = true
	}
	if _, err := partialPolicy.ModeFor(modPkg); err != nil {
		uncoveredFailed = true
	}
	if !uncoveredFailed {
		t.Error("policy covered a class its contract never declared")
	}
}
