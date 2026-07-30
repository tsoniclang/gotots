package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type genericReceiverMethodCall struct {
	owner         *types.Func
	facet         api.CallableFacet
	memberSuffix  string
	receiver      tsgo.Expression
	before        []tsgo.Statement
	typeArguments []tsgo.TypeNode
	arguments     []tsgo.Expression
	requests      []api.RootRequest
}

func emitGenericReceiverMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, error) {
	call, err := prepareGenericReceiverMethodCall(
		context,
		children,
		source,
		selector,
		method,
		selection,
		signature,
		discarded,
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	expression, expressionRequests, err :=
		call.expression(context, call.arguments)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		call.before,
		expression,
		api.CombineRequests(call.requests, expressionRequests),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if detached {
		return cooperativecall.DetachedGenericCall(
			context,
			source,
			call.facet,
			target,
		)
	}
	return cooperativecall.GenericCall(
		context,
		source,
		call.facet,
		target,
	)
}

func emitDeferredGenericReceiverMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
) (api.ExpressionEmission, error) {
	call, err := prepareGenericReceiverMethodCall(
		context,
		children,
		source,
		selector,
		method,
		selection,
		signature,
		true,
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := append(
		append([]tsgo.Expression(nil), call.arguments...),
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	control, err := api.NewDirectCallableControlRequest(
		call.owner,
		api.CallableControlRecovery,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	expression, expressionRequests, err :=
		call.expression(context, arguments)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	cooperative, contractRequests, err :=
		cooperativecall.GenericContract(context, call.facet)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return deferredInvocation(
		context,
		call.before,
		nil,
		expression,
		cooperative,
		api.CombineRequests(
			call.requests,
			expressionRequests,
			contractRequests,
			[]api.RootRequest{control},
		),
	)
}

func prepareGenericReceiverMethodCall(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
	discarded bool,
	capture bool,
) (genericReceiverMethodCall, error) {
	owner := method.Origin()
	callableContract, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	arguments := genericinstance.ReceiverTypeArguments(signature.Recv().Type())
	if !ok ||
		arguments == nil ||
		arguments.Len() != len(callableContract.Parameters()) {
		return genericReceiverMethodCall{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	concrete, err := genericinstance.ConcreteCallableSignature(signature)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	declarationSignature, ok := owner.Type().(*types.Signature)
	if !ok {
		return genericReceiverMethodCall{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	memberSuffix, callableFacet, _, selectionRequests, err :=
		cooperativecall.SelectGenericClassMethod(
			context,
			owner,
			declarationSignature,
			concrete,
		)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	if err := validateResults(context, source, concrete, discarded); err != nil {
		return genericReceiverMethodCall{}, err
	}
	var receiver api.ExpressionEmission
	var resolvedMethod *types.Func
	if capture {
		receiver, resolvedMethod, err = selectionvalue.MethodReceiver(
			context,
			children,
			selector,
			selection,
		)
	} else {
		receiver, resolvedMethod, err =
			selectionvalue.DirectMethodReceiver(
				context,
				children,
				selector,
				selection,
			)
	}
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	if resolvedMethod != method {
		return genericReceiverMethodCall{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceArguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		concrete,
		capture,
	)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 || capture {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return genericReceiverMethodCall{}, err
		}
	}
	before = append(before, argumentBefore...)
	typeArguments, typeRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		arguments,
	)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	capabilities, capabilityRequests, err :=
		genericinstance.EmitCapabilities(
			context,
			source,
			callableContract,
			arguments,
		)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	sourceBindings, err := genericabi.SourceParameters(owner, sourceArguments)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	callArguments, err := genericabi.JoinClassMethod(
		owner,
		callableContract.Operations(),
		genericabi.Combine(
			capabilities,
			sourceBindings,
		),
	)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	return genericReceiverMethodCall{
		owner:         owner,
		facet:         callableFacet,
		memberSuffix:  memberSuffix,
		receiver:      receiverValue,
		before:        before,
		typeArguments: classMethodTypeArguments(owner, typeArguments),
		arguments:     callArguments,
		requests: api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
			typeRequests,
			capabilityRequests,
			selectionRequests,
		),
	}, nil
}

func (c genericReceiverMethodCall) expression(
	context api.Context,
	arguments []tsgo.Expression,
) (tsgo.CallExpression, []api.RootRequest, error) {
	return callable.SelectedMethodCall(
		context,
		c.owner,
		c.memberSuffix,
		c.receiver,
		c.typeArguments,
		arguments,
	)
}

func classMethodTypeArguments(
	method *types.Func,
	arguments []tsgo.TypeNode,
) []tsgo.TypeNode {
	if api.ValueReceiverTypeName(method) != nil {
		return nil
	}
	return arguments
}
