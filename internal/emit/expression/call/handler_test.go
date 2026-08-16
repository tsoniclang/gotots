package call

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func TestCalleeObjectRejectsEverySelectedToolchainBuiltin(t *testing.T) {
	info := &types.Info{Uses: make(map[*ast.Ident]types.Object)}
	builtins := 0
	for _, name := range types.Universe.Names() {
		object := types.Universe.Lookup(name)
		if _, ok := object.(*types.Builtin); !ok {
			continue
		}
		builtins++
		identifier := &ast.Ident{Name: name}
		info.Uses[identifier] = object
		if actual, ok := calleeObject(api.DirectTypeInfo(info), identifier); ok {
			t.Fatalf("builtin %s resolved as source function %v", name, actual)
		}
	}
	if builtins == 0 {
		t.Fatal("selected toolchain exposes no builtins")
	}

	sourcePackage := types.NewPackage("example.com/source", "source")
	function := types.NewFunc(
		token.Pos(1),
		sourcePackage,
		"Run",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
	sourcePackage.Scope().Insert(function)
	identifier := &ast.Ident{Name: "Run"}
	info.Uses[identifier] = function
	if actual, ok := calleeObject(api.DirectTypeInfo(info), identifier); !ok || actual != function {
		t.Fatalf("source function resolved as %v, %v", actual, ok)
	}
}
