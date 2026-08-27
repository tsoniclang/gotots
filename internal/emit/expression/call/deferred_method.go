package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
)

func emitDeferredMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
) (api.ExpressionEmission, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if signature.RecvTypeParams().Len() != 0 {
		return emitDeferredGenericReceiverMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			signature,
		)
	}
	if err := validateResults(context, source, signature, true); err != nil {
		return api.ExpressionEmission{}, err
	}
	if _, interfaceReceiver := interfacetype.Resolve(
		selection.Recv(),
	); interfaceReceiver {
		return emitDeferredInterfaceMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			signature,
		)
	}
	invocation, err := methodcall.Resolve(
		context,
		children,
		source,
		method,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, resolvedMethod, err := selectionvalue.MethodReceiver(
		context,
		children,
		selector,
		selection,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if resolvedMethod != method {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		receiver.Before(),
		constantDeclaration(
			context,
			receiverName,
			nil,
			receiver.Value(),
		),
	)
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, argumentBefore...)
	recoveryObservation, err :=
		context.ObserveRecoveryCallable(invocation.Facet())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !recoveryObservation.Recovery() {
		call, callErr := invocation.Invoke(
			context,
			children,
			context.Factory().Identifier(receiverName),
			arguments,
		)
		if callErr != nil {
			return api.ExpressionEmission{}, callErr
		}
		return deferredInvocation(
			context,
			append(before, call.Before()...),
			nil,
			call.Value(),
			api.CombineRequests(
				receiver.Requests(),
				argumentRequests,
				call.Requests(),
				recoveryObservation.Requests(),
			),
		)
	}
	call, err :=
		invocation.RecoveryCall(
			context,
			children,
			context.Factory().Identifier(receiverName),
			arguments,
			context.Factory().Identifier(
				callable.RecoveryAuthorityName,
			),
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, call.Before()...)
	return deferredInvocation(
		context,
		before,
		nil,
		call.Value(),
		api.CombineRequests(
			receiver.Requests(),
			argumentRequests,
			call.Requests(),
			recoveryObservation.Requests(),
		),
	)
}

func emitDeferredInterfaceMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
) (api.ExpressionEmission, error) {
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(selection.Recv()),
		selector.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := interfaceoperation.ApplyDeferred(
		context,
		children,
		selector,
		selection.Recv(),
		receiver,
		method,
		signature,
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err = api.NewExpressionEmission(
		append(call.Before(), argumentBefore...),
		call.Value(),
		api.CombineRequests(call.Requests(), argumentRequests),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return deferredInvocation(
		context,
		call.Before(),
		nil,
		call.Value(),
		call.Requests(),
	)
}
