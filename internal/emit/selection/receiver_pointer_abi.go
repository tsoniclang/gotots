package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	source ast.Node,
	declarationOwner types.Object,
	declared *types.Pointer,
	effective *types.Pointer,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if declared == nil || effective == nil {
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
	if target.Representation() == api.PointerRepresentationDirectClass &&
		sourceRepresentation.Representation() !=
			api.PointerRepresentationDirectClass {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role: context.Role(),
			Reason: "pointer receiver occurrence diverged from its " +
				"declaration-family ABI",
		}
	}
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
