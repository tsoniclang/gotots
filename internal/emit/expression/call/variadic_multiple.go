package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
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
	emissions := make([]api.ExpressionEmission, 0, fixedCount+1)
	variadicValues := make(
		[]api.ExpressionEmission,
		0,
		results.Len()-fixedCount,
	)
	requests := capture.Requests()
	for index := range results.Len() {
		expected := variadicType.Elem()
		if index < fixedCount {
			expected = signature.Params().At(index).Type()
		}
		element, err := capture.Element(context, index)
		if err != nil {
			return nil, nil, nil, err
		}
		copied, err := context.Values().Transfer(
			context.WithRole(api.RoleCallArgument),
			source.Args[0],
			results.At(index).Type(),
			expected,
			api.ValueTransferCopy,
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
			append(capture.Statements(), captured...),
			api.CombineRequests(requests, capturedRequests),
			err
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	for _, emission := range emissions {
		arguments = append(arguments, emission.Value())
		requests = append(requests, emission.Requests()...)
	}
	return arguments, capture.Statements(), requests, nil
}
