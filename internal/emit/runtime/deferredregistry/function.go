package deferredregistry

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func entries(factory tsgo.Factory) tsgo.PropertyDeclaration {
	source := typeReference(factory, sourceType)
	deferred := typeReference(factory, deferredType)
	mapType := factory.TypeReferenceNode(
		factory.Identifier("WeakMap"),
		[]tsgo.TypeNode{source, deferred},
	)
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(entriesMember),
		nil,
		mapType,
		factory.NewExpression(
			factory.Identifier("WeakMap"),
			[]tsgo.TypeNode{source, deferred},
			nil,
		),
	)
}

func register(factory tsgo.Factory) tsgo.MethodDeclaration {
	source := factory.Identifier("source")
	deferred := factory.Identifier("deferred")
	sourceTarget := typeReference(factory, sourceType)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier("register"),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, source.Text(), sourceTarget),
			parameter(factory, deferred.Text(), typeReference(factory, deferredType)),
		},
		sourceTarget,
		factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.CallExpression(
				memberAccess(factory, entriesMember, "set"),
				nil,
				nil,
				[]tsgo.Expression{source, deferred},
				tsgo.NodeFlagsNone,
			)),
			factory.ReturnStatement(source),
		}, true),
	)
}

func resolve(factory tsgo.Factory) tsgo.MethodDeclaration {
	source := factory.Identifier("source")
	undefined := undefinedType(factory)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier("resolve"),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(
			factory,
			source.Text(),
			factory.UnionTypeNode([]tsgo.TypeNode{
				typeReference(factory, sourceType),
				undefined,
			}),
		)},
		factory.UnionTypeNode([]tsgo.TypeNode{
			typeReference(factory, deferredType),
			undefined,
		}),
		factory.Block([]tsgo.Statement{
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					source,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
				factory.Block([]tsgo.Statement{
					factory.ReturnStatement(factory.Identifier("undefined")),
				}, true),
				nil,
			),
			factory.ReturnStatement(factory.CallExpression(
				memberAccess(factory, entriesMember, "get"),
				nil,
				nil,
				[]tsgo.Expression{source},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}
