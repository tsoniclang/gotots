package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func reflectedStructFieldTarget(
	context api.Context,
	sourceType types.Type,
	field *types.Var,
	receiver api.ExpressionEmission,
) (api.StoreTargetEmission, error) {
	target, storageBacked, err := namedstructstorage.FieldTarget(
		context,
		nil,
		sourceType,
		field,
		receiver,
	)
	if err != nil || storageBacked {
		return target, err
	}
	named, namedOK := types.Unalias(sourceType).(*types.Named)
	if namedOK && named.Obj() != nil && !field.Exported() &&
		field.Pkg() != named.Obj().Pkg() {
		owner, member, ownerErr := namedstructstorage.DemandFieldOwner(
			context,
			sourceType,
			field,
			receiver,
		)
		if ownerErr != nil {
			return api.StoreTargetEmission{}, ownerErr
		}
		return api.NewCanonicalStoragePropertyStoreTargetEmission(
			context.Factory(),
			owner,
			member,
			field.Type(),
		)
	}
	member, err := context.Names().Member(field)
	if err != nil {
		return api.StoreTargetEmission{}, err
	}
	return api.NewPropertyStoreTargetEmission(
		context.Factory(),
		receiver,
		member,
		field.Type(),
	)
}

func storageStructFieldCallbacks(
	context api.Context,
	field *types.Var,
	target api.StoreTargetEmission,
	settable bool,
) (
	tsgo.ConciseBody,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	factory := context.Factory()
	read, err := target.ReadValue(
		context.WithRole(api.RoleStructField),
		nil,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	var get tsgo.ConciseBody = factory.ParenthesizedExpression(read.Value())
	if len(read.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), read.Before()...)
		statements = append(statements, factory.ReturnStatement(read.Value()))
		get = factory.Block(statements, true)
	}
	if !settable {
		return get, nil, read.Requests(), nil
	}
	storedValue := api.DirectExpression(factory.Identifier("value"))
	if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); !isInterface {
		storedValue, err = context.Values().Transfer(
			context.WithRole(api.RoleStructCopyField),
			nil,
			field.Type(),
			field.Type(),
			api.ValueTransferCopy,
			storedValue,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleStructField),
		nil,
		storedValue,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	setStatements := append([]tsgo.Statement(nil), stored.Before()...)
	setStatements = append(
		setStatements,
		factory.ExpressionStatement(stored.Value()),
	)
	return get, factory.Block(setStatements, true),
		api.CombineRequests(
			read.Requests(),
			stored.Requests(),
		), nil
}
