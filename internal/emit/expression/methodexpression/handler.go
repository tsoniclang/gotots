package methodexpression

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, error) {
	if selected == nil || selected.Kind() != types.MethodExpr {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if method, direct := selectionvalue.DirectMethodExpression(
		context,
		source,
		selected,
	); direct {
		reference, err := context.Names().Reference(method)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(
			context.Factory().Identifier(reference.Name()),
			reference.Requests()...,
		), nil
	}
	signature, ok := selected.Type().(*types.Signature)
	if !ok ||
		!callable.Supports(signature) ||
		signature.Params().Len() == 0 ||
		!types.Identical(
			signature.Params().At(0).Type(),
			selected.Recv(),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetSignature, err := callable.EmitAdapter(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parameters := targetSignature.ParameterReferences(context.Factory())
	receiver, method, err := selectionvalue.MethodExpressionReceiver(
		context,
		children,
		source,
		selected,
		api.DirectExpression(parameters[0]),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Reference(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := append(
		[]tsgo.Expression{receiver.Value()},
		parameters[1:]...,
	)
	call := context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	var body tsgo.ConciseBody = call
	if len(receiver.Before()) != 0 {
		statements := receiver.Before()
		if signature.Results().Len() == 0 {
			statements = append(
				statements,
				context.Factory().ExpressionStatement(call),
			)
		} else {
			statements = append(
				statements,
				context.Factory().ReturnStatement(call),
			)
		}
		body = context.Factory().Block(statements, true)
	}
	return api.DirectExpression(
		context.Factory().ArrowFunction(
			nil,
			nil,
			targetSignature.Parameters(),
			targetSignature.Result(),
			context.Factory().EqualsGreaterThanToken(),
			body,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			reference.Requests(),
		)...,
	), nil
}
