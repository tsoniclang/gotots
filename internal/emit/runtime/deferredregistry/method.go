package deferredregistry

import (
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func methodEntries(
	factory tsgo.Factory,
	member string,
	deferredTypeName string,
) tsgo.PropertyDeclaration {
	targetType := factory.TypeReferenceNode(
		factory.Identifier("Map"),
		[]tsgo.TypeNode{
			objectType(factory),
			mapType(factory, typeReference(factory, deferredTypeName)),
		},
	)
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(member),
		nil,
		targetType,
		factory.NewExpression(
			factory.Identifier("Map"),
			targetType.TypeArguments(),
			nil,
		),
	)
}

func registerMethod(
	factory tsgo.Factory,
	entriesMember string,
	methodName string,
	deferredTypeName string,
) tsgo.MethodDeclaration {
	method := factory.Identifier("method")
	dynamicType := factory.Identifier("dynamicType")
	deferred := factory.Identifier("deferred")
	entries := factory.Identifier("entries")
	deferredType := typeReference(factory, deferredTypeName)
	innerType := mapType(factory, deferredType)
	undefined := undefinedType(factory)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(methodName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, method.Text(), objectType(factory)),
			parameter(factory, dynamicType.Text(), objectType(factory)),
			parameter(factory, deferred.Text(), deferredType),
		},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindVoidKeyword),
		factory.Block([]tsgo.Statement{
			factory.VariableStatement(
				nil,
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{factory.VariableDeclaration(
						entries,
						nil,
						factory.UnionTypeNode([]tsgo.TypeNode{innerType, undefined}),
						factory.CallExpression(
							memberAccess(factory, entriesMember, "get"),
							nil,
							nil,
							[]tsgo.Expression{method},
							tsgo.NodeFlagsNone,
						),
					)},
					tsgo.NodeFlagsLet,
				),
			),
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					entries,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsEqualsEqualsToken,
					),
					factory.Identifier("undefined"),
				),
				factory.Block([]tsgo.Statement{
					factory.ExpressionStatement(factory.BinaryExpression(
						nil,
						entries,
						nil,
						factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
						factory.NewExpression(
							factory.Identifier("Map"),
							[]tsgo.TypeNode{objectType(factory), deferredType},
							nil,
						),
					)),
					factory.ExpressionStatement(factory.CallExpression(
						memberAccess(factory, entriesMember, "set"),
						nil,
						nil,
						[]tsgo.Expression{method, entries},
						tsgo.NodeFlagsNone,
					)),
				}, true),
				nil,
			),
			factory.ExpressionStatement(factory.CallExpression(
				factory.PropertyAccessExpression(
					entries,
					nil,
					factory.Identifier("set"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{dynamicType, deferred},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}

func resolveMethod(
	factory tsgo.Factory,
	entriesMember string,
	methodName string,
	interfaceValueName string,
	deferredTypeName string,
) tsgo.MethodDeclaration {
	method := factory.Identifier("method")
	receiver := factory.Identifier("receiver")
	entries := factory.Identifier("entries")
	undefined := undefinedType(factory)
	deferredType := typeReference(factory, deferredTypeName)
	return factory.MethodDeclaration(
		nil,
		nil,
		factory.Identifier(methodName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, method.Text(), objectType(factory)),
			parameter(
				factory,
				receiver.Text(),
				typeReference(factory, interfaceValueName),
			),
		},
		factory.UnionTypeNode([]tsgo.TypeNode{deferredType, undefined}),
		factory.Block([]tsgo.Statement{
			factory.VariableStatement(
				nil,
				factory.VariableDeclarationList(
					[]tsgo.VariableDeclaration{factory.VariableDeclaration(
						entries,
						nil,
						nil,
						factory.CallExpression(
							memberAccess(factory, entriesMember, "get"),
							nil,
							nil,
							[]tsgo.Expression{method},
							tsgo.NodeFlagsNone,
						),
					)},
					tsgo.NodeFlagsConst,
				),
			),
			factory.IfStatement(
				factory.BinaryExpression(
					nil,
					entries,
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
				factory.PropertyAccessExpression(
					entries,
					nil,
					factory.Identifier("get"),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{factory.PropertyAccessExpression(
					receiver,
					nil,
					factory.Identifier(interfacecontract.DynamicTypeMember),
					tsgo.NodeFlagsNone,
				)},
				tsgo.NodeFlagsNone,
			)),
		}, true),
	)
}
