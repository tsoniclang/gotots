package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
)

func emitUnsafeBuiltin(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	kind := unsafeoperation.Classify(builtin)
	switch {
	case kind.Constant():
		facts, ok := context.TypesInfo().TypeAndValue(source)
		if !ok || facts.Type == nil || facts.Value == nil {
			return api.ExpressionEmission{},
				true,
				api.Unsupported(
					context,
					api.CategoryExpression,
					source,
				)
		}
		target, err := constantvalue.EmitValue(
			context.WithRole(api.RoleBuiltinArgument),
			source,
			facts.Type,
			facts.Value,
		)
		return target, true, err
	case kind.Runtime():
		if kind == unsafeoperation.Add {
			target, err := emitUnsafeAdd(context, children, source)
			return target, true, err
		}
		if kind == unsafeoperation.String {
			target, err := emitUnsafeString(
				context,
				children,
				source,
				discarded,
			)
			return target, true, err
		}
		return api.ExpressionEmission{}, true, api.Unsupported(
			context,
			api.CategoryExpression,
			source,
		)
	default:
		return api.ExpressionEmission{}, false, nil
	}
}
