package naming

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionInterfaceMethodSetDemandIsSingleAndIsolated(t *testing.T) {
	sourceType, first, second := interfaceDemandTypes()
	empty := interfaceDemandSelection(
		"empty-interface",
		types.NewInterfaceType(nil, nil).Complete(),
	)
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}
	registry := NewRegistry()
	firstBinding := internDemandAdapter(
		t,
		registry,
		"a",
		sourceType,
		placement,
	)
	for _, contract := range []interfaceContractSelection{
		interfaceDemandSelection("first", first),
		interfaceDemandSelection("second", second),
	} {
		if requests, err := registry.recordInterfaceContractDemand(
			empty,
			contract,
		); err != nil {
			t.Fatal(err)
		} else if got := countAdapterContractRequests(requests); got != 0 {
			t.Fatalf("pre-adapter contract requests = %d, want 0", got)
		}
	}
	requests, err := registry.recordReflectionInterfaceAdapter(
		firstBinding,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := countAdapterCompleteMethodSetRequests(requests); got != 1 {
		t.Fatalf("complete method-set requests = %d, want 1", got)
	}
	if got := countAdapterContractRequests(requests); got != 0 {
		t.Fatalf("adapter contracts = %d, want 0", got)
	}
	if got := countReflectionRequests(requests); got != 0 {
		t.Fatalf("reflection requests = %d, want 0", got)
	}
	if reached := registry.interfaceAdaptersByContract[empty.contractKey]; reached != nil {
		if _, leaked := reached[firstBinding.key]; leaked {
			t.Fatal("reflection method-set request populated ordinary interface reachability")
		}
	}
}

func countAdapterCompleteMethodSetRequests(requests []api.RootRequest) int {
	count := 0
	err := api.WalkRootRequests(requests, func(request api.RootRequest) error {
		requirement, ok := request.DeclarationRequirement()
		if !ok {
			return nil
		}
		if _, complete := requirement.InterfaceAdapterCompleteMethodSet(); complete {
			count++
		}
		return nil
	})
	if err != nil {
		return -1
	}
	return count
}
