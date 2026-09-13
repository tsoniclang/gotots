package memoryview

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestIndexedRegionAddressCapturesSelectedBacking(t *testing.T) {
	statement, err := BuildOperation(tsgo.NewFactory(), api.RuntimeRegionAddress, "goRegionAddress", "GoPanic")
	if err != nil {
		t.Fatal(err)
	}
	function := statement.(tsgo.FunctionDeclaration)
	branch := function.Body().(tsgo.Block).Statements()[0].(tsgo.IfStatement).ThenStatement().(tsgo.Block).Statements()
	if len(branch) != 2 {
		t.Fatalf("indexed address branch has %d statements", len(branch))
	}
	declaration := branch[0].(tsgo.VariableStatement).DeclarationList().Declarations()[0]
	backing := declaration.Initializer().(tsgo.PropertyAccessExpression)
	if backing.Expression().(tsgo.Identifier).Text() != "region" || backing.Name().(tsgo.Identifier).Text() != ValuesMember {
		t.Fatal("indexed address does not capture its exact backing field")
	}
	call := branch[1].(tsgo.ReturnStatement).Expression().(tsgo.CallExpression)
	addressed := call.Arguments()[0].(tsgo.ElementAccessExpression)
	if addressed.Expression().(tsgo.Identifier).Text() != declaration.Name().(tsgo.Identifier).Text() {
		t.Fatal("indexed address does not use the captured backing")
	}
}
