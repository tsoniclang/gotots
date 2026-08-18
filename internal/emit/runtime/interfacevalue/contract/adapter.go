package contract

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func AdapterConstructor(
	factory tsgo.Factory,
	valueName string,
	payload tsgo.TypeNode,
) tsgo.TypeNode {
	instance := factory.IntersectionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(factory.Identifier(valueName), nil),
		factory.TypeLiteralNode([]tsgo.TypeElement{
			factory.PropertySignatureDeclaration(
				[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
				factory.Identifier(PayloadMember),
				nil,
				payload,
				factory.OmittedExpression(),
			),
		}),
	})
	value := factory.Identifier("value")
	return factory.TypeLiteralNode([]tsgo.TypeElement{
		factory.ConstructSignatureDeclaration(
			nil,
			[]tsgo.ParameterDeclaration{adapterParameter(
				factory,
				PayloadMember,
				payload,
			)},
			instance,
		),
		factory.MethodSignatureDeclaration(
			nil,
			factory.Identifier("$is"),
			nil,
			nil,
			[]tsgo.ParameterDeclaration{adapterParameter(
				factory,
				"value",
				factory.UnionTypeNode([]tsgo.TypeNode{
					factory.TypeReferenceNode(factory.Identifier(valueName), nil),
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			)},
			factory.TypePredicateNode(nil, value, instance),
		),
	})
}

func adapterParameter(
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
