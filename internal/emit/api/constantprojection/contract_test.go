package constantprojection_test

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestConstantProjectionKindBoundaryIsTotal(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/projection", "projection")
	selected := types.NewConst(
		token.Pos(1),
		sourcePackage,
		"Width",
		types.Typ[types.UntypedInt],
		constant.MakeInt64(64),
	)
	owner := types.NewFunc(
		token.Pos(2),
		sourcePackage,
		"Use",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)

	valid := map[types.BasicKind]struct{}{
		types.Bool:       {},
		types.Int:        {},
		types.Int8:       {},
		types.Int16:      {},
		types.Int32:      {},
		types.Int64:      {},
		types.Uint:       {},
		types.Uint8:      {},
		types.Uint16:     {},
		types.Uint32:     {},
		types.Uint64:     {},
		types.Uintptr:    {},
		types.Float32:    {},
		types.Float64:    {},
		types.Complex64:  {},
		types.Complex128: {},
		types.String:     {},
	}
	for index := range types.Typ {
		kind := types.BasicKind(index)
		_, want := valid[kind]
		_, got := api.ConstantProjectionType(kind)
		if got != want {
			t.Fatalf("projection kind %d accepted = %v, want %v", kind, got, want)
		}
		_, nameError := api.ConstantProjectionName("Width", kind)
		_, packageError := api.NewConstantProjectionRequirement(selected, kind)
		_, localError := api.NewLocalConstantProjectionRequirement(owner, selected, kind)
		if (nameError == nil) != want ||
			(packageError == nil) != want ||
			(localError == nil) != want {
			t.Fatalf(
				"projection kind %d boundaries = name %v package %v local %v, want accepted %v",
				kind,
				nameError,
				packageError,
				localError,
				want,
			)
		}
	}
	invalid := []types.BasicKind{
		types.BasicKind(-1),
		types.BasicKind(len(types.Typ)),
		types.BasicKind(len(types.Typ) + 100),
	}
	for _, kind := range invalid {
		if _, ok := api.ConstantProjectionType(kind); ok {
			t.Fatalf("invalid projection kind %d was accepted", kind)
		}
		if _, err := api.ConstantProjectionName("Width", kind); err == nil {
			t.Fatalf("name accepted out-of-range projection kind %d", kind)
		}
		if _, err := api.NewConstantProjectionRequirement(selected, kind); err == nil {
			t.Fatalf("package requirement accepted out-of-range projection kind %d", kind)
		}
		if _, err := api.NewLocalConstantProjectionRequirement(owner, selected, kind); err == nil {
			t.Fatalf("local requirement accepted out-of-range projection kind %d", kind)
		}
	}
}
