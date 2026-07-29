package callable

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitVariadicParameter(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	parameter *types.Var,
	name string,
	parameterRole api.Role,
) (tsgo.ParameterDeclaration, []api.RootRequest, error) {
	if parameter == nil || name == "" {
		return nil, nil, api.Unsupported(context, api.CategoryType, source)
	}
	parameterType, ok := types.Unalias(parameter.Type()).(*types.Slice)
	if !ok {
		return nil, nil, api.Unsupported(context, api.CategoryType, source)
	}
	targetType, err := children.RepresentedType(
		context.WithRole(parameterRole),
		source,
		parameterType,
	)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType.Value(),
		nil,
	), targetType.Requests(), nil
}
