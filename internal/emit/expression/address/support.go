package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func captureBefore(
	context api.Context,
	receiver api.ExpressionEmission,
	following []tsgo.Statement,
) (tsgo.Expression, []tsgo.Statement, error) {
	before := receiver.Before()
	value := receiver.Value()
	if len(following) != 0 {
		name, err := context.Names().Temporary(api.TemporaryAddressOperand)
		if err != nil {
			return nil, nil, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						value,
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		value = context.Factory().Identifier(name)
	}
	before = append(before, following...)
	return value, before, nil
}

func fresh(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	element types.Type,
) (api.ExpressionEmission, error) {
	value, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(element),
		source,
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
	return pointermarker.Operation(
		context,
		tsoniccore.SymbolAllocatePointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{value},
	)
}

func cancelDereference(
	context api.Context,
	children api.ChildEmitter,
	source *ast.StarExpr,
	element types.Type,
) (api.ExpressionEmission, error) {
	pointerType := context.TypesInfo().TypeOf(source.X)
	_, pointerElement, ok := pointertype.Resolve(pointerType)
	defined, definedOK := definedtype.ResolvePointer(pointerType)
	if definedOK {
		pointer, _ := defined.Pointer()
		pointerElement = pointer.Elem()
		ok = true
	}
	if !ok || !types.Identical(pointerElement, element) {
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
	return pointer, nil
}
