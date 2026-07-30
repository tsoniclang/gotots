package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/call/interfaceoperation"
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
		valueSignature, valueOK := selection.Type().(*types.Signature)
		if valueOK {
			valueSignature, valueOK =
				callable.ValueSignature(valueSignature)
		}
		if !valueOK {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return emitInterfaceMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			signature,
			valueSignature,
			detached,
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
	reference, err := context.Names().Reference(method)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments = append([]tsgo.Expression{receiverValue}, arguments...)
	target, err := api.NewExpressionEmission(
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
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if detached {
		return cooperativecall.DetachedSourceCall(
			context,
			source,
			method,
			target,
		)
	}
	return cooperativecall.SourceCall(context, source, method, target)
}

func emitInterfaceMethod(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	selector *ast.SelectorExpr,
	method *types.Func,
	selection *types.Selection,
	signature *types.Signature,
	valueSignature *types.Signature,
	detached bool,
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
	target, err := interfaceoperation.Apply(
		context,
		children,
		selector.X,
		selection.Recv(),
		receiver,
		method,
		arguments,
		argumentBefore,
		argumentRequests,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if detached {
		return cooperativecall.DetachedValueCall(
			context,
			source,
			valueSignature,
			target,
		)
	}
	return cooperativecall.ValueCall(
		context,
		source,
		valueSignature,
		target,
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
