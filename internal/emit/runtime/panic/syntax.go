package panicruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func parameter(
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

func parameterProperty(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
	modifiers ...tsgo.ModifierLike,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		modifiers,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}

func readonlyObjectSet(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("ReadonlySet"),
		[]tsgo.TypeNode{
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword),
		},
	)
}

func readonlyObjectArray(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeOperatorNode(
		tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
		factory.ArrayTypeNode(
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword),
		),
	)
}
