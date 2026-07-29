package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapindexexpression "github.com/tsoniclang/gotots/internal/emit/expression/mapindex"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	if _, ok := maprepresentation.Source(context, operandType); ok {
		return mapindexexpression.Emit(context, children, source)
	}
	if api.ContainsGenericTypeParameter(operandType) {
		return emitGeneric(context, children, source, operandType)
	}
	if array, ok := arrayvalue.Resolve(context, operandType); ok {
		return array.EmitIndex(context, children, source)
	}
	if _, pointedType, ok := pointertype.Resolve(operandType); ok {
		if array, arrayOK := arrayvalue.Resolve(context, pointedType); arrayOK {
			return emitPointerArrayIndex(context, children, source, array)
		}
	}
	if defined, ok := definedtype.ResolvePointer(operandType); ok {
		pointer, _ := defined.Pointer()
		if array, arrayOK := arrayvalue.Resolve(
			context,
			pointer.Elem(),
		); arrayOK {
			return emitPointerArrayIndex(context, children, source, array)
		}
	}
	if _, _, ok := slicevalue.Resolve(operandType); ok {
		return emitSliceIndex(context, children, source)
	}
	if _, ok := definedtype.ResolveSlice(operandType); ok {
		return emitSliceIndex(context, children, source)
	}
	indexType := context.TypesInfo().TypeOf(source.Index)
	resultType := context.TypesInfo().TypeOf(source)
	if !basictype.SupportsString(operandType) ||
		!basictype.SupportsStringIndex(context.TypesSizes(), indexType) ||
		!isByte(resultType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(resultType, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleIndexOperand).
			WithExpectedType(types.Typ[types.String]),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	indexExpected := indexType
	if isUntypedInteger(indexExpected) {
		indexExpected = types.Typ[types.Int]
	}
	index, err := children.Expression(
		context.
			WithRole(api.RoleIndexValue).
			WithExpectedType(indexExpected),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, handled, err := Apply(
		context,
		source,
		operandType,
		indexType,
		resultType,
		operand,
		index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return target, nil
}

func emitPointerArrayIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	array arrayvalue.RuntimeArray,
) (api.ExpressionEmission, error) {
	elementType := array.ElementType()
	if !types.Identical(context.TypesInfo().TypeOf(source), elementType) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(elementType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	address, err := children.Address(
		context.
			WithRole(api.RoleIndexOperand).
			WithExpectedType(types.NewPointer(elementType)),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleIndexOperand),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		elementType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := api.NewExpressionEmission(
		address.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			targetElement.Value(),
			storageType.Value(),
			address.Value(),
		),
		api.CombineRequests(
			address.Requests(),
			targetElement.Requests(),
			storageType.Requests(),
			runtime.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().FromStorage(
		context,
		source,
		elementType,
		stored,
	)
}

func isByte(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok && basic.Kind() == types.Uint8
}

func isUntypedInteger(sourceType types.Type) bool {
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Info()&types.IsInteger != 0
}

func emitSliceIndex(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source.X)
	_, elementType, ok := slicevalue.Resolve(sourceType)
	defined, definedOK := definedtype.ResolveSlice(sourceType)
	if definedOK {
		sliceType, _ := defined.Slice()
		elementType = sliceType.Elem()
		ok = true
	}
	if !ok ||
		context.TypesInfo().TypeOf(source) == nil ||
		!types.Identical(context.TypesInfo().TypeOf(source), elementType) ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(elementType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	indexType := context.TypesInfo().TypeOf(source.Index)
	if !basictype.SupportsInteger(context.TypesSizes(), indexType) {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleSliceIndex),
				api.CategoryExpression,
				source.Index,
			)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceReceiver).
			WithExpectedType(sourceType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := children.Expression(
		context.
			WithRole(api.RoleSliceIndex).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, handled, err := Apply(
		context,
		source,
		sourceType,
		indexType,
		elementType,
		receiver,
		index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return target, nil
}

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
	operandType types.Type,
) (api.ExpressionEmission, error) {
	indexType := context.TypesInfo().TypeOf(source.Index)
	resultType := context.TypesInfo().TypeOf(source)
	if indexType == nil ||
		resultType == nil ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleIndexOperand).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	index, err := children.Expression(
		context.
			WithRole(api.RoleIndexValue).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		expressionoperands.Present(operand),
		expressionoperands.Present(index),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := ordered.Values()
	target, err := genericoperation.Call(
		context,
		source,
		api.GenericOperationIndex,
		[]types.Type{operandType, indexType},
		[]types.Type{resultType},
		[]tsgo.Expression{values[0], values[1]},
		ordered.Requests()...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		ordered.Before(),
		target.Value(),
		target.Requests(),
	)
}
