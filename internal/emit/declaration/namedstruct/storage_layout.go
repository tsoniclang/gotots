package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type layoutEmission struct {
	declarations []tsgo.Statement
	members      []tsgo.ClassElement
	requests     []api.RootRequest
}

type layoutField struct {
	field       field
	logicalType tsgo.TypeNode
	storageType tsgo.TypeNode
}

func emitLayout(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	className string,
	fields []field,
	storage bool,
	moduleExport bool,
) (layoutEmission, error) {
	selected, requests, err := emitLayoutFields(
		context,
		children,
		fields,
		storage,
	)
	if err != nil {
		return layoutEmission{}, err
	}
	if !storage {
		members := []tsgo.ClassElement{
			directConstructor(context, selected),
			makeMethod(context, className, selected, false),
		}
		return layoutEmission{members: members, requests: requests}, nil
	}
	storageName := className + "$Storage"
	storageType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(storageName),
		nil,
	)
	members, memberRequests, err := storageMembers(
		context,
		source,
		className,
		storageType,
		selected,
	)
	if err != nil {
		return layoutEmission{}, err
	}
	return layoutEmission{
		declarations: []tsgo.Statement{storageAlias(
			context,
			storageName,
			selected,
			moduleExport,
		)},
		members:  members,
		requests: api.CombineRequests(requests, memberRequests),
	}, nil
}

func emitLayoutFields(
	context api.Context,
	children api.ChildEmitter,
	fields []field,
	storage bool,
) ([]layoutField, []api.RootRequest, error) {
	result := make([]layoutField, 0, len(fields))
	var requests []api.RootRequest
	for _, sourceField := range fields {
		logical, err := children.RepresentedType(
			context.WithRole(api.RoleStructFieldType),
			sourceField.typeSource,
			sourceField.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		storageType := logical
		if storage {
			storageType, err = context.Values().StorageType(
				context.WithRole(api.RoleStorageType),
				sourceField.typeSource,
				sourceField.object.Type(),
			)
			if err != nil {
				return nil, nil, err
			}
		}
		result = append(result, layoutField{
			field:       sourceField,
			logicalType: logical.Value(),
			storageType: storageType.Value(),
		})
		requests = append(
			requests,
			api.CombineRequests(
				logical.Requests(),
				storageType.Requests(),
			)...,
		)
	}
	return result, requests, nil
}

func directConstructor(
	context api.Context,
	fields []layoutField,
) tsgo.ConstructorDeclaration {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	for _, selected := range fields {
		var modifiers []tsgo.ModifierLike
		if !selected.field.blank {
			modifiers = []tsgo.ModifierLike{context.Factory().PublicKeyword()}
		}
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			modifiers,
			nil,
			context.Factory().Identifier(selected.field.name),
			nil,
			selected.logicalType,
			nil,
		))
	}
	return context.Factory().ConstructorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PrivateKeyword()},
		nil,
		parameters,
		nil,
		context.Factory().Block(nil, true),
	)
}

func makeMethod(
	context api.Context,
	className string,
	fields []layoutField,
	storage bool,
) tsgo.MethodDeclaration {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	arguments := make([]tsgo.Expression, 0, len(fields))
	for _, selected := range fields {
		name := context.Factory().Identifier(selected.field.name)
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			name,
			nil,
			selected.logicalType,
			nil,
		))
		arguments = append(arguments, name)
	}
	var value tsgo.Expression = context.Factory().NewExpression(
		context.Factory().Identifier(className),
		nil,
		arguments,
	)
	if storage {
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
		value = context.Factory().NewExpression(
			context.Factory().Identifier(className),
			nil,
			[]tsgo.Expression{
				context.Factory().ObjectLiteralExpression(properties, true),
			},
		)
	}
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
	)
}

func storageAlias(
	context api.Context,
	name string,
	fields []layoutField,
	moduleExport bool,
) tsgo.TypeAliasDeclaration {
	members := make([]tsgo.TypeElement, 0, len(fields))
	for _, selected := range fields {
		members = append(members,
			context.Factory().PropertySignatureDeclaration(
				nil,
				context.Factory().Identifier(selected.field.name),
				nil,
				selected.storageType,
				context.Factory().OmittedExpression(),
			),
		)
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	return context.Factory().TypeAliasDeclaration(
		modifiers,
		context.Factory().Identifier(name),
		nil,
		context.Factory().TypeLiteralNode(members),
	)
}

func storageMembers(
	context api.Context,
	source ast.Node,
	className string,
	storageType tsgo.TypeNode,
	fields []layoutField,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	constructor := context.Factory().ConstructorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			nil,
			context.Factory().Identifier("$storage"),
			nil,
			storageType,
			nil,
		)},
		nil,
		context.Factory().Block(nil, true),
	)
	makeMember, makeRequests, err := storageMakeMethod(
		context,
		source,
		className,
		fields,
	)
	if err != nil {
		return nil, nil, err
	}
	members := []tsgo.ClassElement{
		constructor,
		makeMember,
		storageOfMethod(context, className, storageType),
		fromStorageMethod(context, className, storageType),
	}
	requests := makeRequests
	for _, selected := range fields {
		if selected.field.blank {
			continue
		}
		getter, setter, fieldRequests, err := storageFieldMembers(
			context,
			source,
			selected,
		)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, getter, setter)
		requests = append(requests, fieldRequests...)
	}
	return members, requests, nil
}
