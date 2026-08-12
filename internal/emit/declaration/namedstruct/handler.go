package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	typefacet "github.com/tsoniclang/gotots/internal/emit/declaration/typefacet"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type field struct {
	source     ast.Node
	typeSource ast.Node
	object     *types.Var
	name       string
	blank      bool
}

type structSource struct {
	spec      *ast.TypeSpec
	literal   *ast.StructType
	structure *types.Struct
	basis     types.Type
}

func emitClass(
	context api.Context,
	children api.ChildEmitter,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
	operations []operationAssembly,
	requirements []api.DeclarationRequirement,
	representationFacets []api.TypeRepresentationFacet,
) (api.DeclarationEmission, error) {
	source, ok := sourceType(
		context,
		declaration,
		typeName,
	)
	if !ok {
		return api.DeclarationEmission{},
			api.Unsupported(context, api.CategoryDeclaration, declaration)
	}
	parameters, err := genericdeclaration.EnterType(
		context,
		source.spec,
		typeName,
		requirements,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	context = parameters.Context()
	className, err := context.Names().Declare(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	moduleExport, err := context.Names().ModuleExport(typeName)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	if source.basis != nil {
		fields, err := derivedFields(
			context,
			typeName,
			source.spec.Type,
			source.structure,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		return emitStructClass(
			context,
			children,
			source.spec.Type,
			typeName.Type(),
			className,
			fields,
			operations,
			moduleExport,
			parameters.Nodes(),
			parameters.References(),
			representationFacets,
		)
	}
	fields, err := fields(context, source.literal, source.structure)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	return emitStructClass(
		context,
		children,
		source.literal,
		typeName.Type(),
		className,
		fields,
		operations,
		moduleExport,
		parameters.Nodes(),
		parameters.References(),
		representationFacets,
	)
}

func EmitAnonymous(
	context api.Context,
	children api.ChildEmitter,
	structType *types.Struct,
	className string,
	operations []api.NamedStructOperation,
	representationFacets []api.TypeRepresentationFacet,
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
		blank := object.Name() == "_"
		name, err := structconstruction.FieldName(context.Names(), object, index)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
		fields = append(fields, field{
			source:     nil,
			typeSource: nil,
			object:     object,
			name:       name,
			blank:      blank,
		})
	}
	selected := make([]operationAssembly, 0, len(operations))
	for _, operation := range operations {
		selected = append(selected, operationAssembly{operation: operation})
	}
	return emitStructClass(
		context,
		children,
		nil,
		structType,
		className,
		fields,
		selected,
		moduleExport,
		nil,
		nil,
		representationFacets,
	)
}

func emitStructClass(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
	className string,
	fields []field,
	operations []operationAssembly,
	moduleExport bool,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	representationFacets []api.TypeRepresentationFacet,
) (api.DeclarationEmission, error) {
	var storageOperation *operationAssembly
	valueOperations := make([]operationAssembly, 0, len(operations))
	for _, operation := range operations {
		if operation.operation == api.NamedStructOperationStorage {
			selected := operation
			storageOperation = &selected
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
		storageOperation,
		moduleExport,
		typeParameters,
		typeArguments,
	)
	if err != nil {
		return api.DeclarationEmission{}, err
	}
	classType := context.Factory().TypeReferenceNode(
		context.Factory().Identifier(className),
		typeArguments,
	)
	markers := typefacet.Emission{}
	if len(representationFacets) != 0 {
		markers, err = typefacet.Build(
			context,
			sourceType,
			layout.storageType,
			representationFacets,
			false,
		)
		if err != nil {
			return api.DeclarationEmission{}, err
		}
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
	members = append(members, markers.Members()...)
	requests := api.CombineRequests(layout.requests, markers.Requests())

	for _, operation := range valueOperations {
		member, operationRequests, err := emitValueOperation(
			context,
			children,
			source,
			className,
			classType,
			layout.fields,
			operation,
			typeParameters,
			typeArguments,
			storageOperation != nil,
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
			typeParameters,
			markers.Heritage(),
			members,
		),
	)
	return declarationEmission(
		declarations,
		requests,
		className,
		storageOperation != nil,
		moduleExport,
	)
}

func sourceType(
	context api.Context,
	declaration *ast.GenDecl,
	typeName *types.TypeName,
) (structSource, bool) {
	if declaration == nil || typeName == nil {
		return structSource{}, false
	}
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.TypeSpec)
		if !ok || context.TypesInfo().DefOf(spec.Name) != typeName {
			continue
		}
		named, namedOK := types.Unalias(typeName.Type()).(*types.Named)
		if !namedOK ||
			spec.Assign.IsValid() ||
			(spec.TypeParams == nil) != (named.TypeParams().Len() == 0) {
			return structSource{}, false
		}
		structType, structOK := named.Underlying().(*types.Struct)
		if !structOK {
			return structSource{}, false
		}
		if sourceStruct, ok := structLiteralSyntax(spec.Type); ok {
			return structSource{
				spec:      spec,
				literal:   sourceStruct,
				structure: structType,
			}, true
		}
		basis := context.TypesInfo().TypeOf(spec.Type)
		if basis == nil ||
			!types.Identical(basis.Underlying(), structType) ||
			types.Identical(basis, typeName.Type()) {
			return structSource{}, false
		}
		return structSource{
			spec:      spec,
			structure: structType,
			basis:     basis,
		}, true
	}
	return structSource{}, false
}

func structLiteralSyntax(source ast.Expr) (*ast.StructType, bool) {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			break
		}
		source = parenthesized.X
	}
	structType, ok := source.(*ast.StructType)
	return structType, ok
}

