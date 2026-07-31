package ordering

import (
	"go/token"
	"go/types"
	"sort"
	"testing"
)

func TestCompareObjectsUsesSemanticOrderBeforeTokenAllocation(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/provider", "provider")
	signature := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	alpha := types.NewFunc(token.Pos(200), sourcePackage, "Alpha", signature)
	beta := types.NewFunc(token.Pos(10), sourcePackage, "Beta", signature)
	objects := []types.Object{beta, alpha}
	sort.Slice(objects, func(left, right int) bool {
		return CompareObjects(objects[left], objects[right]) < 0
	})
	if objects[0] != alpha || objects[1] != beta {
		t.Fatal("token allocation selected declaration order")
	}
}

func TestCompareObjectsOrdersSameNamedMethodsByReceiverIdentity(t *testing.T) {
	sourcePackage := types.NewPackage("example.com/provider", "provider")
	method := func(position token.Pos, receiverName string) *types.Func {
		typeName := types.NewTypeName(
			position,
			sourcePackage,
			receiverName,
			nil,
		)
		named := types.NewNamed(
			typeName,
			types.NewStruct(nil, nil),
			nil,
		)
		receiver := types.NewVar(
			position,
			sourcePackage,
			"",
			types.NewPointer(named),
		)
		return types.NewFunc(
			position,
			sourcePackage,
			"Read",
			types.NewSignatureType(
				receiver,
				nil,
				nil,
				nil,
				nil,
				false,
			),
		)
	}
	alpha := method(token.Pos(200), "Alpha")
	beta := method(token.Pos(10), "Beta")
	if CompareObjects(alpha, beta) >= 0 ||
		CompareObjects(beta, alpha) <= 0 {
		t.Fatal("same-named methods ignored receiver identity")
	}
}
