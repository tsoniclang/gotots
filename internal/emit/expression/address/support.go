package address

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
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
	return context.AddressableStorage().Cell(
		context,
		children,
		source,
		element,
		value,
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
	return dereference(
		context,
		children,
		source,
		pointerType,
		pointer,
	)
}

func dereference(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	pointerType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	_, element, ok := pointertype.Resolve(pointerType)
	defined, definedOK := definedtype.ResolvePointer(pointerType)
	if definedOK {
		pointer, _ := defined.Pointer()
		element = pointer.Elem()
		ok = true
	}
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if definedOK {
		var err error
		pointer, err = defined.Project(context, pointer)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(element),
		api.PointerRepresentationDemandNone,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleFieldReceiver),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	runtime, err := pointerRuntime(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if representation.Representation().DirectClass() {
		return api.NewExpressionEmission(
			pointer.Before(),
			pointerruntime.Direct(
				context.Factory(),
				runtime.Name(),
				targetElement.Value(),
				pointer.Value(),
			),
			api.CombineRequests(
				pointer.Requests(),
				targetElement.Requests(),
				runtime.Requests(),
				representation.Requests(),
			),
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
	return api.NewExpressionEmission(
		pointer.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(runtime.Name()),
				nil,
				context.Factory().Identifier(pointerruntime.DereferenceName),
				tsgo.NodeFlagsNone,
			),
			nil,
			[]tsgo.TypeNode{targetElement.Value(), storageType.Value()},
			[]tsgo.Expression{pointer.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			pointer.Requests(),
			targetElement.Requests(),
			storageType.Requests(),
			runtime.Requests(),
			representation.Requests(),
		),
	)
}

func pointerRuntime(context api.Context) (api.NameReference, error) {
	return context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
}
