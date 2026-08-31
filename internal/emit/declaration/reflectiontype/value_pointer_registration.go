package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func pointerValueOperationsStatement(
	context api.Context,
	names api.ReflectionNames,
	operations api.NameReference,
	reflectionType *types.TypeName,
	descriptorType api.NameReference,
	sourceType types.Type,
	descriptorName string,
	pointerType *types.Pointer,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	factory := context.Factory()
	adapter, err := names.ReflectionInterfaceAdapter(sourceType)
	if err != nil {
		return nil, nil, false, err
	}
	scaffold := &locationScaffold{
		factory:        factory,
		adapter:        adapter,
		descriptorType: descriptorType,
	}
	properties := make([]tsgo.ObjectLiteralElementLike, 0, 2)
	pointee := pointerType.Elem()
	var element tsgo.Expression
	var elementErr error
	if _, isInterface := types.Unalias(pointee).Underlying().(*types.Interface); isInterface {
		element, elementErr = pointerInterfaceElementOperations(
			context,
			names,
			reflectionType,
			pointee,
			scaffold,
		)
	} else {
		element, elementErr = pointerStoredElementOperations(
			context,
			names,
			reflectionType,
			pointee,
			scaffold,
		)
	}
	if elementErr != nil {
		return nil, nil, false, elementErr
	}
	properties = append(
		properties,
		expressionProperty(factory, "element", element),
	)
	if _, basic := types.Unalias(pointee).Underlying().(*types.Basic); basic {
		newPointer, newRequests, supported, newErr :=
			pointerNewOperation(context, pointee, scaffold)
		if newErr != nil {
			return nil, nil, false, newErr
		}
		scaffold.requests = append(scaffold.requests, newRequests...)
		if supported {
			properties = append(
				properties,
				expressionProperty(factory, "newPointer", newPointer),
			)
		}
	}
	statement := factory.ExpressionStatement(factory.CallExpression(
		factory.PropertyAccessExpression(
			operations.Expression(factory),
			nil,
			factory.Identifier("$registerPointer"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{
			factory.Identifier(descriptorName),
			factory.ArrowFunction(
				nil,
				nil,
				nil,
				nil,
				factory.EqualsGreaterThanToken(),
				factory.ParenthesizedExpression(adapter.Expression(factory)),
			),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{untypedParameter(factory, "elements")},
				nil,
				factory.EqualsGreaterThanToken(),
				factory.ParenthesizedExpression(
					factory.ObjectLiteralExpression(properties, true),
				),
			),
		},
		tsgo.NodeFlagsNone,
	))
	return statement, api.CombineRequests(
		operations.Requests(),
		adapter.Requests(),
		scaffold.requests,
	), true, nil
}

func pointerStoredElementOperations(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	scaffold *locationScaffold,
) (tsgo.Expression, error) {
	factory := scaffold.factory
	elemAdapter, err := context.Names().InterfaceAdapter(pointee, nil)
	if err != nil {
		return nil, err
	}
	descriptor, err := names.ReflectionValueType(pointee, reflectionType)
	if err != nil {
		return nil, err
	}
	loadPointer, err := context.Names().TsonicCore(tsoniccore.SymbolLoadPointer)
	if err != nil {
		return nil, err
	}
	storePointer, err := context.Names().TsonicCore(tsoniccore.SymbolStorePointer)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, elemAdapter.Requests()...)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	scaffold.requests = append(scaffold.requests, loadPointer.Requests()...)
	scaffold.requests = append(scaffold.requests, storePointer.Requests()...)
	value, err := context.Values().Transfer(
		context.WithRole(api.RoleStructCopyField),
		nil,
		pointee,
		pointee,
		api.ValueTransferCopy,
		api.DirectExpression(factory.Identifier("value")),
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, value.Requests()...)
	setStatements := append([]tsgo.Statement(nil), value.Before()...)
	setStatements = append(setStatements, factory.ExpressionStatement(
		factory.CallExpression(
			storePointer.Expression(factory),
			nil,
			nil,
			[]tsgo.Expression{
				factory.Identifier("pointer"),
				value.Value(),
			},
			tsgo.NodeFlagsNone,
		),
	))
	return pointerElementBuilderCall(factory, "value", []tsgo.Expression{
		arrow(factory, scaffold.descriptorType, descriptor.Expression(factory)),
		factory.ArrowFunction(
			nil,
			nil,
			nil,
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(elemAdapter.Expression(factory)),
		),
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "pointer")},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(factory.CallExpression(
				loadPointer.Expression(factory),
				nil,
				nil,
				[]tsgo.Expression{factory.Identifier("pointer")},
				tsgo.NodeFlagsNone,
			)),
		),
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "pointer"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.Block(setStatements, true),
		),
	}), nil
}

