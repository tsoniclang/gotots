package operand

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

type model struct {
	expected types.Type
	defined  definedtype.Model
}

func Supports(sizes types.Sizes, sourceType types.Type) bool {
	_, ok := resolve(sizes, sourceType)
	return ok
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.ExpressionEmission, error) {
	if children == nil || source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "integer operand input is invalid",
		}
	}
	selected, ok := resolve(
		context.TypesSizes(),
		context.TypesInfo().TypeOf(source),
	)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emission, err := children.Expression(
		context.WithExpectedType(selected.expected),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if selected.defined.Type() == nil {
		return emission, nil
	}
	return selected.defined.Project(context, emission)
}

func resolve(sizes types.Sizes, sourceType types.Type) (model, bool) {
	if sizes == nil || sourceType == nil {
		return model{}, false
	}
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 &&
		basic.Info()&types.IsInteger != 0 {
		return model{expected: types.Typ[types.Int]}, true
	}
	if _, ok := integervalue.Describe(sizes, sourceType); ok {
		return model{expected: sourceType}, true
	}
	defined, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return model{}, false
	}
	if _, ok := integervalue.Describe(sizes, defined.Underlying()); !ok {
		return model{}, false
	}
	return model{expected: sourceType, defined: defined}, true
}
