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
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	var target tsgo.Expression = context.Factory().Identifier("$target")
	var value tsgo.Expression = context.Factory().Identifier("$value")
	if canonicalStorage {
		target = context.Factory().PropertyAccessExpression(
			target,
			nil,
			context.Factory().Identifier("$storage"),
			tsgo.NodeFlagsNone,
		)
		value = context.Factory().PropertyAccessExpression(
			value,
			nil,
			context.Factory().Identifier("$storage"),
			tsgo.NodeFlagsNone,
		)
	}
	body, requests, err := assignFields(
		context,
		fields,
		target,
		value,
		canonicalStorage,
	)
	if err != nil {
		return nil, nil, err
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
	), requests, nil
}

func assignFields(
	context api.Context,
	fields []field,
	target tsgo.Expression,
	value tsgo.Expression,
	canonicalStorage bool,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var body []tsgo.Statement
	var requests []api.RootRequest
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
		body = append(body, assignmentStatement(
			context,
			targetField,
			valueField,
		))
	}
	return body, api.CombineRequests(requests), nil
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
