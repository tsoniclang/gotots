package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/namedstructstorage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func nonBlankStructFieldCallbacks(
	context api.Context,
	sourceType types.Type,
	field *types.Var,
	scaffold *locationScaffold,
) (
	tsgo.Expression,
	tsgo.Block,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	factory := scaffold.factory
	target, storageBacked, err := namedstructstorage.FieldTarget(
		context.WithRole(api.RoleStructField),
		nil,
		sourceType,
		field,
		api.DirectExpression(factory.Identifier("instance")),
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if storageBacked {
		return storageStructFieldCallbacks(context, field, target, scaffold)
	}
	member, err := context.Names().Member(field)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	fieldAccess := tsgo.Expression(memberAccess(
		factory,
		"instance",
		member,
	))
	if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); isInterface {
		boxed, set, requests, callbackErr := interfaceFieldCallbacks(
			context,
			field,
			fieldAccess,
			scaffold,
		)
		return boxed, nil, set, requests, callbackErr
	}
	fieldAdapter, err := context.Names().InterfaceAdapter(field.Type(), nil)
	if err != nil {
		return nil, nil, nil, nil, err
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
	setStatements := append([]tsgo.Statement(nil), copied.Before()...)
	setStatements = append(
		setStatements,
		factory.ExpressionStatement(factory.BinaryExpression(
			nil,
			fieldAccess,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			copied.Value(),
		)),
	)
	return factory.NewExpression(
			fieldAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{fieldAccess},
		), nil, factory.Block(setStatements, true), api.CombineRequests(
			fieldAdapter.Requests(),
			copied.Requests(),
		), nil
}

func storageStructFieldCallbacks(
	context api.Context,
	field *types.Var,
	target api.StoreTargetEmission,
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
	scaffold *locationScaffold,
) (
	tsgo.Expression,
	tsgo.Block,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	assigned, contractRequests, err := admittedInterfaceFieldValue(
		context,
		field,
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
	get := read.Value()
	var getBlock tsgo.Block
	if len(read.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), read.Before()...)
		statements = append(statements, scaffold.factory.ReturnStatement(get))
		get = nil
		getBlock = scaffold.factory.Block(statements, true)
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
