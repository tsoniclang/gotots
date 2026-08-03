package stringvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func stringToSlice(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	targetSlice *types.Slice,
	targetElement types.Type,
	kind sliceKind,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if targetSlice == nil ||
		targetElement == nil ||
		(kind != sliceBytes && kind != sliceRunes) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	element, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElementType),
		source,
		targetElement,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		source,
		targetElement,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimeSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	textName, resultName, indexName, err := conversionNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	text := context.Factory().Identifier(textName)
	result := context.Factory().Identifier(resultName)
	index := context.Factory().Identifier(indexName)
	before := append(operand.Before(), zero.Before()...)
	before = append(
		before,
		variable(context, tsgo.NodeFlagsConst, textName, operand.Value()),
	)
	if kind == sliceBytes {
		before = append(
			before,
			variable(
				context,
				tsgo.NodeFlagsConst,
				resultName,
				makeSlice(
					context,
					runtime.Name(),
					element.Value(),
					property(context, text, "length"),
					context.Factory().NullLiteral(),
					zero.Value(),
				),
			),
			forLoop(
				context,
				index,
				property(context, text, "length"),
				[]tsgo.Statement{
					context.Factory().ExpressionStatement(callMember(
						context,
						result,
						runtimeslice.MemberName(runtimeslice.MemberSet),
						index,
						targetInteger(
							context,
							callMember(context, text, "charCodeAt", index),
						),
					)),
				},
			),
		)
		return api.NewExpressionEmission(
			before,
			result,
			api.CombineRequests(
				operand.Requests(),
				element.Requests(),
				zero.Requests(),
				runtime.Requests(),
			),
		)
	}
	return decodeStringRunes(
		context,
		text,
		result,
		resultName,
		index,
		indexName,
		element,
		zero,
		runtime,
		before,
		operand.Requests(),
	)
}

func decodeStringRunes(
	context api.Context,
	text tsgo.Identifier,
	result tsgo.Identifier,
	resultName string,
	offset tsgo.Identifier,
	offsetName string,
	element api.TypeEmission,
	zero api.ExpressionEmission,
	runtime api.NameReference,
	before []tsgo.Statement,
	operandRequests []api.RootRequest,
) (api.ExpressionEmission, error) {
	decodedName, err := context.Names().Temporary(
		api.TemporaryConversionOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	decoder, err := context.Names().Runtime(
		api.RuntimeStringDecodeRune,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	decoded := context.Factory().Identifier(decodedName)
	decodedRune := context.Factory().ElementAccessExpression(
		decoded,
		nil,
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
	decodedWidth := context.Factory().ElementAccessExpression(
		decoded,
		nil,
		context.Factory().NumericLiteral("1", tsgo.TokenFlagsNone),
		tsgo.NodeFlagsNone,
	)
	before = append(
		before,
		variable(
			context,
			tsgo.NodeFlagsLet,
			resultName,
			makeSlice(
				context,
				runtime.Name(),
				element.Value(),
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
				property(context, text, "length"),
				zero.Value(),
			),
		),
		variable(
			context,
			tsgo.NodeFlagsLet,
			offsetName,
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
		),
		context.Factory().WhileStatement(
			binary(
				context,
				offset,
				tsgo.BinaryOperatorLessThanToken,
				property(context, text, "length"),
			),
			context.Factory().Block([]tsgo.Statement{
				variable(
					context,
					tsgo.NodeFlagsConst,
					decodedName,
					context.Factory().CallExpression(
						context.Factory().Identifier(decoder.Name()),
						nil,
						nil,
						[]tsgo.Expression{text, offset},
						tsgo.NodeFlagsNone,
					),
				),
				context.Factory().ExpressionStatement(binary(
					context,
					result,
					tsgo.BinaryOperatorEqualsToken,
					callMember(
						context,
						result,
						runtimeslice.MemberName(runtimeslice.MemberAppend),
						zero.Value(),
						context.Factory().ArrayLiteralExpression(
							[]tsgo.Expression{
								targetInteger(context, decodedRune),
							},
							false,
						),
					),
				)),
				context.Factory().ExpressionStatement(binary(
					context,
					offset,
					tsgo.BinaryOperatorPlusEqualsToken,
					decodedWidth,
				)),
			}, true),
		),
	)
	return api.NewExpressionEmission(
		before,
		result,
		api.CombineRequests(
			operandRequests,
			element.Requests(),
			zero.Requests(),
			runtime.Requests(),
			decoder.Requests(),
		),
	)
}

func makeSlice(
	context api.Context,
	runtimeName string,
	elementType tsgo.TypeNode,
	length tsgo.Expression,
	capacity tsgo.Expression,
	zero tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		property(
			context,
			context.Factory().Identifier(runtimeName),
			runtimeslice.MemberName(runtimeslice.MemberMake),
		),
		nil,
		[]tsgo.TypeNode{elementType},
		[]tsgo.Expression{length, capacity, zero},
		tsgo.NodeFlagsNone,
	)
}

func targetInteger(
	context api.Context,
	value tsgo.Expression,
) tsgo.Expression {
	if context.IntegerRepresentation() != api.IntegerRepresentationBigInt {
		return value
	}
	return context.Factory().CallExpression(
		api.TargetIntrinsicBigInt.Expression(context.Factory()),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
