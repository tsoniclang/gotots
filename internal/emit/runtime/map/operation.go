package mapruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	if symbol != api.RuntimeMapClear && symbol != api.RuntimeMapKeys {
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	mapContract, err := api.RuntimeContract(api.RuntimeMap)
	if err != nil {
		return nil, err
	}
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)
	member := MemberClear
	if symbol == api.RuntimeMapKeys {
		member = MemberKeys
	}
	memberName, err := Name(member)
	if err != nil {
		return nil, err
	}
	target := factory.Identifier("target")
	resultType := tsgo.TypeNode(
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
	)
	bodyExpression := tsgo.Expression(factory.CallExpression(
		factory.PropertyAccessExpression(
			target,
			nil,
			factory.Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		nil,
		tsgo.NodeFlagsNone,
	))
	body := tsgo.Statement(factory.ExpressionStatement(bodyExpression))
	if symbol == api.RuntimeMapKeys {
		resultType = factory.ArrayTypeNode(keyType)
		body = factory.ReturnStatement(bodyExpression)
	}
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(contract.ExportedName()),
		[]tsgo.TypeParameterDeclaration{
			keyTypeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			target.Text(),
			runtimeMapType(
				factory,
				mapContract.ExportedName(),
				keyType,
				valueType,
			),
		)},
		resultType,
		factory.Block([]tsgo.Statement{body}, true),
	), nil
}
