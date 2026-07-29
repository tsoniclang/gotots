package methodvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, error) {
	if selected == nil || selected.Kind() != types.MethodVal {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := selected.Type().(*types.Signature)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	targetSignatureSource := signature
	typeArguments := genericinstance.ReceiverTypeArguments(selected.Recv())
	generic := signature.RecvTypeParams().Len() != 0
	if generic {
		var err error
		targetSignatureSource, err =
			genericinstance.ConcreteCallableSignature(signature)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	} else if !callable.Supports(signature) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, interfaceReceiver := interfacetype.Resolve(
		selected.Recv(),
	); interfaceReceiver {
		return emitInterface(context, children, source, selected, signature)
	}
	receiver, method, err := selectionvalue.MethodReceiver(
		context,
		children,
		source,
		selected,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitAdapter(
		context,
		children,
		source,
		targetSignatureSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := targetSignature.ParameterReferences(context.Factory())
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						receiver.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	reference, err := context.Names().Reference(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var targetTypeArguments []tsgo.TypeNode
	var typeRequests []api.RootRequest
	var capabilities []genericabi.Binding[tsgo.Expression]
	var capabilityRequests []api.RootRequest
	var operationSet api.GenericOperationSet
	owner := method.Origin()
	if generic {
		var resolved bool
		var resolveErr error
		operationSet, resolved, resolveErr =
			context.ResolveGenericCallable(owner)
		if resolveErr != nil {
			return api.ExpressionEmission{}, resolveErr
		}
		if !resolved ||
			typeArguments == nil ||
			typeArguments.Len() != len(operationSet.Parameters()) {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		targetTypeArguments, typeRequests, err =
			genericinstance.EmitTypeArguments(
				context,
				children,
				source,
				typeArguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		capabilities, capabilityRequests, err =
			genericinstance.EmitCapabilities(
				context,
				source,
				operationSet,
				typeArguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	callArguments := append(
		[]tsgo.Expression{
			context.Factory().Identifier(receiverName),
		},
		arguments...,
	)
	if generic {
		receiverBinding, bindErr := genericabi.Receiver[tsgo.Expression](
			owner,
			context.Factory().Identifier(receiverName),
		)
		if bindErr != nil {
			return api.ExpressionEmission{}, bindErr
		}
		sourceBindings, bindErr := genericabi.SourceParameters(
			owner,
			arguments,
		)
		if bindErr != nil {
			return api.ExpressionEmission{}, bindErr
		}
		callArguments, err = genericabi.JoinMethod(
			owner,
			operationSet.Operations(),
			genericabi.Combine(
				capabilities,
				[]genericabi.Binding[tsgo.Expression]{receiverBinding},
				sourceBindings,
			),
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call := context.Factory().CallExpression(
		context.Factory().Identifier(reference.Name()),
		nil,
		targetTypeArguments,
		callArguments,
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			nil,
			nil,
			targetSignature.Parameters(),
			targetSignature.Result(),
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			typeRequests,
			capabilityRequests,
			reference.Requests(),
		),
	)
}

func emitInterface(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
	signature *types.Signature,
) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(selected.Recv()),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitAdapter(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	nonNil, err := context.Names().Runtime(
		api.RuntimeInterfaceNonNil,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	member, err := context.Names().InterfaceMethodName(
		selected.Obj().(*types.Func),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(receiverName),
						nil,
						nil,
						context.Factory().CallExpression(
							context.Factory().Identifier(nonNil.Name()),
							nil,
							nil,
							[]tsgo.Expression{receiver.Value()},
							tsgo.NodeFlagsNone,
						),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(receiverName),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		targetSignature.ParameterReferences(context.Factory()),
		tsgo.NodeFlagsNone,
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			nil,
			nil,
			targetSignature.Parameters(),
			targetSignature.Result(),
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			nonNil.Requests(),
		),
	)
}
