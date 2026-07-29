package methodvalue

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
	if selected == nil || selected.Kind() != types.MethodVal {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := selected.Type().(*types.Signature)
	if !ok || !callable.Supports(signature) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, method, err := selectionvalue.MethodReceiver(
		context,
		children,
		source,
		selected,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
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
	arguments := targetSignature.ParameterReferences(context.Factory())
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						receiver.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	reference, err := context.Names().Reference(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	callArguments := append(
		[]tsgo.Expression{context.Factory().Identifier(receiverName)},
		arguments...,
	)
	call := context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		nil,
		callArguments,
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			nil,
			nil,
			targetSignature.Parameters(),
			targetSignature.Result(),
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			reference.Requests(),
		),
	)
}
