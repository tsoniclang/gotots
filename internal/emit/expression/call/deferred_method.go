package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	call, callRequests, err := invocation.Call(
		context,
		context.Factory().Identifier(receiverName),
		arguments,
		context.Factory().Identifier(
			callable.RecoveryAuthorityName,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	cooperative, contractRequests, err :=
		cooperativecall.GenericContract(
			context,
			invocation.Facet(),
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return deferredInvocation(
		context,
		before,
		nil,
		call,
		cooperative,
		api.CombineRequests(
			receiver.Requests(),
			argumentRequests,
			callRequests,
			contractRequests,
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
	receiverName, err := context.Names().Temporary(
		api.TemporaryReceiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverContract, err := interfaceoperation.NonNilType(
		context,
		children,
		selector.X,
		selection.Recv(),
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
	guardedReceiver := context.Factory().CallExpression(
		context.Factory().Identifier(nonNil.Name()),
		nil,
		[]tsgo.TypeNode{receiverContract.Value()},
		[]tsgo.Expression{receiver.Value()},
		tsgo.NodeFlagsNone,
	)
	before := append(
		receiver.Before(),
		constantDeclaration(
			context,
			receiverName,
			receiverContract.Value(),
			guardedReceiver,
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
	member, err := context.Names().InterfaceMethodName(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments = append(
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
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
		arguments,
		tsgo.NodeFlagsNone,
	)
	callableReference, err :=
		context.Names().InterfaceMethodCallable(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	cooperative, contractRequests, err :=
		cooperativecall.InterfaceMethodContract(
			context,
			callableReference,
		)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return deferredInvocation(
		context,
		before,
		nil,
		call,
		cooperative,
		api.CombineRequests(
			receiver.Requests(),
			argumentRequests,
			nonNil.Requests(),
			receiverContract.Requests(),
			contractRequests,
		),
	)
}
