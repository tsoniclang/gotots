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
	properties := make([]tsgo.ObjectLiteralElementLike, 0, 3)
	pointee := pointerType.Elem()
	switch types.Unalias(pointee).Underlying().(type) {
	case *types.Struct:
		properties = append(properties, booleanProperty(factory, "zero", true))
		named, namedOK := types.Unalias(pointee).(*types.Named)
		if namedOK && named.Obj() != nil {
			if err := attachPointerPanic(context, scaffold); err != nil {
				return nil, nil, false, err
			}
			element, elementErr := pointerStoredElementOperations(
				context,
				names,
				reflectionType,
				pointee,
				true,
				scaffold,
			)
			if elementErr != nil {
				return nil, nil, false, elementErr
			}
			properties = append(
				properties,
				expressionProperty(factory, "element", element),
			)
		}
	case *types.Slice, *types.Basic, *types.Map:
		if err := attachPointerPanic(context, scaffold); err != nil {
			return nil, nil, false, err
		}
		properties = append(properties, booleanProperty(factory, "zero", true))
		element, elementErr := pointerStoredElementOperations(
			context,
			names,
			reflectionType,
			pointee,
			false,
			scaffold,
		)
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
			factory.ObjectLiteralExpression(properties, true),
		},
		tsgo.NodeFlagsNone,
	))
	return statement, api.CombineRequests(
		operations.Requests(),
		adapter.Requests(),
		scaffold.requests,
	), true, nil
}

func attachPointerPanic(
	context api.Context,
	scaffold *locationScaffold,
) error {
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return err
	}
	scaffold.panicRef = panicReference
	scaffold.requests = append(
		scaffold.requests,
		panicReference.Requests()...,
	)
	return nil
}

func pointerStoredElementOperations(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	copyValue bool,
	scaffold *locationScaffold,
) (tsgo.ObjectLiteralExpression, error) {
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
	value := api.DirectExpression(guardedForeignPayload(
		scaffold,
		elemAdapter,
		"Value.Set",
	))
	if copyValue {
		value, err = context.Values().Transfer(
			context.WithRole(api.RoleStructCopyField),
			nil,
			pointee,
			pointee,
			api.ValueTransferCopy,
			value,
		)
		if err != nil {
			return nil, err
		}
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
	return factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "type", arrow(
			factory,
			scaffold.descriptorType,
			descriptor.Expression(factory),
		)),
		expressionProperty(factory, "get", factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "pointer")},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(factory.NewExpression(
				elemAdapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.CallExpression(
					loadPointer.Expression(factory),
					nil,
					nil,
					[]tsgo.Expression{factory.Identifier("pointer")},
					tsgo.NodeFlagsNone,
				)},
			)),
		)),
		expressionProperty(factory, "set", factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "pointer"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			factory.Block(setStatements, true),
		)),
	}, true), nil
}

func pointerNewOperation(
	context api.Context,
	pointee types.Type,
	scaffold *locationScaffold,
) (tsgo.Expression, []api.RootRequest, bool, error) {
	basic, ok := types.Unalias(pointee).Underlying().(*types.Basic)
	if !ok {
		return nil, nil, false, nil
	}
	supportedZero, err := scalarZeroExpression(context, scaffold.factory, basic)
	if err != nil || supportedZero == nil {
		return nil, nil, false, err
	}
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
