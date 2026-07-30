package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func valuePointerMethodReceiver(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	resolved path,
	method *types.Func,
	declared types.Type,
) (api.ExpressionEmission, error) {
	pointer, _, _, ok := pointerType(declared)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	representation, err := pointertype.ObserveSource(
		context,
		method.Origin(),
		pointer,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var receiver api.ExpressionEmission
	if representation.Representation() ==
		api.PointerRepresentationDirectClass {
		root, expressionErr := children.Expression(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(resolved.root),
			source.X,
		)
		if expressionErr != nil {
			return api.ExpressionEmission{}, expressionErr
		}
		receiver, err = projectValue(
			context,
			children,
			source,
			resolved,
			root,
		)
	} else {
		receiver, err = addressSource(
			context,
			children,
			source,
			resolved,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		receiver.Value(),
		api.CombineRequests(
			receiver.Requests(),
			representation.Requests(),
		),
	)
}

func methodABIReceiver(method *types.Func) (types.Type, bool) {
	if method == nil || method.Origin() == nil {
		return nil, false
	}
	signature, ok := method.Origin().Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, false
	}
	return signature.Recv().Type(), true
}

func adaptPointerMethodReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	declarationOwner types.Object,
	declared *types.Pointer,
	effective *types.Pointer,
	element types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if declared == nil || effective == nil || element == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err := pointertype.ObserveSource(
		context,
		declarationOwner,
		declared,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceRepresentation, err := pointertype.Observe(
		context,
		effective,
		target.Representation() != api.PointerRepresentationDirectClass,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if target.Representation() != api.PointerRepresentationDirectClass ||
		sourceRepresentation.Representation() ==
			api.PointerRepresentationDirectClass {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			api.CombineRequests(
				value.Requests(),
				target.Requests(),
				sourceRepresentation.Requests(),
			),
		)
	}
	return carrierLogicalMethodReceiver(
		context,
		source,
		element,
		sourceRepresentation,
		target,
		value,
	)
}

func carrierLogicalMethodReceiver(
	context api.Context,
	source ast.Node,
	element types.Type,
	sourceRepresentation api.PointerRepresentationObservation,
	target api.PointerRepresentationObservation,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	runtime, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	name, err := context.Names().Temporary(api.TemporaryReceiverValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	identifier := context.Factory().Identifier(name)
	before := append(
		value.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						identifier,
						nil,
						nil,
						pointerruntime.OptionalStorage(
							context.Factory(),
							runtime.Name(),
							value.Value(),
						),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	logical, err := context.ContainerStorage().FromPointerStorage(
		context.WithRole(api.RoleReceiverValue),
		source,
		element,
		sourceRepresentation,
		api.DirectExpression(identifier),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(logical.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "pointer receiver storage projection is not expression-local",
		}
	}
	condition := context.Factory().BinaryExpression(
		nil,
		identifier,
		nil,
		context.Factory().BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		context.Factory().Identifier("undefined"),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			condition,
			context.Factory().QuestionToken(),
			context.Factory().Identifier("undefined"),
			context.Factory().ColonToken(),
			logical.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			logical.Requests(),
			runtime.Requests(),
			target.Requests(),
			sourceRepresentation.Requests(),
		),
	)
}
