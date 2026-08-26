package certify

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestPortableSortHasOneBoundedSynchronousWorkPath(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"src",
		"internal",
		"portable",
		"slices",
		"sort.ts",
	))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	start := strings.Index(text, "export function sortValues")
	if start < 0 {
		t.Fatal("portable sort owner is absent")
	}
	body := text[start:]
	for _, required := range []string{
		"for (let width = 1; width < values.length; width *= 2)",
		"compare(leftValue, rightValue)",
		"[source, target] = [target, source]",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("portable sort lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"async ",
		"await ",
		"callComparison(",
		"[Symbol.iterator]",
		".slice(",
		".sort(",
		"instanceof Promise",
		"Promise.resolve",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portable sort contains %q", forbidden)
		}
	}
	if strings.Count(body, "sortValues") != 1 {
		t.Fatal("portable sort recurses or has duplicate owners")
	}
}

func TestCallbackKernelDenominatorIsClosed(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"contract",
		"manifest.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{
		"maps|kind=4|receiver=|name=EqualFunc",
		"slices|kind=4|receiver=|name=BinarySearchFunc",
		"slices|kind=4|receiver=|name=CompactFunc",
		"slices|kind=4|receiver=|name=CompareFunc",
		"slices|kind=4|receiver=|name=ContainsFunc",
		"slices|kind=4|receiver=|name=DeleteFunc",
		"slices|kind=4|receiver=|name=EqualFunc",
		"slices|kind=4|receiver=|name=IndexFunc",
		"slices|kind=4|receiver=|name=SortFunc",
		"slices|kind=4|receiver=|name=SortStableFunc",
		"slices|kind=4|receiver=|name=SortedFunc",
	}
	actual := make([]string, 0, len(expected))
	for _, module := range manifest.FacetModules() {
		for _, facet := range module.Facets() {
			if facet.Kind() != gostdlib.FacetGenericCallableKernel ||
				!slices.Equal(facet.Capabilities(), []gostdlib.FacetCapability{
					gostdlib.FacetCapabilityKernel,
				}) || len(facet.CallableParameters()) == 0 {
				continue
			}
			actual = append(actual, facet.SourceIdentity())
		}
	}
	slices.Sort(actual)
	if !slices.Equal(actual, expected) {
		t.Fatalf("callback kernel denominator = %#v", actual)
	}
}

func TestProviderDefinedCallableEffectDenominatorIsClosed(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(
		repository,
		"gostdlib",
		"contract",
		"manifest.json",
	))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := gostdlib.Parse(payload)
	if err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{
		"context|kind=2|receiver=|name=CancelCauseFunc",
		"context|kind=2|receiver=|name=CancelFunc",
		"io/fs|kind=2|receiver=|name=WalkDirFunc",
	} {
		binding, ok := manifest.Binding(identity)
		if !ok || binding.DefinedValueRepresentation() !=
			gostdlib.DefinedValueRepresentationCanonical ||
			binding.Effect() != gostdlib.EffectSynchronous {
			t.Fatalf("canonical callable %s = %#v, %t", identity, binding, ok)
		}
	}
	for _, identity := range []string{
		"iter|kind=2|receiver=|name=Seq",
		"iter|kind=2|receiver=|name=Seq2",
	} {
		binding, ok := manifest.Binding(identity)
		if !ok || binding.DefinedValueRepresentation() !=
			gostdlib.DefinedValueRepresentationOperations ||
			binding.Effect() != gostdlib.EffectInvalid {
			t.Fatalf("operation callable %s = %#v, %t", identity, binding, ok)
		}
		facet, ok := manifest.Facet(
			identity,
			gostdlib.FacetDefinedValueOperations,
			gostdlib.FacetCapabilityProject,
		)
		if !ok || facet.Effect() != gostdlib.EffectSynchronous {
			t.Fatalf("operation callable facet %s = %#v, %t", identity, facet, ok)
		}
	}
}

func TestCallbackKernelsContainNoSuspendingDispatch(t *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path      string
		functions []string
	}{
		{
			path: "gostdlib/src/internal/portable/slices/read.ts",
			functions: []string{
				"BinarySearchFunc",
				"CompareFunc",
				"ContainsFunc",
				"EqualFunc",
				"IndexFunc",
			},
		},
		{
			path: "gostdlib/src/internal/portable/slices/transform.ts",
			functions: []string{
				"CompactFunc",
				"DeleteFunc",
			},
		},
		{
			path:      "gostdlib/src/internal/portable/maps/operations.ts",
			functions: []string{"EqualFunc"},
		},
	}
	for _, test := range tests {
		source, readErr := os.ReadFile(filepath.Join(repository, test.path))
		if readErr != nil {
			t.Fatal(readErr)
		}
		for _, name := range test.functions {
			body := exportedFunctionBody(t, string(source), name)
			for _, forbidden := range []string{
				"async ",
				"await ",
				"Promise",
				"instanceof",
				"callComparison(",
				"callEquality(",
				"callPredicate(",
			} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("%s contains suspending dispatch %q", name, forbidden)
				}
			}
		}
	}
}

func exportedFunctionBody(t *testing.T, source, name string) string {
	t.Helper()
	start := strings.Index(source, "export function "+name+"<")
	if start < 0 {
		start = strings.Index(source, "export function "+name+"(")
	}
	if start < 0 {
		t.Fatalf("exported function %s is absent", name)
	}
	brace := strings.Index(source[start:], "{")
	if brace < 0 {
		t.Fatalf("exported function %s has no body", name)
	}
	brace += start
	depth := 0
	for index := brace; index < len(source); index++ {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : index+1]
			}
		}
	}
	t.Fatalf("exported function %s has an unterminated body", name)
	return ""
}
