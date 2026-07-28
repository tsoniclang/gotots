package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitVariadicArguments(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	captureAll bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if source == nil || signature == nil || !signature.Variadic() {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	parameterCount := signature.Params().Len()
	fixedCount := parameterCount - 1
	if fixedCount < 0 || len(source.Args) < fixedCount {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if source.Ellipsis != token.NoPos &&
		len(source.Args) != fixedCount+1 {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emissions := make([]api.ExpressionEmission, 0, parameterCount)
	for index := 0; index < fixedCount; index++ {
		emission, err := emitVariadicArgument(
			context,
			children,
			source.Args[index],
			signature.Params().At(index).Type(),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		emissions = append(emissions, emission)
	}
	variadicParameter := signature.Params().At(fixedCount)
	variadicType, ok := types.Unalias(variadicParameter.Type()).(*types.Slice)
	if !ok {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if source.Ellipsis != token.NoPos {
		emission, err := emitVariadicArgument(
			context,
			children,
			source.Args[fixedCount],
			variadicParameter.Type(),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		emissions = append(emissions, emission)
	} else {
		values := make([]api.ExpressionEmission, 0, len(source.Args)-fixedCount)
		for _, argument := range source.Args[fixedCount:] {
			emission, err := emitVariadicArgument(
				context,
				children,
				argument,
				variadicType.Elem(),
			)
			if err != nil {
				return nil, nil, nil, err
			}
			values = append(values, emission)
		}
		emission, err := emitVariadicSlice(
			context,
			children,
			source,
			variadicType.Elem(),
			values,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		emissions = append(emissions, emission)
	}
	if captureAll || variadicEmissionsNeedCapture(emissions) {
		return captureArguments(context, children, source, signature, emissions)
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var requests []api.RootRequest
	for _, emission := range emissions {
		arguments = append(arguments, emission.Value())
		requests = append(requests, emission.Requests()...)
	}
	return arguments, nil, requests, nil
}

func emitVariadicArgument(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	expected types.Type,
) (api.ExpressionEmission, error) {
	actual := context.TypesInfo().TypeOf(source)
	if actual == nil || !types.AssignableTo(actual, expected) {
		return api.ExpressionEmission{}, api.Unsupported(
			context.WithRole(api.RoleCallArgument),
			api.CategoryExpression,
			source,
		)
	}
	emission, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(expected),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().Copy(
		context.WithRole(api.RoleCallArgument),
		source,
		expected,
		emission,
	)
}

func variadicEmissionsNeedCapture(
	emissions []api.ExpressionEmission,
) bool {
	for _, emission := range emissions {
		if len(emission.Before()) != 0 {
			return true
		}
	}
	return false
}

func emitVariadicSlice(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	elementType types.Type,
	values []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	element, err := children.RepresentedType(
		context.WithRole(api.RoleCallArgument),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var arguments []tsgo.Expression
	var before []tsgo.Statement
	var requests []api.RootRequest
	if variadicEmissionsNeedCapture(values) {
		var valueRequests []api.RootRequest
		arguments, before, valueRequests, err = captureArguments(
			context,
			children,
			source,
			nil,
			values,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		requests = api.CombineRequests(requests, valueRequests)
	} else {
		arguments = make([]tsgo.Expression, 0, len(values))
		for _, value := range values {
			arguments = append(arguments, value.Value())
			requests = append(requests, value.Requests()...)
		}
	}
	aggregate := context.Values().RequiresStructuralCopy(context, elementType)
	var zeroValue tsgo.Expression
	var zeroRequests []api.RootRequest
	if aggregate {
		zeroFactory, requestsForZero, err := slicevalue.AggregateZeroFactory(
			context,
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		zeroValue = zeroFactory
		zeroRequests = requestsForZero
	} else {
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleCallArgument),
			source,
			elementType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(zero.Before()) != 0 {
			return api.ExpressionEmission{}, api.Unsupported(
				context,
				api.CategoryExpression,
				source,
			)
		}
		zeroValue = zero.Value()
		zeroRequests = zero.Requests()
	}
	var runtimeSymbol api.RuntimeSymbol
	var member runtimeslice.Member
	switch {
	case aggregate && len(values) == 0:
		runtimeSymbol = api.RuntimeSliceNilWith
	case aggregate:
		runtimeSymbol = api.RuntimeSliceLiteralWith
	case len(values) == 0:
		runtimeSymbol = api.RuntimeSlice
		member = runtimeslice.MemberNil
	default:
		runtimeSymbol = api.RuntimeSlice
		member = runtimeslice.MemberLiteral
	}
	runtime, err := context.Names().Runtime(
		runtimeSymbol,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments = append([]tsgo.Expression{zeroValue}, arguments...)
	var target tsgo.Expression
	if aggregate {
		target = context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			[]tsgo.TypeNode{element.Value()},
			arguments,
			tsgo.NodeFlagsNone,
		)
	} else {
		target = context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(runtimeslice.MemberName(member)),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{element.Value()},
			arguments,
			tsgo.NodeFlagsNone,
		)
	}
	return api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(
			requests,
			zeroRequests,
			element.Requests(),
			runtime.Requests(),
		),
	)
}
