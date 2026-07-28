package namedstruct

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type field struct {
	source     ast.Node
	typeSource ast.Node
	object     *types.Var
	name       string
	blank      bool
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

	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return emitStructClass(
		context,
		children,
		sourceStruct,
		className,
		fields,
		operations,
		moduleExport,
	)
}

func EmitAnonymous(
	context api.Context,
	children api.ChildEmitter,
	structType *types.Struct,
	className string,
	operations []api.NamedStructOperation,
	moduleExport bool,
) (api.DeclarationEmission, error) {
	if structType == nil || className == "" {
		return api.DeclarationEmission{}, &api.GeneratedArtifactShapeError{
			Artifact: className,
			Reason:   "anonymous struct declaration input is invalid",
		}
	}
	fields := make([]field, 0, structType.NumFields())
	for index := range structType.NumFields() {
		object := structType.Field(index)
		if object.Embedded() {
			return api.DeclarationEmission{}, &api.GeneratedArtifactShapeError{
				Artifact: className,
				Reason:   "embedded fields belong to the embedding lane",
			}
		}
		blank := object.Name() == "_"
		name := fmt.Sprintf("$blank%d", index)
		if !blank {
			var err error
			name, err = context.Names().Member(object)
			if err != nil {
				return api.DeclarationEmission{}, err
			}
		}
		fields = append(fields, field{
			source:     nil,
			typeSource: nil,
			object:     object,
			name:       name,
			blank:      blank,
		})
	}
	return emitStructClass(
		context,
		children,
		nil,
		className,
		fields,
		operations,
		moduleExport,
	)
}

func emitStructClass(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	className string,
	fields []field,
	operations []api.NamedStructOperation,
	moduleExport bool,
) (api.DeclarationEmission, error) {
	storageLayout := false
	valueOperations := make([]api.NamedStructOperation, 0, len(operations))
	for _, operation := range operations {
		if operation == api.NamedStructOperationStorage {
			storageLayout = true
			continue
		}
		valueOperations = append(valueOperations, operation)
	}
	layout, err := emitLayout(
		context,
		children,
		source,
		className,
		fields,
		storageLayout,
		moduleExport,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	members := make([]tsgo.ClassElement, 0, 3+len(valueOperations))
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
	members = append(members, layout.members...)
	requests := layout.requests

	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		nil,
	)
	for _, operation := range valueOperations {
		member, operationRequests, err := emitValueOperation(
			context,
			children,
			source,
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

	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{context.Factory().ExportKeyword()}
	}
	declarations := append(
		layout.declarations,
		context.Factory().ClassDeclaration(
			modifiers,
			context.Factory().Identifier(className),
			nil,
			nil,
			members,
		),
	)
	return api.NewDeclarationEmission(declarations, requests)
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
		if len(sourceField.Names) == 0 {
			return nil, api.Unsupported(
				context.WithRole(api.RoleStructField),
				api.CategoryDeclaration,
				sourceField,
			)
		}
		sourceType := context.TypesInfo().TypeOf(sourceField.Type)
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
				context.TypesInfo().Defs[sourceName] != object ||
				sourceType == nil ||
				!types.Identical(sourceType, object.Type()) {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceName,
				)
			}
			blank := object.Name() == "_"
			name := fmt.Sprintf("$blank%d", fieldIndex)
			if !blank {
				var err error
				name, err = context.Names().Member(object)
				if err != nil {
					return nil, err
				}
			}
			result = append(result, field{
				source:     sourceName,
				typeSource: sourceField.Type,
				object:     object,
				name:       name,
				blank:      blank,
			})
			fieldIndex++
		}
	}
	if fieldIndex != structType.NumFields() {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
	return result, nil
}
