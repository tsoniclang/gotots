package api_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestTargetIntrinsicHasClosedGlobalIdentity(t *testing.T) {
	factory := tsgo.NewFactory()
	for intrinsic := api.TargetIntrinsicNumber; intrinsic <= api.TargetIntrinsicError; intrinsic++ {
		assertTargetIntrinsic(t, intrinsic, intrinsic.Expression(factory))
		if name := intrinsic.UnshadowedExpression(factory); name.Text() != intrinsic.String() {
			t.Fatalf("unshadowed target intrinsic = %q, want %q", name.Text(), intrinsic.String())
		}
	}
	if name := api.TargetIntrinsicPromise.TypeName(factory); name.Text() != "Promise" {
		t.Fatalf("target intrinsic type name = %q, want Promise", name.Text())
	}
	if !api.TargetIntrinsicPromise.ReservesTypeName() ||
		!api.TargetIntrinsicObject.ReservesTypeName() ||
		!api.IsReservedTargetTypeName("Promise") ||
		!api.IsReservedTargetTypeName("Object") ||
		api.TargetIntrinsicString.ReservesTypeName() ||
		api.IsReservedTargetTypeName("String") {
		t.Fatal("target intrinsic type-name reservation is not exact")
	}
	if catchesTargetIntrinsic(factory.Identifier("Number"), api.TargetIntrinsicNumber) {
		t.Fatal("bare Number identifier passed the target-intrinsic identity gate")
	}
	if api.TargetIntrinsicNumber != 1 || api.TargetIntrinsicError != 7 ||
		api.TargetIntrinsic(8).String() != "target-intrinsic(8)" {
		t.Fatal("target-intrinsic IDs or names drifted")
	}
}

func assertTargetIntrinsic(
	t *testing.T,
	intrinsic api.TargetIntrinsic,
	expression tsgo.Expression,
) {
	t.Helper()
	if !catchesTargetIntrinsic(expression, intrinsic) {
		t.Fatalf("target intrinsic = %T, want globalThis.%s", expression, intrinsic)
	}
}

func catchesTargetIntrinsic(
	expression tsgo.Expression,
	intrinsic api.TargetIntrinsic,
) bool {
	member, ok := expression.(tsgo.PropertyAccessExpression)
	if !ok {
		return false
	}
	anchor, ok := member.Expression().(tsgo.Identifier)
	name, nameOK := member.Name().(tsgo.Identifier)
	return ok &&
		nameOK &&
		anchor.Text() == api.TargetGlobalAnchorName &&
		name.Text() == intrinsic.String()
}
