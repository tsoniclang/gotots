package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	definedmodel "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func typeDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
) (api.DeclarationEmission, error) {
	if typeName.IsAlias() {
		return aliasDeclaration(context, children, typeName)
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment named type has no named identity",
		}
	}
	switch underlying := named.Underlying().(type) {
	case *types.Interface:
		return interfaceDeclaration(
			context,
			children,
			typeName,
			underlying.Complete(),
		)
	case *types.Struct:
		return structDeclaration(
			context,
			children,
			typeName,
			underlying,
		)
	default:
		return definedDeclaration(
			context,
			children,
			typeName,
			underlying,
		)
	}
}

func aliasDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
) (api.DeclarationEmission, error) {
	generic, err := enterGeneric(context, typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	target, err := children.RepresentedType(
		generic.context.WithRole(api.RoleDefinedUnderlyingType),
		nil,
		types.Unalias(typeName.Type()),
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(
		context.Factory().TypeAliasDeclaration(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			context.Factory().Identifier(name),
			generic.parameters,
			target.Value(),
		),
		target.Requests()...,
	), nil
}

func interfaceDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
	source *types.Interface,
) (api.DeclarationEmission, error) {
	generic, err := enterGeneric(context, typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = generic.context
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	requests := runtimeValue.Requests()
	members := make([]tsgo.TypeElement, 0, source.NumMethods())
	for index := range source.NumMethods() {
		method := source.Method(index)
		signature, ok := method.Type().(*types.Signature)
		if !ok {
			return api.DeclarationEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "environment interface method has no signature",
			}
		}
		target, err := callable.EmitEnvironmentContract(
			context,
			children,
			signature,
		)
		if err != nil {
			return api.DeclarationEmission{}, &MemberError{
				Owner:  typeName,
				Member: method,
				Cause:  err,
			}
		}
		memberName, err := context.Names().InterfaceMethodName(method)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members,
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(memberName),
				nil,
				nil,
				target.Parameters(),
				target.Result(),
			),
		)
		requests = append(requests, target.Requests()...)
	}
	interfaceType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(name),
		generic.arguments,
	)
	valueType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(runtimeValue.Name()),
		nil,
	)
	statements := []tsgo.Statement{
		context.Factory().InterfaceDeclaration(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			context.Factory().Identifier(name),
			generic.parameters,
			[]tsgo.HeritageClause{
				context.Factory().HeritageClause(
					tsgo.HeritageClauseTokenKindExtendsKeyword,
					[]tsgo.ExpressionWithTypeArguments{
						context.Factory().ExpressionWithTypeArguments(
							context.Factory().Identifier(runtimeValue.Name()),
							nil,
						),
					},
				),
			},
			members,
		),
		ambientConstant(
			context,
			name+"$contract",
			context.Factory().TypeOperatorNode(
				tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
				context.Factory().ArrayTypeNode(
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindObjectKeyword,
					),
				),
			),
		),
		context.Factory().FunctionDeclaration(
			exportDeclare(context),
			nil,
			context.Factory().Identifier(name+"$is"),
			generic.parameters,
			[]tsgo.ParameterDeclaration{parameter(
				context,
				"$value",
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					valueType,
					context.Factory().KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
					),
				}),
			)},
			context.Factory().TypePredicateNode(
				nil,
				context.Factory().Identifier("$value"),
				interfaceType,
			),
			nil,
		),
	}
	return api.NewDeclarationEmission(statements, requests)
}

func structDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
	source *types.Struct,
) (api.DeclarationEmission, error) {
	generic, err := enterGeneric(context, typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = generic.context
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(name),
		generic.arguments,
	)
	fields, parameters, requests, err := structFields(
		context,
		children,
		source,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	storageName := name + api.StructStorageTypeSuffix
	storageType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(storageName),
		generic.arguments,
	)
	members := make([]tsgo.ClassElement, 0, len(fields)+10)
	members = append(members, context.Factory().PropertyDeclaration(
		[]tsgo.ModifierLike{
			context.Factory().PrivateKeyword(),
			context.Factory().ReadonlyKeyword(),
		},
		context.Factory().Identifier("$goType"),
		nil,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		),
		nil,
	))
	for _, field := range fields {
		members = append(members, context.Factory().PropertyDeclaration(
			nil,
			field.Name(),
			nil,
			field.Type(),
			nil,
		))
	}
	members = append(members, context.Factory().ConstructorDeclaration(
		[]tsgo.ModifierLike{context.Factory().PrivateKeyword()},
		nil,
		parameters,
		nil,
		nil,
	))
	members = append(
		members,
		structStaticMembers(
			context,
			generic.parameters,
			parameters,
			classType,
			storageType,
			fields,
		)...,
	)
	statements := []tsgo.Statement{
		context.Factory().TypeAliasDeclaration(
			[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
			context.Factory().Identifier(storageName),
			generic.parameters,
			context.Factory().TypeLiteralNode(typeElements(fields)),
		),
		context.Factory().ClassDeclaration(
			exportDeclare(context),
			context.Factory().Identifier(name),
			generic.parameters,
			nil,
			members,
		),
	}
	return api.NewDeclarationEmission(statements, requests)
}

