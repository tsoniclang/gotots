package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type field struct {
	source     *ast.Ident
	typeSource ast.Expr
	object     *types.Var
	name       string
}

func emitClass(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (api.DeclarationEmission, error) {
	sourceStruct, structType, ok := sourceType(
		context,
		declaration,
		typeName,
	)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, declaration)
	}
	fields, err := fields(context, sourceStruct, structType)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	className, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}

	members := make([]tsgo.ClassElement, 0, 2)
	members = append(members, context.Factory().PropertyDeclaration(
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
	))

	constructor, requests, err := constructor(
		context,
		children,
		fields,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	members = append(members, constructor)

	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	return api.DirectDeclaration(
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(className),
			nil,
			nil,
			members,
		),
		requests...,
	), nil
}

func emitCompanion(
	context api.Context,
	children api.ChildEmitter,
	declaration ast.Decl,
	typeName *types.TypeName,
	operation api.CompanionOperation,
) (api.DeclarationEmission, error) {
	general, ok := declaration.(*ast.GenDecl)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, declaration)
	}
	sourceStruct, structType, ok := sourceType(context, general, typeName)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, declaration)
	}
	fields, err := fields(context, sourceStruct, structType)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	className, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	companionName, err := api.CompanionExportName(className, operation)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		nil,
	)

	var target tsgo.FunctionDeclaration
	var requests []api.PlacementRequest
	switch operation {
	case api.CompanionZero:
		target, requests, err = zeroFunction(
			context,
			sourceStruct,
			companionName,
			className,
			classType,
			fields,
		)
	case api.CompanionCopy:
		target, requests, err = copyFunction(
			context,
			sourceStruct,
			companionName,
			className,
			classType,
			fields,
		)
	case api.CompanionEqual:
		target, requests, err = equalFunction(
			context,
			children,
			sourceStruct,
			companionName,
			classType,
			fields,
		)
	default:
		return api.DeclarationEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "companion operation is invalid",
		}
	}
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return api.DirectDeclaration(target, requests...), nil
}

func sourceType(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (*ast.StructType, *types.Struct, bool) {
	if declaration == nil || typeName == nil {
		return nil, nil, false
	}
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.TypeSpec)
		if !ok || context.TypesInfo().Defs[spec.Name] != typeName {
			continue
		}
		sourceStruct, syntaxOK := spec.Type.(*ast.StructType)
		named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
		if !syntaxOK || !namedOK ||
			spec.Assign.IsValid() ||
			spec.TypeParams != nil ||
			named.TypeParams().Len() != 0 {
			return nil, nil, false
		}
		structType, structOK := named.Underlying().(*types.Struct)
		return sourceStruct, structType, structOK
	}
	return nil, nil, false
}

func fields(
	context api.Context,
	source *ast.StructType,
	structType *types.Struct,
) ([]field, error) {
	if source == nil || source.Incomplete || source.Fields == nil || structType == nil {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
	result := make([]field, 0, structType.NumFields())
	fieldIndex := 0
	for _, sourceField := range source.Fields.List {
		if sourceField.Tag != nil ||
			len(sourceField.Names) == 0 {
			return nil, api.Unsupported(
				context.WithRole(api.RoleStructField),
				api.CategoryDeclaration,
				sourceField,
			)
		}
		sourceType := context.TypesInfo().TypeOf(sourceField.Type)
		if !supportedFieldType(context, sourceType) {
			return nil, api.Unsupported(
				context.WithRole(api.RoleStructFieldType),
				api.CategoryType,
				sourceField.Type,
			)
		}
		for _, sourceName := range sourceField.Names {
			if fieldIndex >= structType.NumFields() {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceName,
				)
			}
			object := structType.Field(fieldIndex)
			if object.Embedded() ||
				object.Name() == "_" ||
				structType.Tag(fieldIndex) != "" ||
				context.TypesInfo().Defs[sourceName] != object ||
				sourceType == nil ||
				!types.Identical(sourceType, object.Type()) {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceName,
				)
			}
			name, err := context.Names().Member(object)
			if err != nil {
				return nil, err
			}
			result = append(result, field{
				source:     sourceName,
				typeSource: sourceField.Type,
				object:     object,
				name:       name,
			})
			fieldIndex++
		}
	}
	if fieldIndex != structType.NumFields() {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
	return result, nil
}

