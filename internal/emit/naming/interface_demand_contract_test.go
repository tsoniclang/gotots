package naming

import (
	"errors"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
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
		"$goInterfaceAdapter$Named_Value",
		"$goReflectType$Named_Value",
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
		if len(repeated) != 1 {
			t.Fatalf(
				"repeated transition %d returned %d requests, want 1",
				occurrence,
				len(repeated),
			)
		}
	}
	repeatedDirect, err := adapterFirst.interfaceAdapterContractRequests(
		binding,
		"first",
		first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(repeatedDirect) != 2 {
		t.Fatalf(
			"repeated direct demand closure requests = %d, want 2",
			len(repeatedDirect),
		)
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
		"$goInterfaceAdapter$Named_Value",
		"$goReflectType$Named_Value",
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

func TestInterfaceDynamicTypeTokenIsCanonicalForLocalGoType(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/local", "local")
	functionScope := types.NewScope(
		sourcePackage.Scope(),
		token.Pos(1),
		token.Pos(100),
		"function",
	)
	typeName := types.NewTypeName(
		token.Pos(10),
		sourcePackage,
		"Local",
		nil,
	)
	localType := types.NewNamed(
		typeName,
		types.NewStruct(
			[]*types.Var{
				types.NewField(
					token.Pos(11),
					sourcePackage,
					"Value",
					types.Typ[types.Int32],
					false,
				),
			},
			nil,
		),
		nil,
	)
	if existing := functionScope.Insert(typeName); existing != nil {
		t.Fatal("local type insertion failed")
	}
	key := strings.Repeat("a", 64)
	registry := NewRegistry()
	first, err := registry.internInterfaceDynamicTypeToken(
		key,
		localType,
		"$goDynamicType$Named_Local",
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.internInterfaceDynamicTypeToken(
		key,
		localType,
		"$goDynamicType$Named_Local",
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedType, ok := first.owner.InterfaceDynamicType()
	if first.owner != second.owner ||
		first.name != second.name ||
		!ok ||
		selectedType != localType ||
		first.owner.Placement() != api.GeneratedArtifactPlacementCompilation ||
		first.owner.OutputPath() != output.InterfaceTypeSupportPath {
		t.Fatalf("canonical dynamic type = %#v / %#v", first, second)
	}
}

func TestInterfaceDynamicTypeTokenRejectsSemanticNameCollision(t *testing.T) {
	registry := NewRegistry()
	first := types.NewNamed(
		types.NewTypeName(token.Pos(1), nil, "First", nil),
		types.Typ[types.Int32],
		nil,
	)
	second := types.NewNamed(
		types.NewTypeName(token.Pos(2), nil, "Second", nil),
		types.Typ[types.Int32],
		nil,
	)
	if _, err := registry.internInterfaceDynamicTypeToken(
		strings.Repeat("1", 64),
		first,
		"$goDynamicType$Shared",
	); err != nil {
		t.Fatal(err)
	}
	_, err := registry.internInterfaceDynamicTypeToken(
		strings.Repeat("2", 64),
		second,
		"$goDynamicType$Shared",
	)
	var nameError *api.NameError
	if !errors.As(err, &nameError) ||
		nameError.Reason !=
			"interface-dynamic-type target-name collision" {
		t.Fatalf("collision error = %#v", err)
	}
}
