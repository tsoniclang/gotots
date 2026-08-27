package naming

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionInterfaceDemandsFlushAsExactBatch(t *testing.T) {
	sourceType, first, second := interfaceDemandTypes()
	empty := interfaceDemandSelection(
		"empty-interface",
		types.NewInterfaceType(nil, nil).Complete(),
	)
	missingMethod := types.NewFunc(
		token.NoPos,
		types.NewPackage("example.com/demand", "demand"),
		"Missing",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	missing := types.NewInterfaceType(
		[]*types.Func{missingMethod},
		nil,
	).Complete()
	contracts := []interfaceContractSelection{
		interfaceDemandSelection("first", first),
		interfaceDemandSelection("missing", missing),
		interfaceDemandSelection("second", second),
	}
	for _, exposureFirst := range []bool{false, true} {
		registry := NewRegistry()
		binding := internDemandAdapter(
			t,
			registry,
			"a",
			sourceType,
			generatedArtifactPlacement{
				kind: api.GeneratedArtifactPlacementCompilation,
			},
		)
		recordExposure := func() {
			t.Helper()
			if err := registry.recordReflectionInterfaceAdapter(
				empty,
				binding,
			); err != nil {
				t.Fatal(err)
			}
		}
		if exposureFirst {
			recordExposure()
			requests, err := registry.FlushReflectionInterfaceDemands()
			if err != nil {
				t.Fatal(err)
			}
			if got := countAdapterContractRequests(requests); got != 0 {
				t.Fatalf("pre-contract flush requests = %d, want 0", got)
			}
		}
		for _, contract := range contracts {
			requests, err := registry.recordInterfaceContractDemand(
				empty,
				contract,
			)
			if err != nil {
				t.Fatal(err)
			}
			if got := countAdapterContractRequests(requests); got != 0 {
				t.Fatalf("eager reflection requests = %d, want 0", got)
			}
		}
		if !exposureFirst {
			recordExposure()
		}
		requests, err := registry.FlushReflectionInterfaceDemands()
		if err != nil {
			t.Fatal(err)
		}
		if got := countAdapterContractRequests(requests); got != 2 {
			t.Fatalf("quiescent adapter contracts = %d, want 2", got)
		}
		if got := countReflectionRequests(requests); got != 0 {
			t.Fatalf("quiescent reflection requests = %d, want 0", got)
		}
		if repeated, err := registry.FlushReflectionInterfaceDemands(); err != nil {
			t.Fatal(err)
		} else if got := countAdapterContractRequests(repeated); got != 0 {
			t.Fatalf("repeated flush contracts = %d, want 0", got)
		}
		if reached := registry.interfaceAdaptersByContract[empty.contractKey]; reached != nil {
			if _, leaked := reached[binding.key]; leaked {
				t.Fatal("reflection exposure populated ordinary interface reachability")
			}
		}
	}
}
