package namedstruct

import (
	"go/ast"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func storageMakeMethod(
	context api.Context,
	source ast.Node,
	className string,
	fields []layoutField,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	var requests []api.RootRequest
	for index, selected := range fields {
		name := context.Factory().Identifier(
			"$field" + strconv.Itoa(index),
		)
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			name,
			nil,
			selected.logicalType,
			nil,
		))
		stored, err := context.Values().ToStorage(
			context.WithRole(api.RoleStructAssignField),
			selected.field.source,
			selected.field.object.Type(),
			api.DirectExpression(name),
		)
		if err != nil {
			return nil, nil, err
		}
		if len(stored.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructAssignField),
				api.CategoryDeclaration,
				source,
			)
		}
		properties = append(properties, context.Factory().PropertyAssignment(
			nil,
			context.Factory().Identifier(selected.field.name),
			nil,
			selected.storageType,
			stored.Value(),
		))
		requests = append(requests, stored.Requests()...)
	}
	value := context.Factory().NewExpression(
		context.Factory().Identifier(className),
		nil,
		[]tsgo.Expression{
			context.Factory().ObjectLiteralExpression(properties, true),
		},
	)
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(api.StructMakeMember),
		nil,
		nil,
		parameters,
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(className),
			nil,
		),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(value)},
			true,
		),
	), requests, nil
}

func storageOfMethod(
	context api.Context,
	className string,
	storageType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	source := context.Factory().Identifier("$source")
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(api.StructStorageOfMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			source,
			nil,
			context.Factory().TypeReferenceNode(
				context.Factory().Identifier(className),
				nil,
			),
			nil,
		)},
		storageType,
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(
				context.Factory().PropertyAccessExpression(
					source,
					nil,
					context.Factory().Identifier("$storage"),
					tsgo.NodeFlagsNone,
				),
			)},
			true,
		),
	)
}

func fromStorageMethod(
	context api.Context,
	className string,
	storageType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	source := context.Factory().Identifier("$source")
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(api.StructFromStorageMember),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			source,
			nil,
			storageType,
			nil,
		)},
		context.Factory().TypeReferenceNode(
			context.Factory().Identifier(className),
			nil,
		),
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ReturnStatement(
				context.Factory().NewExpression(
					context.Factory().Identifier(className),
					nil,
					[]tsgo.Expression{source},
				),
			)},
			true,
		),
	)
}

func storageFieldMembers(
	context api.Context,
	source ast.Node,
	selected layoutField,
) (tsgo.GetAccessorDeclaration, tsgo.SetAccessorDeclaration, []api.RootRequest, error) {
	storageValue := context.Factory().PropertyAccessExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().ThisExpression(),
			nil,
			context.Factory().Identifier("$storage"),
			tsgo.NodeFlagsNone,
		),
		nil,
		context.Factory().Identifier(selected.field.name),
		tsgo.NodeFlagsNone,
	)
	restored, err := context.Values().FromStorage(
		context.WithRole(api.RoleStructField),
		selected.field.source,
		selected.field.object.Type(),
		api.DirectExpression(storageValue),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(restored.Before()) != 0 {
		return nil, nil, nil, api.Unsupported(
			context.WithRole(api.RoleStructField),
			api.CategoryDeclaration,
			source,
		)
	}
	value := context.Factory().Identifier("$value")
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleStructAssignField),
		selected.field.source,
		selected.field.object.Type(),
		api.DirectExpression(value),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(stored.Before()) != 0 {
		return nil, nil, nil, api.Unsupported(
			context.WithRole(api.RoleStructAssignField),
			api.CategoryDeclaration,
			source,
		)
	}
	getter := context.Factory().GetAccessorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
		context.Factory().Identifier(selected.field.name),
		nil,
		nil,
		selected.logicalType,
		context.Factory().Block(
			[]tsgo.Statement{
				context.Factory().ReturnStatement(restored.Value()),
			},
			true,
		),
	)
	setter := context.Factory().SetAccessorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
		context.Factory().Identifier(selected.field.name),
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			value,
			nil,
			selected.logicalType,
			nil,
		)},
		nil,
		context.Factory().Block(
			[]tsgo.Statement{context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					storageValue,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					stored.Value(),
				),
			)},
			true,
		),
	)
	return getter, setter, api.CombineRequests(
		restored.Requests(),
		stored.Requests(),
	), nil
}
