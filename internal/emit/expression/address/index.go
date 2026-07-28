package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
		!basictype.SupportsInteger(
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
	indexType := context.TypesInfo().TypeOf(source.Index)
	index, err := children.Expression(
		context.
			WithRole(api.RoleArrayIndex).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	parentValue, before, err := captureBefore(
		context,
		parent,
		index.Before(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementTarget, err := children.RepresentedType(
		context.WithRole(api.RoleArrayIndex),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayTarget, err := array.EmitType(
		context.WithRole(api.RoleArrayReceiver),
		children,
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arrayStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source.X,
		arrayType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if throughPointer {
		parentValue = context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(
					pointerruntime.DereferenceName,
				),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				arrayTarget.Value(),
				arrayStorage.Value(),
			},
			[]tsgo.Expression{parentValue},
			tsgo.NodeFlagsNone,
		)
	}
	method := pointerruntime.IndexName
	elementStorage := elementTarget
	typeArguments := []tsgo.TypeNode{
		elementTarget.Value(),
		elementStorage.Value(),
		arrayTarget.Value(),
		arrayStorage.Value(),
	}
	arguments := []tsgo.Expression{parentValue, index.Value()}
	var projectionRequests []api.RootRequest
	if context.Values().RequiresStorageProjection(context, element) {
		projection, err := buildStorageProjection(
			context,
			source,
			element,
			elementTarget,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		method = pointerruntime.IndexViewName
		elementStorage = projection.storage
		typeArguments = []tsgo.TypeNode{
			elementTarget.Value(),
			elementStorage.Value(),
			arrayTarget.Value(),
			elementTarget.Value(),
			arrayStorage.Value(),
		}
		arguments = append(
			arguments,
			projection.toStorage,
			projection.fromStorage,
		)
		projectionRequests = projection.requests
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(method),
				tsgo.NodeFlagsNone,
			),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			parent.Requests(),
			index.Requests(),
			elementTarget.Requests(),
			elementStorage.Requests(),
			arrayTarget.Requests(),
			arrayStorage.Requests(),
			runtime.Requests(),
			projectionRequests,
		),
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
	if !basictype.SupportsInteger(context.TypesSizes(), indexType) {
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
	index, err := children.Expression(
		context.
			WithRole(api.RoleSliceIndex).
			WithExpectedType(indexType),
		source.Index,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverValue, before, err := captureBefore(
		context,
		receiver,
		index.Before(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	elementTarget, err := children.RepresentedType(
		context.WithRole(api.RoleSliceElement),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtimeSymbol := api.RuntimeSliceAddress
	typeArguments := []tsgo.TypeNode{elementTarget.Value()}
	arguments := []tsgo.Expression{receiverValue, index.Value()}
	var projectionRequests []api.RootRequest
	if context.Values().RequiresStorageProjection(context, element) {
		projection, err := buildStorageProjection(
			context,
			source,
			element,
			elementTarget,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		runtimeSymbol = api.RuntimeSliceAddressView
		typeArguments = []tsgo.TypeNode{
			elementTarget.Value(),
			projection.storage.Value(),
			elementTarget.Value(),
		}
		arguments = append(
			arguments,
			projection.toStorage,
			projection.fromStorage,
		)
		projectionRequests = projection.requests
	}
	runtime, err := context.Names().Runtime(
		runtimeSymbol,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(runtime.Name()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			index.Requests(),
			elementTarget.Requests(),
			runtime.Requests(),
			projectionRequests,
		),
	)
}
