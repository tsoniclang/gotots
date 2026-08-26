package reflectiontype

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e reflectionSliceElement) appendOperation(
	context api.Context,
	runtimeSlice api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	factory := scaffold.factory
	stored, err := e.copyToStorage(
		context,
		"Value.Append",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, stored.Requests()...)
	convert := reflectionSliceStorageArrow(e, stored, scaffold)
	converted := reflectionSliceCall(
		factory,
		factory.Identifier("values"),
		"map",
		convert,
	)
	if !e.structural {
		result := reflectionSliceCall(
			factory,
			scaffoldPayload(scaffold),
			runtimeslice.MemberName(runtimeslice.MemberAppend),
			e.zero.Value(),
			converted,
		)
		return reflectionSliceAppendArrow(
			scaffold,
			factory.ParenthesizedExpression(guardedProjection(
				scaffold,
				"Value.Append",
				reflectionSliceBox(scaffold, wrapSlice, result),
			)),
		), nil
	}
	return e.structuralAppendOperation(
		context,
		runtimeSlice,
		wrapSlice,
		converted,
		scaffold,
	)
}

func reflectionSliceStorageArrow(
	element reflectionSliceElement,
	stored api.ExpressionEmission,
	scaffold *locationScaffold,
) tsgo.ArrowFunction {
	factory := scaffold.factory
	parameter := factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier("value"),
		nil,
		optionalInterfaceBoxType(factory, scaffold.boxType),
		nil,
	)
	if len(stored.Before()) == 0 {
		return factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{parameter},
			element.storage.Value(),
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(stored.Value()),
		)
	}
	statements := append([]tsgo.Statement(nil), stored.Before()...)
	statements = append(statements, factory.ReturnStatement(stored.Value()))
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter},
		element.storage.Value(),
		factory.EqualsGreaterThanToken(),
		factory.Block(statements, true),
	)
}

func reflectionSliceAppendArrow(
	scaffold *locationScaffold,
	body tsgo.ConciseBody,
) tsgo.ArrowFunction {
	factory := scaffold.factory
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("values"),
				nil,
				factory.TypeOperatorNode(
					tsgo.TypeOperatorNodeOperatorKindReadonlyKeyword,
					factory.ArrayTypeNode(optionalInterfaceBoxType(
						factory,
						scaffold.boxType,
					)),
				),
				nil,
			),
		},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
}

func (e reflectionSliceElement) structuralAppendOperation(
	context api.Context,
	runtimeSlice api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	converted tsgo.Expression,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	factory := scaffold.factory
	allocation, err := context.Names().Runtime(
		api.RuntimeSliceStorage,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, err
	}
	existing, err := e.copyStoredElement(
		context,
		reflectionSliceCall(
			factory,
			factory.Identifier("instance"),
			runtimeslice.MemberName(runtimeslice.MemberGet),
			factory.Identifier("index"),
		),
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, allocation.Requests()...)
	scaffold.requests = append(scaffold.requests, existing.Requests()...)
	statements := []tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Append"),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "instance", scaffoldPayload(scaffold)),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "incoming", converted),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "nextLength", reflectionSliceBinary(
			factory,
			reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberLength)),
			tsgo.BinaryOperatorPlusToken,
			reflectionSliceMember(factory, factory.Identifier("incoming"), "length"),
		)),
		reflectionSliceVariable(factory, tsgo.NodeFlagsLet, "result", factory.Identifier("instance")),
		factory.IfStatement(
			reflectionSliceBinary(
				factory,
				factory.Identifier("nextLength"),
				tsgo.BinaryOperatorLessThanEqualsToken,
				reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberCapacity)),
			),
			factory.Block(reflectionSliceAppendReuse(factory), true),
			factory.Block(e.reflectionSliceAppendGrowth(
				factory,
				runtimeSlice,
				allocation,
				existing,
			), true),
		),
		factory.ReturnStatement(reflectionSliceBox(
			scaffold,
			wrapSlice,
			factory.Identifier("result"),
		)),
	}
	return reflectionSliceAppendArrow(scaffold, factory.Block(statements, true)), nil
}

