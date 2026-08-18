package array

import (
	arraymember "github.com/tsoniclang/gotots/internal/emit/runtime/array/member"
	indexedstorage "github.com/tsoniclang/gotots/internal/emit/runtime/indexedstorage"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func typeParameters(factory tsgo.Factory) []tsgo.TypeParameterDeclaration {
	return []tsgo.TypeParameterDeclaration{
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("T"),
			nil,
			nil,
			nil,
		),
		factory.TypeParameterDeclaration(
			nil,
			factory.Identifier("N"),
			factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNumberKeyword),
			nil,
			nil,
		),
	}
}

func typeReference(factory tsgo.Factory, name string) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(factory.Identifier(name), nil)
}

func arrayType(
	factory tsgo.Factory,
	exportedName string,
	elementType tsgo.TypeNode,
	lengthType tsgo.TypeNode,
) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(
		factory.Identifier(exportedName),
		[]tsgo.TypeNode{elementType, lengthType},
	)
}

func method(
	factory tsgo.Factory,
	modifiers []tsgo.ModifierLike,
	name string,
	typeParameters []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body []tsgo.Statement,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		modifiers,
		nil,
		factory.Identifier(name),
		nil,
		typeParameters,
		parameters,
		result,
		factory.Block(body, true),
	)
}

func runtimeMethod(
	factory tsgo.Factory,
	modifiers []tsgo.ModifierLike,
	member arraymember.Identity,
	typeParameters []tsgo.TypeParameterDeclaration,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	body []tsgo.Statement,
) tsgo.MethodDeclaration {
	return method(
		factory,
		modifiers,
		member.Name(),
		typeParameters,
		parameters,
		result,
		body,
	)
}

func parameter(
	factory tsgo.Factory,
	modifiers []tsgo.ModifierLike,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		modifiers,
		nil,
		factory.Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func variable(
	factory tsgo.Factory,
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(name),
				nil,
				targetType,
				value,
			)},
			flags,
		),
	)
}

func property(
	factory tsgo.Factory,
	value tsgo.Expression,
	name string,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		value,
		nil,
		factory.Identifier(name),
		tsgo.NodeFlagsNone,
	)
}

func runtimeProperty(
	factory tsgo.Factory,
	value tsgo.Expression,
	member arraymember.Identity,
) tsgo.PropertyAccessExpression {
	return property(factory, value, member.Name())
}

func element(
	factory tsgo.Factory,
	value tsgo.Expression,
	index tsgo.Expression,
) tsgo.ElementAccessExpression {
	return factory.ElementAccessExpression(
		value,
		nil,
		index,
		tsgo.NodeFlagsNone,
	)
}

func definedElement(
	factory tsgo.Factory,
	panicName string,
	value tsgo.Expression,
	index tsgo.Expression,
	targetType tsgo.TypeNode,
) tsgo.AsExpression {
	return indexedstorage.Element(
		factory,
		panicName,
		value,
		index,
		targetType,
	)
}

func call(
	factory tsgo.Factory,
	callee tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	arguments ...tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		callee,
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func binary(
	factory tsgo.Factory,
	left tsgo.Expression,
	operator tsgo.BinaryOperator,
	right tsgo.Expression,
) tsgo.BinaryExpression {
	return factory.BinaryExpression(
		nil,
		left,
		nil,
		factory.BinaryOperatorToken(operator),
		right,
	)
}

func boundsPanic(
	factory tsgo.Factory,
	panicName string,
	message string,
) tsgo.ExpressionStatement {
	return factory.ExpressionStatement(panicruntime.Call(
		factory,
		panicName,
		factory.StringLiteral(message, tsgo.TokenFlagsNone),
	))
}
