package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type constructionForm uint8

const (
	constructionFormInvalid constructionForm = iota
	constructionFormNamedObject
	constructionFormProviderFacet
)

type constructionCaptureReason uint8

const (
	constructionCaptureNone constructionCaptureReason = iota
	constructionCapturePrerequisite
	constructionCaptureProviderOrder
)

type arrangedField struct {
	index int
	value tsgo.Expression
}

func arrange(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CompositeLit,
	named *types.Named,
	structType *types.Struct,
	elements []element,
	canonicalStorage bool,
	form constructionForm,
) (
	[]tsgo.Statement,
	[]api.RootRequest,
	[]arrangedField,
	error,
) {
	if form != constructionFormNamedObject &&
		form != constructionFormProviderFacet {
		return nil, nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "struct construction form is invalid",
		}
	}
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
	captureThrough, _ := constructionCaptureBoundary(
		context,
		structType,
		elements,
		zeroByField,
		form,
	)
	byField := make(map[int]tsgo.Expression, len(elements))
	ordered := make([]arrangedField, 0, structType.NumFields())
	var before []tsgo.Statement
	var requests []api.RootRequest
	for index, element := range elements {
		requests = append(requests, element.value.Requests()...)
		value := element.value.Value()
		if index <= captureThrough {
			name, err := context.Names().Temporary(api.TemporaryCompositeField)
			if err != nil {
				return nil, nil, nil, err
			}
			captured := context.Factory().Identifier(name)
			before = append(before, element.value.Before()...)
			before = append(before, context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
						captured,
						nil,
						nil,
						value,
					)},
					tsgo.NodeFlagsConst,
				),
			))
			value = captured
		}
		byField[element.fieldIndex] = value
		ordered = append(ordered, arrangedField{
			index: element.fieldIndex,
			value: value,
		})
	}
	for fieldIndex := range structType.NumFields() {
		if value := byField[fieldIndex]; value != nil {
			continue
		}
		zero := zeroByField[fieldIndex]
		before = append(before, zero.Before()...)
		requests = append(requests, zero.Requests()...)
		byField[fieldIndex] = zero.Value()
		ordered = append(ordered, arrangedField{
			index: fieldIndex,
			value: zero.Value(),
		})
	}
	if form == constructionFormNamedObject {
		return before, requests, ordered, nil
	}
	positional := make([]arrangedField, 0, structType.NumFields())
	for fieldIndex := range structType.NumFields() {
		positional = append(positional, arrangedField{
			index: fieldIndex,
			value: byField[fieldIndex],
		})
	}
	return before, requests, positional, nil
}

func constructionCaptureBoundary(
	context api.Context,
	structType *types.Struct,
	elements []element,
	zeros map[int]api.ExpressionEmission,
	form constructionForm,
) (int, constructionCaptureReason) {
	boundary := -1
	reason := constructionCaptureNone
	for index, element := range elements {
		if len(element.value.Before()) != 0 {
			boundary = index
			reason = constructionCapturePrerequisite
			continue
		}
		if form == constructionFormProviderFacet &&
			(element.fieldIndex != index ||
				structType.Field(element.fieldIndex).Name() == "_") &&
			context.EvaluationOrder() == api.EvaluationOrderPreserveGo {
			boundary = index
			if reason == constructionCaptureNone {
				reason = constructionCaptureProviderOrder
			}
		}
	}
	for _, zero := range zeros {
		if len(zero.Before()) != 0 && len(elements) != 0 {
			return len(elements) - 1, constructionCapturePrerequisite
		}
	}
	return boundary, reason
}
