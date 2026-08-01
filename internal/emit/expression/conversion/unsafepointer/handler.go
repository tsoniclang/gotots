package unsafepointer

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unsafepointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/unsafepointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	sourceInteger := uintptrType(sourceType)
	targetInteger := uintptrType(targetType)
	_, sourceDefined, sourcePointerOK := resolvePointer(sourceType)
	targetPointer, targetDefined, targetPointerOK := resolvePointer(targetType)
	if !types.ConvertibleTo(sourceType, targetType) ||
		!sourceUnsafe && !targetUnsafe ||
		sourceUnsafe && !targetPointerOK && !targetInteger ||
		targetUnsafe && !sourcePointerOK && !sourceInteger {
		return api.ExpressionEmission{}, false, nil
	}
	reference, err := context.Names().Runtime(
		api.RuntimeUnsafePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if sourceUnsafe && targetInteger {
		target, targetErr := integerBoundary(
			context,
			reference,
			unsafepointerruntime.ToIntegerName,
			value,
		)
		if targetErr != nil {
			return api.ExpressionEmission{}, true, targetErr
		}
		if targetModel, defined := definedtype.Resolve(targetType); defined {
			target, targetErr = targetModel.Wrap(context, target)
		}
		return target, true, targetErr
	}
	if targetUnsafe && sourceInteger {
		if sourceModel, defined := definedtype.Resolve(sourceType); defined {
			value, err = sourceModel.Project(context, value)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
		target, targetErr := integerBoundary(
			context,
			reference,
			unsafepointerruntime.FromIntegerName,
			value,
		)
		return target, true, targetErr
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
	targetPointerType, err := pointertype.EmitNonNilRepresented(
		context.WithRole(api.RoleConversionOperand),
		children,
		source.Fun,
		targetPointer,
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
				targetPointerType.Value(),
			},
			value.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			reference.Requests(),
			targetPointerType.Requests(),
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

func uintptrType(source types.Type) bool {
	if source == nil {
		return false
	}
	basic, ok := types.Unalias(source).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Uintptr
}

func integerBoundary(
	context api.Context,
	reference api.NameReference,
	member string,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	zero, err := api.IntegerLiteral(
		context.Factory(),
		context.IntegerRepresentation(),
		"0",
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{value.Value(), zero},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
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
