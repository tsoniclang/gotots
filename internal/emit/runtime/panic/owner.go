package panicruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

const RaiseName = "raise"

func Build(factory tsgo.Factory, className string) tsgo.Statement {
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{typeParameter(factory)},
		nil,
		[]tsgo.ClassElement{
			constructor(factory),
			raise(factory, className),
		},
	)
}

func Call(
	factory tsgo.Factory,
	className string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(className),
			nil,
			factory.Identifier(RaiseName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}

func constructor(factory tsgo.Factory) tsgo.ConstructorDeclaration {
	return factory.ConstructorDeclaration(
		[]tsgo.ModifierLike{factory.PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			[]tsgo.ModifierLike{
				factory.PublicKeyword(),
				factory.ReadonlyKeyword(),
			},
			nil,
			factory.Identifier("value"),
			nil,
			typeReference(factory),
			nil,
		)},
		nil,
		factory.Block(nil, true),
	)
}

func raise(
	factory tsgo.Factory,
	className string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(RaiseName),
		nil,
		[]tsgo.TypeParameterDeclaration{typeParameter(factory)},
		[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
			nil,
			nil,
			factory.Identifier("value"),
			nil,
			typeReference(factory),
			nil,
		)},
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNeverKeyword,
		),
		factory.Block([]tsgo.Statement{factory.ThrowStatement(
			factory.NewExpression(
				factory.Identifier(className),
				[]tsgo.TypeNode{typeReference(factory)},
				[]tsgo.Expression{factory.Identifier("value")},
			),
		)}, true),
	)
}

func typeParameter(
	factory tsgo.Factory,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier("T"),
		nil,
		nil,
		nil,
	)
}

func typeReference(factory tsgo.Factory) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(factory.Identifier("T"), nil)
}