func pointerInterfaceElementOperations(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	scaffold *locationScaffold,
) (tsgo.Expression, error) {
	factory := scaffold.factory
	descriptor, err := names.ReflectionValueType(pointee, reflectionType)
	if err != nil {
		return nil, err
	}
	loadPointer, err := context.Names().TsonicCore(tsoniccore.SymbolLoadPointer)
	if err != nil {
		return nil, err
	}
	storePointer, err := context.Names().TsonicCore(tsoniccore.SymbolStorePointer)
	if err != nil {
		return nil, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, err
	}
	scaffold.panicRef = panicReference
	admitted, admissionRequests, err := admittedInterfaceValue(
		context,
		pointee,
		factory.Identifier("value"),
		"Value.Set",
		scaffold,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = api.CombineRequests(
		scaffold.requests,
		panicReference.Requests(),
		descriptor.Requests(),
		loadPointer.Requests(),
		storePointer.Requests(),
		admissionRequests,
	)
	set := factory.Block([]tsgo.Statement{factory.ExpressionStatement(
		factory.CallExpression(
			storePointer.Expression(factory),
			nil,
			nil,
			[]tsgo.Expression{
				factory.Identifier("pointer"),
				factory.Identifier("value"),
			},
			tsgo.NodeFlagsNone,
		),
	)}, true)
	return pointerElementBuilderCall(factory, "interfaceValue", []tsgo.Expression{
		arrow(factory, scaffold.descriptorType, descriptor.Expression(factory)),
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "value")},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(admitted),
		),
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "pointer")},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(factory.CallExpression(
				loadPointer.Expression(factory),
				nil,
				nil,
				[]tsgo.Expression{factory.Identifier("pointer")},
				tsgo.NodeFlagsNone,
			)),
		),
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "pointer"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			set,
		),
	}), nil
}

func pointerElementBuilderCall(
	factory tsgo.Factory,
	member string,
	arguments []tsgo.Expression,
) tsgo.Expression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("elements"),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
}

func pointerNewOperation(
	context api.Context,
	pointee types.Type,
	scaffold *locationScaffold,
) (tsgo.Expression, []api.RootRequest, bool, error) {
	logicalZero, err := context.Values().Zero(context, nil, pointee)
	if err != nil {
		return nil, nil, false, err
	}
	if logicalZero.Value() == nil || len(logicalZero.Before()) != 0 {
		return nil, nil, false, &api.GeneratedArtifactShapeError{
			Artifact: pointee.String(),
			Reason:   "reflection pointer zero is not a direct value",
		}
	}
	allocatePointer, err := context.Names().TsonicCore(
		tsoniccore.SymbolAllocatePointer,
	)
	if err != nil {
		return nil, nil, false, err
	}
	factory := scaffold.factory
	return factory.ArrowFunction(
			nil,
			nil,
			nil,
			nil,
			factory.EqualsGreaterThanToken(),
			factory.CallExpression(
				allocatePointer.Expression(factory),
				nil,
				nil,
				[]tsgo.Expression{logicalZero.Value()},
				tsgo.NodeFlagsNone,
			),
		), api.CombineRequests(
			logicalZero.Requests(),
			allocatePointer.Requests(),
		), true, nil
}
