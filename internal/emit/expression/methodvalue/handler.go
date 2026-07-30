package methodvalue

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
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
	owner := method.Origin()
	var (
		memberSuffix      string
		abiCooperative    bool
		sourceRequests    []api.RootRequest
		selectionRequests []api.RootRequest
	)
	if generic {
		declarationSignature, ok := owner.Type().(*types.Signature)
		if !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		var facet api.CallableFacet
		memberSuffix, facet, _, selectionRequests, err =
			cooperativecall.SelectGenericClassMethod(
				context,
				owner,
				declarationSignature,
				targetSignatureSource,
			)
		if err == nil {
			_, abiCooperative, sourceRequests, err =
				cooperativecall.GenericValueContract(
					context,
					facet,
					targetSignatureSource,
				)
		}
	} else {
		_, abiCooperative, sourceRequests, err =
			cooperativecall.SourceValueContract(
				context,
				method,
				targetSignatureSource,
			)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitABIAdapter(
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
	var targetTypeArguments []tsgo.TypeNode
	var typeRequests []api.RootRequest
	var capabilities []genericabi.Binding[tsgo.Expression]
	var capabilityRequests []api.RootRequest
	var operationSet api.GenericOperationSet
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
	controlRequest, err := api.NewDirectCallableControlRequest(
		owner,
		api.CallableControlRecovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	callArguments := arguments
	if generic {
		sourceArguments := targetSignature.SourceParameterReferences(
			context.Factory(),
		)
		recoveryAuthority, recoveryOK :=
			targetSignature.RecoveryAuthorityReference(context.Factory())
		if !recoveryOK {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic method value lacks recovery authority",
			}
		}
		sourceBindings, bindErr := genericabi.SourceParameters(
			owner,
			sourceArguments,
		)
		if bindErr != nil {
			return api.ExpressionEmission{}, bindErr
		}
		callArguments, err = genericabi.JoinClassMethod(
			owner,
			operationSet.Operations(),
			genericabi.Combine(
				capabilities,
				sourceBindings,
			),
		)
		callArguments = append(callArguments, recoveryAuthority)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if api.ValueReceiverTypeName(owner) != nil {
		targetTypeArguments = nil
	}
	call, callRequests, err := callable.SelectedMethodCall(
		context,
		owner,
		memberSuffix,
		context.Factory().Identifier(receiverName),
		targetTypeArguments,
		callArguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if abiCooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	target, err := api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			targetSignature.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			typeRequests,
			capabilityRequests,
			selectionRequests,
			callRequests,
			[]api.RootRequest{controlRequest},
			sourceRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return target, nil
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
	cooperative, contractRequests, err :=
		cooperativecall.ValueContract(context, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetSignature, err := callable.EmitABIAdapter(
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
	var modifiers []tsgo.ModifierLike
	resultType := targetSignature.Result()
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			targetSignature.Parameters(),
			resultType,
			context.Factory().EqualsGreaterThanToken(),
			call,
		),
		api.CombineRequests(
			receiver.Requests(),
			targetSignature.Requests(),
			nonNil.Requests(),
			contractRequests,
		),
	)
}
