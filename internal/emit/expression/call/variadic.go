package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitVariadicArguments(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.CallExpr,
	_ *types.Signature,
	_ bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	return nil, nil, nil,
		api.Unsupported(context, api.CategoryExpression, source)
}
