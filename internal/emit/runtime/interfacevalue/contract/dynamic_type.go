package contract

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func DynamicType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.TypeLiteralNode([]tsgo.TypeElement{
		factory.PropertySignatureDeclaration(
			[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
			factory.Identifier(DynamicTypeComparable),
			nil,
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
			factory.OmittedExpression(),
		),
	})
}
