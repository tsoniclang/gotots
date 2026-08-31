package namedstruct

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func operationMethod(
	context api.Context,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements []tsgo.Statement,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	parameters = append(
		append([]tsgo.ParameterDeclaration(nil), capabilities...),
		parameters...,
	)
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().StaticKeyword()},
		nil,
		context.Factory().Identifier(name),
		nil,
		typeParameters,
		parameters,
		result,
		context.Factory().Block(statements, true),
	)
}

func construct(
	context api.Context,
	className string,
	typeArguments []tsgo.TypeNode,
	fields []field,
	constructionTypes []tsgo.TypeNode,
	arguments []tsgo.Expression,
	canonicalStorage bool,
) tsgo.NewExpression {
	constructorArguments := arguments
	if canonicalStorage {
		properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
		for index, selected := range fields {
			properties = append(properties, context.Factory().PropertyAssignment(
				nil,
				context.Factory().Identifier(selected.name),
				nil,
				constructionTypes[index],
				arguments[index],
			))
		}
		constructorArguments = []tsgo.Expression{
			context.Factory().ObjectLiteralExpression(properties, true),
		}
	}
	return context.Factory().NewExpression(
		context.Factory().Identifier(className),
		typeArguments,
		constructorArguments,
	)
}

func parameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func property(
	context api.Context,
	receiver string,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(receiver),
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}
