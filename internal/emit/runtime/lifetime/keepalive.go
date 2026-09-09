package lifetime

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeKeepAlive)
	if err != nil {
		return nil, err
	}
	valueContract, err := api.RuntimeContract(api.RuntimeInterfaceValue)
	if err != nil {
		return nil, err
	}
	marker, err := tsoniccore.Resolve(tsoniccore.SymbolKeepAlive)
	if err != nil {
		return nil, err
	}
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()}, nil,
		factory.Identifier(contract.ExportedName()), nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(nil, nil,
			factory.Identifier("value"), nil, factory.UnionTypeNode([]tsgo.TypeNode{
				factory.TypeReferenceNode(factory.Identifier(valueContract.ExportedName()), nil),
				factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword),
			}), nil)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{factory.ExpressionStatement(factory.CallExpression(
			factory.Identifier(marker.Export()), nil, nil,
			[]tsgo.Expression{factory.Identifier("value")}, tsgo.NodeFlagsNone,
		))}, true),
	), nil
}