func reflectionSliceAppendReuse(factory tsgo.Factory) []tsgo.Statement {
	result := factory.Identifier("result")
	instance := factory.Identifier("instance")
	incomingIndex := factory.Identifier("incomingIndex")
	statements := []tsgo.Statement{
		reflectionSliceAssignment(factory, result, reflectionSliceCall(
			factory,
			instance,
			runtimeslice.StorageWithLengthMember,
			factory.Identifier("nextLength"),
		)),
	}
	return append(statements, reflectionSliceIncomingLoop(
		factory,
		[]tsgo.Statement{factory.ExpressionStatement(reflectionSliceCall(
			factory,
			result,
			runtimeslice.MemberName(runtimeslice.MemberSet),
			reflectionSliceBinary(
				factory,
				reflectionSliceMember(factory, instance, runtimeslice.MemberName(runtimeslice.MemberLength)),
				tsgo.BinaryOperatorPlusToken,
				incomingIndex,
			),
			factory.Identifier("incomingValue"),
		))},
	)...)
}

func (e reflectionSliceElement) reflectionSliceAppendGrowth(
	factory tsgo.Factory,
	runtimeSlice api.NameReference,
	allocation api.NameReference,
	existing api.ExpressionEmission,
) []tsgo.Statement {
	result := factory.Identifier("result")
	instance := factory.Identifier("instance")
	index := factory.Identifier("index")
	incomingIndex := factory.Identifier("incomingIndex")
	statements := []tsgo.Statement{
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "nextCapacity", reflectionSliceCall(
			factory,
			runtimeSlice.Expression(factory),
			runtimeslice.StorageGrownCapacityMember,
			reflectionSliceMember(factory, instance, runtimeslice.MemberName(runtimeslice.MemberCapacity)),
			factory.Identifier("nextLength"),
		)),
		reflectionSliceAssignment(factory, result, factory.CallExpression(
			allocation.Expression(factory),
			nil,
			[]tsgo.TypeNode{e.storage.Value()},
			[]tsgo.Expression{factory.Identifier("nextLength"), factory.Identifier("nextCapacity")},
			tsgo.NodeFlagsNone,
		)),
		reflectionSliceLoop(
			factory,
			"index",
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			reflectionSliceMember(factory, instance, runtimeslice.MemberName(runtimeslice.MemberLength)),
			append(
				existing.Before(),
				factory.ExpressionStatement(reflectionSliceCall(
					factory,
					result,
					runtimeslice.MemberName(runtimeslice.MemberSet),
					index,
					existing.Value(),
				)),
			),
		),
	}
	statements = append(statements, reflectionSliceIncomingLoop(
		factory,
		[]tsgo.Statement{factory.ExpressionStatement(reflectionSliceCall(
			factory,
			result,
			runtimeslice.MemberName(runtimeslice.MemberSet),
			reflectionSliceBinary(
				factory,
				reflectionSliceMember(factory, instance, runtimeslice.MemberName(runtimeslice.MemberLength)),
				tsgo.BinaryOperatorPlusToken,
				incomingIndex,
			),
			factory.Identifier("incomingValue"),
		))},
	)...)
	return append(statements, e.reflectionSliceInitializeTail(
		factory,
		factory.Identifier("nextLength"),
	)...)
}

func (e reflectionSliceElement) reflectionSliceInitializeTail(
	factory tsgo.Factory,
	start tsgo.Expression,
) []tsgo.Statement {
	body := append([]tsgo.Statement(nil), e.zero.Before()...)
	body = append(body, factory.ExpressionStatement(reflectionSliceCall(
		factory,
		factory.Identifier("result"),
		runtimeslice.StorageInitializeMember,
		factory.Identifier("index"),
		e.zero.Value(),
	)))
	return []tsgo.Statement{reflectionSliceLoop(
		factory,
		"index",
		start,
		reflectionSliceMember(factory, factory.Identifier("result"), runtimeslice.MemberName(runtimeslice.MemberCapacity)),
		body,
	)}
}

func (e reflectionSliceElement) copyStoredElement(
	context api.Context,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	loaded, err := context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleSliceElement),
		nil,
		e.member.sourceType,
		api.DirectExpression(value),
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
		loaded,
	)
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

func (e reflectionSliceElement) makeOperation(
	context api.Context,
	indexType api.NameReference,
	runtimeSlice api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	factory := scaffold.factory
	var made api.ExpressionEmission
	var err error
	if e.structural {
		made, err = slicevalue.MakeAggregate(
			context,
			e.storage,
			nil,
			e.member.sourceType,
			factory.Identifier("length"),
			factory.Identifier("capacity"),
			nil,
			nil,
		)
	} else {
		made = api.DirectExpression(reflectionSliceCall(
			factory,
			runtimeSlice.Expression(factory),
			runtimeslice.MemberName(runtimeslice.MemberMake),
			factory.Identifier("length"),
			factory.Identifier("capacity"),
			e.zero.Value(),
		))
	}
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, made.Requests()...)
	statements := append([]tsgo.Statement(nil), made.Before()...)
	statements = append(statements, factory.ReturnStatement(reflectionSliceBox(
		scaffold,
		wrapSlice,
		made.Value(),
	)))
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			reflectionSliceIndexParameter(factory, indexType, "length"),
			reflectionSliceIndexParameter(factory, indexType, "capacity"),
		},
		factory.TypeReferenceNode(scaffold.boxType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.Block(statements, true),
	), nil
}