func supportedFieldType(context api.Context, sourceType types.Type) bool {
	if alias, ok := api.PrimitiveAliasFor(
		context.TypesSizes(),
		sourceType,
	); ok {
		return alias != api.PrimitiveInvalid
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 {
		return false
	}
	_, ok = named.Underlying().(*types.Struct)
	return ok
}

func constructor(
	context api.Context,
	children api.ChildEmitter,
	fields []field,
) (tsgo.ConstructorDeclaration, []api.PlacementRequest, error) {
	parameters := make([]tsgo.ParameterDeclaration, 0, len(fields))
	var requests []api.PlacementRequest
	for _, field := range fields {
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleStructFieldType),
			field.typeSource,
			field.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		parameters = append(parameters, context.Factory().ParameterDeclaration(
			[]tsgo.ModifierLike{context.Factory().PublicKeyword()},
			nil,
			context.Factory().Identifier(field.name),
			nil,
			targetType.Value(),
			nil,
		))
		requests = append(requests, targetType.Requests()...)
	}
	return context.Factory().ConstructorDeclaration(
		nil,
		nil,
		parameters,
		nil,
		context.Factory().Block(nil, true),
	), requests, nil
}

func zeroFunction(
	context api.Context,
	source ast.Node,
	companionName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.FunctionDeclaration, []api.PlacementRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(fields))
	var requests []api.PlacementRequest
	for _, field := range fields {
		value, err := context.Values().Zero(
			context.WithRole(api.RoleStructZeroField),
			field.source,
			field.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		if len(value.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructZeroField),
				api.CategoryDeclaration,
				source,
			)
		}
		arguments = append(arguments, value.Value())
		requests = append(requests, value.Requests()...)
	}
	return companionFunction(
		context,
		companionName,
		nil,
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			context.Factory().NewExpression(
				context.Factory().Identifier(className),
				nil,
				arguments,
			),
		)},
	), requests, nil
}

func copyFunction(
	context api.Context,
	source ast.Node,
	companionName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.FunctionDeclaration, []api.PlacementRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(fields))
	var requests []api.PlacementRequest
	for _, field := range fields {
		value := api.DirectExpression(property(context, "$source", field.name))
		copied, err := context.Values().Copy(
			context.WithRole(api.RoleStructCopyField),
			field.source,
			field.object.Type(),
			value,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(copied.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructCopyField),
				api.CategoryDeclaration,
				source,
			)
		}
		arguments = append(arguments, copied.Value())
		requests = append(requests, copied.Requests()...)
	}
	return companionFunction(
		context,
		companionName,
		[]tsgo.ParameterDeclaration{parameter(context, "$source", classType)},
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			context.Factory().NewExpression(
				context.Factory().Identifier(className),
				nil,
				arguments,
			),
		)},
	), requests, nil
}

func equalFunction(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	companionName string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.FunctionDeclaration, []api.PlacementRequest, error) {
	var expression tsgo.Expression = context.Factory().TrueLiteral()
	var requests []api.PlacementRequest
	for index, field := range fields {
		equal, err := context.Values().Equal(
			context.WithRole(api.RoleStructEqualField),
			field.source,
			field.object.Type(),
			property(context, "$left", field.name),
			property(context, "$right", field.name),
		)
		if err != nil {
			return nil, nil, err
		}
		if index == 0 {
			expression = equal.Value()
		} else {
			expression = context.Factory().BinaryExpression(
				nil,
				expression,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandAmpersandToken,
				),
				equal.Value(),
			)
		}
		requests = append(requests, equal.Requests()...)
	}
	resultType, err := children.RepresentedType(
		context.WithRole(api.RoleResultType),
		source,
		types.Typ[types.Bool],
	)
	if err != nil {
		return nil, nil, err
	}
	requests = append(requests, resultType.Requests()...)
	return companionFunction(
		context,
		companionName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$left", classType),
			parameter(context, "$right", classType),
		},
		resultType.Value(),
		[]tsgo.Statement{context.Factory().ReturnStatement(expression)},
	), requests, nil
}

func companionFunction(
	context api.Context,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements []tsgo.Statement,
) tsgo.FunctionDeclaration {
	return context.Factory().FunctionDeclaration(
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		nil,
		context.Factory().Identifier(name),
		nil,
		parameters,
		result,
		context.Factory().Block(statements, true),
	)
}

func parameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func property(
	context api.Context,
	receiver string,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(receiver),
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}
