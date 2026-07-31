package namedstruct

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func assignMethod(
	context api.Context,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
	typeParameters []tsgo.TypeParameterDeclaration,
	canonicalStorage bool,
) tsgo.MethodDeclaration {
	target := context.Factory().Identifier("$target")
	value := context.Factory().Identifier("$value")
	var body []tsgo.Statement
	for _, field := range fields {
		if field.blank {
			continue
		}
		targetField := tsgo.Expression(context.Factory().PropertyAccessExpression(
			target,
			nil,
			context.Factory().Identifier(field.name),
			tsgo.NodeFlagsNone,
		))
		valueField := tsgo.Expression(context.Factory().PropertyAccessExpression(
			value,
			nil,
			context.Factory().Identifier(field.name),
			tsgo.NodeFlagsNone,
		))
		if canonicalStorage {
			targetField = context.Factory().PropertyAccessExpression(
				property(context, "$target", "$storage"),
				nil,
				context.Factory().Identifier(field.name),
				tsgo.NodeFlagsNone,
			)
			valueField = context.Factory().PropertyAccessExpression(
				property(context, "$value", "$storage"),
				nil,
				context.Factory().Identifier(field.name),
				tsgo.NodeFlagsNone,
			)
		}
		body = append(body, assignmentStatement(
			context,
			targetField,
			valueField,
		))
	}
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$target", classType),
			parameter(context, "$value", classType),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
		body,
		nil,
		typeParameters,
	)
}

func assignmentStatement(
	context api.Context,
	target tsgo.Expression,
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	return context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			target,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			value,
		),
	)
}
