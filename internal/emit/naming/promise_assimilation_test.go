package naming

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
)

func TestStructMemberOwnerReservesPromiseAssimilationMember(t *testing.T) {
	thenField := types.NewField(
		token.NoPos,
		nil,
		typescriptclass.PromiseAssimilationMember,
		types.Typ[types.Int],
		false,
	)
	neighbor := types.NewField(
		token.NoPos,
		nil,
		typescriptclass.PromiseAssimilationMember+"__field_1",
		types.Typ[types.Int],
		false,
	)
	structure := types.NewStruct([]*types.Var{thenField, neighbor}, nil)

	registry := NewRegistry()
	owner := NewOwner(nil, &types.Info{}, registry)
	owner.preallocateStructMembers(structure)
	names := &File{owner: owner}
	thenName, err := names.Member(thenField)
	if err != nil {
		t.Fatal(err)
	}
	neighborName, err := names.Member(neighbor)
	if err != nil {
		t.Fatal(err)
	}
	if thenName == typescriptclass.PromiseAssimilationMember ||
		neighborName == typescriptclass.PromiseAssimilationMember ||
		thenName == neighborName {
		t.Fatalf("source member names = %q, %q", thenName, neighborName)
	}

	environment := NewRegistry()
	environment.indexEnvironmentFields(structure)
	if environment.memberNameByObject[thenField] ==
		typescriptclass.PromiseAssimilationMember {
		t.Fatal("environment member owner admitted Promise assimilation member")
	}
}
