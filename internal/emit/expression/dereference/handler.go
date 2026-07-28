package dereference

import (
	"go/ast"
	"go/types"

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
	return api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			reference.Name(),
			targetElement.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			targetElement.Requests(),
			reference.Requests(),
		),
	)
}
