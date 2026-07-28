package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitVariadicParameter(
	context api.Context,
	_ api.ChildEmitter,
	source ast.Node,
	_ *types.Var,
	_ api.Role,
) (tsgo.ParameterDeclaration, []api.RootRequest, error) {
	return nil, nil, api.Unsupported(context, api.CategoryType, source)
}
