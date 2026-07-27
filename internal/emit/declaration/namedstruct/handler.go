package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
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
	operations []api.NamedStructOperation,
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

	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		nil,
	)
	for _, operation := range operations {
		member, operationRequests, err := emitValueOperation(
			context,
			children,
			sourceStruct,
			className,
			classType,
			fields,
			operation,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		members = append(members, member)
		requests = append(requests, operationRequests...)
	}

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
	if _, ok := arrayvalue.Resolve(context, sourceType); ok {
		return true
	}
	if alias, ok := basictype.PrimitiveAlias(
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