func derivedFields(
	context api.Context,
	typeName *types.TypeName,
	source ast.Node,
	structType *types.Struct,
) ([]field, error) {
	if typeName == nil || structType == nil {
		return nil, api.Unsupported(
			context,
			api.CategoryDeclaration,
			source,
		)
	}
	result := make([]field, 0, structType.NumFields())
	for index := range structType.NumFields() {
		object := structType.Field(index)
		if context.SourceImplementationContract() && !object.Exported() {
			continue
		}
		if !object.Exported() && object.Pkg() != typeName.Pkg() {
			continue
		}
		blank := object.Name() == "_"
		name, err := structconstruction.FieldName(context.Names(), object, index)
		if err != nil {
			return nil, err
		}
		result = append(result, field{
			source:     source,
			typeSource: source,
			object:     object,
			name:       name,
			blank:      blank,
		})
	}
	return result, nil
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
			if fieldIndex >= structType.NumFields() {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceField,
				)
			}
			objectIndex := fieldIndex
			object := structType.Field(objectIndex)
			fieldIndex++
			if context.SourceImplementationContract() && !object.Exported() {
				continue
			}
			sourceType := context.TypesInfo().TypeOf(sourceField.Type)
			if !object.Embedded() ||
				sourceType == nil ||
				!types.Identical(sourceType, object.Type()) {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceField,
				)
			}
			name, err := context.Names().Member(object)
			if err != nil {
				return nil, err
			}
			result = append(result, field{
				source:     sourceField,
				typeSource: sourceField.Type,
				object:     object,
				name:       name,
			})
			continue
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
			objectIndex := fieldIndex
			object := structType.Field(objectIndex)
			fieldIndex++
			if context.SourceImplementationContract() && !object.Exported() {
				continue
			}
			if object.Embedded() ||
				context.TypesInfo().DefOf(sourceName) != object ||
				sourceType == nil ||
				!types.Identical(sourceType, object.Type()) {
				return nil, api.Unsupported(
					context.WithRole(api.RoleStructField),
					api.CategoryDeclaration,
					sourceName,
				)
			}
			blank := object.Name() == "_"
			name, err := structconstruction.FieldName(
				context.Names(), object, objectIndex,
			)
			if err != nil {
				return nil, err
			}
			result = append(result, field{
				source:     sourceName,
				typeSource: sourceField.Type,
				object:     object,
				name:       name,
				blank:      blank,
			})
		}
	}
	if fieldIndex != structType.NumFields() {
		return nil, api.Unsupported(context, api.CategoryDeclaration, source)
	}
	return result, nil
}