func reflectionSliceIndexParameter(
	factory tsgo.Factory,
	indexType api.NameReference,
	name string,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		factory.TypeReferenceNode(indexType.EntityName(factory), nil),
		nil,
	)
}

func (e reflectionSliceElement) growOperation(
	context api.Context,
	indexType api.NameReference,
	runtimeSlice api.NameReference,
	wrapSlice func(tsgo.Expression) tsgo.Expression,
	scaffold *locationScaffold,
) (tsgo.ArrowFunction, error) {
	factory := scaffold.factory
	allocation, err := context.Names().Runtime(api.RuntimeSliceStorage, api.ImportPhaseValue)
	if err != nil {
		return nil, err
	}
	var existing api.ExpressionEmission
	if e.structural {
		existing, err = e.copyStoredElement(context, reflectionSliceCall(
			factory,
			factory.Identifier("instance"),
			runtimeslice.MemberName(runtimeslice.MemberGet),
			factory.Identifier("index"),
		))
	} else {
		existing = api.DirectExpression(reflectionSliceCall(
			factory,
			factory.Identifier("instance"),
			runtimeslice.MemberName(runtimeslice.MemberGet),
			factory.Identifier("index"),
		))
	}
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, allocation.Requests()...)
	scaffold.requests = append(scaffold.requests, existing.Requests()...)
	statements := []tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Grow"),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "instance", scaffoldPayload(scaffold)),
		factory.IfStatement(
			reflectionSliceBinary(
				factory,
				reflectionSliceNumberConversion(factory, factory.Identifier("count")),
				tsgo.BinaryOperatorLessThanToken,
				factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			),
			factory.ReturnStatement(runtimePanic(
				scaffold,
				"reflect.Value.Grow: negative len",
			)),
			nil,
		),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "required", reflectionSliceBinary(
			factory,
			reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberLength)),
			tsgo.BinaryOperatorPlusToken,
			reflectionSliceNumberConversion(factory, factory.Identifier("count")),
		)),
		factory.IfStatement(
			reflectionSliceBinary(
				factory,
				factory.Identifier("required"),
				tsgo.BinaryOperatorLessThanEqualsToken,
				reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberCapacity)),
			),
			factory.ReturnStatement(reflectionSliceBox(scaffold, wrapSlice, factory.Identifier("instance"))),
			nil,
		),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "nextCapacity", reflectionSliceCall(
			factory,
			runtimeSlice.Expression(factory),
			runtimeslice.StorageGrownCapacityMember,
			reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberCapacity)),
			factory.Identifier("required"),
		)),
		reflectionSliceVariable(factory, tsgo.NodeFlagsConst, "result", factory.CallExpression(
			allocation.Expression(factory),
			nil,
			[]tsgo.TypeNode{e.storage.Value()},
			[]tsgo.Expression{
				reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberLength)),
				factory.Identifier("nextCapacity"),
			},
			tsgo.NodeFlagsNone,
		)),
		reflectionSliceLoop(
			factory,
			"index",
			factory.NumericLiteral("0", tsgo.TokenFlagsNone),
			reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberLength)),
			append(
				existing.Before(),
				factory.ExpressionStatement(reflectionSliceCall(
					factory,
					factory.Identifier("result"),
					runtimeslice.MemberName(runtimeslice.MemberSet),
					factory.Identifier("index"),
					existing.Value(),
				)),
			),
		),
	}
	statements = append(statements, e.reflectionSliceInitializeTail(
		factory,
		reflectionSliceMember(factory, factory.Identifier("instance"), runtimeslice.MemberName(runtimeslice.MemberLength)),
	)...)
	statements = append(statements, factory.ReturnStatement(reflectionSliceBox(
		scaffold,
		wrapSlice,
		factory.Identifier("result"),
	)))
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			reflectionSliceIndexParameter(factory, indexType, "count"),
		},
		factory.TypeReferenceNode(scaffold.boxType.EntityName(factory), nil),
		factory.EqualsGreaterThanToken(),
		factory.Block(statements, true),
	), nil
}
