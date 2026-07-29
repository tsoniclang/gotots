package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(
		context.TypesInfo(),
		source.Fun,
	)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := instance.Type.(*types.Signature)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	callable, ok, err := context.ResolveGenericCallable(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !ok ||
		instance.TypeArgs == nil ||
		instance.TypeArgs.Len() != len(callable.Parameters()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	typeArguments, typeRequests, err := genericinstance.EmitTypeArguments(
		context,
		children,
		source,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	capabilities, capabilityRequests, err := genericinstance.EmitCapabilities(
		context,
		source,
		callable,
		instance.TypeArgs,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments = append(capabilities, arguments...)
	result, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			typeArguments,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			reference.Requests(),
			typeRequests,
			capabilityRequests,
			argumentRequests,
		),
	)
	return result, true, err
}

func genericFunctionInstance(
	info *types.Info,
	source ast.Expr,
) (*types.Func, types.Instance, bool) {
	if info == nil || source == nil {
		return nil, types.Instance{}, false
	}
	base := source
	switch selected := source.(type) {
	case *ast.IndexExpr:
		base = selected.X
	case *ast.IndexListExpr:
		base = selected.X
	}
	var identifier *ast.Ident
	switch selected := base.(type) {
	case *ast.Ident:
		identifier = selected
	case *ast.SelectorExpr:
		if info.Selections[selected] != nil {
			return nil, types.Instance{}, false
		}
		identifier = selected.Sel
	default:
		return nil, types.Instance{}, false
	}
	instance, instantiated := info.Instances[identifier]
	owner, function := info.Uses[identifier].(*types.Func)
	if !instantiated || !function {
		return nil, types.Instance{}, false
	}
	return owner.Origin(), instance, true
}
