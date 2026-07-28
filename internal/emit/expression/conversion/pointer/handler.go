package pointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Convert(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	sourceType types.Type,
	targetType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	sourcePointer, sourceDefined, sourceOK := resolve(sourceType)
	targetPointer, targetDefined, targetOK := resolve(targetType)
	if !sourceOK || !targetOK {
		return api.ExpressionEmission{}, false, nil
	}
	if !types.ConvertibleTo(sourceType, targetType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var err error
	if sourceDefined.Type() != nil {
		value, err = sourceDefined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	sourceLogical, err := children.RepresentedType(
		context.WithRole(api.RoleConversionOperand),
		source.Args[0],
		sourcePointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetLogical, err := children.RepresentedType(
		context.WithRole(api.RoleConversionOperand),
		source.Fun,
		targetPointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	targetStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		targetPointer.Elem(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.ViewName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				sourceLogical.Value(),
				targetLogical.Value(),
				targetStorage.Value(),
			},
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			value.Requests(),
			sourceLogical.Requests(),
			targetLogical.Requests(),
			targetStorage.Requests(),
			runtime.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if targetDefined.Type() != nil {
		target, err = targetDefined.Wrap(context, target)
	}
	return target, true, err
}

func resolve(sourceType types.Type) (*types.Pointer, definedtype.Model, bool) {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		return pointer, definedtype.Model{}, true
	}
	defined, ok := definedtype.ResolvePointer(sourceType)
	if !ok {
		return nil, definedtype.Model{}, false
	}
	pointer, _ := defined.Pointer()
	return pointer, defined, true
}
