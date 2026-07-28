package index

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapindexexpression "github.com/tsoniclang/gotots/internal/emit/expression/mapindex"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IndexExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	if _, ok := types.Unalias(operandType).(*types.Map); ok {
		return mapindexexpression.Emit(context, children, source)
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
	if len(operand.Before()) != 0 || len(index.Before()) != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Runtime(
		api.RuntimeStringIndex,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target := tsgo.Expression(context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		nil,
		[]tsgo.Expression{operand.Value(), index.Value()},
		tsgo.NodeFlagsNone,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		target = context.Factory().CallExpression(
			context.Factory().Identifier("BigInt"),
			nil,
			nil,
			[]tsgo.Expression{target},
			tsgo.NodeFlagsNone,
		)
	}
	return api.DirectExpression(
		target,
		api.CombineRequests(
			operand.Requests(),
			index.Requests(),
			reference.Requests(),
		)...,
	), nil
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
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		address.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			runtime.Name(),
			targetElement.Value(),
			address.Value(),
		),
		api.CombineRequests(
			address.Requests(),
			targetElement.Requests(),
			runtime.Requests(),
		),
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
	if definedOK {
		receiver, err = defined.Project(context, receiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
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
	targetReceiver := receiver.Value()
	before := receiver.Before()
	if len(index.Before()) != 0 {
		name, err := context.Names().Temporary(api.TemporarySliceReceiver)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						targetReceiver,
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		targetReceiver = context.Factory().Identifier(name)
	}
	before = append(before, index.Before()...)
	target := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			targetReceiver,
			nil,
			context.Factory().Identifier(
				runtimeslice.MemberName(runtimeslice.MemberGet),
			),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{index.Value()},
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		target,
		api.CombineRequests(receiver.Requests(), index.Requests()),
	)
}
