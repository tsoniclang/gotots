package namedstruct

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func assignMethod(
	context api.Context,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
	capabilities []tsgo.ParameterDeclaration,
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
		capabilities,
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
		_, generic := api.GenericTypeParameter(field.object.Type())
		_, nestedArray := field.object.Type().Underlying().(*types.Array)
		_, structure := field.object.Type().Underlying().(*types.Struct)
		if generic || nestedArray || structure {
			left := api.DirectExpression(targetField)
			right := api.DirectExpression(valueField)
			var err error
			if canonicalStorage {
				left, err = context.Values().FromStorage(context, field.source, field.object.Type(), left)
				if err != nil {
					return nil, nil, err
				}
				right, err = context.Values().FromStorage(context, field.source, field.object.Type(), right)
				if err != nil {
					return nil, nil, err
				}
			}
			assigned, err := context.StableAssignments().AssignStable(context, field.source, field.object.Type(), left.Value(), right)
			if err != nil {
				return nil, nil, err
			}
			if generic && canonicalStorage {
				assigned, err = context.Values().ToStorage(context, field.source, field.object.Type(), assigned)
				if err != nil {
					return nil, nil, err
				}
			}
			body = append(body, left.Before()...)
			body = append(body, assigned.Before()...)
			if generic {
				body = append(body, assignmentStatement(context, targetField, assigned.Value()))
			} else {
				body = append(body, context.Factory().ExpressionStatement(assigned.Value()))
			}
			requests = api.CombineRequests(requests, left.Requests(), assigned.Requests())
			continue
		}
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
