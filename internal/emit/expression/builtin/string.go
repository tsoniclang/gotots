package builtin

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitStringLength(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	if len(source.Args) != 1 ||
		discarded ||
		context.ExpectedResults() != nil ||
		!basictype.SupportsInteger(
			context.TypesSizes(),
			context.TypesInfo().TypeOf(source),
		) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	if !supportsStringArgument(argumentType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(context.TypesInfo().TypeOf(source), expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	expectedType, ok := stringArgumentExpectedType(argumentType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source.Args[0])
	}
	defined, definedArgument := definedtype.ResolveBasic(argumentType)
	value, err := children.Expression(
		context.
			WithRole(api.RoleCallArgument).
			WithExpectedType(expectedType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedArgument {
		value, err = defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	length := tsgo.Expression(context.Factory().PropertyAccessExpression(
		value.Value(),
		nil,
		context.Factory().Identifier("length"),
		tsgo.NodeFlagsNone,
	))
	if context.IntegerRepresentation() == api.IntegerRepresentationBigInt {
		length = context.Factory().CallExpression(
			context.Factory().Identifier("BigInt"),
			nil,
			nil,
			[]tsgo.Expression{length},
			tsgo.NodeFlagsNone,
		)
	}
	return api.NewExpressionEmission(
		value.Before(),
		length,
		value.Requests(),
	)
}

func supportsStringArgument(sourceType types.Type) bool {
	_, ok := stringArgumentExpectedType(sourceType)
	return ok
}

func stringArgumentExpectedType(sourceType types.Type) (types.Type, bool) {
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
		if !basictype.SupportsString(defined.Underlying()) {
			return nil, false
		}
		return defined.Type(), true
	}
	if !basictype.SupportsString(sourceType) {
		return nil, false
	}
	return types.Typ[types.String], true
}

func projectDefinedString(
	context api.Context,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	defined, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return value, nil
	}
	if !basictype.SupportsString(defined.Underlying()) {
		return api.ExpressionEmission{},
			&api.InvariantError{
				Role:   context.Role(),
				Reason: "defined-string projection received a non-string type",
			}
	}
	return defined.Project(context, value)
}
