package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func reflectedStructFieldAccesses(
	context api.Context,
	sourceType types.Type,
	structType *types.Struct,
) (
	[]*reflectedStructFieldAccess,
	tsgo.Expression,
	[]api.RootRequest,
	bool,
	error,
) {
	factory := context.Factory()
	receiver := api.DirectExpression(factory.Identifier("instance"))
	accesses := make([]*reflectedStructFieldAccess, structType.NumFields())
	var first *reflectedStructFieldAccess
	uniform := true
	for index := range structType.NumFields() {
		field := structType.Field(index)
		if field.Name() == "_" {
			continue
		}
		access, err := reflectedStructFieldTarget(
			context.WithRole(api.RoleStructField),
			sourceType,
			field,
			receiver,
		)
		if err != nil {
			return nil, nil, nil, false, err
		}
		accesses[index] = &access
		if first == nil {
			first = &access
			continue
		}
		if first.storageBacked != access.storageBacked {
			uniform = false
		}
	}
	storage := api.DirectExpression(factory.Identifier("instance"))
	var requests []api.RootRequest
	if uniform && first != nil && first.storageBacked {
		if len(first.storage.Before()) != 0 {
			return nil, nil, nil, false, &api.GeneratedArtifactShapeError{
				Artifact: sourceType.String(),
				Reason:   "reflected struct storage resolver has prerequisites",
			}
		}
		storage = first.storage
		requests = first.storage.Requests()
	}
	resolver := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{untypedParameter(factory, "instance")},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(storage.Value()),
	)
	return accesses, resolver, requests, uniform, nil
}

func structValuePropertyFact(
	context api.Context,
	names api.ReflectionNames,
	field *types.Var,
	access *reflectedStructFieldAccess,
	descriptor api.NameReference,
	adapter api.NameReference,
	settable bool,
	scaffold *locationScaffold,
) (tsgo.Expression, bool, error) {
	if access == nil || field.Name() == "_" {
		return nil, false, nil
	}
	if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); isInterface {
		return nil, false, nil
	}
	projection, err := context.Values().RequiresStorageProjection(
		context.WithRole(api.RoleStructField),
		field.Type(),
	)
	if err != nil {
		return nil, false, err
	}
	if access.target.UsesCanonicalStorage() && projection {
		return nil, false, nil
	}

	factory := scaffold.factory
	member := "readonlyValueProperty"
	arguments := []tsgo.Expression{
		arrow(factory, scaffold.descriptorType, descriptor.Expression(factory)),
		factory.ArrowFunction(
			nil,
			nil,
			nil,
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(adapter.Expression(factory)),
		),
		factory.StringLiteral(access.member, tsgo.TokenFlagsNone),
	}
	if settable {
		member = "valueProperty"
		if context.Values().RequiresStructuralCopy(context, field.Type()) {
			copied, copyErr := context.Values().Transfer(
				context.WithRole(api.RoleStructCopyField),
				nil,
				field.Type(),
				field.Type(),
				api.ValueTransferCopy,
				api.DirectExpression(factory.Identifier("value")),
			)
			if copyErr != nil {
				return nil, false, copyErr
			}
			member = "copyingValueProperty"
			arguments = append(arguments, expressionCallback(
				factory,
				"value",
				copied,
			))
			scaffold.requests = append(
				scaffold.requests,
				copied.Requests()...,
			)
		}
	}
	address, addressErr := reflectedStructPropertyAddress(
		context,
		names,
		field,
		access,
	)
	if addressErr != nil {
		return nil, false, addressErr
	}
	arguments = append(arguments, addressCallback(
		factory,
		[]tsgo.ParameterDeclaration{untypedParameter(factory, "storage")},
		address,
	))
	scaffold.requests = append(scaffold.requests, address.Requests()...)
	return structFieldBuilderCall(factory, member, arguments), true, nil
}

func reflectedStructPropertyAddress(
	context api.Context,
	names api.ReflectionNames,
	field *types.Var,
	access *reflectedStructFieldAccess,
) (api.ExpressionEmission, error) {
	factory := context.Factory()
	storage := api.DirectExpression(factory.Identifier("storage"))
	var target api.StoreTargetEmission
	var err error
	if access.target.UsesCanonicalStorage() {
		target, err = api.NewCanonicalStoragePropertyStoreTargetEmission(
			factory,
			storage,
			access.member,
			field.Type(),
		)
	} else {
		target, err = api.NewPropertyStoreTargetEmission(
			factory,
			storage,
			access.member,
			field.Type(),
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return reflectedStoreTargetAddress(
		context,
		names,
		field.Type(),
		target,
	)
}

func expressionCallback(
	factory tsgo.Factory,
	parameter string,
	value api.ExpressionEmission,
) tsgo.Expression {
	var body tsgo.ConciseBody = factory.ParenthesizedExpression(value.Value())
	if len(value.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), value.Before()...)
		statements = append(statements, factory.ReturnStatement(value.Value()))
		body = factory.Block(statements, true)
	}
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{untypedParameter(factory, parameter)},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
}
