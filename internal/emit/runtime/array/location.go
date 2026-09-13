package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const StorageLocationMember = "$location"

func regionValue(factory tsgo.Factory) tsgo.Expression {
	return property(factory, factory.ThisExpression(), "$region")
}

func locationMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	return method(factory, nil, StorageLocationMember, nil, nil,
		memoryview.RegionType(factory, typeReference(factory, "T")),
		[]tsgo.Statement{factory.ReturnStatement(regionValue(factory))})
}

func buildLocationOperation(factory tsgo.Factory) (tsgo.Statement, error) {
	locationContract, err := api.RuntimeContract(api.RuntimeArrayLocation)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, factory.Identifier(locationContract.ExportedName()),
		typeParameters(factory), []tsgo.ParameterDeclaration{parameter(factory, nil, "value",
			arrayType(factory, arrayContract.ExportedName(), elementType, lengthType))}, memoryview.RegionType(factory, elementType),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(call(factory,
			property(factory, factory.Identifier("value"), StorageLocationMember), nil))}, true)), nil
}
