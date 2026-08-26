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
	scaffold *locationScaffold,
) (
	tsgo.Expression,
	tsgo.Block,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	factory := scaffold.factory
	read, err := target.ReadValue(
		context.WithRole(api.RoleStructField),
		nil,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); isInterface {
		return storageInterfaceFieldCallbacks(
			context,
			field,
			target,
			read,
			settable,
			scaffold,
		)
	}
	fieldAdapter, err := context.Names().InterfaceAdapter(field.Type(), nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	boxed := factory.NewExpression(
		fieldAdapter.Expression(factory),
		nil,
		[]tsgo.Expression{read.Value()},
	)
	var get tsgo.Expression = boxed
	var getBlock tsgo.Block
	if len(read.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), read.Before()...)
		statements = append(statements, factory.ReturnStatement(boxed))
		get = nil
		getBlock = factory.Block(statements, true)
	}
	if !settable {
		return get, getBlock, unaddressableFieldSetter(scaffold),
			api.CombineRequests(
				fieldAdapter.Requests(),
				read.Requests(),
			), nil
	}
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleStructCopyField),
		nil,
		field.Type(),
		field.Type(),
		api.ValueTransferCopy,
		api.DirectExpression(guardedForeignPayload(
			scaffold,
			fieldAdapter,
			"Value.Set",
		)),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleStructField),
		nil,
		copied,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	setStatements := append([]tsgo.Statement(nil), stored.Before()...)
	setStatements = append(
		setStatements,
		factory.ExpressionStatement(stored.Value()),
	)
	return get, getBlock, factory.Block(setStatements, true),
		api.CombineRequests(
			fieldAdapter.Requests(),
			read.Requests(),
			stored.Requests(),
		), nil
}

func storageInterfaceFieldCallbacks(
	context api.Context,
	field *types.Var,
	target api.StoreTargetEmission,
	read api.ExpressionEmission,
	settable bool,
	scaffold *locationScaffold,
) (
	tsgo.Expression,
	tsgo.Block,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	get := read.Value()
	var getBlock tsgo.Block
	if len(read.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), read.Before()...)
		statements = append(statements, scaffold.factory.ReturnStatement(get))
		get = nil
		getBlock = scaffold.factory.Block(statements, true)
	}
	if !settable {
		return get, getBlock, unaddressableFieldSetter(scaffold),
			read.Requests(), nil
	}
	assigned, contractRequests, err := admittedInterfaceValue(
		context,
		field.Type(),
		scaffold.factory.Identifier("value"),
		"Value.Set",
		scaffold,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleStructField),
		nil,
		api.DirectExpression(assigned),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	setStatements := append([]tsgo.Statement(nil), stored.Before()...)
	setStatements = append(
		setStatements,
		scaffold.factory.ExpressionStatement(stored.Value()),
	)
	return get, getBlock, scaffold.factory.Block(setStatements, true),
		api.CombineRequests(
			contractRequests,
			read.Requests(),
			stored.Requests(),
		), nil
}

func unaddressableFieldSetter(scaffold *locationScaffold) tsgo.Block {
	return scaffold.factory.Block(
		[]tsgo.Statement{scaffold.factory.ExpressionStatement(runtimePanic(
			scaffold,
			"reflect: Value.Set using unaddressable value",
		))},
		true,
	)
}
