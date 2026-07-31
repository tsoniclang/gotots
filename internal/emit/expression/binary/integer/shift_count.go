package integer

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func projectDefinedShiftCount(
	context api.Context,
	source *ast.BinaryExpr,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if source.Op != token.SHL && source.Op != token.SHR {
		return target, nil
	}
	model, ok := definedtype.ResolveBasic(
		context.TypesInfo().TypeOf(source.Y),
	)
	if !ok {
		return target, nil
	}
	underlying, ok := model.Basic()
	if !ok {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   api.RoleBinaryRight,
			Reason: "defined shift count has no basic underlying type",
		}
	}
	if _, ok := integervalue.Describe(
		context.TypesSizes(),
		underlying,
	); !ok {
		return api.ExpressionEmission{},
			api.Unsupported(
				context.WithRole(api.RoleBinaryRight),
				api.CategoryExpression,
				source.Y,
			)
	}
	return model.Project(context.WithRole(api.RoleBinaryRight), target)
}

func isDefinedIntegerShiftCount(
	sizes types.Sizes,
	sourceType types.Type,
) bool {
	model, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return false
	}
	underlying, ok := model.Basic()
	if !ok {
		return false
	}
	_, ok = integervalue.Describe(sizes, underlying)
	return ok
}
