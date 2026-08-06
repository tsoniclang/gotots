package namedstruct

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
		if !namedStructField(field.object.Type()) {
			body = append(body, assignmentStatement(
				context,
				targetField,
				valueField,
			))
			continue
		}
		pointerRepresentation, err := pointertype.Observe(
			context,
			types.NewPointer(field.object.Type()),
			api.PointerRepresentationDemandNone,
		)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, pointerRepresentation.Requests()...)
		if !pointerRepresentation.UsesStorageIdentity() {
			body = append(body, assignmentStatement(
				context,
				targetField,
				valueField,
			))
			continue
		}
		targetValue := api.DirectExpression(targetField)
		assignedValue := api.DirectExpression(valueField)
		if canonicalStorage {
			targetValue, err = context.Values().FromStorage(
				context.WithRole(api.RoleAssignmentTarget),
				field.source,
				field.object.Type(),
				targetValue,
			)
			if err == nil {
				assignedValue, err = context.Values().FromStorage(
					context.WithRole(api.RoleAssignmentValue),
					field.source,
					field.object.Type(),
					assignedValue,
				)
			}
		}
		if err != nil {
			return nil, nil, err
		}
		stableAssignments := context.StableAssignments()
		if stableAssignments == nil {
			return nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "named-struct assignment has no stable-assignment service",
			}
		}
		assigned, err := stableAssignments.AssignStable(
			context.WithRole(api.RoleAssignmentTarget),
			field.source,
			field.object.Type(),
			targetValue.Value(),
			assignedValue,
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(body, targetValue.Before()...)
		body = append(body, assigned.Before()...)
		body = append(body, context.Factory().ExpressionStatement(
			assigned.Value(),
		))
		requests = append(requests, targetValue.Requests()...)
		requests = append(requests, assigned.Requests()...)
	}
	return body, api.CombineRequests(requests), nil
}

func namedStructField(sourceType types.Type) bool {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		return false
	}
	_, ok = named.Underlying().(*types.Struct)
	return ok
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
