package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointerruntime "github.com/tsoniclang/gotots/internal/emit/runtime/pointer"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
		signature.TypeParams().Len() != 0 ||
		signature.RecvTypeParams().Len() != 0 ||
		len(selection.Index()) != 1 ||
		!receiverTypesMatch(selection.Recv(), signature.Recv().Type()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	declaredReceiver := signature.Recv().Type()
	receiverBase := declaredReceiver
	if pointer, _, ok := pointertype.Resolve(receiverBase); ok {
		receiverBase = pointer.Elem()
	}
	named, ok := types.Unalias(receiverBase).(*types.Named)
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
	receiver, err := emitSelectedReceiver(
		context,
		children,
		selector,
		selection,
		declaredReceiver,
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

func receiverTypesMatch(actual types.Type, declared types.Type) bool {
	if types.Identical(actual, declared) {
		return true
	}
	if pointer, _, ok := pointertype.Resolve(actual); ok &&
		types.Identical(pointer.Elem(), declared) {
		return true
	}
	if pointer, _, ok := pointertype.Resolve(declared); ok &&
		types.Identical(actual, pointer.Elem()) {
		return true
	}
	return false
}

func emitSelectedReceiver(
	context api.Context,
	children api.ChildEmitter,
	selector *ast.SelectorExpr,
	selection *types.Selection,
	declared types.Type,
) (api.ExpressionEmission, error) {
	actual := selection.Recv()
	_, declaredElement, declaredPointer := pointertype.Resolve(declared)
	_, actualElement, actualPointer := pointertype.Resolve(actual)
	switch {
	case declaredPointer && actualPointer:
		if !types.Identical(declaredElement, actualElement) {
			break
		}
		return children.Expression(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(declared),
			selector.X,
		)
	case declaredPointer:
		if !types.Identical(actual, declaredElement) {
			break
		}
		return children.Address(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(declared),
			selector.X,
		)
	case actualPointer:
		if !types.Identical(actualElement, declared) {
			break
		}
		pointer, err := children.Expression(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(actual),
			selector.X,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value, err := dereferenceReceiver(
			context,
			children,
			selector.X,
			declared,
			pointer,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return context.Values().Copy(
			context.WithRole(api.RoleReceiverValue),
			selector.X,
			declared,
			value,
		)
	default:
		if !types.Identical(actual, declared) {
			break
		}
		value, err := children.Expression(
			context.
				WithRole(api.RoleReceiverValue).
				WithExpectedType(declared),
			selector.X,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return context.Values().Copy(
			context.WithRole(api.RoleReceiverValue),
			selector.X,
			declared,
			value,
		)
	}
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryExpression, selector)
}

func dereferenceReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	element types.Type,
	pointer api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	targetElement, err := children.RepresentedType(
		context.WithRole(api.RoleReceiverValue),
		source,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimePointer,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		pointer.Before(),
		pointerruntime.CellValue(
			context.Factory(),
			reference.Name(),
			targetElement.Value(),
			pointer.Value(),
		),
		api.CombineRequests(
			pointer.Requests(),
			targetElement.Requests(),
			reference.Requests(),
		),
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
