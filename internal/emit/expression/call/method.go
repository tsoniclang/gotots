package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func selectedMethod(
	info *types.Info,
	source ast.Expr,
) (*ast.SelectorExpr, *types.Func, *types.Selection, bool) {
	selector, ok := source.(*ast.SelectorExpr)
	if !ok || info == nil {
		return nil, nil, nil, false
	}
	selection := info.Selections[selector]
	if selection == nil || selection.Kind() != types.MethodVal {
		return nil, nil, nil, false
	}
	method, ok := selection.Obj().(*types.Func)
	return selector, method, selection, ok
}

func emitMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	discarded bool,
) (api.ExpressionEmission, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if api.ContainsGenericTypeParameter(selection.Recv()) {
		return emitConstraintMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			discarded,
		)
	}
	if signature.RecvTypeParams().Len() != 0 {
		return emitGenericReceiverMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			signature,
			discarded,
		)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	if _, interfaceReceiver := interfacetype.Resolve(
		selection.Recv(),
	); interfaceReceiver {
		return emitInterfaceMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			signature,
		)
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
			api.Unsupported(context, api.CategoryExpression, source)
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before = append(before, argumentBefore...)
	reference, err := context.Names().Reference(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments = append([]tsgo.Expression{receiverValue}, arguments...)
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
			reference.Requests(),
		),
	)
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
) (api.ExpressionEmission, error) {
	owner := method.Origin()
	callable, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := genericinstance.ReceiverTypeArguments(signature.Recv().Type())
	if !ok ||
		arguments == nil ||
		arguments.Len() != len(callable.Parameters()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	concrete, err := genericinstance.ConcreteCallableSignature(signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if err := validateResults(context, source, concrete, discarded); err != nil {
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
			api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceArguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		concrete,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
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
		return api.ExpressionEmission{}, err
	}
	capabilities, capabilityRequests, err := genericinstance.EmitCapabilities(
		context,
		source,
		callable,
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverBinding, err := genericabi.Receiver(
		owner,
		receiverValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceBindings, err := genericabi.SourceParameters(
		owner,
		sourceArguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	callArguments, err := genericabi.JoinMethod(
		owner,
		callable.Operations(),
		genericabi.Combine(
			capabilities,
			[]genericabi.Binding[tsgo.Expression]{receiverBinding},
			sourceBindings,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			typeArguments,
			callArguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
			typeRequests,
			capabilityRequests,
			reference.Requests(),
		),
	)
}

func emitInterfaceMethod(
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
	return ApplyInterfaceMethod(
		context,
		selector.X,
		selection.Recv(),
		receiver,
		method,
		arguments,
		argumentBefore,
		argumentRequests,
	)
}

func emitConstraintMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	discarded bool,
) (api.ExpressionEmission, error) {
	signature, ok := selection.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		!types.Identical(signature.Recv().Type(), selection.Recv()) ||
		signature.TypeParams().Len() != 0 ||
		signature.RecvTypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
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
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverValue := receiver.Value()
	before := receiver.Before()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before = append(before, argumentBefore...)
	parameterTypes := make([]types.Type, 0, signature.Params().Len()+1)
	parameterTypes = append(parameterTypes, selection.Recv())
	for index := range signature.Params().Len() {
		parameterTypes = append(
			parameterTypes,
			signature.Params().At(index).Type(),
		)
	}
	resultTypes := make([]types.Type, 0, signature.Results().Len())
	for index := range signature.Results().Len() {
		resultTypes = append(
			resultTypes,
			signature.Results().At(index).Type(),
		)
	}
	target, err := genericoperation.ConstraintMethod(
		context,
		source,
		method,
		parameterTypes,
		resultTypes,
		append([]tsgo.Expression{receiverValue}, arguments...),
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
		)...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		target.Value(),
		target.Requests(),
	)
}

func captureReceiver(
	context api.Context,
	receiver api.ExpressionEmission,
) (
	tsgo.Expression,
	[]api.RootRequest,
	[]tsgo.Statement,
	error,
) {
	name, err := context.Names().Temporary(api.TemporaryReceiverValue)
	if err != nil {
		return nil, nil, nil, err
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(name),
		nil,
		nil,
		receiver.Value(),
	)
	before := receiver.Before()
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsConst,
		),
	))
	return context.Factory().Identifier(name),
		nil,
		before,
		nil
}
