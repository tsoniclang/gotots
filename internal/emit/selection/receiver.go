package selection

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func MethodReceiver(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (api.ExpressionEmission, *types.Func, error) {
	resolved, method, ok := methodPath(selected)
	if !ok || !Valid(context, source, selected, types.MethodVal) {
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature := method.Type().(*types.Signature)
	declared := signature.Recv().Type()
	_, declaredElement, _, declaredPointer := pointerType(declared)
	_, _, _, effectivePointer := pointerType(resolved.effective)
	if declaredPointer &&
		!effectivePointer &&
		types.Identical(declaredElement, resolved.effective) {
		receiver, err := addressSource(
			context,
			children,
			source,
			resolved,
		)
		return receiver, method, err
	}
	root, err := children.Expression(
		context.
			WithRole(api.RoleReceiverValue).
			WithExpectedType(resolved.root),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	receiver, err := methodSetReceiver(
		context,
		children,
		source,
		resolved,
		method,
		root,
	)
	return receiver, method, err
}

func MethodSetReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	selected *types.Selection,
	root api.ExpressionEmission,
) (api.ExpressionEmission, types.Type, *types.Func, error) {
	resolved, method, ok := methodPath(selected)
	if !ok ||
		selected.Kind() != types.MethodVal ||
		!types.Identical(selected.Recv(), resolved.root) {
		return api.ExpressionEmission{}, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := methodSetReceiver(
		context,
		children,
		source,
		resolved,
		method,
		root,
	)
	return receiver, resolved.effective, method, err
}

func methodSetReceiver(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	resolved path,
	method *types.Func,
	root api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	signature := method.Type().(*types.Signature)
	declared := signature.Recv().Type()
	_, declaredElement, _, declaredPointer := pointerType(declared)
	_, effectiveElement, _, effectivePointer :=
		pointerType(resolved.effective)
	if declaredPointer &&
		!effectivePointer &&
		types.Identical(declaredElement, resolved.effective) {
		if _, _, _, rootPointer := pointerType(resolved.root); !rootPointer {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return projectAddress(
			context,
			children,
			source,
			resolved,
			root,
			false,
		)
	}
	value, err := projectValue(
		context,
		children,
		source,
		resolved,
		root,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	switch {
	case declaredPointer &&
		effectivePointer &&
		types.Identical(declaredElement, effectiveElement):
		return value, nil
	case !declaredPointer &&
		effectivePointer &&
		types.Identical(declared, effectiveElement):
		value, _, err = dereferenceValue(
			context,
			children,
			source,
			resolved.effective,
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	case !declaredPointer &&
		!effectivePointer &&
		types.Identical(declared, resolved.effective):
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return context.Values().Copy(
		context.WithRole(api.RoleReceiverValue),
		source,
		declared,
		value,
	)
}

func MethodExpressionReceiver(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SelectorExpr,
	selected *types.Selection,
	root api.ExpressionEmission,
) (api.ExpressionEmission, *types.Func, error) {
	resolved, method, ok := methodPath(selected)
	if !ok || !Valid(context, source, selected, types.MethodExpr) {
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature := method.Type().(*types.Signature)
	declared := signature.Recv().Type()
	_, declaredElement, _, declaredPointer := pointerType(declared)
	_, effectiveElement, _, effectivePointer :=
		pointerType(resolved.effective)

	if declaredPointer &&
		!effectivePointer &&
		types.Identical(declaredElement, resolved.effective) {
		if _, _, _, rootPointer := pointerType(resolved.root); !rootPointer {
			return api.ExpressionEmission{}, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		receiver, err := projectAddress(
			context,
			children,
			source,
			resolved,
			root,
			false,
		)
		return receiver, method, err
	}
	value, err := projectValue(
		context,
		children,
		source,
		resolved,
		root,
	)
	if err != nil {
		return api.ExpressionEmission{}, nil, err
	}
	switch {
	case declaredPointer &&
		effectivePointer &&
		types.Identical(declaredElement, effectiveElement):
		return value, method, nil
	case !declaredPointer &&
		effectivePointer &&
		types.Identical(declared, effectiveElement):
		value, _, err = dereferenceValue(
			context,
			children,
			source,
			resolved.effective,
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, nil, err
		}
	case !declaredPointer &&
		!effectivePointer &&
		types.Identical(declared, resolved.effective):
	default:
		return api.ExpressionEmission{}, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleReceiverValue),
		source,
		declared,
		value,
	)
	return value, method, err
}

func DirectMethodExpression(
	context api.Context,
	source *ast.SelectorExpr,
	selected *types.Selection,
) (*types.Func, bool) {
	if !Valid(context, source, selected, types.MethodExpr) {
		return nil, false
	}
	resolved, method, ok := methodPath(selected)
	if !ok || len(resolved.fields) != 0 {
		return nil, false
	}
	signature := method.Type().(*types.Signature)
	return method, types.Identical(
		resolved.root,
		signature.Recv().Type(),
	)
}
