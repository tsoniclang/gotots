package environmentobligation_test

import (
	"context"
	"path/filepath"
	"slices"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

// TestSettledEnvironmentEvidenceDistinguishesDemandAndRoute proves the
// settled environment projection records one closed monotonic use demand and
// one sole implementation route per canonical declaration: a called provider
// function, a provider function taken as a value, a declaration mentioned
// only as a type, a provider callback boundary, and the compiler-intrinsic
// reflect.TypeOf route excluded from provider demand.
func TestSettledEnvironmentEvidenceDistinguishesDemandAndRoute(t *testing.T) {
	project := t.TempDir()
	writeProgramFile(
		t,
		filepath.Join(project, "go.mod"),
		"module example.com/settleduse\n\ngo 1.26.4\n",
	)
	writeProgramFile(t, filepath.Join(project, "source.go"), `package settleduse

import (
	"reflect"
	"sort"
	"strings"
)

type Payload struct{ Name string }

func Trim(input string) string {
	return strings.TrimSpace(input)
}

func Transform() func(string) string {
	return strings.ToUpper
}

func Describe(reader *strings.Reader) bool {
	return reader == nil
}

func Position(values []int, target int) int {
	return sort.Search(len(values), func(index int) bool {
		return values[index] >= target
	})
}

func Kind(payload Payload) reflect.Kind {
	return reflect.TypeOf(payload).Kind()
}
`)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    project,
		Pattern:      ".",
		BuildProfile: linkedProviderBuildProfile(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	roots := make([]emit.Root, 0, 5)
	for _, name := range []string{
		"Trim",
		"Transform",
		"Describe",
		"Position",
		"Kind",
	} {
		root, rootErr := emit.NewRoot(scope.Lookup(name))
		if rootErr != nil {
			t.Fatal(rootErr)
		}
		roots = append(roots, root)
	}
	options := emit.DefaultOptions()
	options.StandardLibrary = linkedProviderCertificate(t)
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}

	profile := emission.EnvironmentProfile()
	if profile.ToolchainVersion() != "go1.26.4" ||
		profile.GOOS() != "linux" ||
		profile.GOARCH() != "amd64" ||
		profile.CgoEnabled() ||
		!slices.Equal(profile.Tags(), []string{"noasm"}) {
		t.Fatalf(
			"environment profile build facts = %q %q/%q cgo=%v tags=%v",
			profile.ToolchainVersion(),
			profile.GOOS(),
			profile.GOARCH(),
			profile.CgoEnabled(),
			profile.Tags(),
		)
	}
	providerDigest, providerLinked := profile.ProviderManifestDigest()
	if !providerLinked || len(providerDigest) != 64 {
		t.Fatalf(
			"provider manifest digest = %q linked=%v",
			providerDigest,
			providerLinked,
		)
	}
	if _, externalLinked := profile.ExternalManifestDigest(); externalLinked {
		t.Fatal("unlinked external provider reported a linked digest")
	}
	if len(profile.BuildFingerprint()) != 64 ||
		len(profile.SchemaContractDigest()) != 64 ||
		profile.SchemaRevision() == "" {
		t.Fatalf(
			"profile fingerprints = %q/%q/%q",
			profile.BuildFingerprint(),
			profile.SchemaContractDigest(),
			profile.SchemaRevision(),
		)
	}
	if profile.PinnedToolVersion() == "" || profile.ProtocolVersion() <= 0 {
		t.Fatalf(
			"pinned target identity = %q protocol %d",
			profile.PinnedToolVersion(),
			profile.ProtocolVersion(),
		)
	}
	if profile.IntegerRepresentation() != emit.IntegerRepresentationNumber {
		t.Fatalf(
			"environment profile integer representation = %v",
			profile.IntegerRepresentation(),
		)
	}

	byName := make(map[string]*emit.EnvironmentObligation)
	for index := range emission.EnvironmentObligations() {
		obligation := &emission.EnvironmentObligations()[index]
		if obligation.Receiver() != "" {
			continue
		}
		key := obligation.PackagePath() + "." + obligation.Name()
		if existing, ok := byName[key]; ok &&
			existing.Kind() == obligation.Kind() {
			t.Fatalf("duplicate settled declaration row %q", key)
		}
		if _, ok := byName[key]; !ok {
			byName[key] = obligation
		}
	}

	trimSpace := requireRow(t, byName, "strings.TrimSpace")
	if !hasDemand(trimSpace, environmentcontract.UseDemandCallable) {
		t.Fatalf(
			"strings.TrimSpace demands = %v, want callable",
			trimSpace.Demands(),
		)
	}
	if trimSpace.Route() != environmentcontract.RouteProvider {
		t.Fatalf(
			"strings.TrimSpace route = %v, want provider",
			trimSpace.Route(),
		)
	}

	toUpper := requireRow(t, byName, "strings.ToUpper")
	if !hasDemand(toUpper, environmentcontract.UseDemandCallable) {
		t.Fatalf(
			"value-taken strings.ToUpper demands = %v, want callable",
			toUpper.Demands(),
		)
	}
	if toUpper.Route() != environmentcontract.RouteProvider {
		t.Fatalf("strings.ToUpper route = %v, want provider", toUpper.Route())
	}

	reader := requireRow(t, byName, "strings.Reader")
	if !hasDemand(reader, environmentcontract.UseDemandTypeContract) {
		t.Fatalf(
			"type-only strings.Reader demands = %v, want type contract",
			reader.Demands(),
		)
	}
	if hasDemand(reader, environmentcontract.UseDemandCallable) ||
		hasDemand(reader, environmentcontract.UseDemandValue) {
		t.Fatalf(
			"type-only strings.Reader gained executable demand: %v",
			reader.Demands(),
		)
	}

	search := requireRow(t, byName, "sort.Search")
	if !hasDemand(search, environmentcontract.UseDemandCallable) {
		t.Fatalf("sort.Search demands = %v, want callable", search.Demands())
	}
	if search.Route() != environmentcontract.RouteProvider {
		t.Fatalf("sort.Search route = %v, want provider", search.Route())
	}
	if selections := search.ProviderSelections(); len(selections) != 0 {
		t.Fatalf(
			"sort.Search provider selections = %v, want direct synchronous binding",
			selections,
		)
	}
	demands := trimSpace.Demands()
	if len(demands) != 0 {
		demands[0] = environmentcontract.UseDemandInvalid
		if trimSpace.Demands()[0] == environmentcontract.UseDemandInvalid {
			t.Fatal("demands exposed mutable backing storage")
		}
	}

	// The settled evidence is the used set, not the provider catalog: a
	// certified binding that no emitted declaration references must not
	// appear as a settled row.
	for _, unused := range []string{
		"strings.NewReplacer",
		"strings.Title",
		"sort.Stable",
	} {
		if _, ok := byName[unused]; ok {
			t.Fatalf(
				"unreferenced catalog binding %q entered the settled evidence",
				unused,
			)
		}
	}

	typeOf := requireRow(t, byName, "reflect.TypeOf")
	if typeOf.Route() != environmentcontract.RouteGeneratedFacet {
		t.Fatalf(
			"reflect.TypeOf route = %v, want sole generated-facet route",
			typeOf.Route(),
		)
	}
	if !hasDemand(typeOf, environmentcontract.UseDemandCallable) {
		t.Fatalf("reflect.TypeOf demands = %v, want callable", typeOf.Demands())
	}
	if typeOf.TargetFingerprint() != "" || typeOf.TargetName() != "" {
		t.Fatalf(
			"generated-facet reflect.TypeOf row carries provider target %q/%q",
			typeOf.TargetName(),
			typeOf.TargetFingerprint(),
		)
	}
}

func requireRow(
	t *testing.T,
	rows map[string]*emit.EnvironmentObligation,
	key string,
) *emit.EnvironmentObligation {
	t.Helper()
	row, ok := rows[key]
	if !ok {
		t.Fatalf("settled environment evidence has no row for %q", key)
	}
	return row
}

func hasDemand(
	row *emit.EnvironmentObligation,
	demand environmentcontract.UseDemand,
) bool {
	return slices.Contains(row.Demands(), demand)
}
