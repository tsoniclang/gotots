package api_test

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestTargetIntrinsicHasClosedGlobalIdentity(t *testing.T) {
	factory := tsgo.NewFactory()
	assertNumberIntrinsic(t, api.TargetIntrinsicNumber.Expression(factory))
	if catchesNumberIntrinsic(factory.Identifier("Number")) {
		t.Fatal("bare Number identifier passed the target-intrinsic identity gate")
	}
	if api.TargetIntrinsicNumber != 1 ||
		api.TargetIntrinsicNumber.String() != "Number" ||
		api.TargetIntrinsic(2).String() != "target-intrinsic(2)" {
		t.Fatal("target-intrinsic IDs or names drifted")
	}
}

func assertNumberIntrinsic(t *testing.T, expression tsgo.Expression) {
	t.Helper()
	if !catchesNumberIntrinsic(expression) {
		t.Fatalf("target intrinsic = %T, want globalThis.Number", expression)
	}
}

func catchesNumberIntrinsic(expression tsgo.Expression) bool {
	member, ok := expression.(tsgo.PropertyAccessExpression)
	if !ok {
		return false
	}
	anchor, ok := member.Expression().(tsgo.Identifier)
	name, nameOK := member.Name().(tsgo.Identifier)
	return ok &&
		nameOK &&
		anchor.Text() == api.TargetGlobalAnchorName &&
		name.Text() == "Number"
}
