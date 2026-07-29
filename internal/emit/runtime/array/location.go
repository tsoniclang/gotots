package array

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const StorageLocationMember = "$location"

func locationMethod(factory tsgo.Factory) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	locationType := factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.TupleTypeNode([]tsgo.TypeNode{
			factory.ArrayTypeNode(elementType),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
		}),
	)
	return method(
		factory,
		[]tsgo.ModifierLike{factory.PublicKeyword()},
		StorageLocationMember,
		nil,
		nil,
		locationType,
		[]tsgo.Statement{factory.ReturnStatement(
			factory.ArrayLiteralExpression(
				[]tsgo.Expression{
					factory.PropertyAccessExpression(
						factory.ThisExpression(),
						nil,
						factory.Identifier("$values"),
						tsgo.NodeFlagsNone,
					),
					factory.PropertyAccessExpression(
						factory.ThisExpression(),
						nil,
						factory.Identifier("$offset"),
						tsgo.NodeFlagsNone,
					),
				},
				false,
			),
		)},
	)
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
	locationType := factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.TupleTypeNode([]tsgo.TypeNode{
			factory.ArrayTypeNode(elementType),
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindNumberKeyword,
			),
		}),
	)
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		nil,
		factory.Identifier(locationContract.ExportedName()),
		typeParameters(factory),
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				nil,
				"value",
				arrayType(
					factory,
					arrayContract.ExportedName(),
					elementType,
					lengthType,
				),
			),
		},
		locationType,
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(call(
				factory,
				property(factory, value, StorageLocationMember),
				nil,
			)),
		}, true),
	), nil
}
