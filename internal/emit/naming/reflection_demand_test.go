package naming

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionDemandFollowsExactInterfaceReachability(t *testing.T) {
	firstSource, first, second := interfaceDemandTypes()
	secondSource := namedDemandType("SecondOnly", "Second")
	lateSource := namedDemandType("LateFirst", "First")
	reflectionType := reflectionContractType()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	registry := NewRegistry()

	firstBinding := internDemandAdapter(t, registry, "a", firstSource, placement)
	firstRequests, err := registry.interfaceAdapterContractRequests(
		firstBinding,
		"first",
		first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countReflectionRequests(firstRequests) != 0 {
		t.Fatal("adapter acquired reflection metadata before a reflection demand")
	}

	secondBinding := internDemandAdapter(t, registry, "b", secondSource, placement)
	secondRequests, err := registry.interfaceAdapterContractRequests(
		secondBinding,
		"second",
		second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countReflectionRequests(secondRequests) != 0 {
		t.Fatal("unrelated adapter acquired reflection metadata")
	}

	demandRequests, err := registry.recordInterfaceReflectionDemand(
		"first",
		first,
		reflectionType,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countReflectionRequests(demandRequests) != 1 {
		t.Fatalf(
			"existing adapter reflection requests = %d, want 1",
			countReflectionRequests(demandRequests),
		)
	}

	lateBinding := internDemandAdapter(t, registry, "c", lateSource, placement)
	lateRequests, err := registry.interfaceAdapterContractRequests(
		lateBinding,
		"first",
		first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countReflectionRequests(lateRequests) != 1 {
		t.Fatalf(
			"late adapter reflection requests = %d, want 1",
			countReflectionRequests(lateRequests),
		)
	}
	if len(registry.reflectionTypes) != 2 {
		t.Fatalf(
			"reflection descriptor owners = %d, want 2",
			len(registry.reflectionTypes),
		)
	}
	secondKey := strings.Repeat("b", 64)
	if _, leaked := registry.reflectionTypes[secondKey]; leaked {
		t.Fatal("unrelated interface adapter leaked into reflection descriptors")
	}
}

func namedDemandType(name string, methods ...string) *types.Named {
	pkg := types.NewPackage("example.com/demand", "demand")
	typeName := types.NewTypeName(token.NoPos, pkg, name, nil)
	named := types.NewNamed(typeName, types.NewStruct(nil, nil), nil)
	for _, methodName := range methods {
		receiver := types.NewVar(token.NoPos, pkg, "", named)
		named.AddMethod(types.NewFunc(
			token.NoPos,
			pkg,
			methodName,
			types.NewSignatureType(
				receiver,
				nil,
				nil,
				types.NewTuple(),
				types.NewTuple(),
				false,
			),
		))
	}
	return named
}

func reflectionContractType() *types.TypeName {
	pkg := types.NewPackage("reflect", "reflect")
	contract := types.NewInterfaceType(nil, nil).Complete()
	name := types.NewTypeName(token.NoPos, pkg, "Type", nil)
	types.NewNamed(name, contract, nil)
	return name
}

func internDemandAdapter(
	t *testing.T,
	registry *Registry,
	key string,
	sourceType types.Type,
	placement generatedArtifactPlacement,
) interfaceAdapterBinding {
	t.Helper()
	binding, err := registry.internInterfaceAdapter(
		strings.Repeat(key, 64),
		sourceType,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	return binding
}

func countReflectionRequests(requests []api.RootRequest) int {
	count := 0
	for _, request := range requests {
		requirement, ok := request.DeclarationRequirement()
		if ok && requirement.Kind() == api.DeclarationRequirementReflectionType {
			count++
		}
	}
	return count
}
