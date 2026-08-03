package stringvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func integerToString(
	context api.Context,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	encoder, err := context.Names().Runtime(
		api.RuntimeStringEncodeRune,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		operand.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(encoder.Name()),
			nil,
			nil,
			[]tsgo.Expression{operand.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(operand.Requests(), encoder.Requests()),
	)
}

func sliceToString(
	context api.Context,
	source ast.Node,
	sourceSlice *types.Slice,
	sourceElement types.Type,
	kind sliceKind,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if sourceSlice == nil ||
		sourceElement == nil ||
		(kind != sliceBytes && kind != sliceRunes) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceName, resultName, indexName, err := conversionNames(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceValue := context.Factory().Identifier(sourceName)
	result := context.Factory().Identifier(resultName)
	index := context.Factory().Identifier(indexName)
	element := callMember(
		context,
		sourceValue,
		runtimeslice.MemberName(runtimeslice.MemberGet),
		index,
	)
	var encoded tsgo.Expression
	var operationRequests []api.RootRequest
	switch kind {
	case sliceBytes:
		encoded = callMember(
			context,
			api.TargetIntrinsicString.Expression(context.Factory()),
			"fromCharCode",
			context.Factory().CallExpression(
				api.TargetIntrinsicNumber.Expression(context.Factory()),
				nil,
				nil,
				[]tsgo.Expression{element},
				tsgo.NodeFlagsNone,
			),
		)
	case sliceRunes:
		encoder, err := context.Names().Runtime(
			api.RuntimeStringEncodeRune,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		encoded = context.Factory().CallExpression(
			context.Factory().Identifier(encoder.Name()),
			nil,
			nil,
			[]tsgo.Expression{element},
			tsgo.NodeFlagsNone,
		)
		operationRequests = encoder.Requests()
	}
	before := append(
		operand.Before(),
		variable(
			context,
			tsgo.NodeFlagsConst,
			sourceName,
			operand.Value(),
		),
		variable(
			context,
			tsgo.NodeFlagsLet,
			resultName,
			context.Factory().StringLiteral("", tsgo.TokenFlagsNone),
		),
		forLoop(
			context,
			index,
			property(
				context,
				sourceValue,
				runtimeslice.MemberName(runtimeslice.MemberLength),
			),
			[]tsgo.Statement{
				context.Factory().ExpressionStatement(binary(
					context,
					result,
					tsgo.BinaryOperatorPlusEqualsToken,
					encoded,
				)),
			},
		),
	)
	return api.NewExpressionEmission(
		before,
		result,
		api.CombineRequests(operand.Requests(), operationRequests),
	)
}
