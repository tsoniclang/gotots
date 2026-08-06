package maprepresentation

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestNilRejectsOpenGenericMapOutsideGenericOperationOwner(
	t *testing.T,
) {
	parameter := types.NewTypeParam(
		types.NewTypeName(token.NoPos, nil, "K", nil),
		types.Universe.Lookup("comparable").Type(),
	)
	sourceType := types.NewMap(parameter, types.Typ[types.Int32])
	targetContext, err := api.NewContext(
		api.RoleMapReceiver,
		token.NewFileSet(),
		types.NewPackage("example.com/specialization", "specialization"),
		&types.Info{},
		types.SizesFor("gc", "amd64"),
		api.MemoryByteOrderLittleEndian,
		tsgo.NewFactory(),
		staticSpecializationNames{},
		staticSpecializationValues{
			key:   parameter,
			value: types.Typ[types.Int32],
		},
		storage.Owner{},
		api.IntegerRepresentationNumber,
		api.EvaluationOrderPreserveGo,
		api.ConcurrencySemanticsDisabled,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Nil(
		targetContext,
		nil,
		nil,
		sourceType,
	); err == nil ||
		!strings.Contains(
			err.Error(),
			"open generic map zero bypassed generic operation ownership",
		) {
		t.Fatalf("open generic map bypass error = %v", err)
	}
}
