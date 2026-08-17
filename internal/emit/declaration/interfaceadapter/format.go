package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func formatStringProperty(
	factory tsgo.Factory,
	sourceType types.Type,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
		factory.Identifier(interfacecontract.FormatStringMember),
		nil,
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		formatStringValue(factory, sourceType),
	)
}

func formatStringValue(
	factory tsgo.Factory,
	sourceType types.Type,
) tsgo.Expression {
	basic, ok := formatBasicType(sourceType)
	if ok && basic.Kind() == types.String {
		return factory.TrueLiteral()
	}
	return factory.FalseLiteral()
}

func formatMethod(
	context api.Context,
	sourceType types.Type,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	body, requests, err := formatOperationBody(
		context,
		sourceType,
		payload(context.Factory(), context.Factory().ThisExpression()),
	)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(interfacecontract.FormatMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			formatParameter(
				factory,
				"verb",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			formatParameter(
				factory,
				"_flags",
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
			),
			formatParameter(factory, "precision", formatPrecisionType(factory)),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindStringKeyword),
		factory.Block(body, true),
	), requests, nil
}

func formatOperationBody(
	context api.Context,
	sourceType types.Type,
	sourceValue tsgo.Expression,
) ([]tsgo.Statement, []api.RootRequest, error) {
	helper, err := context.Names().Runtime(
		api.RuntimeInterfaceFormat,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	factory := context.Factory()
	value := api.DirectExpression(factory.Identifier("undefined"))
	basic, basicOK := formatBasicType(sourceType)
	arguments := make([]tsgo.Expression, 0, 4)
	member := interfacecontract.FormatOtherMember
	if basicOK {
		switch basic.Info() & (types.IsBoolean | types.IsString | types.IsInteger | types.IsFloat) {
		case types.IsBoolean:
			member = interfacecontract.FormatBooleanMember
		case types.IsString:
			member = interfacecontract.FormatStringValueMember
			arguments = append(arguments, factory.Identifier("precision"))
		case types.IsInteger:
			member = interfacecontract.FormatIntegerMember
		case types.IsFloat:
			member = interfacecontract.FormatFloatMember
			arguments = append(arguments, factory.Identifier("precision"))
		default:
			basicOK = false
		}
		if basicOK {
			value, _, _, err = formatValue(context, sourceType, sourceValue)
			if err != nil {
				return nil, nil, err
			}
			arguments = append([]tsgo.Expression{value.Value()}, arguments...)
		}
	}
	arguments = append(
		arguments,
		factory.StringLiteral(dynamicTypeSpelling(sourceType), tsgo.TokenFlagsNone),
		factory.Identifier("verb"),
	)
	call := factory.CallExpression(
		factory.PropertyAccessExpression(
			helper.Expression(factory),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	body := append([]tsgo.Statement(nil), value.Before()...)
	body = append(body, factory.ReturnStatement(call))
	return body, api.CombineRequests(
		value.Requests(),
		helper.Requests(),
	), nil
}

func formatValue(
	context api.Context,
	sourceType types.Type,
	sourceValue tsgo.Expression,
) (api.ExpressionEmission, *types.Basic, bool, error) {
	value := api.DirectExpression(sourceValue)
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
		basic, basicOK := defined.Basic()
		projected, err := defined.Project(context, value)
		return projected, basic, basicOK, err
	}
	basic, basicOK := formatBasicType(sourceType)
	return value, basic, basicOK, nil
}

func formatBasicType(sourceType types.Type) (*types.Basic, bool) {
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
		return defined.Basic()
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	return basic, ok
}

func dynamicTypeSpelling(sourceType types.Type) string {
	return types.TypeString(sourceType, func(source *types.Package) string {
		return source.Name()
	})
}

func formatParameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func formatPrecisionType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
	})
}
