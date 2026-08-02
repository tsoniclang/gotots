package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type genericReceiverMethodCall struct {
	invocation methodcall.Selection
	receiver   tsgo.Expression
	before     []tsgo.Statement
	arguments  []tsgo.Expression
	requests   []api.RootRequest
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
	invoked, err := call.expression(context, children)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		append(call.before, invoked.Before()...),
		invoked.Value(),
		api.CombineRequests(call.requests, invoked.Requests()),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if detached {
		target, err = cooperativecall.DetachedGenericCall(
			context,
			source,
			call.invocation.Facet(),
			target,
		)
	} else {
		target, err = cooperativecall.GenericCall(
			context,
			source,
			call.invocation.Facet(),
			target,
		)
	}
	if err != nil || discarded {
		return target, err
	}
	return call.invocation.FromProviderResults(context, children, target)
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
	recoveryObservation, err :=
		context.ObserveRecoveryCallable(call.invocation.Facet())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !recoveryObservation.Recovery() {
		expression, expressionErr := call.expression(context, children)
		if expressionErr != nil {
			return api.ExpressionEmission{}, expressionErr
		}
		cooperative, contractRequests, contractErr :=
			cooperativecall.GenericContract(
				context,
				call.invocation.Facet(),
			)
		if contractErr != nil {
			return api.ExpressionEmission{}, contractErr
		}
		return deferredInvocation(
			context,
			append(call.before, expression.Before()...),
			nil,
			expression.Value(),
			cooperative,
			api.CombineRequests(
				call.requests,
				expression.Requests(),
				contractRequests,
				recoveryObservation.Requests(),
			),
		)
	}
	expression, providerRecovery, recoveryCooperative, err :=
		call.recoveryExpression(
			context,
			children,
			call.arguments,
			context.Factory().Identifier(
				callable.RecoveryAuthorityName,
			),
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	cooperative := recoveryCooperative
	var contractRequests []api.RootRequest
	if !providerRecovery {
		cooperative, contractRequests, err =
			cooperativecall.GenericContract(
				context,
				call.invocation.Facet(),
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	return deferredInvocation(
		context,
		append(call.before, expression.Before()...),
		nil,
		expression.Value(),
		cooperative,
		api.CombineRequests(
			call.requests,
			expression.Requests(),
			contractRequests,
			recoveryObservation.Requests(),
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
	invocation, err := methodcall.Resolve(
		context,
		children,
		source,
		method,
		signature,
	)
	if err != nil {
		return genericReceiverMethodCall{}, err
	}
	concrete := invocation.Signature()
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
	return genericReceiverMethodCall{
		invocation: invocation,
		receiver:   receiverValue,
		before:     before,
		arguments:  sourceArguments,
		requests: api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
		),
	}, nil
}

func (c genericReceiverMethodCall) expression(
	context api.Context,
	children api.ChildEmitter,
) (api.ExpressionEmission, error) {
	return c.invocation.Invoke(
		context,
		children,
		c.receiver,
		c.arguments,
	)
}

func (c genericReceiverMethodCall) recoveryExpression(
	context api.Context,
	children api.ChildEmitter,
	arguments []tsgo.Expression,
	recovery tsgo.Expression,
) (
	api.ExpressionEmission,
	bool,
	bool,
	error,
) {
	return c.invocation.RecoveryCall(
		context,
		children,
		c.receiver,
		arguments,
		recovery,
	)
}
