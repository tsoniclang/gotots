package array

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const StorageViewMember = "$view"

func viewMethod(
	factory tsgo.Factory,
	exportedName string,
) tsgo.MethodDeclaration {
	elementType := typeReference(factory, "T")
	lengthType := typeReference(factory, "N")
	return method(
		factory,
		[]tsgo.ModifierLike{
			factory.PublicKeyword(),
			factory.StaticKeyword(),
		},
		StorageViewMember,
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
		arrayType(factory, exportedName, elementType, lengthType),
		[]tsgo.Statement{factory.ReturnStatement(factory.NewExpression(
			factory.Identifier(exportedName),
			[]tsgo.TypeNode{elementType, lengthType},
			[]tsgo.Expression{
				factory.Identifier("values"),
				factory.Identifier("offset"),
				factory.Identifier("length"),
			},
		))},
	)
}
