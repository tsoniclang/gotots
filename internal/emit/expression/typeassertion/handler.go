package typeassertion

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeAssertExpr,
) (api.ExpressionEmission, error) {
	sourceType, targetType, results, ok := assertionTypes(context, source)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleConversionOperand).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceTarget, err := children.RepresentedType(
		context,
		source.X,
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
	var resultType tsgo.TypeNode
	var body []tsgo.Statement
	var failureRequests []api.RootRequest
	if results != nil {
		resultType = context.Factory().TupleTypeNode([]tsgo.TypeNode{
			targetTarget.Value(),
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindBooleanKeyword,
			),
		})
		zero, err := context.Values().Zero(
			context,
			source,
			targetType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(zero.Before()) != 0 {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		body = []tsgo.Statement{
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					condition.Value(),
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
						success.Value(),
						context.Factory().TrueLiteral(),
					},
					false,
				),
			),
		}
		failureRequests = zero.Requests()
	} else {
		resultType = targetTarget.Value()
		panicReference, err := context.Names().Runtime(
			api.RuntimePanic,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		body = []tsgo.Statement{
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					condition.Value(),
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
			context.Factory().ReturnStatement(success.Value()),
		}
		failureRequests = panicReference.Requests()
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

func assertionTypes(
	context api.Context,
	source *ast.TypeAssertExpr,
) (types.Type, types.Type, *types.Tuple, bool) {
	if source == nil || source.X == nil || source.Type == nil {
		return nil, nil, nil, false
	}
	sourceType := context.TypesInfo().TypeOf(source.X)
	sourceInterface, ok := interfacetype.Resolve(sourceType)
	if !ok {
		return nil, nil, nil, false
	}
	targetType := context.TypesInfo().TypeOf(source.Type)
	if targetType == nil || !types.AssertableTo(sourceInterface, targetType) {
		return nil, nil, nil, false
	}
	if results := context.ExpectedResults(); results != nil {
		actual, ok := context.TypesInfo().TypeOf(source).(*types.Tuple)
		if !ok ||
			!types.Identical(actual, results) ||
			results.Len() != 2 ||
			!types.Identical(results.At(0).Type(), targetType) ||
			!types.Identical(
				results.At(1).Type(),
				types.Typ[types.Bool],
			) {
			return nil, nil, nil, false
		}
		return sourceType, targetType, results, true
	}
	actual := context.TypesInfo().TypeOf(source)
	expected := context.ExpectedType()
	if actual == nil ||
		expected == nil ||
		!types.Identical(actual, targetType) ||
		!types.AssignableTo(targetType, expected) {
		return nil, nil, nil, false
	}
	return sourceType, targetType, nil, true
}
