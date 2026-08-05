package reflectiontype

import (
	"go/types"
	"strconv"

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
		expressionProperty(factory, "zero", factory.ArrowFunction(
			nil,
			nil,
			nil,
			factory.TypeReferenceNode(
				scaffold.boxType.EntityName(factory),
				nil,
			),
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{zero},
			)),
		)),
	}
	boxing, err := scalarBoxingProperty(context, scaffold, basic)
	if err != nil {
		return nil, err
	}
	if boxing != nil {
		properties = append(properties, boxing)
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

// scalarBoxingProperty derives the typed boxing callback of one basic
// scalar: signed and unsigned integers truncate through the exact
// BigInt.asIntN/asUintN width before narrowing to the product carrier,
// float32 rounds through Math.fround, and bool boxes identically. The
// truncation is the Go conversion semantics of SetInt, SetUint, SetFloat,
// SetBool, and Convert.
func scalarBoxingProperty(
	context api.Context,
	scaffold *locationScaffold,
	basic *types.Basic,
) (tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	info := basic.Info()
	var name string
	var wide api.PrimitiveAlias
	switch {
	case info&types.IsBoolean != 0:
		name, wide = "boxBool", api.PrimitiveBool
	case info&types.IsString != 0:
		return nil, nil
	case info&types.IsUnsigned != 0:
		name, wide = "boxUint", api.PrimitiveUint64
	case info&types.IsInteger != 0:
		name, wide = "boxInt", api.PrimitiveInt64
	case info&types.IsFloat != 0:
		name, wide = "boxFloat", api.PrimitiveFloat64
	default:
		return nil, nil
	}
	parameterType, err := context.Names().ProviderPrimitive(wide)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, parameterType.Requests()...)
	var narrowed tsgo.Expression = factory.Identifier("value")
	switch {
	case info&types.IsInteger != 0:
		bits := context.TypesSizes().Sizeof(basic) * 8
		truncator := "asIntN"
		if info&types.IsUnsigned != 0 {
			truncator = "asUintN"
		}
		alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), basic)
		if !ok {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: basic.String(),
				Reason:   "reflection value scalar has no primitive alias",
			}
		}
		carrier, carrierErr := api.IntegerCarrierRepresentation(
			alias,
			context.ScalarABI(),
		)
		if carrierErr != nil {
			return nil, carrierErr
		}
		if bits != 64 || carrier == api.IntegerCarrierNumber {
			narrowed = factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("globalThis"),
						nil,
						factory.Identifier("BigInt"),
						tsgo.NodeFlagsNone,
					),
					nil,
					factory.Identifier(truncator),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{
					factory.NumericLiteral(
						strconv.FormatInt(bits, 10),
						tsgo.TokenFlagsNone,
					),
					narrowed,
				},
				tsgo.NodeFlagsNone,
			)
		}
		if carrier == api.IntegerCarrierNumber {
			narrowed = factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.Identifier("globalThis"),
					nil,
					factory.Identifier("Number"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{narrowed},
				tsgo.NodeFlagsNone,
			)
		}
	case info&types.IsFloat != 0:
		if context.TypesSizes().Sizeof(basic) == 4 {
			narrowed = factory.CallExpression(
				factory.PropertyAccessExpression(
					factory.PropertyAccessExpression(
						factory.Identifier("globalThis"),
						nil,
						factory.Identifier("Math"),
						tsgo.NodeFlagsNone,
					),
					nil,
					factory.Identifier("fround"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{narrowed},
				tsgo.NodeFlagsNone,
			)
		}
	}
	return expressionProperty(factory, name, factory.ArrowFunction(
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
		factory.ParenthesizedExpression(factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{narrowed},
		)),
	)), nil
}
