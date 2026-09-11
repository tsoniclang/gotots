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
	case api.RuntimeArrayLocation:
		return buildLocationOperation(factory)
	case api.RuntimeArrayPacked:
		return buildPackedOperation(factory)
	case api.RuntimeArrayFromRegion:
		return buildRegionOperation(factory)
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
