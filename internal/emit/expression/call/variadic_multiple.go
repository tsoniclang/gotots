package call

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitVariadicMultipleArgument(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	results *types.Tuple,
	captureAll bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	fixedCount := signature.Params().Len() - 1
	variadicType, ok := types.Unalias(
		signature.Params().At(fixedCount).Type(),
	).(*types.Slice)
	if source.Ellipsis != token.NoPos ||
		!ok ||
		results == nil ||
		results.Len() < fixedCount {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	for index := range results.Len() {
		expected := variadicType.Elem()
		if index < fixedCount {
			expected = signature.Params().At(index).Type()
		}
		if !types.AssignableTo(results.At(index).Type(), expected) {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedResults(results),
		source.Args[0],
	)
	if err != nil {
		return nil, nil, nil, err
	}
	temporaryName, err := context.Names().Temporary(
		api.TemporaryMultipleResults,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	before := append([]tsgo.Statement(nil), value.Before()...)
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(temporaryName),
				nil,
				nil,
				value.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	emissions := make([]api.ExpressionEmission, 0, fixedCount+1)
	variadicValues := make(
		[]api.ExpressionEmission,
		0,
		results.Len()-fixedCount,
	)
	requests := value.Requests()
	for index := range results.Len() {
		expected := variadicType.Elem()
		if index < fixedCount {
			expected = signature.Params().At(index).Type()
		}
		element := context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporaryName),
			nil,
			context.Factory().NumericLiteral(
				strconv.Itoa(index),
				tsgo.TokenFlagsNone,
			),
			tsgo.NodeFlagsNone,
		)
		copied, err := context.Values().Copy(
			context.WithRole(api.RoleCallArgument),
			source.Args[0],
			expected,
			api.DirectExpression(element),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if index < fixedCount {
			emissions = append(emissions, copied)
		} else {
			variadicValues = append(variadicValues, copied)
		}
	}
	packed, err := emitVariadicSlice(
		context,
		children,
		source,
		variadicType.Elem(),
		variadicValues,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	emissions = append(emissions, packed)
	if captureAll || variadicEmissionsNeedCapture(emissions) {
		arguments, captured, capturedRequests, err := captureArguments(
			context,
			children,
			source,
			signature,
			emissions,
		)
		return arguments,
			append(before, captured...),
			api.CombineRequests(requests, capturedRequests),
			err
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	for _, emission := range emissions {
		arguments = append(arguments, emission.Value())
		requests = append(requests, emission.Requests()...)
	}
	return arguments, before, requests, nil
}
