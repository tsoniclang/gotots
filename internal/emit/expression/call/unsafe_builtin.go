package call

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
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
		if kind == unsafeoperation.SliceData {
			target, err := emitUnsafeSliceData(context, children, source)
			return target, true, err
		}
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

func emitUnsafeSliceData(context api.Context, children api.ChildEmitter, source *ast.CallExpr) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 || source.Ellipsis != token.NoPos {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	sourceType := context.TypesInfo().TypeOf(source.Args[0])
	_, element, ok := slicevalue.Source(sourceType)
	if !ok {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(context.WithRole(api.RoleCallArgument).WithExpectedType(sourceType), source.Args[0])
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiver, err = slicevalue.Project(context, sourceType, receiver)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return slicevalue.Data(context, source, element, receiver)
}
