package slicing

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	resultType types.Type,
	receiver api.ExpressionEmission,
	bounds []api.ExpressionEmission,
	full bool,
) (api.ExpressionEmission, bool, error) {
	wantBounds := 2
	if full {
		wantBounds = 3
	}
	if sourceType == nil || resultType == nil || len(bounds) != wantBounds {
		return api.ExpressionEmission{}, false, nil
	}
	if _, _, ok := slicevalue.Source(sourceType); ok &&
		types.Identical(sourceType, resultType) {
		projected, err := slicevalue.Project(context, sourceType, receiver)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		target, err := sliceRuntimeCall(context, projected, bounds, full)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		target, err = slicevalue.Wrap(context, resultType, target)
		return target, true, err
	}
	if model, ok := stringModel(sourceType, resultType, full); ok {
		projected := receiver
		var err error
		if model.Type() != nil {
			projected, err = model.Project(context, receiver)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
		target, err := stringRuntimeCall(context, projected, bounds)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		if model.Type() != nil {
			target, err = model.Wrap(context, target)
		}
		return target, true, err
	}
	array, ok := arrayvalue.Resolve(context, sourceType)
	resultSlice, _, resultOK := slicevalue.Source(resultType)
	if !ok || !resultOK ||
		!types.Identical(array.ElementType(), resultSlice.Elem()) {
		return api.ExpressionEmission{}, false, nil
	}
	storage, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		sourceType,
		receiver,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := arrayRuntimeCall(context, storage, bounds, full)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err = slicevalue.Wrap(context, resultType, target)
	return target, true, err
}

func stringModel(
	sourceType types.Type,
	resultType types.Type,
	full bool,
) (definedtype.Model, bool) {
	if full || !types.Identical(sourceType, resultType) {
		return definedtype.Model{}, false
	}
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basictype.SupportsString(basic) {
		return definedtype.Model{}, true
	}
	model, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return definedtype.Model{}, false
	}
	basic, ok := model.Basic()
	return model, ok && basictype.SupportsString(basic)
}

func sliceRuntimeCall(
	context api.Context,
	receiver api.ExpressionEmission,
	bounds []api.ExpressionEmission,
	full bool,
) (api.ExpressionEmission, error) {
	ordered, err := orderedSliceOperands(context, receiver, bounds)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	arguments := append([]tsgo.Expression(nil), values[1:]...)
	if !full {
		arguments = append(arguments, context.Factory().NullLiteral())
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				values[0],
				nil,
				context.Factory().Identifier(
					runtimeslice.MemberName(runtimeslice.MemberSlice),
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		ordered.Requests(),
	)
}

func stringRuntimeCall(
	context api.Context,
	receiver api.ExpressionEmission,
	bounds []api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	ordered, err := orderedSliceOperands(context, receiver, bounds)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeStringSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			ordered.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(ordered.Requests(), reference.Requests()),
	)
}

func arrayRuntimeCall(
	context api.Context,
	receiver api.ExpressionEmission,
	bounds []api.ExpressionEmission,
	full bool,
) (api.ExpressionEmission, error) {
	ordered, err := orderedSliceOperands(context, receiver, bounds)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	arguments := append([]tsgo.Expression(nil), values...)
	if !full {
		arguments = append(arguments, context.Factory().NullLiteral())
	}
	reference, err := context.Names().Runtime(
		api.RuntimeArraySlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(ordered.Requests(), reference.Requests()),
	)
}

func orderedSliceOperands(
	context api.Context,
	receiver api.ExpressionEmission,
	bounds []api.ExpressionEmission,
) (expressionoperands.Sequence, error) {
	items := make([]expressionoperands.Item, 0, len(bounds)+1)
	items = append(items, expressionoperands.Present(receiver))
	for _, bound := range bounds {
		items = append(items, expressionoperands.Present(bound))
	}
	return expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		items...,
	)
}
