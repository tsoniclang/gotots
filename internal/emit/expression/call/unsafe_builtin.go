package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	unsafeoperation "github.com/tsoniclang/gotots/internal/emit/expression/builtin/unsafeoperation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
		facts, ok := context.TypesInfo().Types[source]
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
		signature, ok := context.TypesInfo().TypeOf(source.Fun).(*types.Signature)
		if !ok || signature == nil {
			return api.ExpressionEmission{},
				true,
				api.Unsupported(
					context,
					api.CategoryExpression,
					source,
				)
		}
		if err := validateResults(
			context,
			source,
			signature,
			discarded,
		); err != nil {
			return api.ExpressionEmission{}, true, err
		}
		arguments, before, requests, err := emitArguments(
			context,
			children,
			source,
			signature,
			false,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		reference, err := context.Names().Reference(builtin)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		requirement, err := api.NewEnvironmentBuiltinRequest(
			builtin,
			signature,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
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
				reference.Requests(),
				requests,
				[]api.RootRequest{requirement},
			),
		)
		return target, true, err
	default:
		return api.ExpressionEmission{}, false, nil
	}
}
