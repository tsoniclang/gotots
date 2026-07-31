package dereference

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
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
	pointer, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(pointerType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedOK {
		pointer, err = defined.Project(context, pointer)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if value, handled, err := genericpointer.Load(
		context,
		source,
		element,
		pointer,
	); handled || err != nil {
		return value, err
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		false,
	)
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
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		guarded, err := api.NewExpressionEmission(
			pointer.Before(),
			pointerruntime.Direct(
				context.Factory(),
				reference.Name(),
				targetElement.Value(),
				pointer.Value(),
			),
			api.CombineRequests(
				pointer.Requests(),
				targetElement.Requests(),
				reference.Requests(),
				representation.Requests(),
			),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return context.Values().Transfer(
			context,
			source,
			element,
			element,
			api.ValueTransferCopy,
			guarded,
		)
	}
	storageType, err := context.ContainerStorage().PointerStorageType(
		context.WithRole(api.RoleStorageType),
		source,
		element,
		representation,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	stored, err := api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			reference.Name(),
			targetElement.Value(),
			storageType.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			targetElement.Requests(),
			storageType.Requests(),
			reference.Requests(),
			representation.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.ContainerStorage().FromPointerStorage(
		context,
		source,
		element,
		representation,
		stored,
	)
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
	return context.TypesInfo().Uses[identifier] ==
		types.Universe.Lookup("new")
}
