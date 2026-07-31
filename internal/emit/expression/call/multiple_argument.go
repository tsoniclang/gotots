package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMultipleArgument(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	results *types.Tuple,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if results == nil || results.Len() != signature.Params().Len() {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	for index := range results.Len() {
		if !types.AssignableTo(results.At(index).Type(), signature.Params().At(index).Type()) {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	capture, err := resulttuple.Emit(
		context,
		children,
		source.Args[0],
		results,
		api.RoleCallArgument,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	arguments := make([]tsgo.Expression, 0, results.Len())
	requests := capture.Requests()
	for index := range results.Len() {
		element, err := capture.Element(context, index)
		if err != nil {
			return nil, nil, nil, err
		}
		copied, err := context.Values().Transfer(
			context.WithRole(api.RoleCallArgument),
			source.Args[0],
			results.At(index).Type(),
			signature.Params().At(index).Type(),
			api.ValueTransferCopy,
			api.DirectExpression(element),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(copied.Before()) != 0 {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		arguments = append(arguments, copied.Value())
		requests = append(requests, copied.Requests()...)
	}
	return arguments, capture.Statements(), requests, nil
}
