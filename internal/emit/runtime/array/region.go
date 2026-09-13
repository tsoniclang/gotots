package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/runtime/memoryview"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const StorageRegionMember = "$fromRegion"

func regionMethod(factory tsgo.Factory, exportedName string) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return method(factory, []tsgo.ModifierLike{factory.StaticKeyword()}, StorageRegionMember, typeParameters(factory),
		[]tsgo.ParameterDeclaration{parameter(factory, nil, "region", memoryview.RegionType(factory, elementType)),
			parameter(factory, nil, "length", lengthType)}, arrayType(factory, exportedName, elementType, lengthType),
		[]tsgo.Statement{factory.ReturnStatement(factory.NewExpression(factory.Identifier(exportedName), []tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{factory.Identifier("region"), factory.Identifier("length")}))})
}

func buildRegionOperation(factory tsgo.Factory) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(api.RuntimeArrayFromRegion)
	if err != nil {
		return nil, err
	}
	arrayContract, err := api.RuntimeContract(api.RuntimeArray)
	if err != nil {
		return nil, err
	}
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return factory.FunctionDeclaration([]tsgo.ModifierLike{factory.ExportKeyword()}, nil, factory.Identifier(contract.ExportedName()), typeParameters(factory),
		[]tsgo.ParameterDeclaration{parameter(factory, nil, "region", memoryview.RegionType(factory, elementType)),
			parameter(factory, nil, "length", lengthType)}, arrayType(factory, arrayContract.ExportedName(), elementType, lengthType),
		factory.Block([]tsgo.Statement{factory.ReturnStatement(call(factory,
			property(factory, factory.Identifier(arrayContract.ExportedName()), StorageRegionMember), []tsgo.TypeNode{elementType, lengthType},
			factory.Identifier("region"), factory.Identifier("length")))}, true)), nil
}
