package operation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	targetType types.Type,
	commaOK bool,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	sourceTarget, err := children.RepresentedType(
		context,
		source,
		sourceType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetTarget, err := children.RepresentedType(
		context,
		source,
		targetType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	value := context.Factory().Identifier("$value")
	condition, err := interfacevalue.Test(
		context,
		sourceType,
		targetType,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	success, err := interfacevalue.Extract(
		context,
		source,
		targetType,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	resultType, body, failureRequests, err := body(
		context,
		source,
		targetType,
		targetTarget.Value(),
		condition.Value(),
		success.Value(),
		commaOK,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := context.Factory().CallExpression(
		context.Factory().ParenthesizedExpression(
			context.Factory().ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					context.Factory().ParameterDeclaration(
						nil,
						nil,
						value,
						nil,
						sourceTarget.Value(),
						nil,
					),
				},
				resultType,
				context.Factory().EqualsGreaterThanToken(),
				context.Factory().Block(body, true),
			),
		),
		nil,
		nil,
		[]tsgo.Expression{receiver.Value()},
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		receiver.Before(),
		target,
		api.CombineRequests(
			receiver.Requests(),
			sourceTarget.Requests(),
			targetTarget.Requests(),
			condition.Requests(),
			success.Requests(),
			failureRequests,
		),
	)
}

func body(
	context api.Context,
	source ast.Node,
	targetType types.Type,
	targetTarget tsgo.TypeNode,
	condition tsgo.Expression,
	success tsgo.Expression,
	commaOK bool,
) (tsgo.TypeNode, []tsgo.Statement, []api.RootRequest, error) {
	if commaOK {
		zero, err := context.Values().Zero(
			context,
			source,
			targetType,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(zero.Before()) != 0 {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return context.Factory().TupleTypeNode([]tsgo.TypeNode{
				targetTarget,
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
			}),
			[]tsgo.Statement{
				context.Factory().IfStatement(
					context.Factory().PrefixUnaryExpression(
						tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
						condition,
					),
					context.Factory().Block(
						[]tsgo.Statement{
							context.Factory().ReturnStatement(
								context.Factory().ArrayLiteralExpression(
									[]tsgo.Expression{
										zero.Value(),
										context.Factory().FalseLiteral(),
									},
									false,
								),
							),
						},
						true,
					),
					nil,
				),
				context.Factory().ReturnStatement(
					context.Factory().ArrayLiteralExpression(
						[]tsgo.Expression{
							success,
							context.Factory().TrueLiteral(),
						},
						false,
					),
				),
			},
			zero.Requests(),
			nil
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return targetTarget,
		[]tsgo.Statement{
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					condition,
				),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ReturnStatement(
							panicruntime.Call(
								context.Factory(),
								panicReference.Name(),
								context.Factory().StringLiteral(
									"runtime error: interface conversion failed",
									tsgo.TokenFlagsNone,
								),
							),
						),
					},
					true,
				),
				nil,
			),
			context.Factory().ReturnStatement(success),
		},
		panicReference.Requests(),
		nil
}
