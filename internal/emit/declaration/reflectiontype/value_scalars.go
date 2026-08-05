package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// basicValueProperties adds the zero-evidence callback for every basic
// scalar and the canonical boxing callback for strings.
func basicValueProperties(
	context api.Context,
	scaffold *locationScaffold,
	basic *types.Basic,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	zero, err := scalarZeroExpression(context, factory, basic)
	if err != nil || zero == nil {
		return nil, err
	}
	isZero := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.IsZero",
			factory.BinaryExpression(
				nil,
				boxPayload(factory),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				zero,
			),
		)),
	)
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "isZero", isZero),
	}
	if basic.Info()&types.IsString != 0 {
		parameterType, typeErr := context.Names().ProviderPrimitive(
			api.PrimitiveString,
		)
		if typeErr != nil {
			return nil, typeErr
		}
		scaffold.requests = append(
			scaffold.requests,
			parameterType.Requests()...,
		)
		boxString := factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				factory.TypeReferenceNode(
					parameterType.EntityName(factory),
					nil,
				),
				nil,
			)},
			factory.TypeReferenceNode(
				scaffold.boxType.EntityName(factory),
				nil,
			),
			factory.EqualsGreaterThanToken(),
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.Identifier("value")},
			),
		)
		properties = append(
			properties,
			expressionProperty(factory, "boxString", boxString),
		)
	}
	return properties, nil
}

// scalarZeroExpression is the exact target zero literal of one basic
// scalar under the product scalar representation. Unsupported basic kinds
// yield no zero evidence rather than an approximate literal.
func scalarZeroExpression(
	context api.Context,
	factory tsgo.Factory,
	basic *types.Basic,
) (tsgo.Expression, error) {
	info := basic.Info()
	switch {
	case info&types.IsBoolean != 0:
		return factory.FalseLiteral(), nil
	case info&types.IsString != 0:
		return factory.StringLiteral("", tsgo.TokenFlagsNone), nil
	case info&types.IsFloat != 0:
		return factory.NumericLiteral("0", tsgo.TokenFlagsNone), nil
	case info&types.IsInteger != 0:
		alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), basic)
		if !ok {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: basic.String(),
				Reason:   "reflection value scalar has no primitive alias",
			}
		}
		return api.IntegerLiteral(factory, context.ScalarABI(), alias, "0")
	default:
		return nil, nil
	}
}
