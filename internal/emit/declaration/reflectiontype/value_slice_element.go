package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type reflectionSliceElement struct {
	member     reflectionMemberBox
	storage    api.TypeEmission
	descriptor api.NameReference
	zero       api.ExpressionEmission
	structural bool
}

func newReflectionSliceElement(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	elementType types.Type,
	scaffold *locationScaffold,
) (reflectionSliceElement, error) {
	member, err := newReflectionMemberBox(context, elementType)
	if err != nil {
		return reflectionSliceElement{}, err
	}
	descriptor, err := names.ReflectionValueType(elementType, reflectionType)
	if err != nil {
		return reflectionSliceElement{}, err
	}
	storage, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
		nil,
		elementType,
	)
	if err != nil {
		return reflectionSliceElement{}, err
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleSliceElement),
		nil,
		elementType,
	)
	if err != nil {
		return reflectionSliceElement{}, err
	}
	zero, err = context.ContainerStorage().ToContainerStorage(
		context.WithRole(api.RoleSliceElement),
		nil,
		elementType,
		zero,
	)
	if err != nil {
		return reflectionSliceElement{}, err
	}
	scaffold.requests = append(scaffold.requests, member.requests()...)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	scaffold.requests = append(scaffold.requests, storage.Requests()...)
	scaffold.requests = append(scaffold.requests, zero.Requests()...)
	return reflectionSliceElement{
		member:     member,
		storage:    storage,
		descriptor: descriptor,
		zero:       zero,
		structural: context.Values().RequiresStructuralCopy(context, elementType),
	}, nil
}

func (e reflectionSliceElement) copyToStorage(
	context api.Context,
	operation string,
	scaffold *locationScaffold,
) (api.ExpressionEmission, error) {
	copied, err := e.copyFromBox(context, operation, scaffold)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.ContainerStorage().ToContainerStorage(
		context.WithRole(api.RoleSliceElement),
		nil,
		e.member.sourceType,
		copied,
	)
}

func (e reflectionSliceElement) copyFromBox(
	context api.Context,
	operation string,
	scaffold *locationScaffold,
) (api.ExpressionEmission, error) {
	logical, err := e.member.admittedValue(
		context,
		"value",
		operation,
		scaffold,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleSliceElement),
		nil,
		e.member.sourceType,
		e.member.sourceType,
		api.ValueTransferCopy,
		logical,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return copied, nil
}

func (e reflectionSliceElement) indexOperation(
	context api.Context,
	indexType api.NameReference,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	factory := scaffold.factory
	target, err := api.NewContainerStorageAccessorStoreTargetEmission(
		api.DirectExpression(factory.Identifier("instance")),
		"get",
		"set",
		[]api.ExpressionEmission{api.DirectExpression(factory.Identifier("index"))},
		e.member.sourceType,
	)
	if err != nil {
		return nil, err
	}
	read, err := target.ReadValue(
		context.WithRole(api.RoleSliceElement),
		nil,
	)
	if err != nil {
		return nil, err
	}
	boxed := e.member.boxedValue(factory, read.Value())
	var get tsgo.Expression = boxed
	var getBlock tsgo.Block
	if len(read.Before()) != 0 {
		statements := append([]tsgo.Statement(nil), read.Before()...)
		statements = append(statements, factory.ReturnStatement(boxed))
		get = nil
		getBlock = factory.Block(statements, true)
	}
	value, err := e.copyFromBox(
		context,
		"Value.Set",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleSliceElement),
		nil,
		value,
	)
	if err != nil {
		return nil, err
	}
	setStatements := append([]tsgo.Statement(nil), stored.Before()...)
	setStatements = append(
		setStatements,
		factory.ExpressionStatement(stored.Value()),
	)
	addressOperation, err := context.Names().Runtime(
		api.RuntimeSliceAddress,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, err
	}
	storagePointer, err := api.NewExpressionEmission(
		nil,
		factory.CallExpression(
			addressOperation.Expression(factory),
			nil,
			[]tsgo.TypeNode{e.storage.Value()},
			[]tsgo.Expression{
				factory.Identifier("instance"),
				factory.Identifier("index"),
			},
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			addressOperation.Requests(),
			e.storage.Requests(),
		),
	)
	if err != nil {
		return nil, err
	}
	pointer, err := context.Values().ProjectStoragePointer(
		context.WithRole(api.RoleSliceElement),
		nil,
		e.member.sourceType,
		storagePointer,
	)
	if err != nil {
		return nil, err
	}
	boxedAddress, err := boxedReflectionAddress(
		context,
		e.member.sourceType,
		pointer,
	)
	if err != nil {
		return nil, err
	}
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: e.descriptor,
		settable:   true,
		get:        get,
		getBlock:   getBlock,
		set:        factory.Block(setStatements, true),
		address:    addressCallback(factory, nil, boxedAddress),
	})
	indexBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Index"),
		constStatement(factory, "instance", scaffoldPayload(scaffold)),
		factory.ReturnStatement(location),
	}, true)
	scaffold.requests = append(scaffold.requests, read.Requests()...)
	scaffold.requests = append(scaffold.requests, stored.Requests()...)
	scaffold.requests = append(scaffold.requests, boxedAddress.Requests()...)
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("index"),
				nil,
				factory.TypeReferenceNode(indexType.EntityName(factory), nil),
				nil,
			),
		},
		nil,
		factory.EqualsGreaterThanToken(),
		indexBody,
	), nil
}
