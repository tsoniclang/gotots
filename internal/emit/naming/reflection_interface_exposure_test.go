package naming

import (
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestReflectionInterfaceExposureIsExactAndOrderIndependent(t *testing.T) {
	sourceType, first, _ := interfaceDemandTypes()
	empty := interfaceDemandSelection(
		"empty-interface",
		types.NewInterfaceType(nil, nil).Complete(),
	)
	firstContract := interfaceDemandSelection("first", first)
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}

	adapterFirst := NewRegistry()
	firstBinding := internDemandAdapter(
		t,
		adapterFirst,
		"a",
		sourceType,
		placement,
	)
	before, err := adapterFirst.recordReflectionInterfaceAdapter(
		empty,
		firstBinding,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionInterfaceExposureRequests(t, "adapter-first before demand", before, 0)
	if reached := adapterFirst.interfaceAdaptersByContract[empty.contractKey]; reached != nil {
		if _, leaked := reached[firstBinding.key]; leaked {
			t.Fatal("reflection exposure leaked into ordinary empty-interface membership")
		}
	}
	after, err := adapterFirst.recordInterfaceContractDemand(
		empty,
		firstContract,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionInterfaceExposureRequests(t, "adapter-first after demand", after, 1)

	contractFirst := NewRegistry()
	before, err = contractFirst.recordInterfaceContractDemand(
		empty,
		firstContract,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionInterfaceExposureRequests(t, "contract-first before adapter", before, 0)
	lateBinding := internDemandAdapter(
		t,
		contractFirst,
		"b",
		sourceType,
		placement,
	)
	late, err := contractFirst.recordReflectionInterfaceAdapter(
		empty,
		lateBinding,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionInterfaceExposureRequests(t, "contract-first after adapter", late, 1)

	unrelated := NewRegistry()
	unrelatedBinding := internDemandAdapter(
		t,
		unrelated,
		"c",
		namedDemandType("SecondOnly", "Second"),
		placement,
	)
	if _, err := unrelated.recordInterfaceContractDemand(empty, firstContract); err != nil {
		t.Fatal(err)
	}
	unrelatedRequests, err := unrelated.recordReflectionInterfaceAdapter(
		empty,
		unrelatedBinding,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertReflectionInterfaceExposureRequests(t, "unrelated adapter", unrelatedRequests, 0)
}

func assertReflectionInterfaceExposureRequests(
	t *testing.T,
	owner string,
	requests []api.RootRequest,
	wantContracts int,
) {
	t.Helper()
	if got := countAdapterContractRequests(requests); got != wantContracts {
		t.Fatalf("%s adapter contracts = %d, want %d", owner, got, wantContracts)
	}
	if got := countReflectionRequests(requests); got != 0 {
		t.Fatalf("%s reflection requests = %d, want 0", owner, got)
	}
}
