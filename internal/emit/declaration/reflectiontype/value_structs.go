package reflectiontype

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// pointerValueProperties adds the elem callback of one pointer to a
// directly represented struct: the returned location aliases the pointee
// instance so field navigation writes original storage, and whole-value
// writes assign every field exactly. A nil pointer yields no location,
// matching the Go zero Value result of Elem.
func pointerValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	structType, err := supportedValueStruct(pointee)
	if err != nil {
		return nil, err
	}
	elemAdapter, err := context.Names().InterfaceAdapter(pointee, nil)
	if err != nil {
		return nil, err
	}
	descriptor, err := names.ReflectionValueType(pointee, reflectionType)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, elemAdapter.Requests()...)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	assignments := make([]tsgo.Statement, 0, structType.NumFields()+1)
	assignments = append(assignments, constStatement(
		factory,
		"replacement",
		guardedForeignPayload(scaffold, elemAdapter, "Value.Set"),
	))
	for index := range structType.NumFields() {
		member, memberErr := context.Names().Member(structType.Field(index))
		if memberErr != nil {
			return nil, memberErr
		}
		assignments = append(assignments, factory.ExpressionStatement(
			factory.BinaryExpression(
				nil,
				memberAccess(factory, "instance", member),
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				memberAccess(factory, "replacement", member),
			),
		))
	}
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			elemAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{factory.Identifier("instance")},
		),
		set: factory.Block(assignments, true),
	})
	body := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Elem"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.IfStatement(
			factory.BinaryExpression(
				nil,
				factory.Identifier("instance"),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				factory.Identifier("undefined"),
			),
			factory.Block([]tsgo.Statement{
				factory.ReturnStatement(factory.Identifier("undefined")),
			}, true),
			nil,
		),
		factory.ReturnStatement(location),
	}, true)
	elem := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
	return []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "elem", elem),
	}, nil
}

// structValueProperties adds the numField and field callbacks of one
// struct: every field location reads and writes the boxed instance through
// its exact generated member spelling, with settability taken from export
// evidence.
func structValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	structType *types.Struct,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	if _, err := supportedValueStruct(structType); err != nil {
		return nil, err
	}
	provider, ok := context.ProviderScalarABI()
	if !ok {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: structType.String(),
			Reason:   "reflection value provider scalar ABI is absent",
		}
	}
	indexType, err := context.Names().ProviderPrimitive(api.PrimitiveInt64)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, indexType.Requests()...)
	numField, err := integerProperty(
		factory,
		provider,
		"numField",
		int64(structType.NumFields()),
		api.PrimitiveInt64,
	)
	if err != nil {
		return nil, err
	}
	cases := make([]tsgo.CaseOrDefaultClause, 0, structType.NumFields()+1)
	for index := range structType.NumFields() {
		field := structType.Field(index)
		fieldAdapter, adapterErr := context.Names().InterfaceAdapter(
			field.Type(),
			nil,
		)
		if adapterErr != nil {
			return nil, adapterErr
		}
		descriptor, descriptorErr := names.ReflectionValueType(
			field.Type(),
			reflectionType,
		)
		if descriptorErr != nil {
			return nil, descriptorErr
		}
		member, memberErr := context.Names().Member(field)
		if memberErr != nil {
			return nil, memberErr
		}
		scaffold.requests = append(
			scaffold.requests,
			fieldAdapter.Requests()...,
		)
		scaffold.requests = append(
			scaffold.requests,
			descriptor.Requests()...,
		)
		fieldAccess := memberAccess(factory, "instance", member)
		location := locationLiteral(scaffold, locationCallbacks{
			descriptor: descriptor,
			settable:   field.Exported(),
			get: factory.NewExpression(
				fieldAdapter.Expression(factory),
				nil,
				[]tsgo.Expression{fieldAccess},
			),
			set: factory.Block([]tsgo.Statement{
				factory.ExpressionStatement(factory.BinaryExpression(
					nil,
					fieldAccess,
					nil,
					factory.BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					guardedForeignPayload(
						scaffold,
						fieldAdapter,
						"Value.Set",
					),
				)),
			}, true),
		})
		caseLiteral, caseErr := api.IntegerLiteral(
			factory,
			provider,
			api.PrimitiveInt64,
			strconv.Itoa(index),
		)
		if caseErr != nil {
			return nil, caseErr
		}
		cases = append(cases, factory.CaseClause(
			caseLiteral,
			[]tsgo.Statement{factory.ReturnStatement(location)},
		))
	}
	cases = append(cases, factory.DefaultClause(nil, []tsgo.Statement{
		factory.ReturnStatement(runtimePanic(
			scaffold,
			"reflect: Field index out of range",
		)),
	}))
	body := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Field"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.SwitchStatement(
			factory.Identifier("index"),
			factory.CaseBlock(cases),
		),
	}, true)
	field := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			boxParameter(scaffold),
			factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("index"),
				nil,
				factory.TypeReferenceNode(
					indexType.EntityName(factory),
					nil,
				),
				nil,
			),
		},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
	return []tsgo.ObjectLiteralElementLike{
		numField,
		expressionProperty(factory, "field", field),
	}, nil
}

// pointerSliceValueProperties adds the elem callback of one pointer to a
// slice: the pointee is a runtime pointer storage cell, so the location
// reads and replaces the slice header through the cell's value member and
// header mutations stay visible to the original variable.
func pointerSliceValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	sliceAdapter, err := context.Names().InterfaceAdapter(pointee, nil)
	if err != nil {
		return nil, err
	}
	descriptor, err := names.ReflectionValueType(pointee, reflectionType)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, sliceAdapter.Requests()...)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	cellValue := memberAccess(factory, "instance", "value")
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			sliceAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{cellValue},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.BinaryExpression(
				nil,
				cellValue,
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				guardedForeignPayload(
					scaffold,
					sliceAdapter,
					"Value.Set",
				),
			)),
		}, true),
	})
	body := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Elem"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.IfStatement(
			factory.BinaryExpression(
				nil,
				factory.Identifier("instance"),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				factory.Identifier("undefined"),
			),
			factory.Block([]tsgo.Statement{
				factory.ReturnStatement(factory.Identifier("undefined")),
			}, true),
			nil,
		),
		factory.ReturnStatement(location),
	}, true)
	elem := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
	return []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "elem", elem),
	}, nil
}
