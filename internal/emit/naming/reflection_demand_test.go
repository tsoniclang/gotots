package naming

import (
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionDemandFollowsExactInterfaceReachability(t *testing.T) {
	firstSource, first, second := interfaceDemandTypes()
	firstDemand := interfaceDemandSelection("first", first)
	secondDemand := interfaceDemandSelection("second", second)
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
		&firstDemand,
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
		&secondDemand,
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
		&firstDemand,
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

func TestReflectionValueContractJoinIsDiscoveryOrderIndependent(t *testing.T) {
	_, contract, _ := interfaceDemandTypes()
	demand := interfaceDemandSelection("first", contract)
	source := namedDemandType("Reflected", "First")
	reflectionType := reflectionContractType()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}

	adapterFirst := NewRegistry()
	firstBinding := internDemandAdapter(
		t,
		adapterFirst,
		"a",
		source,
		placement,
	)
	adapterFirst.reflectionValueDemands[firstBinding.key] = struct{}{}
	before, err := adapterFirst.interfaceAdapterContractRequests(
		firstBinding,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countAdapterContractRequests(before) != 0 {
		t.Fatal("reflected adapter acquired an absent interface contract")
	}
	after, err := adapterFirst.recordReflectionValueContract(
		demand,
		reflectionType,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionValueJoin(t, "adapter-first", after)

	contractFirst := NewRegistry()
	before, err = contractFirst.recordReflectionValueContract(
		demand,
		reflectionType,
	)
	if err != nil {
		t.Fatal(err)
	}
	if countAdapterContractRequests(before) != 0 {
		t.Fatal("reflection contract selected an absent adapter")
	}
	lateKey := strings.Repeat("b", 64)
	contractFirst.reflectionValueDemands[lateKey] = struct{}{}
	lateBinding := internDemandAdapter(
		t,
		contractFirst,
		"b",
		source,
		placement,
	)
	late, err := contractFirst.interfaceAdapterContractRequests(
		lateBinding,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionValueJoin(t, "contract-first", late)
}

func TestReflectionReplaySelectsOnlyObservedExactSurfaces(t *testing.T) {
	source, contract, _ := interfaceDemandTypes()
	registry := NewRegistry()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	binding := internDemandAdapter(t, registry, "a", source, placement)
	registry.reflectionValueDemands[binding.key] = struct{}{}
	reflectionType := reflectionContractType()

	for index := 0; index < 2; index++ {
		selection := interfaceDemandSelection("first", contract)
		selection.sourceType = namedInterfaceDemandType(
			"Observed"+strconv.Itoa(index),
			contract,
		)
		selection.surfaceKey = "observed-" + strconv.Itoa(index)
		if _, err := registry.recordReflectionValueContract(
			selection,
			reflectionType,
		); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 128; index++ {
		selection := interfaceDemandSelection("first", contract)
		selection.sourceType = namedInterfaceDemandType(
			"Unobserved"+strconv.Itoa(index),
			contract,
		)
		selection.surfaceKey = "unobserved-" + strconv.Itoa(index)
		if _, err := registry.internInterfaceContract(selection); err != nil {
			t.Fatal(err)
		}
	}

	requests, err := registry.interfaceAdapterContractRequests(binding, nil)
	if err != nil {
		t.Fatal(err)
	}
	if count := countAdapterContractRequests(requests); count != 2 {
		t.Fatalf("observed exact-surface requests = %d, want 2", count)
	}
}

func assertReflectionValueJoin(
	t *testing.T,
	order string,
	requests []api.RootRequest,
) {
	t.Helper()
	if countAdapterContractRequests(requests) != 1 {
		t.Fatalf(
			"%s adapter contract requests = %d, want 1",
			order,
			countAdapterContractRequests(requests),
		)
	}
	if countReflectionRequests(requests) != 1 {
		t.Fatalf(
			"%s reflection requests = %d, want 1",
			order,
			countReflectionRequests(requests),
		)
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
		"$goInterfaceAdapter$"+key,
		"$goReflectType$"+key,
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

func countAdapterContractRequests(requests []api.RootRequest) int {
	count := 0
	for _, request := range requests {
		requirement, ok := request.DeclarationRequirement()
		if !ok {
			continue
		}
		_, _, _, _, contract := requirement.InterfaceAdapterContract()
		if contract {
			count++
		}
	}
	return count
}
