package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	"github.com/tsoniclang/gotots/internal/emit/methodcall"
	selectionvalue "github.com/tsoniclang/gotots/internal/emit/selection"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func selectedMethod(
	info api.TypeInfoView,
	source ast.Expr,
) (*ast.SelectorExpr, *types.Func, *types.Selection, bool) {
	selector, ok := source.(*ast.SelectorExpr)
	if !ok || !info.Valid() {
		return nil, nil, nil, false
	}
	selection := info.SelectionOf(selector)
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
	detached bool,
) (api.ExpressionEmission, error) {
	signature, ok := method.Type().(*types.Signature)
	if !ok ||
		signature.Recv() == nil ||
		signature.TypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, constraintReceiver := api.GenericTypeParameter(
		selection.Recv(),
	); constraintReceiver {
		contextualReceiver := selection.Recv()
		if _, stillOpen := api.GenericTypeParameter(
			contextualReceiver,
		); !stillOpen {
			return emitConcretizedConstraintMethod(
				context,
				children,
				source,
				selector,
				method,
				selection,
				contextualReceiver,
				discarded,
				detached,
			)
		}
		return emitConstraintMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			discarded,
			detached,
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
			detached,
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
			discarded,
			detached,
		)
	}
	target, selected, profileRequests, err := emitProviderProfileMethod(
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
	if selected || err != nil {
		return target, err
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
	receiver, resolvedMethod, err := selectionvalue.DirectMethodReceiver(
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
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 || detached {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before = append(before, argumentBefore...)
	invoked, err := invocation.Invoke(
		context,
		children,
		receiverValue,
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err = api.NewExpressionEmission(
		append(before, invoked.Before()...),
		invoked.Value(),
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
			invoked.Requests(),
			profileRequests,
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if discarded {
		return target, nil
	}
	return invocation.FromProviderResults(context, children, target)
}

func emitConcretizedConstraintMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	contractMethod *types.Func,
	contractSelection *types.Selection,
	receiverType types.Type,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, error) {
	methodSet := types.NewMethodSet(receiverType)
	selected := methodSet.Lookup(contractMethod.Pkg(), contractMethod.Name())
	expected := contractSelection.Type()
	if selected == nil || selected.Kind() != types.MethodVal ||
		!types.Identical(selected.Type(), expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	method, ok := selected.Obj().(*types.Func)
	signature, signatureOK := method.Type().(*types.Signature)
	if !ok || !signatureOK || signature.Recv() == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
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
	concrete := invocation.Signature()
	if err := validateResults(context, source, concrete, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	root, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(receiverType),
		selector.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, _, resolvedMethod, err := selectionvalue.DirectMethodSetReceiver(
		context,
		children,
		selector,
		selected,
		root,
	)
	if err != nil || resolvedMethod != method {
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		concrete,
		detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.RootRequest
	if len(argumentBefore) != 0 || detached {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			receiver,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before = append(before, argumentBefore...)
	invoked, err := invocation.Invoke(
		context,
		children,
		receiverValue,
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		append(before, invoked.Before()...),
		invoked.Value(),
		api.CombineRequests(
			receiver.Requests(),
			receiverRequests,
			argumentRequests,
			invoked.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if discarded {
		return target, nil
	}
	return invocation.FromProviderResults(context, children, target)
}

func emitConstraintMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, error) {
	if detached {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
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
	genericArguments := make(
		[]api.ExpressionEmission,
		0,
		len(arguments)+1,
	)
	genericArguments = append(
		genericArguments,
		api.DirectExpression(
			receiverValue,
			api.CombineRequests(
				receiver.Requests(),
				receiverRequests,
				argumentRequests,
			)...,
		),
	)
	for _, argument := range arguments {
		genericArguments = append(
			genericArguments,
			api.DirectExpression(argument),
		)
	}
	target, err := genericoperation.ConstraintMethod(
		context,
		source,
		method,
		parameterTypes,
		resultTypes,
		genericArguments,
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
