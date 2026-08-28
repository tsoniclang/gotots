package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func storageZeroMethod(
	context api.Context,
	source ast.Node,
	storageType tsgo.TypeNode,
	fields []layoutField,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(fields))
	var body []tsgo.Statement
	var requests []api.RootRequest
	for _, selected := range fields {
		value, err := context.Values().StorageZero(
			context.WithRole(api.RoleStructZeroField),
			selected.field.source,
			selected.field.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(body, value.Before()...)
		arguments = append(arguments, value.Value())
		requests = append(requests, value.Requests()...)
	}
	body = append(body, context.Factory().ReturnStatement(
		storageObject(context, fields, arguments),
	))
	return operationMethod(
		context,
		api.StructStorageZeroMember,
		nil,
		storageType,
		body,
		capabilities,
		typeParameters,
	), requests, nil
}

func storageObject(
	context api.Context,
	fields []layoutField,
	arguments []tsgo.Expression,
) tsgo.ObjectLiteralExpression {
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	for index, selected := range fields {
		properties = append(properties, context.Factory().PropertyAssignment(
			nil,
			context.Factory().Identifier(selected.field.name),
			nil,
			selected.storageType,
			arguments[index],
		))
	}
	return context.Factory().ObjectLiteralExpression(properties, true)
}
