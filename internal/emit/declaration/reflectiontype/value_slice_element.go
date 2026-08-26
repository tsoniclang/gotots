package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type reflectionSliceElement struct {
	sourceType     types.Type
	storage        api.TypeEmission
	adapter        api.NameReference
	descriptor     api.NameReference
	zero           api.ExpressionEmission
	interfaceValue bool
	structural     bool
}

func newReflectionSliceElement(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	elementType types.Type,
	scaffold *locationScaffold,
) (reflectionSliceElement, error) {
	_, interfaceValue := types.Unalias(elementType).Underlying().(*types.Interface)
	var adapter api.NameReference
	var err error
	if !interfaceValue {
		adapter, err = context.Names().InterfaceAdapter(elementType, nil)
		if err != nil {
			return reflectionSliceElement{}, err
		}
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
	if !interfaceValue {
		scaffold.requests = append(scaffold.requests, adapter.Requests()...)
	}
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	scaffold.requests = append(scaffold.requests, storage.Requests()...)
	scaffold.requests = append(scaffold.requests, zero.Requests()...)
	return reflectionSliceElement{
		sourceType:     elementType,
		storage:        storage,
		adapter:        adapter,
		descriptor:     descriptor,
		zero:           zero,
		interfaceValue: interfaceValue,
		structural:     context.Values().RequiresStructuralCopy(context, elementType),
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
		e.sourceType,
		copied,
	)
}

func (e reflectionSliceElement) copyFromBox(
	context api.Context,
	operation string,
	scaffold *locationScaffold,
) (api.ExpressionEmission, error) {
	value := scaffold.factory.Identifier("value")
	logical := api.DirectExpression(value)
	if e.interfaceValue {
		admitted, requests, err := admittedInterfaceValue(
			context,
			e.sourceType,
			value,
			operation,
			scaffold,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		logical = api.DirectExpression(admitted, requests...)
	} else {
		logical = api.DirectExpression(guardedForeignPayload(
			scaffold,
			e.adapter,
			operation,
		))
	}
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleSliceElement),
		nil,
		e.sourceType,
		e.sourceType,
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
		e.sourceType,
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
	boxed := read.Value()
	if !e.interfaceValue {
		boxed = factory.NewExpression(
			e.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{read.Value()},
		)
	}
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
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: e.descriptor,
		settable:   true,
		get:        get,
		getBlock:   getBlock,
		set:        factory.Block(setStatements, true),
	})
	indexBody := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Index"),
		constStatement(factory, "instance", scaffoldPayload(scaffold)),
		factory.ReturnStatement(location),
	}, true)
	scaffold.requests = append(scaffold.requests, read.Requests()...)
	scaffold.requests = append(scaffold.requests, stored.Requests()...)
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
