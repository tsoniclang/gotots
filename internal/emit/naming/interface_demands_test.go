package naming

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestInterfaceContractReachabilityIsIncrementalAndOrderIndependent(
	t *testing.T,
) {
	sourceType, first, second := interfaceDemandTypes()
	placement := generatedArtifactPlacement{
		kind: api.GeneratedArtifactPlacementCompilation,
	}

	adapterFirst := NewRegistry()
	binding, err := adapterFirst.internInterfaceAdapter(
		strings.Repeat("a", 64),
		sourceType,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	firstRequests, err := adapterFirst.interfaceAdapterContractRequests(
		binding,
		"first",
		first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstRequests) != 1 {
		t.Fatalf("first direct demand requests = %d, want 1", len(firstRequests))
	}
	secondRequests, err := adapterFirst.recordInterfaceContractDemand(
		"first",
		first,
		"second",
		second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondRequests) != 1 {
		t.Fatalf("first transition requests = %d, want 1", len(secondRequests))
	}
	for occurrence := 0; occurrence < 1_000; occurrence++ {
		repeated, repeatErr := adapterFirst.recordInterfaceContractDemand(
			"first",
			first,
			"second",
			second,
		)
		if repeatErr != nil {
			t.Fatal(repeatErr)
		}
		if len(repeated) != 0 {
			t.Fatalf(
				"repeated transition %d scheduled %d requests",
				occurrence,
				len(repeated),
			)
		}
	}

	transitionFirst := NewRegistry()
	beforeAdapter, err := transitionFirst.recordInterfaceContractDemand(
		"first",
		first,
		"second",
		second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeAdapter) != 0 {
		t.Fatalf("transition without adapters scheduled %d requests", len(beforeAdapter))
	}
	lateBinding, err := transitionFirst.internInterfaceAdapter(
		strings.Repeat("b", 64),
		sourceType,
		placement,
	)
	if err != nil {
		t.Fatal(err)
	}
	lateRequests, err := transitionFirst.interfaceAdapterContractRequests(
		lateBinding,
		"first",
		first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(lateRequests) != 2 {
		t.Fatalf("late adapter closure requests = %d, want 2", len(lateRequests))
	}
}

func interfaceDemandTypes() (types.Type, *types.Interface, *types.Interface) {
	sourcePackage := types.NewPackage("example.com/demand", "demand")
	typeName := types.NewTypeName(
		token.NoPos,
		sourcePackage,
		"Value",
		nil,
	)
	sourceType := types.NewNamed(
		typeName,
		types.NewStruct(nil, nil),
		nil,
	)
	for _, name := range []string{"First", "Second"} {
		receiver := types.NewVar(token.NoPos, sourcePackage, "", sourceType)
		sourceType.AddMethod(types.NewFunc(
			token.NoPos,
			sourcePackage,
			name,
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
	contract := func(name string) *types.Interface {
		method := types.NewFunc(
			token.NoPos,
			sourcePackage,
			name,
			types.NewSignatureType(
				nil,
				nil,
				nil,
				types.NewTuple(),
				types.NewTuple(),
				false,
			),
		)
		return types.NewInterfaceType([]*types.Func{method}, nil).Complete()
	}
	return sourceType, contract("First"), contract("Second")
}
