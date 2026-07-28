package slicing

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitArray(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
) (api.ExpressionEmission, bool, error) {
	if source == nil || source.X == nil {
		return api.ExpressionEmission{}, false, nil
	}
	operandType := context.TypesInfo().TypeOf(source.X)
	array, arrayOK := arrayvalue.Resolve(context, operandType)
	pointerElement, pointerModel, pointerOK := arrayPointer(operandType)
	if pointerOK {
		array, arrayOK = arrayvalue.Resolve(context, pointerElement)
	}
	if !arrayOK {
		return api.ExpressionEmission{}, false, nil
	}
	if err := validateArraySliceResult(context, source, array.ElementType()); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var receiver api.ExpressionEmission
	var err error
	if pointerOK {
		receiver, err = pointerArrayStorage(
			context,
			children,
			source,
			operandType,
			pointerElement,
			pointerModel,
		)
	} else {
		receiver, err = arrayStorage(
			context,
			children,
			source,
			operandType,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	low, err := emitBound(
		context.WithRole(api.RoleSliceLow),
		children,
		source.Low,
		context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	high, err := emitBound(
		context.WithRole(api.RoleSliceHigh),
		children,
		source.High,
		context.Factory().NullLiteral(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	max := expressionoperands.Omitted(context.Factory().NullLiteral())
	if source.Slice3 {
		max, err = emitBound(
			context.WithRole(api.RoleSliceMax),
			children,
			source.Max,
			nil,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		expressionoperands.Present(receiver),
		low,
		high,
		max,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeArraySlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	returnValue, err := api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			ordered.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			ordered.Requests(),
			reference.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	returnValue, err = slicevalue.Wrap(
		context,
		context.ExpectedType(),
		returnValue,
	)
	return returnValue, true, err
}

func validateArraySliceResult(
	context api.Context,
	source *ast.SliceExpr,
	elementType types.Type,
) error {
	resultType := context.TypesInfo().TypeOf(source)
	resultSlice, ok := types.Unalias(resultType).(*types.Slice)
	if !ok ||
		!types.Identical(resultSlice.Elem(), elementType) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) ||
		(source.Slice3 && source.Max == nil) {
		return api.Unsupported(context, api.CategoryExpression, source)
	}
	return nil
}

func arrayStorage(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
	operandType types.Type,
) (api.ExpressionEmission, error) {
	value, err := children.Expression(
		context.
			WithRole(api.RoleSliceOperand).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source.X,
		operandType,
		value,
	)
}

func pointerArrayStorage(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
	operandType types.Type,
	elementType types.Type,
	definedPointer definedtype.Model,
) (api.ExpressionEmission, error) {
	value, err := children.Expression(
		context.
			WithRole(api.RoleSliceOperand).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedPointer.Type() != nil {
		value, err = definedPointer.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	logicalType, err := children.RepresentedType(
		context.WithRole(api.RoleSliceOperand),
		source.X,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source.X,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			reference.Name(),
			logicalType.Value(),
			storageType.Value(),
			value.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			logicalType.Requests(),
			storageType.Requests(),
			reference.Requests(),
		),
	)
}

func arrayPointer(
	sourceType types.Type,
) (types.Type, definedtype.Model, bool) {
	if _, elementType, ok := pointertype.Resolve(sourceType); ok {
		return elementType, definedtype.Model{}, true
	}
	definedPointer, ok := definedtype.ResolvePointer(sourceType)
	if !ok {
		return nil, definedtype.Model{}, false
	}
	pointer, ok := definedPointer.Pointer()
	if !ok {
		return nil, definedtype.Model{}, false
	}
	return pointer.Elem(), definedPointer, true
}
