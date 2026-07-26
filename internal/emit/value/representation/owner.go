package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct{}

func (Owner) RequiresCustomEquality(sourceType types.Type) bool {
	_, _, ok := namedStruct(sourceType)
	return ok
}

func (Owner) Zero(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.ExpressionEmission, error) {
	if alias, ok := exactPrimitive(context, sourceType); ok {
		reference, err := context.Names().Primitive(alias)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		var literal tsgo.Expression
		if alias == api.PrimitiveBool {
			literal = context.Factory().FalseLiteral()
		} else {
			literal = context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone)
		}
		return api.DirectExpression(
			context.Factory().AsExpression(
				literal,
				context.Factory().TypeReferenceNode(
					context.Factory().Identifier(reference.Name()),
					nil,
				),
			),
			reference.Requests()...,
		), nil
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(typeName)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		staticCall(context, reference.Name(), "$zero", nil),
		reference.Requests()...,
	), nil
}

func (Owner) Copy(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, ok := primitive(context, sourceType); ok {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if ownsFreshValue(context, source) {
		return api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			value.Requests(),
		)
	}
	reference, err := context.Names().Reference(typeName)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		staticCall(
			context,
			reference.Name(),
			"$copy",
			[]tsgo.Expression{value.Value()},
		),
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
}

func ownsFreshValue(context api.Context, source ast.Node) bool {
	switch source := source.(type) {
	case *ast.CallExpr, *ast.CompositeLit:
		return true
	case *ast.ParenExpr:
		return ownsFreshValue(context, source.X)
	case *ast.SelectorExpr:
		selection := context.TypesInfo().Selections[source]
		return selection != nil &&
			selection.Kind() == types.FieldVal &&
			!selection.Indirect() &&
			len(selection.Index()) == 1 &&
			ownsFreshValue(context, source.X)
	default:
		return false
	}
}

func (Owner) Assign(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	target tsgo.Expression,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if _, ok := primitive(context, sourceType); ok {
		return api.NewExpressionEmission(
			value.Before(),
			context.Factory().BinaryExpression(
				nil,
				target,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				value.Value(),
			),
			value.Requests(),
		)
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(typeName)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		staticCall(
			context,
			reference.Name(),
			"$assign",
			[]tsgo.Expression{target, value.Value()},
		),
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
}

func (Owner) Equal(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if _, ok := exactPrimitive(context, sourceType); ok {
		return api.DirectExpression(context.Factory().BinaryExpression(
			nil,
			left,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			right,
		)), nil
	}
	typeName, _, ok := namedStruct(sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reference, err := context.Names().Reference(typeName)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(
		staticCall(
			context,
			reference.Name(),
			"$equal",
			[]tsgo.Expression{left, right},
		),
		reference.Requests()...,
	), nil
}

func primitive(
	context api.Context,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	return api.PrimitiveAliasFor(context.TypesSizes(), sourceType)
}

func exactPrimitive(
	context api.Context,
	sourceType types.Type,
) (api.PrimitiveAlias, bool) {
	alias, ok := primitive(context, sourceType)
	return alias, ok && alias != api.PrimitiveInt64
}

func namedStruct(
	sourceType types.Type,
) (*types.TypeName, *types.Struct, bool) {
	if sourceType == nil {
		return nil, nil, false
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return nil, nil, false
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok || named.Obj() == nil {
		return nil, nil, false
	}
	return named.Obj(), structType, true
}

func staticCall(
	context api.Context,
	typeName string,
	operation string,
	arguments []tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(typeName),
			nil,
			context.Factory().Identifier(operation),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}
