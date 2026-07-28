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

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	pointerType := context.TypesInfo().TypeOf(source)
	if _, unary := source.(*ast.UnaryExpr); !unary {
		pointerType = context.ExpectedType()
	}
	pointer, element, ok := pointertype.Resolve(pointerType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	expected := context.ExpectedType()
	if expected != nil && !types.AssignableTo(pointer, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch source := source.(type) {
	case *ast.UnaryExpr:
		return emitOperand(context, children, source.X, element)
	default:
		return emitOperand(context, children, source, element)
	}
}

func emitOperand(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	element types.Type,
) (api.ExpressionEmission, error) {
	switch source := source.(type) {
	case *ast.Ident:
		return identifier(context, children, source, element)
	case *ast.SelectorExpr:
		return selector(context, children, source, element)
	case *ast.IndexExpr:
		return indexed(context, children, source, element)
	case *ast.CompositeLit:
		return fresh(context, children, source, element)
	case *ast.StarExpr:
		return cancelDereference(context, children, source, element)
	case *ast.ParenExpr:
		return emitOperand(context, children, source.X, element)
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
}

func identifier(
	context api.Context,
	children api.ChildEmitter,
	source *ast.Ident,
	element types.Type,
) (api.ExpressionEmission, error) {
	variable, ok := context.TypesInfo().Uses[source].(*types.Var)
	if !ok ||
		variable.IsField() ||
		!types.Identical(variable.Type(), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if variable.Pkg() != nil &&
		variable.Parent() == variable.Pkg().Scope() {
		return packageVariable(context, variable)
	}
	if name, selected := context.AddressableStorage().Name(
		context,
		variable,
	); selected {
		return api.DirectExpression(context.Factory().Identifier(name)), nil
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(variable.Type()),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	cell, err := context.AddressableStorage().Cell(
		context,
		children,
		source,
		variable.Type(),
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	requirement, err := context.AddressableStorage().Requirement(context, variable)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		cell.Before(),
		cell.Value(),
		api.CombineRequests(
			cell.Requests(),
			[]api.RootRequest{requirement},
		),
	)
}

func packageVariable(
	context api.Context,
	variable *types.Var,
) (api.ExpressionEmission, error) {
	target, err := context.Names().PackageVariable(variable)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.ObjectFieldName),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{
				context.Factory().Identifier(target.StateName()),
				context.Factory().StringLiteral(
					target.FieldName(),
					tsgo.TokenFlagsNone,
				),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			target.Requests(),
			runtime.Requests(),
		)...,
	), nil
}

func selector(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	element types.Type,
) (api.ExpressionEmission, error) {
	selection := context.TypesInfo().Selections[source]
	if selection == nil {
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		packageName, ok := context.TypesInfo().Uses[qualifier].(*types.PkgName)
		variable, variableOK := context.TypesInfo().Uses[source.Sel].(*types.Var)
		if !ok ||
			!variableOK ||
			variable.Pkg() != packageName.Imported() ||
			variable.Parent() != variable.Pkg().Scope() ||
			!types.Identical(variable.Type(), element) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return packageVariable(context, variable)
	}
	field, ok := selection.Obj().(*types.Var)
	if !ok ||
		selection.Kind() != types.FieldVal ||
		field.Embedded() ||
		len(selection.Index()) != 1 ||
		!types.Identical(field.Type(), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var receiver api.ExpressionEmission
	var err error
	if selection.Indirect() {
		receiverType := context.TypesInfo().TypeOf(source.X)
		receiver, err = children.Expression(
			context.
				WithRole(api.RoleFieldReceiver).
				WithExpectedType(receiverType),
			source.X,
		)
		if err == nil {
			receiver, err = dereference(
				context,
				children,
				source.X,
				receiverType,
				receiver,
			)
		}
	} else {
		receiverType := context.TypesInfo().TypeOf(source.X)
		receiver, err = children.Address(
			context.
				WithRole(api.RoleFieldReceiver).
				WithExpectedType(types.NewPointer(receiverType)),
			source.X,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.FieldName),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{
				receiver.Value(),
				context.Factory().StringLiteral(name, tsgo.TokenFlagsNone),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(receiver.Requests(), runtime.Requests()),
	)
}

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
			[]tsgo.TypeNode{arrayTarget.Value()},
			[]tsgo.Expression{parentValue},
			tsgo.NodeFlagsNone,
		)
	}
	requirement, required, err := array.AddressIndexRequirement()
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var requirementRequests []api.RootRequest
	if required {
		requirementRequests = []api.RootRequest{requirement}
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.IndexName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{elementTarget.Value(), arrayTarget.Value()},
			[]tsgo.Expression{parentValue, index.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			parent.Requests(),
			index.Requests(),
			elementTarget.Requests(),
			arrayTarget.Requests(),
			runtime.Requests(),
			requirementRequests,
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
	runtime, err := context.Names().Runtime(
		api.RuntimeSliceAddress,
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
			[]tsgo.TypeNode{elementTarget.Value()},
			[]tsgo.Expression{receiverValue, index.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			index.Requests(),
			elementTarget.Requests(),
			runtime.Requests(),
		),
	)
}
