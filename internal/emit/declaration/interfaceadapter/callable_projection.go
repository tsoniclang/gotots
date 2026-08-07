package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func projectMethodArguments(
	context api.Context,
	signature *types.Signature,
	method *types.Func,
	arguments []tsgo.Expression,
	direct bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if !direct {
		return arguments, nil, nil, nil
	}
	selected, _ := context.ResolveCallableABI(method.Origin())
	return callable.ProjectArguments(
		context,
		nil,
		signature,
		arguments,
		selected,
	)
}

func adapterCallBody(
	context api.Context,
	signature *types.Signature,
	call api.ExpressionEmission,
) []tsgo.Statement {
	body := call.Before()
	if signature.Results().Len() == 0 {
		return append(body, context.Factory().ExpressionStatement(call.Value()))
	}
	return append(body, context.Factory().ReturnStatement(call.Value()))
}