func structFields(
	context api.Context,
	children api.ChildEmitter,
	source *types.Struct,
) (
	[]tsgo.PropertySignatureDeclaration,
	[]tsgo.ParameterDeclaration,
	[]api.RootRequest,
	error,
) {
	fields := make(
		[]tsgo.PropertySignatureDeclaration,
		0,
		source.NumFields(),
	)
	parameters := make(
		[]tsgo.ParameterDeclaration,
		0,
		source.NumFields(),
	)
	var requests []api.RootRequest
	for index := range source.NumFields() {
		field := source.Field(index)
		if !field.Exported() {
			continue
		}
		name := "$blank" + genericName(index)[2:]
		if field.Name() != "_" {
			var err error
			name, err = context.Names().Member(field)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		target, err := children.RepresentedType(
			context.WithRole(api.RoleStructFieldType),
			nil,
			field.Type(),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		fields = append(fields,
			context.Factory().PropertySignatureDeclaration(
				nil,
				context.Factory().Identifier(name),
				nil,
				target.Value(),
				context.Factory().OmittedExpression(),
			),
		)
		parameters = append(parameters, parameter(context, name, target.Value()))
		requests = append(requests, target.Requests()...)
	}
	return fields, parameters, requests, nil
}

func structStaticMembers(
	context api.Context,
	typeParameters []tsgo.TypeParameterDeclaration,
	fields []tsgo.ParameterDeclaration,
	classType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	sourceFields []tsgo.PropertySignatureDeclaration,
) []tsgo.ClassElement {
	static := []tsgo.ModifierLike{
		context.Factory().PublicKeyword(),
		context.Factory().StaticKeyword(),
	}
	method := func(
		name string,
		parameters []tsgo.ParameterDeclaration,
		result tsgo.TypeNode,
	) tsgo.MethodDeclaration {
		return context.Factory().MethodDeclaration(
			static,
			nil,
			context.Factory().Identifier(name),
			nil,
			typeParameters,
			parameters,
			result,
			nil,
		)
	}
	booleanType := context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindBooleanKeyword,
	)
	numberType := context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindNumberKeyword,
	)
	return []tsgo.ClassElement{
		method(api.StructMakeMember, fields, classType),
		method("$zero", nil, classType),
		method("$copy", []tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		}, classType),
		method("$equal", []tsgo.ParameterDeclaration{
			parameter(context, "$left", classType),
			parameter(context, "$right", classType),
		}, booleanType),
		method("$hash", []tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		}, numberType),
		method("$convert", []tsgo.ParameterDeclaration{
			parameter(
				context,
				"$source",
				context.Factory().TypeLiteralNode(typeElements(sourceFields)),
			),
		}, classType),
		method(api.StructStorageOfMember, []tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		}, storageType),
		method(api.StructFromStorageMember, []tsgo.ParameterDeclaration{
			parameter(context, "$source", storageType),
		}, classType),
	}
}

func typeElements(
	fields []tsgo.PropertySignatureDeclaration,
) []tsgo.TypeElement {
	result := make([]tsgo.TypeElement, 0, len(fields))
	for _, field := range fields {
		result = append(result, field)
	}
	return result
}

func definedDeclaration(
	context api.Context,
	children api.ChildEmitter,
	typeName *types.TypeName,
	underlying types.Type,
) (api.DeclarationEmission, error) {
	generic, err := enterGeneric(context, typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = generic.context
	var target api.TypeEmission
	if signature, ok := underlying.(*types.Signature); ok {
		target, err = callable.EmitDefinedNonNilType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			children,
			nil,
			signature,
		)
	} else {
		target, err = children.RepresentedType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			nil,
			underlying,
		)
	}
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	name, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(name),
		generic.arguments,
	)
	optionalClass := context.Factory().UnionTypeNode([]tsgo.TypeNode{
		classType,
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
	members := []tsgo.ClassElement{
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PrivateKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier(definedmodel.BrandMember),
			nil,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			nil,
		),
		context.Factory().PropertyDeclaration(
			[]tsgo.ModifierLike{
				context.Factory().PublicKeyword(),
				context.Factory().ReadonlyKeyword(),
			},
			context.Factory().Identifier(definedmodel.ValueMember),
			nil,
			target.Value(),
			nil,
		),
		context.Factory().ConstructorDeclaration(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				parameter(context, definedmodel.ValueMember, target.Value()),
			},
			nil,
			nil,
		),
	}
	static := []tsgo.ModifierLike{
		context.Factory().PublicKeyword(),
		context.Factory().StaticKeyword(),
	}
	method := func(
		methodName string,
		source tsgo.TypeNode,
		result tsgo.TypeNode,
	) tsgo.MethodDeclaration {
		return context.Factory().MethodDeclaration(
			static,
			nil,
			context.Factory().Identifier(methodName),
			nil,
			generic.parameters,
			[]tsgo.ParameterDeclaration{
				parameter(context, "$source", source),
			},
			result,
			nil,
		)
	}
	members = append(members,
		method(definedmodel.FromMember, target.Value(), optionalClass),
		method(definedmodel.ValueOfMember, optionalClass, target.Value()),
		method(definedmodel.MapReadMember, optionalClass, target.Value()),
		method(definedmodel.MapStoreMember, optionalClass, target.Value()),
		method(definedmodel.MapWrapMember, target.Value(), optionalClass),
	)
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			exportDeclare(context),
			context.Factory().Identifier(name),
			generic.parameters,
			nil,
			members,
		),
		target.Requests()...,
	), nil
}
