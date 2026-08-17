package api_test

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"testing"
	"unsafe"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestRootRequestIsPointerScaleImmutableHandle(t *testing.T) {
	pointerSize := unsafe.Sizeof(uintptr(0))
	if size := unsafe.Sizeof(api.RootRequest{}); size > 2*pointerSize {
		t.Fatalf(
			"RootRequest size = %d bytes, want at most %d",
			size,
			2*pointerSize,
		)
	}
}

func TestCombinedRootRequestsPreserveOrderedImmutableLeaves(t *testing.T) {
	requests := namedImportRequests(t, "alpha", "beta", "gamma", "delta")
	leftSource := []api.RootRequest{requests[0]}
	rightSource := []api.RootRequest{requests[2]}
	left := api.CombineRequests(leftSource, []api.RootRequest{requests[1]})
	right := api.CombineRequests(rightSource, []api.RootRequest{requests[3]})
	combined := api.CombineRequests(left, right)

	leftSource[0] = api.RootRequest{}
	rightSource[0] = api.RootRequest{}
	if len(combined) != 1 {
		t.Fatalf("top-level carriers = %d, want 1", len(combined))
	}
	assertRequestNames(t, combined, "alpha", "beta", "gamma", "delta")

	exposed := append([]api.RootRequest(nil), combined...)
	exposed[0] = api.RootRequest{}
	assertRequestNames(t, combined, "alpha", "beta", "gamma", "delta")
}

func TestCombinedRootRequestsRejectInvalidNestedLeaf(t *testing.T) {
	request := namedImportRequests(t, "valid")[0]
	combined := api.CombineRequests(
		[]api.RootRequest{request},
		api.CombineRequests(
			[]api.RootRequest{request},
			[]api.RootRequest{{}},
		),
	)
	err := api.WalkRootRequests(combined, func(api.RootRequest) error {
		return nil
	})
	var requestError *api.RootRequestError
	if !errors.As(err, &requestError) {
		t.Fatalf("error = %#v, want RootRequestError", err)
	}
}

func TestCombinedRootRequestCarrierStaysBoundedAsLeavesGrow(t *testing.T) {
	request := namedImportRequests(t, "value")[0]
	for _, count := range []int{256, 512, 1024, 2048} {
		var combined []api.RootRequest
		for range count {
			combined = api.CombineRequests(
				combined,
				[]api.RootRequest{request},
			)
		}
		if len(combined) != 1 {
			t.Fatalf(
				"leaf count %d produced %d top-level carriers, want 1",
				count,
				len(combined),
			)
		}
		visited := 0
		if err := api.WalkRootRequests(
			combined,
			func(api.RootRequest) error {
				visited++
				return nil
			},
		); err != nil {
			t.Fatal(err)
		}
		if visited != count {
			t.Fatalf("visited leaves = %d, want %d", visited, count)
		}
	}
}

func TestUniqueRootRequestPayloadWalkIsLinearInPersistentGraph(t *testing.T) {
	request := namedImportRequests(t, "value")[0]
	combined := []api.RootRequest{request}
	for range 24 {
		combined = api.CombineRequests(combined, combined)
	}

	visited := 0
	if err := api.WalkUniqueRootRequestPayloads(
		combined,
		func(api.RootRequest) error {
			visited++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if visited != 1 {
		t.Fatalf("visited payloads = %d, want 1", visited)
	}
}

func TestUniqueRootRequestPayloadWalkPreservesDistinctPayloads(t *testing.T) {
	factory := tsgo.NewFactory()
	first, err := api.NewImportRequest(
		factory,
		api.ImportPhaseType,
		"./values.js",
		"Value",
		"First",
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := api.NewImportRequest(
		factory,
		api.ImportPhaseValue,
		"./values.js",
		"Value",
		"Second",
	)
	if err != nil {
		t.Fatal(err)
	}

	var localNames []string
	err = api.WalkUniqueRootRequestPayloads(
		api.CombineRequests(
			[]api.RootRequest{first},
			[]api.RootRequest{second},
		),
		func(request api.RootRequest) error {
			localNames = append(localNames, request.LocalName())
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(localNames); got != "[First Second]" {
		t.Fatalf("visited local names = %s, want [First Second]", got)
	}
}

func TestDeclarationRequestSelectionPreservesSharedSubgraph(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/requests", "requests")
	typeName := types.NewTypeName(token.Pos(1), sourcePackage, "Value", nil)
	copyRequirement, err := api.NewNamedStructOperationRequirement(
		typeName,
		api.NamedStructOperationCopy,
	)
	if err != nil {
		t.Fatal(err)
	}
	equalRequirement, err := api.NewNamedStructOperationRequirement(
		typeName,
		api.NamedStructOperationEqual,
	)
	if err != nil {
		t.Fatal(err)
	}
	copyRequest, err := api.NewDeclarationRequirementRequest(copyRequirement)
	if err != nil {
		t.Fatal(err)
	}
	equalRequest, err := api.NewDeclarationRequirementRequest(equalRequirement)
	if err != nil {
		t.Fatal(err)
	}
	declarations := api.CombineRequests(
		[]api.RootRequest{copyRequest},
		[]api.RootRequest{equalRequest},
	)
	imports := namedImportRequests(t, "Value")
	selected, err := api.SelectDeclarationRequests(
		api.CombineRequests(imports, declarations),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0] != declarations[0] {
		t.Fatal("declaration selection rebuilt the immutable shared subgraph")
	}
	visited := 0
	if err := api.WalkRootRequests(
		selected,
		func(request api.RootRequest) error {
			if request.Kind() != api.RootRequestDeclarationRequirement {
				t.Fatalf("selected request kind = %v", request.Kind())
			}
			visited++
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	if visited != 2 {
		t.Fatalf("selected declaration requests = %d, want 2", visited)
	}
}

func namedImportRequests(t *testing.T, names ...string) []api.RootRequest {
	t.Helper()
	factory := tsgo.NewFactory()
	requests := make([]api.RootRequest, 0, len(names))
	for _, name := range names {
		request, err := api.NewImportRequest(
			factory,
			api.ImportPhaseValue,
			"./values.js",
			name,
			name,
		)
		if err != nil {
			t.Fatal(err)
		}
		requests = append(requests, request)
	}
	return requests
}

func assertRequestNames(
	t *testing.T,
	requests []api.RootRequest,
	want ...string,
) {
	t.Helper()
	var got []string
	err := api.WalkRootRequests(
		requests,
		func(request api.RootRequest) error {
			got = append(got, request.ExportedName())
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("request order = %v, want %v", got, want)
	}
}
