package mapruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	if symbol != api.RuntimeMapClear {
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
	clearName, err := Name(MemberClear)
	if err != nil {
		return nil, err
	}
	target := factory.Identifier("target")
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
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{factory.ExpressionStatement(
			factory.CallExpression(
				factory.PropertyAccessExpression(
					target,
					nil,
					factory.Identifier(clearName),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		)}, true),
	), nil
}
