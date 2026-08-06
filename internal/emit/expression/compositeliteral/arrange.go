package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func arrange(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	named *types.Named,
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
	present := make(map[int]struct{}, len(elements))
	for _, element := range elements {
		present[element.fieldIndex] = struct{}{}
	}
	zeroByField := make(map[int]api.ExpressionEmission)
	for fieldIndex := range structType.NumFields() {
		if _, ok := present[fieldIndex]; ok {
			continue
		}
		fieldType := structType.Field(fieldIndex).Type()
		zero, err := context.Values().Zero(
			context.WithRole(api.RoleStructZeroField),
			source,
			fieldType,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if named != nil {
			zero, _, err = providerboundary.ToProviderStructField(
				context.WithRole(api.RoleStructZeroField),
				children,
				source,
				named,
				structType.Field(fieldIndex),
				zero,
			)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		if canonicalStorage {
			zero, err = context.Values().ToStorage(
				context.WithRole(api.RoleStructZeroField),
				source,
				fieldType,
				zero,
			)
			if err != nil {
				return nil, nil, nil, err
			}
		}
		zeroByField[fieldIndex] = zero
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
	if !capture {
		for _, zero := range zeroByField {
			if len(zero.Before()) != 0 {
				capture = true
				break
			}
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
		zero := zeroByField[fieldIndex]
		before = append(before, zero.Before()...)
		values = append(values, zero.Value())
		requests = append(requests, zero.Requests()...)
	}
	return before, requests, values, nil
}
