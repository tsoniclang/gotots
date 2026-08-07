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
	receiverABI api.MethodReceiverABI,
) (api.ExpressionEmission, error) {
	pointer, _, _, ok := pointerType(declared)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	var representation api.PointerRepresentationObservation
	if receiverABI == api.MethodReceiverABISourceRepresentation {
		var err error
		representation, err = pointertype.ObserveSource(
			context,
			method.Origin(),
			pointer,
			api.PointerRepresentationDemandNone,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else if receiverABI != api.MethodReceiverABIContractDirect {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver ABI is invalid",
		}
	}
	var (
		receiver api.ExpressionEmission
		err      error
	)
	switch {
	case receiverABI == api.MethodReceiverABIContractDirect:
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
	case representation.Representation().DirectClass() && len(resolved.fields) == 0:
		target, targetErr := children.StoreTarget(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(resolved.root),
			source.X,
		)
		if targetErr != nil {
			return api.ExpressionEmission{}, targetErr
		}
		receiver, err = target.MutableValue(
			context.WithRole(api.RoleReceiverValue),
			source.X,
		)
	default:
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
	source ast.Node,
	declarationOwner types.Object,
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
		sourceRepresentation, err := pointertype.Observe(
			context,
			effective,
			api.PointerRepresentationDemandNone,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if sourceRepresentation.Representation().DirectClass() {
			return api.NewExpressionEmission(
				value.Before(),
				value.Value(),
				api.CombineRequests(
					value.Requests(),
					sourceRepresentation.Requests(),
				),
			)
		}
		return projectContractDirectReceiver(
			context,
			source,
			effective.Elem(),
			sourceRepresentation,
			value,
		)
	}
	if receiverABI != api.MethodReceiverABISourceRepresentation {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "method receiver ABI is invalid",
		}
	}
	target, err := pointertype.ObserveSource(
		context,
		declarationOwner,
		declared,
		api.PointerRepresentationDemandNone,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceDemand := api.PointerRepresentationDemandNone
	if !target.Representation().DirectClass() {
		sourceDemand = api.PointerRepresentationDemandStableLocation
	}
	sourceRepresentation, err := pointertype.Observe(
		context,
		effective,
		sourceDemand,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if target.Representation().DirectClass() &&
		!sourceRepresentation.Representation().DirectClass() {
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

func projectContractDirectReceiver(
	context api.Context,
	source ast.Node,
	element types.Type,
	representation api.PointerRepresentationObservation,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryReceiverValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	identifier := context.Factory().Identifier(name)
	storage := api.DirectExpression(
		context.Factory().PropertyAccessExpression(
			identifier,
			nil,
			context.Factory().Identifier(pointerruntime.CellValueName),
			tsgo.NodeFlagsNone,
		),
	)
	logical, err := context.ContainerStorage().FromPointerStorage(
		context.WithRole(api.RoleReceiverValue),
		source,
		element,
		representation,
		storage,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if len(logical.Before()) != 0 {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "contract-direct receiver projection has nested prerequisites",
		}
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
			logical.Value(),
		),
		api.CombineRequests(
			value.Requests(),
			logical.Requests(),
			representation.Requests(),
		),
	)
}
