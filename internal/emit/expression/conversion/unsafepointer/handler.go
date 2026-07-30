package unsafepointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unsafepointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafepointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
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
	sourceUnsafe := basictype.SupportsUnsafePointer(sourceType)
	targetUnsafe := basictype.SupportsUnsafePointer(targetType)
	_, sourceDefined, sourcePointerOK := resolvePointer(sourceType)
	targetPointer, targetDefined, targetPointerOK := resolvePointer(targetType)
	if !types.ConvertibleTo(sourceType, targetType) ||
		sourceUnsafe == targetUnsafe ||
		!sourceUnsafe && !sourcePointerOK ||
		!targetUnsafe && !targetPointerOK {
		return api.ExpressionEmission{}, false, nil
	}
	reference, err := context.Names().Runtime(
		api.RuntimeUnsafePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if targetUnsafe {
		if sourceDefined.Type() != nil {
			value, err = sourceDefined.Project(context, value)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
		target, targetErr := api.NewExpressionEmission(
			value.Before(),
			call(
				context,
				reference.Name(),
				unsafepointerruntime.FromName,
				nil,
				value.Value(),
			),
			api.CombineRequests(value.Requests(), reference.Requests()),
		)
		return target, true, targetErr
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
	target, err := api.NewExpressionEmission(
		value.Before(),
		call(
			context,
			reference.Name(),
			unsafepointerruntime.ToName,
			[]tsgo.TypeNode{
				targetLogical.Value(),
				targetStorage.Value(),
			},
			value.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			reference.Requests(),
			targetLogical.Requests(),
			targetStorage.Requests(),
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

func resolvePointer(
	sourceType types.Type,
) (*types.Pointer, definedtype.Model, bool) {
	if pointer, ok := types.Unalias(sourceType).(*types.Pointer); ok {
		if defined, definedOK := definedtype.ResolvePointer(sourceType); definedOK {
			return pointer, defined, true
		}
		return pointer, definedtype.Model{}, true
	}
	return nil, definedtype.Model{}, false
}

func call(
	context api.Context,
	runtimeName string,
	memberName string,
	typeArguments []tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(runtimeName),
			nil,
			context.Factory().Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		typeArguments,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}
