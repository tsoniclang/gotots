package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type layoutEmission struct {
	declarations []tsgo.Statement
	members      []tsgo.ClassElement
	fields       []layoutField
	storageType  tsgo.TypeNode
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
	storageOperation *operationAssembly,
	moduleExport bool,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) (layoutEmission, error) {
	storage := storageOperation != nil
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
		members := []tsgo.ClassElement{directConstructor(context, selected)}
		return layoutEmission{
			members:  members,
			fields:   selected,
			requests: requests,
		}, nil
	}
	if len(storageOperation.capabilities) != 0 {
		return layoutEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "storage projection cannot own primitive generic capabilities",
		}
	}
	storageName := className + "$Storage"
	storageType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(storageName),
		typeArguments,
	)
	members, memberRequests, err := storageMembers(
		context,
		source,
		className,
		storageType,
		selected,
		typeParameters,
		typeArguments,
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
			typeParameters,
		)},
		members:     members,
		fields:      selected,
		storageType: storageType,
		requests:    api.CombineRequests(requests, memberRequests),
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
		fieldContext := context.WithRole(api.RoleStructFieldType)
		logical, err := children.RepresentedType(
			fieldContext,
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
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
		nil,
		parameters,
		nil,
		context.Factory().Block(nil, true),
	)
}

func storageAlias(
	context api.Context,
	name string,
	fields []layoutField,
	moduleExport bool,
	typeParameters []tsgo.TypeParameterDeclaration,
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
		typeParameters,
		context.Factory().TypeLiteralNode(members),
	)
}

func storageMembers(
	context api.Context,
	source ast.Node,
	className string,
	storageType tsgo.TypeNode,
	fields []layoutField,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.ClassElement, []api.RootRequest, error) {
	constructor := context.Factory().ConstructorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
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
	members := []tsgo.ClassElement{
		constructor,
		storageOfMethod(
			context,
			className,
			storageType,
			typeParameters,
			typeArguments,
		),
		fromStorageMethod(
			context,
			className,
			storageType,
			typeParameters,
			typeArguments,
		),
	}
	var requests []api.RootRequest
	if len(typeParameters) != 0 {
		return members, requests, nil
	}
	for _, selected := range fields {
		if selected.field.blank {
			continue
		}
		getter, setter, fieldRequests, err := storageFieldMembers(
			context,
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
