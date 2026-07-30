package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
)

func indexed(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	element types.Type,
) (api.ExpressionEmission, error) {
	receiverType := context.TypesInfo().TypeOf(source.X)
	if array, ok := arrayvalue.Resolve(context, receiverType); ok {
		return arrayIndex(
			context,
			children,
			source,
			receiverType,
			array,
			element,
			false,
			definedtype.Model{},
		)
	}
	if _, pointedType, pointerOK := pointertype.Resolve(receiverType); pointerOK {
		if array, arrayOK := arrayvalue.Resolve(context, pointedType); arrayOK {
			return arrayIndex(
				context,
				children,
				source,
				pointedType,
				array,
				element,
				true,
				definedtype.Model{},
			)
		}
	}
	if defined, pointerOK := definedtype.ResolvePointer(receiverType); pointerOK {
		pointer, _ := defined.Pointer()
		pointedType := pointer.Elem()
		if array, arrayOK := arrayvalue.Resolve(context, pointedType); arrayOK {
			return arrayIndex(
				context,
				children,
				source,
				pointedType,
				array,
				element,
				true,
				defined,
			)
		}
	}
	if _, sliceElement, ok := slicevalue.Resolve(
		receiverType,
	); ok && types.Identical(sliceElement, element) {
		return sliceIndex(context, children, source, receiverType, element)
	}
	if defined, ok := definedtype.ResolveSlice(receiverType); ok {
		sliceType, _ := defined.Slice()
		if types.Identical(sliceType.Elem(), element) {
			return sliceIndex(
				context,
				children,
				source,
				receiverType,
				element,
			)
		}
	}
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, source)
}

func arrayIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	arrayType types.Type,
	array arrayvalue.RuntimeArray,
	element types.Type,
	throughPointer bool,
	definedPointer definedtype.Model,
) (api.ExpressionEmission, error) {
	if !types.Identical(array.ElementType(), element) ||
		!integeroperand.Supports(
			context.TypesSizes(),
			context.TypesInfo().TypeOf(source.Index),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var parent api.ExpressionEmission
	var err error
	if throughPointer {
		expectedType := types.Type(types.NewPointer(arrayType))
		if definedPointer.Type() != nil {
			expectedType = definedPointer.Type()
		}
		parent, err = children.Expression(
			context.
				WithRole(api.RoleArrayReceiver).
				WithExpectedType(expectedType),
			source.X,
		)
		if err == nil && definedPointer.Type() != nil {
			parent, err = definedPointer.Project(context, parent)
		}
	} else {
		parent, err = children.Address(
			context.
				WithRole(api.RoleArrayReceiver).
				WithExpectedType(types.NewPointer(arrayType)),
			source.X,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := integeroperand.Emit(
		context.WithRole(api.RoleArrayIndex),
		children,
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if parameter, generic := api.GenericTypeParameter(element); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationIndexAddress,
			[]types.Type{
				types.NewPointer(arrayType),
				types.Typ[types.Int],
			},
			[]types.Type{types.NewPointer(parameter)},
			[]api.ExpressionEmission{parent, index},
		)
	}
	return array.Address(
		context,
		children,
		source,
		parent,
		index,
		throughPointer,
	)
}

func sliceIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	receiverType types.Type,
	element types.Type,
) (api.ExpressionEmission, error) {
	indexType := context.TypesInfo().TypeOf(source.Index)
	if !integeroperand.Supports(context.TypesSizes(), indexType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceReceiver).
			WithExpectedType(receiverType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if defined, ok := definedtype.ResolveSlice(receiverType); ok {
		receiver, err = defined.Project(context, receiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	index, err := integeroperand.Emit(
		context.WithRole(api.RoleSliceIndex),
		children,
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if parameter, generic := api.GenericTypeParameter(element); generic {
		return genericoperation.Call(
			context,
			source,
			api.GenericOperationIndexAddress,
			[]types.Type{
				types.NewSlice(parameter),
				types.Typ[types.Int],
			},
			[]types.Type{types.NewPointer(parameter)},
			[]api.ExpressionEmission{receiver, index},
		)
	}
	return slicevalue.Address(
		context,
		children,
		source,
		element,
		receiver,
		index,
	)
}
