package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/mapliteral"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type element struct {
	fieldIndex       int
	declarationField *types.Var
	selectedField    *types.Var
	source           ast.Expr
	value            api.ExpressionEmission
}

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
) (api.ExpressionEmission, error) {
	sourceType := literalType(context, source)
	if sourceType != nil {
		if _, ok := types.Unalias(sourceType).Underlying().(*types.Map); ok {
			return mapliteral.Emit(context, children, source, sourceType)
		}
	}
	if array, ok := arrayvalue.Resolve(
		context,
		sourceType,
	); ok {
		return array.EmitLiteral(context, children, source, sourceType)
	}
	if target, handled, err := emitSlice(
		context,
		children,
		source,
		sourceType,
	); handled || err != nil {
		return target, err
	}
	named, structType, ok := structSourceType(context, source, sourceType)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elements, err := emitElements(
		context,
		children,
		source,
		named,
		structType,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if named != nil && hasInaccessibleFields(context, structType) {
		return emitRestrictedNamedStruct(
			context,
			source,
			named,
			structType,
			elements,
		)
	}
	canonicalStorage := false
	form := constructionFormDirectPositional
	var constructorReference api.NameReference
	var anonymousReference api.NameReference
	if named != nil {
		constructorReference, err = context.Names().NamedStructConstructor(
			named.Origin().Obj(),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		_, canonicalStorage, err = namedstructstorage.Selected(
			context,
			named,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		form = constructionFormForReference(
			constructorReference,
			canonicalStorage,
		)
	} else {
		anonymousReference, err = context.Names().AnonymousStruct(
			structType,
			api.AnonymousStructDemandDefinition,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if structType.NumFields() != 0 {
			artifact, artifactErr := anonymousStructArtifact(
				anonymousReference,
			)
			if artifactErr != nil {
				return api.ExpressionEmission{}, artifactErr
			}
			canonicalStorage, err = context.ResolveAnonymousStructDemand(
				artifact,
				api.AnonymousStructDemandStorage,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
	}
	before, requests, fields, err := arrange(
		context,
		children,
		source,
		named,
		structType,
		elements,
		canonicalStorage,
		form,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if named == nil {
		value, typeRequests, err := sourceStructConstruction(
			context,
			source,
			anonymousReference.Expression(context.Factory()),
			nil,
			structType,
			fields,
			canonicalStorage,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			before,
			value,
			api.CombineRequests(
				requests,
				anonymousReference.Requests(),
				typeRequests,
			),
		)
	}
	var reference api.NameReference
	var typeArguments []tsgo.TypeNode
	if canonicalStorage {
		if form == constructionFormProviderFacet {
			reference = constructorReference
		} else {
			reference, err = context.Names().NamedStructOperation(
				named.Origin().Obj(),
				api.NamedStructOperationStorage,
			)
		}
		if err == nil && named.TypeParams().Len() != 0 {
			typeArguments, requests, err =
				genericNamedStructTypeArguments(
					context,
					children,
					source,
					named,
					requests,
				)
		}
	} else {
		reference = constructorReference
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if form == constructionFormProviderFacet {
		values := make([]tsgo.Expression, 0, len(fields))
		for _, selected := range fields {
			values = append(values, selected.value)
		}
		return api.NewExpressionEmission(
			before,
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					reference.Expression(context.Factory()),
					nil,
					context.Factory().Identifier(api.StructMakeMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				typeArguments,
				values,
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(requests, reference.Requests()),
		)
	}
	value, typeRequests, err := sourceStructConstruction(
		context,
		source,
		reference.Expression(context.Factory()),
		typeArguments,
		structType,
		fields,
		canonicalStorage,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		value,
		api.CombineRequests(
			requests,
			constructorReference.Requests(),
			reference.Requests(),
			typeRequests,
		),
	)
}

func RequiresAddress(
	context api.Context,
	source *ast.CompositeLit,
) bool {
	if source == nil || source.Type != nil {
		return false
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return false
	}
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	expected := context.ExpectedType()
	return ok &&
		pointer.Elem() != nil &&
		expected != nil &&
		types.AssignableTo(sourceType, expected)
}

func literalType(
	context api.Context,
	source *ast.CompositeLit,
) types.Type {
	if source == nil {
		return nil
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if source.Type != nil || sourceType == nil {
		return sourceType
	}
	pointer, ok := types.Unalias(sourceType).(*types.Pointer)
	expected := context.ExpectedType()
	if ok &&
		expected != nil &&
		types.Identical(pointer.Elem(), expected) {
		return pointer.Elem()
	}
	return sourceType
}

func hasInaccessibleFields(
	context api.Context,
	structType *types.Struct,
) bool {
	for index := range structType.NumFields() {
		field := structType.Field(index)
		if !field.Exported() &&
			field.Pkg() != nil &&
			field.Pkg() != context.TypesPackage() {
			return true
		}
	}
	return false
}

func emitRestrictedNamedStruct(
	context api.Context,
	source *ast.CompositeLit,
	named *types.Named,
	structType *types.Struct,
	elements []element,
) (api.ExpressionEmission, error) {
	before := make([]tsgo.Statement, 0, len(elements)*2+2)
	requests := make([]api.RootRequest, 0, len(elements)+1)
	values := make([]tsgo.Expression, 0, len(elements))
	for _, element := range elements {
		before = append(before, element.value.Before()...)
		name, err := context.Names().Temporary(api.TemporaryCompositeField)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		value := context.Factory().Identifier(name)
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						value,
						nil,
						nil,
						element.value.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		))
		values = append(values, value)
		requests = append(requests, element.value.Requests()...)
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleStructZeroField),
		source,
		named,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, zero.Before()...)
	resultName, err := context.Names().Temporary(api.TemporaryStructSource)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := context.Factory().Identifier(resultName)
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					result,
					nil,
					nil,
					zero.Value(),
				),
			},
			tsgo.NodeFlagsConst,
		),
	))
	requests = append(requests, zero.Requests()...)
	for index, element := range elements {
		field := element.selectedField
		if !field.Exported() &&
			field.Pkg() != nil &&
			field.Pkg() != context.TypesPackage() {
			return api.ExpressionEmission{},
				api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					element.source,
				)
		}
		if field.Name() == "_" {
			continue
		}
		target, selected, err := namedstructstorage.FieldTarget(
			context.WithRole(api.RoleStructAssignField),
			element.source,
			named,
			field,
			api.DirectExpression(result),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if selected {
			stored, storeErr := target.StoreValue(
				context.WithRole(api.RoleStructAssignField),
				element.source,
				api.DirectExpression(values[index]),
			)
			if storeErr != nil {
				return api.ExpressionEmission{}, storeErr
			}
			before = append(before, stored.Before()...)
			before = append(
				before,
				context.Factory().ExpressionStatement(stored.Value()),
			)
			requests = append(requests, stored.Requests()...)
			continue
		}
		name, err := context.Names().Member(element.declarationField)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(before, context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().PropertyAccessExpression(
					result,
					nil,
					context.Factory().Identifier(name),
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
	return api.NewExpressionEmission(before, result, requests)
}

func structSourceType(
	context api.Context,
	source *ast.CompositeLit,
	sourceType types.Type,
) (*types.Named, *types.Struct, bool) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok {
		structType, structOK := types.Unalias(sourceType).(*types.Struct)
		expected := context.ExpectedType()
		return nil, structType,
			structOK &&
				!source.Incomplete &&
				expected != nil &&
				types.AssignableTo(sourceType, expected)
	}
	if source.Incomplete ||
		(named.TypeParams().Len() != 0 &&
			named.TypeArgs().Len() != named.TypeParams().Len()) {
		return nil, nil, false
	}
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, nil, false
	}
	expected := context.ExpectedType()
	return named, structType,
		expected != nil && types.AssignableTo(sourceType, expected)
}

func emitElements(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	named *types.Named,
	structType *types.Struct,
) ([]element, error) {
	result := make([]element, 0, len(source.Elts))
	seen := make(map[int]struct{}, len(source.Elts))
	keyedCount := 0
	for sourceIndex, sourceElement := range source.Elts {
		field, fieldOK := context.TypesInfo().StructFieldAt(source, sourceIndex)
		valueSource := sourceElement
		if keyed, ok := sourceElement.(*ast.KeyValueExpr); ok {
			keyedCount++
			identifier, identifierOK := keyed.Key.(*ast.Ident)
			if !identifierOK {
				return nil, api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					sourceElement,
				)
			}
			field, fieldOK = context.TypesInfo().StructFieldOf(source, identifier)
			if !fieldOK {
				return nil, api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					sourceElement,
				)
			}
			valueSource = keyed.Value
		}
		if !fieldOK {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				sourceElement,
			)
		}
		fieldIndex := field.Index()
		if fieldIndex < 0 || fieldIndex >= structType.NumFields() {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				sourceElement,
			)
		}
		if _, duplicate := seen[fieldIndex]; duplicate {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				sourceElement,
			)
		}
		seen[fieldIndex] = struct{}{}
		selectedField := field.Selected()
		fieldType := selectedField.Type()
		valueType := context.TypesInfo().TypeOf(valueSource)
		if valueType == nil || !types.AssignableTo(valueType, fieldType) {
			return nil, api.Unsupported(
				context.WithRole(api.RoleCompositeElement),
				api.CategoryExpression,
				valueSource,
			)
		}
		value, err := children.Expression(
			context.
				WithRole(api.RoleCompositeElement).
				WithExpectedType(fieldType),
			valueSource,
		)
		if err != nil {
			return nil, err
		}
		value, err = context.Values().Transfer(
			context.WithRole(api.RoleCompositeElement),
			valueSource,
			valueType,
			fieldType,
			api.ValueTransferCopy,
			value,
		)
		if err != nil {
			return nil, err
		}
		requests, err := cooperative.JoinNominalFieldCallableABIs(
			context.WithRole(api.RoleCompositeElement),
			named,
			selectedField,
		)
		if err != nil {
			return nil, err
		}
		value, err = api.NewExpressionEmission(
			value.Before(),
			value.Value(),
			api.CombineRequests(value.Requests(), requests),
		)
		if err != nil {
			return nil, err
		}
		if named != nil {
			value, _, err = providerboundary.ToProviderStructField(
				context.WithRole(api.RoleCompositeElement),
				children,
				valueSource,
				named,
				selectedField,
				value,
			)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, element{
			fieldIndex:       fieldIndex,
			declarationField: field.Declaration(),
			selectedField:    selectedField,
			source:           valueSource,
			value:            value,
		})
	}
	if keyedCount != 0 && keyedCount != len(source.Elts) {
		return nil, api.Unsupported(
			context.WithRole(api.RoleCompositeElement),
			api.CategoryExpression,
			source,
		)
	}
	if keyedCount == 0 &&
		len(source.Elts) != 0 &&
		len(source.Elts) != structType.NumFields() {
		return nil, api.Unsupported(
			context.WithRole(api.RoleCompositeElement),
			api.CategoryExpression,
			source,
		)
	}
	return result, nil
}
