package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
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
		signature.Variadic() ||
		signature.TypeParams().Len() != 0 ||
		signature.RecvTypeParams().Len() != 0 ||
		selection.Indirect() ||
		len(selection.Index()) != 1 ||
		!types.Identical(selection.Recv(), signature.Recv().Type()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	named, ok := types.Unalias(signature.Recv().Type()).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if _, ok := named.Underlying().(*types.Struct); !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(signature.Recv().Type()),
		selector.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = context.Values().Copy(
		context.WithRole(api.RoleReceiverValue),
		selector.X,
		signature.Recv().Type(),
		receiver,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := receiver.Before()
	receiverValue := receiver.Value()
	var receiverRequests []api.PlacementRequest
	if len(argumentBefore) != 0 {
		receiverValue, receiverRequests, before, err = captureReceiver(
			context,
			children,
			selector.X,
			signature.Recv().Type(),
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

func captureReceiver(
	context api.Context,
	_ api.ChildEmitter,
	_ ast.Node,
	_ types.Type,
	receiver api.ExpressionEmission,
) (
	tsgo.Expression,
	[]api.PlacementRequest,
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
