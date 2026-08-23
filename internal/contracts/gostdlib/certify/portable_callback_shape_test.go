package certify

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func TestPortableCooperativeSortHasOneBoundedWorkPath(t *testing.T) {
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
	start := strings.Index(text, "export async function sortValues")
	if start < 0 {
		t.Fatal("portable cooperative sort owner is absent")
	}
	body := text[start:]
	for _, required := range []string{
		"for (let width = 1; width < values.length; width *= 2)",
		"await callComparison(compare, leftValue, rightValue)",
		"[source, target] = [target, source]",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("portable cooperative sort lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"await sortValues(",
		"[Symbol.iterator]",
		".slice(",
		".sort(",
		"instanceof Promise",
		"Promise.resolve",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portable cooperative sort contains %q", forbidden)
		}
	}
}

func TestPortableSynchronousSortHasNoCooperativeDispatch(t *testing.T) {
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
	body := string(source)
	start := strings.Index(body, "export function sortValuesSynchronous")
	if start < 0 {
		t.Fatal("portable synchronous sort owner is absent")
	}
	body = body[start:]
	for _, required := range []string{
		"for (let width = 1; width < values.length; width *= 2)",
		"compare(leftValue, rightValue)",
		"[source, target] = [target, source]",
	} {
		if !strings.Contains(body, required) {
			t.Fatalf("portable synchronous sort lacks %q", required)
		}
	}
	for _, forbidden := range []string{
		"await ",
		"callComparison(",
		"Promise",
		".sort(",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("portable synchronous sort contains %q", forbidden)
		}
	}
	if strings.Count(body, "sortValuesSynchronous") != 1 {
		t.Fatal("portable synchronous sort recurses or has duplicate owners")
	}
}

func TestSynchronousCallbackKernelDenominatorIsClosed(t *testing.T) {
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
	}
	actual := make([]string, 0, len(expected))
	for _, module := range manifest.FacetModules() {
		for _, facet := range module.Facets() {
			if facet.Kind() != gostdlib.FacetGenericCallableKernel ||
				!slices.Equal(facet.Capabilities(), []gostdlib.FacetCapability{
					gostdlib.FacetCapabilitySynchronousKernel,
				}) {
				continue
			}
			actual = append(actual, facet.SourceIdentity())
		}
	}
	slices.Sort(actual)
	if !slices.Equal(actual, expected) {
		t.Fatalf("synchronous callback kernel denominator = %#v", actual)
	}
	if _, ok := manifest.SynchronousGenericCallableKernel(
		"slices|kind=4|receiver=|name=SortedFunc",
	); ok {
		t.Fatal("sequence-driven SortedFunc was incorrectly narrowed")
	}
	expectedTransports := map[string]struct {
		current     string
		replacement string
		parameter   int
	}{
		"maps|kind=4|receiver=|name=EqualFunc": {
			"MapsEqualFuncKernel", "MapsEqualFuncSynchronousKernel", 6,
		},
		"slices|kind=4|receiver=|name=BinarySearchFunc": {
			"SlicesBinarySearchFuncKernel", "SlicesBinarySearchFuncSynchronousKernel", 5,
		},
		"slices|kind=4|receiver=|name=CompactFunc": {
			"SlicesCompactFuncKernel", "SlicesCompactFuncSynchronousKernel", 7,
		},
		"slices|kind=4|receiver=|name=CompareFunc": {
			"SlicesCompareFuncKernel", "SlicesCompareFuncSynchronousKernel", 8,
		},
		"slices|kind=4|receiver=|name=ContainsFunc": {
			"SlicesContainsFuncKernel", "SlicesContainsFuncSynchronousKernel", 4,
		},
		"slices|kind=4|receiver=|name=DeleteFunc": {
			"SlicesDeleteFuncKernel", "SlicesDeleteFuncSynchronousKernel", 7,
		},
		"slices|kind=4|receiver=|name=EqualFunc": {
			"SlicesEqualFuncKernel", "SlicesEqualFuncSynchronousKernel", 8,
		},
		"slices|kind=4|receiver=|name=IndexFunc": {
			"SlicesIndexFuncKernel", "SlicesIndexFuncSynchronousKernel", 4,
		},
		"slices|kind=4|receiver=|name=SortFunc": {
			"SlicesSortFuncKernel", "SlicesSortFuncSynchronousKernel", 5,
		},
		"slices|kind=4|receiver=|name=SortStableFunc": {
			"SlicesSortStableFuncKernel", "SlicesSortStableFuncSynchronousKernel", 5,
		},
	}
	actualTransports := make(map[string]gostdlib.InvocationTransportDocument)
	for _, transport := range manifest.InvocationTransports() {
		if transport.Conditional != nil {
			actualTransports[transport.SourceIdentity] = transport
		}
	}
	if len(actualTransports) != len(expectedTransports) {
		t.Fatalf("conditional transport denominator = %d", len(actualTransports))
	}
	for identity, expectedTransport := range expectedTransports {
		transport, ok := actualTransports[identity]
		if !ok || transport.Conditional == nil {
			t.Fatalf("conditional transport %q is absent", identity)
		}
		if transport.Target.Access != gostdlib.InvocationTransportAccessExport ||
			transport.Target.Export != expectedTransport.current ||
			!slices.Equal(transport.InputParameters, []int{expectedTransport.parameter}) ||
			!slices.Equal(
				transport.Conditional.CallableParameters,
				[]int{expectedTransport.parameter},
			) ||
			transport.Conditional.Replacement.Access !=
				gostdlib.InvocationTransportAccessExport ||
			transport.Conditional.Replacement.Export != expectedTransport.replacement ||
			transport.Target.Specifier != transport.Conditional.Replacement.Specifier {
			t.Fatalf("conditional transport %q = %#v", identity, transport)
		}
	}
}

func TestSynchronousCallbackKernelsContainNoCooperativeDispatch(t *testing.T) {
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
				"BinarySearchFuncSynchronous",
				"CompareFuncSynchronous",
				"ContainsFuncSynchronous",
				"EqualFuncSynchronous",
				"IndexFuncSynchronous",
			},
		},
		{
			path: "gostdlib/src/internal/portable/slices/transform.ts",
			functions: []string{
				"CompactFuncSynchronous",
				"DeleteFuncSynchronous",
			},
		},
		{
			path:      "gostdlib/src/internal/portable/maps/operations.ts",
			functions: []string{"EqualFuncSynchronous"},
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
					t.Fatalf("%s contains cooperative dispatch %q", name, forbidden)
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
