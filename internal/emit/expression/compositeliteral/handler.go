package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/expression/mapliteral"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
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
	if _, ok := types.Unalias(
		context.TypesInfo().TypeOf(source),
	).(*types.Map); ok {
		return mapliteral.Emit(context, children, source)
	}
	if array, ok := arrayvalue.Resolve(
		context,
		context.TypesInfo().TypeOf(source),
	); ok {
		return array.EmitLiteral(context, children, source)
	}
	if target, handled, err := emitSlice(context, children, source); handled || err != nil {
		return target, err
	}
	named, structType, ok := structSourceType(context, source)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	elements, err := emitElements(context, children, source, structType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before, requests, values, err := arrange(
		context,
		children,
		source,
		structType,
		elements,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Reference(named.Obj())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().NewExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			values,
		),
		api.CombineRequests(requests, reference.Requests()),
	)
}

func structSourceType(
	context api.Context,
	source *ast.CompositeLit,
) (*types.Named, *types.Struct, bool) {
	sourceType := context.TypesInfo().TypeOf(source)
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.TypeParams().Len() != 0 || source.Incomplete {
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
		fieldType := structType.Field(fieldIndex).Type()
		value, err := children.Expression(
			context.
				WithRole(api.RoleCompositeElement).
				WithExpectedType(fieldType),
			valueSource,
		)
		if err != nil {
			return nil, err
		}
		value, err = context.Values().Copy(
			context.WithRole(api.RoleCompositeElement),
			valueSource,
			fieldType,
			value,
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
) (
	[]tsgo.Statement,
	[]api.RootRequest,
	[]tsgo.Expression,
	error,
) {
	capture := false
	for index, element := range elements {
		reordersSource := element.fieldIndex != index &&
			context.EvaluationOrder() == api.EvaluationOrderPreserveGo
		if reordersSource || len(element.value.Before()) != 0 {
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
