package dereference

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.StarExpr,
) (api.ExpressionEmission, error) {
	if source == nil || source.X == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	pointerType := context.TypesInfo().TypeOf(source.X)
	_, element, ok := pointertype.Resolve(pointerType)
	defined, definedOK := definedtype.ResolvePointer(pointerType)
	if definedOK {
		pointer, _ := defined.Pointer()
		element = pointer.Elem()
		ok = true
	}
	if !ok ||
		!types.Identical(context.TypesInfo().TypeOf(source), element) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(element, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if zeroFromNew(context, source, pointerType, element) {
		return context.Values().Zero(context, source, element)
	}
	if projected, ok, err := projectedParameter(
		context,
		children,
		source.X,
		element,
	); err != nil {
		return api.ExpressionEmission{}, err
	} else if ok {
		return context.Values().Transfer(
			context,
			source,
			element,
			element,
			api.ValueTransferCopy,
			projected,
		)
	}
	pointer, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(pointerType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.PointeeValues().Pointee(
		context,
		source,
		pointerType,
		pointer,
	)
}

func projectedParameter(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	element types.Type,
) (api.ExpressionEmission, bool, error) {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			break
		}
		source = parenthesized.X
	}
	identifier, ok := source.(*ast.Ident)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	parameter, ok := context.TypesInfo().UseOf(identifier).(*types.Var)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	selected, ok := context.CallableParameterABI(parameter)
	if !ok || selected.Projection() != callableabi.ProjectionPointeeValue {
		return api.ExpressionEmission{}, false, nil
	}
	pointer, ok := types.Unalias(parameter.Type()).(*types.Pointer)
	if !ok || !types.Identical(pointer.Elem(), element) {
		return api.ExpressionEmission{}, false, nil
	}
	reference, err := context.Names().Reference(parameter)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	projected := api.DirectExpression(
		reference.Expression(context.Factory()),
		reference.Requests()...,
	)
	if selected.NilPolicy() == callableabi.NilPolicyRejectAtBoundary {
		return projected, true, nil
	}
	if selected.NilPolicy() != callableabi.NilPolicyPreserve {
		return api.ExpressionEmission{}, false, &api.InvariantError{
			Role:   context.Role(),
			Reason: "projected parameter has an invalid nil policy",
		}
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleUnaryOperand),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	return api.DirectExpression(
		pointerruntime.Direct(
			context.Factory(),
			runtime.Name(),
			targetElement.Value(),
			projected.Value(),
		),
		api.CombineRequests(
			projected.Requests(),
			targetElement.Requests(),
			runtime.Requests(),
		)...,
	), true, nil
}

func zeroFromNew(
	context api.Context,
	source *ast.StarExpr,
	pointerType types.Type,
	element types.Type,
) bool {
	operand := ast.Expr(source.X)
	for {
		parenthesized, ok := operand.(*ast.ParenExpr)
		if !ok {
			break
		}
		operand = parenthesized.X
	}
	call, ok := operand.(*ast.CallExpr)
	if !ok ||
		call.Ellipsis != token.NoPos ||
		len(call.Args) != 1 ||
		!types.Identical(context.TypesInfo().TypeOf(call), pointerType) ||
		!types.Identical(context.TypesInfo().TypeOf(call.Args[0]), element) {
		return false
	}
	identifier, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return context.TypesInfo().UseOf(identifier) ==
		types.Universe.Lookup("new")
}
