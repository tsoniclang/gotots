package compositeliteral

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func constructionFormForReference(reference api.NameReference) constructionForm {
	if reference.ProviderBoundary() {
		return constructionFormProviderFacet
	}
	return constructionFormNamedObject
}

func namedObjectConstruction(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	reference tsgo.Expression,
	typeArguments []tsgo.TypeNode,
	structType *types.Struct,
	fields []arrangedField,
	canonicalStorage bool,
) (tsgo.NewExpression, []api.RootRequest, error) {
	if structType == nil || len(fields) != structType.NumFields() {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "named struct construction field set is incomplete",
		}
	}
	if len(fields) == 0 {
		return context.Factory().NewExpression(
			reference,
			typeArguments,
			nil,
		), nil, nil
	}
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	var requests []api.RootRequest
	for _, selected := range fields {
		field := structType.Field(selected.index)
		name, err := structconstruction.FieldName(
			context.Names(),
			field,
			selected.index,
		)
		if err != nil {
			return nil, nil, err
		}
		fieldType, err := children.RepresentedType(
			context.WithRole(api.RoleStructFieldType),
			source,
			field.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		if canonicalStorage {
			fieldType, err = context.Values().StorageType(
				context.WithRole(api.RoleStorageType),
				source,
				field.Type(),
			)
			if err != nil {
				return nil, nil, err
			}
		}
		requests = append(requests, fieldType.Requests()...)
		properties = append(properties, context.Factory().PropertyAssignment(
			nil,
			context.Factory().Identifier(name),
			nil,
			fieldType.Value(),
			selected.value,
		))
	}
	return context.Factory().NewExpression(
		reference,
		typeArguments,
		[]tsgo.Expression{
			context.Factory().ObjectLiteralExpression(properties, true),
		},
	), requests, nil
}
