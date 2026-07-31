package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
		return packageVariable(context, children, source, variable)
	}
	if name, selected := context.AddressableStorage().Name(
		context,
		variable,
	); selected {
		return api.DirectExpression(context.Factory().Identifier(name)), nil
	}
	var receiverRequest []api.RootRequest
	if receiver, ok := context.ValueReceiver(variable); ok {
		request, err := receiver.CopyRequest()
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		receiverRequest = []api.RootRequest{request}
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
			receiverRequest,
			[]api.RootRequest{requirement},
		),
	)
}

func packageVariable(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	variable *types.Var,
) (api.ExpressionEmission, error) {
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(variable.Type()),
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := context.Names().PackageVariable(variable)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	logicalType, err := children.RepresentedType(
		context.WithRole(api.RoleUnaryOperand),
		source,
		variable.Type(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	state := context.Factory().Identifier(target.StateName())
	field := context.Factory().StringLiteral(
		target.FieldName(),
		tsgo.TokenFlagsNone,
	)
	return api.DirectExpression(
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.ObjectFieldName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{
				logicalType.Value(),
				context.Factory().TypeQueryNode(state, nil),
				context.Factory().LiteralTypeNode(field),
			},
			[]tsgo.Expression{
				state,
				field,
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			target.Requests(),
			logicalType.Requests(),
			runtime.Requests(),
			representation.Requests(),
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
		return packageVariable(context, children, source, variable)
	}
	field, ok := selection.Obj().(*types.Var)
	if !ok ||
		selection.Kind() != types.FieldVal ||
		!types.Identical(field.Type(), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return selectionvalue.FieldAddress(
		context,
		children,
		source,
		selection,
		element,
	)
}
