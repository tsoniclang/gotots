package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const derivedStorageMember = "$storage"

func emitDerivedClass(
	context api.Context,
	children api.ChildEmitter,
	source *ast.TypeSpec,
	basis types.Type,
	className string,
	fields []field,
	operations []operationAssembly,
	moduleExport bool,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
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
		false,
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
		basisStorage.Value(),
		layoutFields,
		typeParameters,
		typeArguments,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	members = append(members, makeMember)
	requests = append(requests, makeRequests...)
	for _, selected := range layoutFields {
		if selected.field.blank {
			continue
		}
		getter, setter, memberRequests, err := derivedFieldMembers(
			context,
			source,
			basis,
			basisStorage.Value(),
			selected,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members, getter, setter)
		requests = append(requests, memberRequests...)
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
			className,
			classType,
			basisStorage.Value(),
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
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	declarations = append(declarations,
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(className),
			typeParameters,
			nil,
			members,
		),
	)
	return api.NewDeclarationEmission(declarations, requests)
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
	storageType tsgo.TypeNode,
	fields []layoutField,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	values := make([]tsgo.Expression, 0, len(fields))
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
		values = append(values, name)
	}
	body, requests, err := derivedConstructBody(
		context,
		source,
		basis,
		className,
		storageType,
		fields,
		values,
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
	_ tsgo.TypeNode,
	fields []layoutField,
	values []tsgo.Expression,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.Statement, []api.RootRequest, error) {
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleStructZeroField),
		source,
		basis,
	)
	if err != nil {
		return nil, nil, err
	}
	body, basisValue := captureDerivedValue(context, "$basis", zero)
	for index, selected := range fields {
		if selected.field.blank {
			continue
		}
		body = append(body, context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().PropertyAccessExpression(
					basisValue,
					nil,
					context.Factory().Identifier(selected.field.name),
					tsgo.NodeFlagsNone,
				),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				values[index],
			),
		))
	}
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		basis,
		api.DirectExpression(basisValue),
	)
	if err != nil {
		return nil, nil, err
	}
	body = append(body, stored.Before()...)
	body = append(body, context.Factory().ReturnStatement(
		context.Factory().NewExpression(
			context.Factory().Identifier(className),
			typeArguments,
			[]tsgo.Expression{stored.Value()},
		),
	))
	return body, api.CombineRequests(
		zero.Requests(),
		stored.Requests(),
	), nil
}

func derivedFieldMembers(
	context api.Context,
	source ast.Node,
	basis types.Type,
	_ tsgo.TypeNode,
	selected layoutField,
) (
	tsgo.GetAccessorDeclaration,
	tsgo.SetAccessorDeclaration,
	[]api.RootRequest,
	error,
) {
	storage := context.Factory().PropertyAccessExpression(
		context.Factory().ThisExpression(),
		nil,
		context.Factory().Identifier(derivedStorageMember),
		tsgo.NodeFlagsNone,
	)
	read, err := context.Values().FromStorage(
		context.WithRole(api.RoleStructField),
		source,
		basis,
		api.DirectExpression(storage),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	readBody, readValue := captureDerivedValue(context, "$basis", read)
	readBody = append(readBody, context.Factory().ReturnStatement(
		context.Factory().PropertyAccessExpression(
			readValue,
			nil,
			context.Factory().Identifier(selected.field.name),
			tsgo.NodeFlagsNone,
		),
	))
	write, err := context.Values().FromStorage(
		context.WithRole(api.RoleStructAssignField),
		source,
		basis,
		api.DirectExpression(storage),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	writeBody, writeValue := captureDerivedValue(context, "$basis", write)
	value := context.Factory().Identifier("$value")
	writeBody = append(writeBody, context.Factory().ExpressionStatement(
		context.Factory().BinaryExpression(
			nil,
			context.Factory().PropertyAccessExpression(
				writeValue,
				nil,
				context.Factory().Identifier(selected.field.name),
				tsgo.NodeFlagsNone,
			),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsToken,
			),
			value,
		),
	))
	return context.Factory().GetAccessorDeclaration(
			[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
			context.Factory().Identifier(selected.field.name),
			nil,
			nil,
			selected.logicalType,
			context.Factory().Block(readBody, true),
		),
		context.Factory().SetAccessorDeclaration(
			[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
			context.Factory().Identifier(selected.field.name),
			nil,
			[]tsgo.ParameterDeclaration{parameter(
				context,
				"$value",
				selected.logicalType,
			)},
			nil,
			context.Factory().Block(writeBody, true),
		),
		api.CombineRequests(read.Requests(), write.Requests()),
		nil
}
