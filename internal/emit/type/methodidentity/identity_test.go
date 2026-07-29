package methodidentity

import (
	"go/token"
	"go/types"
	"testing"
)

func TestMethodIdentityUsesVisibilityPackageAndExactSignature(t *testing.T) {
	firstPackage := types.NewPackage("example.com/first", "first")
	secondPackage := types.NewPackage("example.com/second", "second")
	receiver := types.NewVar(token.NoPos, firstPackage, "receiver", types.Typ[types.Int32])
	result := types.NewVar(token.NoPos, firstPackage, "", types.Typ[types.Int32])
	signature := func(parameterType types.Type) *types.Signature {
		return types.NewSignatureType(
			receiver,
			nil,
			nil,
			types.NewTuple(types.NewVar(
				token.NoPos,
				firstPackage,
				"value",
				parameterType,
			)),
			types.NewTuple(result),
			false,
		)
	}
	exportedFirst := types.NewFunc(
		token.Pos(1),
		firstPackage,
		"Read",
		signature(types.Typ[types.Int32]),
	)
	exportedSecond := types.NewFunc(
		token.Pos(2),
		secondPackage,
		"Read",
		signature(types.Typ[types.Int32]),
	)
	privateFirst := types.NewFunc(
		token.Pos(3),
		firstPackage,
		"read",
		signature(types.Typ[types.Int32]),
	)
	privateSecond := types.NewFunc(
		token.Pos(4),
		secondPackage,
		"read",
		signature(types.Typ[types.Int32]),
	)
	different := types.NewFunc(
		token.Pos(5),
		firstPackage,
		"Read",
		signature(types.Typ[types.String]),
	)
	identity := func(object *types.TypeName) (string, error) {
		if object.Pkg() == nil {
			return object.Name(), nil
		}
		return object.Pkg().Path() + "." + object.Name(), nil
	}
	key := func(method *types.Func) string {
		t.Helper()
		result, err := BuildKey(method, identity)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if key(exportedFirst) != key(exportedSecond) ||
		!Equivalent(exportedFirst, exportedSecond) {
		t.Fatal("exported equivalent methods do not share identity")
	}
	if key(privateFirst) == key(privateSecond) ||
		Equivalent(privateFirst, privateSecond) {
		t.Fatal("unexported methods from different packages share identity")
	}
	if key(exportedFirst) == key(different) ||
		Equivalent(exportedFirst, different) {
		t.Fatal("different method signatures share identity")
	}
}
