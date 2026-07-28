package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func BuildAggregateOperation(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimeArrayAllocate:
		return buildAllocateOperation(factory)
	case api.RuntimeArrayView:
		return buildViewOperation(factory)
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func buildAllocateOperation(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeArrayAllocate)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(contract.ExportedName()),
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(factory, nil, "length", lengthType),
		},
		arrayType(
			factory,
			arrayContract.ExportedName(),
			elementType,
			lengthType,
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(call(
				factory,
				property(
					factory,
					factory.Identifier(arrayContract.ExportedName()),
					StorageAllocateMember,
				),
				[]tsgo.TypeNode{elementType, lengthType},
				factory.Identifier("length"),
			)),
		}, true),
	), nil
}

func buildViewOperation(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeArrayView)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(contract.ExportedName()),
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				nil,
				"values",
				factory.ArrayTypeNode(elementType),
			),
			parameter(
				factory,
				nil,
				"offset",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			parameter(factory, nil, "length", lengthType),
		},
		arrayType(
			factory,
			arrayContract.ExportedName(),
			elementType,
			lengthType,
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(call(
				factory,
				property(
					factory,
					factory.Identifier(arrayContract.ExportedName()),
					StorageViewMember,
				),
				[]tsgo.TypeNode{elementType, lengthType},
				factory.Identifier("values"),
				factory.Identifier("offset"),
				factory.Identifier("length"),
			)),
		}, true),
	), nil
}
