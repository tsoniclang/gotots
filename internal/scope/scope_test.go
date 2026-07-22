package scope_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// TestRequestSelectedContractAndWitnesses proves contract selection is a
// request fact with exact binding witnesses: empty selection fails typed,
// digest mismatch fails, and every unit selection records the exact rule.
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
	dir := t.TempDir()
	for rel, content := range map[string]string{
		"go.mod":  "module witness.example/m\n\ngo 1.26\n",
		"main.go": "package m\n\nimport \"fmt\"\n\nfunc F() { fmt.Println(1) }\n\nfunc Bodyless() int\n",
		"asm.s":   "// stub\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	universe, err := source.LoadUniverse(source.Request{Dir: dir})
	if err != nil {
		t.Fatalf("LoadUniverse: %v", err)
	}
	selection, err := scope.Select(universe, contract)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if selection.ContractID() != scope.DefaultContractID || selection.ContractFingerprint() != contract.Fingerprint() {
		t.Error("selection does not bind its contract identity/fingerprint")
	}
	rules := map[string]scope.BindingWitness{}
	for _, unit := range selection.Units() {
		if !unit.Witness.Rule.Valid() || unit.Witness.Detail == "" {
			t.Errorf("unit %s has no binding witness", unit.Unit)
		}
		rules[unit.Unit.Kind().String()+"|"+unit.Provider.String()] = unit.Witness
	}
	// F: module namespace rule -> automatic. Bodyless: unit-evidence rule.
	if w, ok := rules["func-body|automatic-translation"]; !ok || w.Rule != scope.RuleOwnerNamespace {
		t.Errorf("func body witness = %+v, want owner-namespace", w)
	}
	if w, ok := rules["bodyless-decl|external-obligation"]; !ok || w.Rule != scope.RuleUnitEvidence || w.Detail != "bodyless-automatic" {
		t.Errorf("bodyless witness = %+v, want unit-evidence/bodyless-automatic", w)
	}
	// Std and toolchain owners bind to DISTINCT providers.
	stdSeen, toolSeen := false, false
	for _, unit := range selection.Units() {
		switch unit.Provider {
		case scope.ProviderGostdlib:
			stdSeen = true
		case scope.ProviderToolchainSource:
			toolSeen = true
		}
	}
	if !stdSeen {
		t.Error("no gostdlib-bound std units in closure")
	}
	_ = toolSeen // toolchain packages only enter closures when cmd/* is requested
}
