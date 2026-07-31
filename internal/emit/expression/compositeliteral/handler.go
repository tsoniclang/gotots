package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/expression/mapliteral"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type element struct {
	fieldIndex int
	source     ast.Expr
	value      api.ExpressionEmission
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
	if named != nil {
		_, canonicalStorage, err = namedstructstorage.Selected(
			context,
			named,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	before, requests, values, err := arrange(
		context,
		children,
		source,
		structType,
		elements,
		canonicalStorage,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if named == nil {
		reference, err := context.Names().AnonymousStruct(
			structType,
			api.AnonymousStructDemandDefinition,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
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
				nil,
				values,
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(requests, reference.Requests()),
		)
	}
	var reference api.NameReference
	var typeArguments []tsgo.TypeNode
	if canonicalStorage {
		reference, err = context.Names().NamedStructOperation(
			named.Origin().Obj(),
			api.NamedStructOperationStorage,
		)
		if err == nil {
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
		reference, err = context.Names().NamedStructConstructor(
			named.Origin().Obj(),
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
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
		field := structType.Field(element.fieldIndex)
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
		name, err := context.Names().Member(field)
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
		fieldIndex := sourceIndex
		valueSource := sourceElement
		if keyed, ok := sourceElement.(*ast.KeyValueExpr); ok {
			keyedCount++
			identifier, identifierOK := keyed.Key.(*ast.Ident)
			field, fieldOK := context.TypesInfo().Uses[identifier].(*types.Var)
			if !identifierOK || !fieldOK {
				return nil, api.Unsupported(
					context.WithRole(api.RoleCompositeElement),
					api.CategoryExpression,
					sourceElement,
				)
			}
			fieldIndex = structFieldIndex(structType, field)
			valueSource = keyed.Value
		}
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
		field := structType.Field(fieldIndex)
		fieldType := field.Type()
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
			field,
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
		result = append(result, element{
			fieldIndex: fieldIndex,
			source:     valueSource,
			value:      value,
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

func arrange(
	context api.Context,
	_ api.ChildEmitter,
	source *ast.CompositeLit,
	structType *types.Struct,
	elements []element,
	canonicalStorage bool,
) (
	[]tsgo.Statement,
	[]api.RootRequest,
	[]tsgo.Expression,
	error,
) {
	if canonicalStorage {
		elements = append([]element(nil), elements...)
		for index := range elements {
			fieldType := structType.Field(elements[index].fieldIndex).Type()
			stored, err := context.Values().ToStorage(
				context.WithRole(api.RoleStructAssignField),
				elements[index].source,
				fieldType,
				elements[index].value,
			)
			if err != nil {
				return nil, nil, nil, err
			}
			elements[index].value = stored
		}
	}
	capture := false
	for index, element := range elements {
		reordersSource := element.fieldIndex != index &&
			context.EvaluationOrder() == api.EvaluationOrderPreserveGo
		blankField := structType.Field(element.fieldIndex).Name() == "_"
		if reordersSource || blankField || len(element.value.Before()) != 0 {
			capture = true
			break
		}
	}
	byField := make(map[int]tsgo.Expression, len(elements))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, element := range elements {
		requests = append(requests, element.value.Requests()...)
		if !capture {
			byField[element.fieldIndex] = element.value.Value()
			continue
		}
		name, err := context.Names().Temporary(api.TemporaryCompositeField)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, element.value.Before()...)
		before = append(before, context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					element.value.Value(),
				)},
				tsgo.NodeFlagsConst,
			),
		))
		byField[element.fieldIndex] = context.Factory().Identifier(name)
	}
	values := make([]tsgo.Expression, 0, structType.NumFields())
	for fieldIndex := range structType.NumFields() {
		if value := byField[fieldIndex]; value != nil {
			values = append(values, value)
			continue
		}
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleStructZeroField),
			source,
			structType.Field(fieldIndex).Type(),
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if canonicalStorage {
			zero, err = context.Values().ToStorage(
				context.WithRole(api.RoleStructZeroField),
				source,
				structType.Field(fieldIndex).Type(),
				zero,
			)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		if len(zero.Before()) != 0 {
			return nil, nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructZeroField),
				api.CategoryExpression,
				source,
			)
		}
		values = append(values, zero.Value())
		requests = append(requests, zero.Requests()...)
	}
	return before, requests, values, nil
}

func structFieldIndex(structType *types.Struct, field *types.Var) int {
	for index := range structType.NumFields() {
		if structType.Field(index) == field {
			return index
		}
	}
	return -1
}
