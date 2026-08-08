package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func valuePointerMethodReceiver(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	resolved path,
	declared types.Type,
	receiverABI api.MethodReceiverABI,
) (api.ExpressionEmission, error) {
	_, _, _, ok := pointerType(declared)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if receiverABI == api.MethodReceiverABISourceRepresentation {
		if len(resolved.fields) == 0 {
			return children.Address(
				context.
					WithRole(api.RoleReceiverValue).
					WithExpectedType(declared),
				source.X,
			)
		}
		return addressSource(context, children, source, resolved)
	}
	if receiverABI != api.MethodReceiverABIContractDirect {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver ABI is invalid",
		}
	}
	var (
		receiver api.ExpressionEmission
		err      error
	)
	target, targetErr := children.StoreTarget(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(resolved.root),
		source.X,
	)
	if targetErr != nil {
		return api.ExpressionEmission{}, targetErr
	}
	root, mutableErr := target.MutableValue(
		context.WithRole(api.RoleReceiverValue),
		source.X,
	)
	if mutableErr != nil {
		return api.ExpressionEmission{}, mutableErr
	}
	receiver, err = projectMutableValue(
		context,
		children,
		source,
		resolved,
		root,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		receiver.Before(),
		receiver.Value(),
		receiver.Requests(),
	)
}

func methodABIReceiver(
	context api.Context,
	method *types.Func,
	dispatchType types.Type,
) (types.Type, api.MethodReceiverABI, error) {
	if method == nil || method.Origin() == nil {
		return nil, api.MethodReceiverABIInvalid, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver owner is invalid",
		}
	}
	signature, ok := method.Origin().Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil, api.MethodReceiverABIInvalid, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver signature is invalid",
		}
	}
	if dispatchType != nil {
		if _, interfaceDispatch :=
			types.Unalias(dispatchType).Underlying().(*types.Interface); interfaceDispatch {
			return signature.Recv().Type(),
				api.MethodReceiverABISourceRepresentation,
				nil
		}
	}
	target, err := context.Names().MethodTarget(method.Origin())
	if err != nil {
		return nil, api.MethodReceiverABIInvalid, err
	}
	if !target.ReceiverABI().Valid() {
		return nil, api.MethodReceiverABIInvalid, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method target receiver ABI is invalid",
		}
	}
	return signature.Recv().Type(),
		target.ReceiverABI(), nil
}

func adaptPointerMethodReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	declared *types.Pointer,
	effective *types.Pointer,
	receiverABI api.MethodReceiverABI,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if declared == nil || effective == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if receiverABI == api.MethodReceiverABIContractDirect {
		return projectContractDirectReceiver(
			context,
			children,
			source,
			effective.Elem(),
			value,
		)
	}
	if receiverABI != api.MethodReceiverABISourceRepresentation {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver ABI is invalid",
		}
	}
	return value, nil
}

func projectContractDirectReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryReceiverValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	identifier := context.Factory().Identifier(name)
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleReceiverValue),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	loaded, err := pointermarker.Operation(
		context,
		tsoniccore.SymbolLoadPointer,
		[]api.TypeEmission{targetElement},
		[]api.ExpressionEmission{api.DirectExpression(identifier)},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
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
						value.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				identifier,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().VoidExpression(
					context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
				),
			),
			context.Factory().QuestionToken(),
			context.Factory().VoidExpression(
				context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			),
			context.Factory().ColonToken(),
			loaded.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			loaded.Requests(),
		),
	)
}
