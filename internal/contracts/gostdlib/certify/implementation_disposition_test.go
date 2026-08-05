package certify

import (
	"path/filepath"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

// TestCertificationDerivesImplementationDispositions proves every
// behavior-bearing provider binding carries one mechanically certified
// implementation disposition derived from the strict checked provider
// project: a body that reaches the canonical placeholder is certified
// `placeholder`, an implemented body is certified `implemented`, and the
// certified private value-level dependencies resolve to exact symbols. A
// certified binding is linkage evidence only; the disposition is the
// behavior evidence.
func TestCertificationDerivesImplementationDispositions(t *testing.T) {
	repository := filepath.Join("..", "..", "..", "..")
	buildProfile, err := environmentcontract.NewBuildProfile(
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := Verify(Config{
		RepositoryRoot: repository,
		ProviderRoot:   filepath.Join(repository, "gostdlib"),
		ManifestPath: filepath.Join(
			repository, "gostdlib", "contract", "manifest.json",
		),
		ModuleMapPath: filepath.Join(
			repository, "gostdlib", "contract", "modules.json",
		),
		FacetMapPath: filepath.Join(
			repository, "gostdlib", "contract", "facets.json",
		),
		RuntimeContractPath: filepath.Join(
			repository, "gostdlib", "contract", "runtime.json",
		),
		TSConfigPath: filepath.Join(
			repository, "gostdlib", "tsconfig.json",
		),
		ScratchDirectory: t.TempDir(),
		GoBinary:         "go",
		BuildProfile:     buildProfile,
		Backend:          "node",
		MinimumGoVersion: "go1.26.4",
		MaximumGoVersion: "go1.26.4",
	})
	if err != nil {
		t.Fatal(err)
	}
	byIdentity := make(map[string]gostdlib.Binding)
	for _, module := range certificate.Modules() {
		for _, binding := range module.Bindings() {
			byIdentity[binding.Identity()] = binding
		}
	}

	elem, ok := byIdentity["reflect|kind=4|receiver=reflect.Value|name=Elem"]
	if !ok {
		t.Fatal("reflect.Value.Elem has no certified binding")
	}
	if elem.Disposition() != gostdlib.DispositionImplemented {
		t.Fatalf(
			"reflect.Value.Elem disposition = %v, want implemented",
			elem.Disposition(),
		)
	}

	typeAssert, ok := byIdentity["reflect|kind=4|receiver=|name=TypeAssert"]
	if !ok {
		t.Fatal("reflect.TypeAssert has no certified binding")
	}
	if typeAssert.Disposition() != gostdlib.DispositionPlaceholder {
		t.Fatalf(
			"reflect.TypeAssert disposition = %v, want placeholder",
			typeAssert.Disposition(),
		)
	}

	trimSpace, ok := byIdentity["strings|kind=4|receiver=|name=TrimSpace"]
	if !ok {
		t.Fatal("strings.TrimSpace has no certified binding")
	}
	if trimSpace.Disposition() != gostdlib.DispositionImplemented {
		t.Fatalf(
			"strings.TrimSpace disposition = %v, want implemented",
			trimSpace.Disposition(),
		)
	}

	// Every behavior-bearing callable binding carries a closed disposition;
	// type-only rows carry none.
	for identity, binding := range byIdentity {
		if binding.Kind() != gostdlib.BindingFunction {
			continue
		}
		if !binding.Disposition().Valid() {
			t.Fatalf(
				"callable binding %q has no certified disposition",
				identity,
			)
		}
	}

	// Placeholder identity derives from the checked caller/symbol evidence:
	// the placeholder body depends on the canonical placeholder symbol, and
	// implemented bodies retain their conservative private value-level
	// dependencies resolved to exact symbols.
	foundCanonical := false
	for _, dependency := range typeAssert.ImplementationDependencies() {
		if dependency == "" {
			t.Fatal("reflect.TypeAssert recorded an anonymous dependency")
		}
		if gostdlib.CanonicalPlaceholderDependency(dependency) {
			foundCanonical = true
		}
	}
	if !foundCanonical {
		t.Fatalf(
			"reflect.TypeAssert placeholder does not depend on the canonical placeholder symbol: %v",
			typeAssert.ImplementationDependencies(),
		)
	}
	for _, dependency := range elem.ImplementationDependencies() {
		if gostdlib.CanonicalPlaceholderDependency(dependency) {
			t.Fatalf(
				"implemented reflect.Value.Elem depends on the canonical placeholder: %v",
				elem.ImplementationDependencies(),
			)
		}
	}
	for _, dependency := range trimSpace.ImplementationDependencies() {
		if gostdlib.CanonicalPlaceholderDependency(dependency) {
			t.Fatalf(
				"implemented strings.TrimSpace depends on the canonical placeholder: %v",
				trimSpace.ImplementationDependencies(),
			)
		}
	}
}
