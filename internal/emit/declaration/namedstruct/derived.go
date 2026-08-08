package namedstruct

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	typefacet "github.com/tsoniclang/gotots/internal/emit/declaration/typefacet"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const derivedStorageMember = "$storage"

func emitDerivedClass(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSpec,
	sourceType types.Type,
	basis types.Type,
	className string,
	fields []field,
	operations []operationAssembly,
	moduleExport bool,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	representationFacets []api.TypeRepresentationFacet,
) (api.DeclarationEmission, error) {
	basisStorage, err := context.Values().StorageType(
		context.WithRole(api.RoleStorageType),
		source.Type,
		basis,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	layoutFields, fieldRequests, err := emitLayoutFields(
		context,
		children,
		fields,
		true,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		typeArguments,
	)
	members := []tsgo.ClassElement{
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().DeclareKeyword(),
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier("$goType"),
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			nil,
		),
		derivedConstructor(context, basisStorage.Value()),
	}
	requests := api.CombineRequests(
		basisStorage.Requests(),
		fieldRequests,
	)
	makeMember, makeRequests, err := derivedMakeMethod(
		context,
		source,
		basis,
		className,
		classType,
		layoutFields,
		typeParameters,
		typeArguments,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	members = append(members, makeMember)
	requests = append(requests, makeRequests...)
	if len(typeParameters) == 0 {
		for _, selected := range layoutFields {
			if selected.field.blank {
				continue
			}
			getter, setter, memberRequests, err := storageFieldMembers(
				context,
				selected,
			)
			if err != nil {
				return api.DeclarationEmission{}, err
			}
			members = append(members, getter, setter)
			requests = append(requests, memberRequests...)
		}
	}

	storageDemanded := false
	for _, operation := range operations {
		if operation.operation == api.NamedStructOperationStorage {
			if len(operation.capabilities) != 0 {
				return api.DeclarationEmission{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "derived-struct storage cannot own generic capabilities",
				}
			}
			storageDemanded = true
			continue
		}
		member, memberRequests, err := emitDerivedValueOperation(
			context,
			children,
			source,
			basis,
			fields,
			className,
			classType,
			operation,
			typeParameters,
			typeArguments,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members, member)
		requests = append(requests, memberRequests...)
	}

	var declarations []tsgo.Statement
	if storageDemanded {
		storageName := className + api.StructStorageTypeSuffix
		var modifiers []tsgo.ModifierLike
		if moduleExport {
			modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
		}
		declarations = append(declarations,
			context.Factory().TypeAliasDeclaration(
				modifiers,
				context.Factory().Identifier(storageName),
				typeParameters,
				basisStorage.Value(),
			),
		)
		storageType := context.Factory().TypeReferenceNode(
			context.Factory().Identifier(storageName),
			typeArguments,
		)
		members = append(
			members,
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
		)
	}
	markers := typefacet.Emission{}
	if len(representationFacets) != 0 {
		var storageType tsgo.TypeNode
		if storageDemanded {
			storageType = context.Factory().TypeReferenceNode(
				context.Factory().Identifier(
					className+api.StructStorageTypeSuffix,
				),
				typeArguments,
			)
		}
		markers, err = typefacet.Build(
			context,
			sourceType,
			storageType,
			representationFacets,
			false,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members, markers.Members()...)
		requests = append(requests, markers.Requests()...)
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	declarations = append(declarations,
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(className),
			typeParameters,
			markers.Heritage(),
			members,
		),
	)
	return declarationEmission(
		declarations,
		requests,
		className,
		storageDemanded,
		moduleExport,
	)
}

func derivedConstructor(
	context api.Context,
	storageType tsgo.TypeNode,
) tsgo.ConstructorDeclaration {
	return context.Factory().ConstructorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PrivateKeyword()},
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			nil,
			context.Factory().Identifier(derivedStorageMember),
			nil,
			storageType,
			nil,
		)},
		nil,
		context.Factory().Block(nil, true),
	)
}

func derivedMakeMethod(
	context api.Context,
	source ast.Node,
	basis types.Type,
	className string,
	classType tsgo.TypeNode,
	fields []layoutField,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	values := make([]tsgo.Expression, 0, len(fields))
	for index, selected := range fields {
		name := context.Factory().Identifier(
			"$field" + strconv.Itoa(index),
		)
		parameterType := selected.logicalType
		if len(typeParameters) != 0 {
			parameterType = selected.storageType
		}
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			nil,
			nil,
			name,
			nil,
			parameterType,
			nil,
		))
		values = append(values, name)
	}
	body, requests, err := derivedConstructBody(
		context,
		source,
		basis,
		className,
		fields,
		values,
		len(typeParameters) != 0,
		typeArguments,
	)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PublicKeyword(),
			context.Factory().StaticKeyword(),
		},
		nil,
		context.Factory().Identifier(api.StructMakeMember),
		nil,
		typeParameters,
		parameters,
		classType,
		context.Factory().Block(body, true),
	), requests, nil
}

func derivedConstructBody(
	context api.Context,
	source ast.Node,
	basis types.Type,
	className string,
	fields []layoutField,
	values []tsgo.Expression,
	valuesStored bool,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if completeDerivedStorage(basis, fields) {
		var body []tsgo.Statement
		properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
		var requests []api.RootRequest
		for index, selected := range fields {
			stored := api.DirectExpression(values[index])
			var err error
			if !valuesStored {
				stored, err = context.Values().ToStorage(
					context.WithRole(api.RoleStructAssignField),
					selected.field.source,
					selected.field.object.Type(),
					stored,
				)
				if err != nil {
					return nil, nil, err
				}
			}
			body = append(body, stored.Before()...)
			properties = append(
				properties,
				context.Factory().PropertyAssignment(
					nil,
					context.Factory().Identifier(selected.field.name),
					nil,
					selected.storageType,
					stored.Value(),
				),
			)
			requests = append(requests, stored.Requests()...)
		}
		body = append(body, context.Factory().ReturnStatement(
			context.Factory().NewExpression(
				context.Factory().Identifier(className),
				typeArguments,
				[]tsgo.Expression{context.Factory().ObjectLiteralExpression(
					properties,
					true,
				)},
			),
		))
		return body, requests, nil
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleStructZeroField),
		source,
		basis,
	)
	if err != nil {
		return nil, nil, err
	}
	storage, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		basis,
		zero,
	)
	if err != nil {
		return nil, nil, err
	}
	body, storageValue := captureDerivedValue(context, "$storage", storage)
	requests := append([]api.RootRequest(nil), storage.Requests()...)
	for index, selected := range fields {
		if selected.field.blank {
			continue
		}
		stored := api.DirectExpression(values[index])
		if !valuesStored {
			stored, err = context.Values().ToStorage(
				context.WithRole(api.RoleStructAssignField),
				selected.field.source,
				selected.field.object.Type(),
				stored,
			)
			if err != nil {
				return nil, nil, err
			}
		}
		body = append(body, stored.Before()...)
		body = append(body, context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().PropertyAccessExpression(
					storageValue,
					nil,
					context.Factory().Identifier(selected.field.name),
					tsgo.NodeFlagsNone,
				),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				stored.Value(),
			),
		))
		requests = append(requests, stored.Requests()...)
	}
	body = append(body, context.Factory().ReturnStatement(
		context.Factory().NewExpression(
			context.Factory().Identifier(className),
			typeArguments,
			[]tsgo.Expression{storageValue},
		),
	))
	return body, requests, nil
}

func completeDerivedStorage(basis types.Type, fields []layoutField) bool {
	structure, ok := basis.Underlying().(*types.Struct)
	return ok && structure.NumFields() == len(fields)
}
