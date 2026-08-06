package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/emit/api"
	genericpointer "github.com/tsoniclang/gotots/internal/emit/generic/pointer"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) ProjectedPointee(
	context api.Context,
	source ast.Node,
	pointerSourceType types.Type,
	pointer api.ExpressionEmission,
	nilPolicy callableabi.NilPolicy,
) (api.ExpressionEmission, error) {
	if nilPolicy == callableabi.NilPolicyRejectAtBoundary {
		return owner.Pointee(context, source, pointerSourceType, pointer)
	}
	if nilPolicy != callableabi.NilPolicyPreserve {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointee projection has an invalid nil policy",
		}
	}
	before := pointer.Before()
	requests := pointer.Requests()
	value := pointer.Value()
	if value.Kind() != tsgo.SyntaxKindIdentifier {
		name, err := context.Names().Temporary(api.TemporaryCallArgument)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				)},
				tsgo.NodeFlagsConst,
			),
		))
		value = context.Factory().Identifier(name)
	}
	loaded, err := owner.Pointee(
		context,
		source,
		pointerSourceType,
		api.DirectExpression(value),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(loaded.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "captured pointee projection produced prerequisites",
		}
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				value,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().Identifier("undefined"),
			),
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			loaded.Value(),
		),
		api.CombineRequests(requests, loaded.Requests()),
	)
}

func (owner Owner) Pointee(
	context api.Context,
	source ast.Node,
	pointerSourceType types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	_, element, ok := pointertype.Resolve(pointerSourceType)
	defined, definedOK := definedtype.ResolvePointer(pointerSourceType)
	if definedOK {
		sourcePointer, _ := defined.Pointer()
		element = sourcePointer.Elem()
		ok = true
	}
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	}
	if definedOK {
		var err error
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
	targetElement, err := owner.children.RepresentedType(
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
	if representation.Representation() == api.PointerRepresentationDirectClass {
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
		return owner.Transfer(
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
