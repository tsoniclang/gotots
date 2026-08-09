package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	variable, ok := context.TypesInfo().UseOf(source).(*types.Var)
	variableType := context.TypesInfo().TypeOfObject(variable)
	if !ok ||
		variable.IsField() ||
		!types.Identical(variableType, element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if variable.Pkg() != nil &&
		variable.Parent() == variable.Pkg().Scope() {
		return packageVariable(context, children, source, variable)
	}
	var receiverRequest []api.RootRequest
	if receiver, ok := context.ValueReceiver(variable); ok {
		request, err := receiver.CopyRequest()
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		receiverRequest = []api.RootRequest{request}
	}
	reference, err := context.Names().Reference(variable)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleUnaryOperand),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	address, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolAddressOf,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{api.DirectExpression(
			reference.Expression(context.Factory()),
			reference.Requests()...,
		)},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		address.Before(),
		address.Value(),
		api.CombineRequests(
			address.Requests(),
			receiverRequest,
		),
	)
}

func packageVariable(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	variable *types.Var,
) (api.ExpressionEmission, error) {
	variableType := context.TypesInfo().TypeOfObject(variable)
	if variableType == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := context.Names().PackageVariable(variable)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	storageType, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source,
		variableType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	address, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolAddressOf,
		[]api.TypeEmission{storageType},
		[]api.ExpressionEmission{api.DirectExpression(
			target.Expression(context.Factory()),
			target.Requests()...,
		)},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().ProjectStoragePointer(
		context.WithRole(api.RoleUnaryOperand),
		source,
		variableType,
		address,
	)
}

func selector(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	element types.Type,
) (api.ExpressionEmission, error) {
	selection := context.TypesInfo().SelectionOf(source)
	if selection == nil {
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		packageName, ok := context.TypesInfo().UseOf(qualifier).(*types.PkgName)
		variable, variableOK := context.TypesInfo().UseOf(source.Sel).(*types.Var)
		if !ok ||
			!variableOK ||
			variable.Pkg() != packageName.Imported() ||
			variable.Parent() != variable.Pkg().Scope() ||
			!types.Identical(
				context.TypesInfo().TypeOfObject(variable),
				element,
			) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return packageVariable(context, children, source, variable)
	}
	field, ok := selection.Obj().(*types.Var)
	if !ok ||
		selection.Kind() != types.FieldVal ||
		!types.Identical(
			field.Type(),
			element,
		) {
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
