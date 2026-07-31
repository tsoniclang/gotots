package namedstructstorage

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Selected(
	context api.Context,
	sourceType types.Type,
) (*types.Named, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok ||
		named.Obj() == nil ||
		named.TypeParams().Len() == 0 ||
		named.TypeArgs().Len() != named.TypeParams().Len() {
		return nil, false, nil
	}
	_, selected, err := context.ResolveGenericNamedStructOperation(
		named.Origin().Obj(),
		api.NamedStructOperationStorage,
	)
	if err != nil {
		return nil, false, err
	}
	if !selected {
		return nil, false, nil
	}
	return named, true, nil
}

func FieldTarget(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	field *types.Var,
	receiver api.ExpressionEmission,
) (api.StoreTargetEmission, bool, error) {
	named, selected, err := Selected(context, sourceType)
	if err != nil || !selected {
		return api.StoreTargetEmission{}, selected, err
	}
	if field == nil || !containsField(named, field) {
		return api.StoreTargetEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic storage field does not belong to its receiver",
		}
	}
	member, err := context.Names().Member(field)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	reference, err := context.Names().NamedStructOperation(
		named.Origin().Obj(),
		api.NamedStructOperationStorage,
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	storage, err := api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(api.StructStorageOfMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{receiver.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(receiver.Requests(), reference.Requests()),
	)
	if err != nil {
		return api.StoreTargetEmission{}, true, err
	}
	target, err := api.NewCanonicalStoragePropertyStoreTargetEmission(
		context.Factory(),
		storage,
		member,
		field.Type(),
	)
	return target, true, err
}

func DemandFieldOwner(
	context api.Context,
	sourceType types.Type,
	field *types.Var,
	receiver api.ExpressionEmission,
) (api.ExpressionEmission, string, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok ||
		named.Obj() == nil ||
		!containsField(named, field) {
		return api.ExpressionEmission{}, "", &api.InvariantError{
			Role:   context.Role(),
			Reason: "field-storage owner is not an exact named struct field",
		}
	}
	member, err := context.Names().Member(field)
	if err != nil {
		return api.ExpressionEmission{}, "", err
	}
	reference, err := context.Names().NamedStructOperation(
		named.Origin().Obj(),
		api.NamedStructOperationStorage,
	)
	if err != nil {
		return api.ExpressionEmission{}, "", err
	}
	storage, err := api.NewExpressionEmission(
		receiver.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(api.StructStorageOfMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{receiver.Value()},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(receiver.Requests(), reference.Requests()),
	)
	return storage, member, err
}

func containsField(named *types.Named, field *types.Var) bool {
	structType, ok := named.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for index := range structType.NumFields() {
		if structType.Field(index) == field {
			return true
		}
	}
	return false
}
