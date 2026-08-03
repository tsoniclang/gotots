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
	sourcePointer, sourceDefined, sourcePointerOK := resolvePointer(sourceType)
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
	if sourceUnsafe {
		if sourceModel, defined := definedtype.ResolveBasic(sourceType); defined {
			value, err = sourceModel.Project(context, value)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
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
		if targetErr == nil {
			target, targetErr = wrapDefinedUnsafe(context, targetType, target)
		}
		return target, true, targetErr
	}
	if targetUnsafe {
		if sourceDefined.Type() != nil {
			value, err = sourceDefined.Project(context, value)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
		logical, storage, codec, pointerRequests, typeErr := pointerContract(
			context,
			children,
			sourcePointer,
		)
		if typeErr != nil {
			return api.ExpressionEmission{}, true, typeErr
		}
		target, targetErr := api.NewExpressionEmission(
			value.Before(),
			call(
				context,
				reference.Name(),
				unsafepointerruntime.FromName,
				[]tsgo.TypeNode{logical, storage},
				value.Value(),
				codec.Expression(context.Factory()),
			),
			api.CombineRequests(
				value.Requests(),
				reference.Requests(),
				codec.Requests(),
				pointerRequests,
			),
		)
		if targetErr == nil {
			target, targetErr = wrapDefinedUnsafe(context, targetType, target)
		}
		return target, true, targetErr
	}
	logical, storage, codec, pointerRequests, err := pointerContract(
		context,
		children,
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
			[]tsgo.TypeNode{logical, storage},
			value.Value(),
			codec.Expression(context.Factory()),
		),
		api.CombineRequests(
			value.Requests(),
			reference.Requests(),
			codec.Requests(),
			pointerRequests,
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

func Prepare(
	context api.Context,
	sourceType types.Type,
	targetType types.Type,
) error {
	sourceUnsafe := basictype.SupportsUnsafePointer(sourceType)
	targetUnsafe := basictype.SupportsUnsafePointer(targetType)
	sourcePointer, _, sourcePointerOK := resolvePointer(sourceType)
	targetPointer, _, targetPointerOK := resolvePointer(targetType)
	var selected *types.Pointer
	switch {
	case targetUnsafe && sourcePointerOK:
		selected = sourcePointer
	case sourceUnsafe && targetPointerOK:
		selected = targetPointer
	default:
		return nil
	}
	observation, err := pointertype.Observe(context, selected, true)
	if err != nil {
		return err
	}
	if !observation.Representation().Valid() {
		return &api.InvariantError{
			Role:   context.Role(),
			Reason: "unsafe conversion selected an invalid pointer representation",
		}
	}
	return nil
}

func pointerContract(
	context api.Context,
	children api.ChildEmitter,
	pointer *types.Pointer,
) (tsgo.TypeNode, tsgo.TypeNode, api.NameReference, []api.RootRequest, error) {
	if pointer == nil {
		return nil, nil, api.NameReference{}, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "unsafe conversion pointer contract is absent",
		}
	}
	observation, err := pointertype.Observe(context, pointer, true)
	if err != nil {
		return nil, nil, api.NameReference{}, nil, err
	}
	logical, err := children.RepresentedType(
		context.WithRole(api.RoleConversionOperand),
		nil,
		pointer.Elem(),
	)
	if err != nil {
		return nil, nil, api.NameReference{}, nil, err
	}
	storage, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		nil,
		pointer.Elem(),
		observation,
	)
	if err != nil {
		return nil, nil, api.NameReference{}, nil, err
	}
	names, ok := context.Names().(api.UnsafeCodecNames)
	if !ok {
		return nil, nil, api.NameReference{}, nil, &api.ContextError{
			Reason: "unsafe-codec names are unavailable",
		}
	}
	codec, err := names.UnsafeCodec(pointer.Elem())
	if err != nil {
		return nil, nil, api.NameReference{}, nil, err
	}
	return logical.Value(), storage.Value(), codec, api.CombineRequests(
		observation.Requests(),
		logical.Requests(),
		storage.Requests(),
	), nil
}

func wrapDefinedUnsafe(
	context api.Context,
	targetType types.Type,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	model, defined := definedtype.ResolveBasic(targetType)
	if !defined {
		return target, nil
	}
	return model.Wrap(context, target)
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
		return pointer, definedtype.Model{}, true
	}
	if defined, definedOK := definedtype.ResolvePointer(sourceType); definedOK {
		pointer, pointerOK := defined.Pointer()
		return pointer, defined, pointerOK
	}
	return nil, definedtype.Model{}, false
}

func call(
	context api.Context,
	runtimeName string,
	memberName string,
	typeArguments []tsgo.TypeNode,
	values ...tsgo.Expression,
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
		values,
		tsgo.NodeFlagsNone,
	)
}
