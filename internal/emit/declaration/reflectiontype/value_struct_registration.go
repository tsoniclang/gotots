package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func structValueOperationsStatement(
	context api.Context,
	names api.ReflectionNames,
	operations api.NameReference,
	reflectionType *types.TypeName,
	descriptorType api.NameReference,
	sourceType types.Type,
	descriptorName string,
	structType *types.Struct,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	factory := context.Factory()
	adapter, err := names.ReflectionInterfaceAdapter(sourceType)
	if err != nil {
		return nil, nil, false, err
	}
	providerRepresented, err := structUsesProviderRepresentation(
		context,
		names,
		sourceType,
	)
	if err != nil {
		return nil, nil, false, err
	}
	if providerRepresented {
		messages := make([]tsgo.Expression, 0, structType.NumFields())
		for index := range structType.NumFields() {
			field := structType.Field(index)
			messages = append(messages, factory.StringLiteral(
				"reflect: field "+field.Name()+" of "+structType.String()+
					" is outside the generated location model",
				tsgo.TokenFlagsNone,
			))
		}
		return structRegistrationCall(
				factory,
				operations,
				"$registerOpaqueStruct",
				descriptorName,
				adapter,
				nil,
				factory.ArrayLiteralExpression(messages, true),
				nil,
			), api.CombineRequests(
				operations.Requests(),
				adapter.Requests(),
			), true, nil
	}
	scaffold := &locationScaffold{
		factory:        factory,
		adapter:        adapter,
		descriptorType: descriptorType,
	}
	accesses, storageResolver, storageRequests, propertyFacts, err :=
		reflectedStructFieldAccesses(context, sourceType, structType)
	if err != nil {
		return nil, nil, false, err
	}
	scaffold.requests = append(scaffold.requests, storageRequests...)
	fields := make([]tsgo.Expression, 0, structType.NumFields())
	for index := range structType.NumFields() {
		field := structType.Field(index)
		descriptor, descriptorErr := names.ReflectionValueType(
			field.Type(),
			reflectionType,
		)
		if descriptorErr != nil {
			return nil, nil, false, descriptorErr
		}
		scaffold.requests = append(
			scaffold.requests,
			descriptor.Requests()...,
		)
		settable := field.Exported()
		_, interfaceField := types.Unalias(field.Type()).Underlying().(*types.Interface)
		var getBody tsgo.ConciseBody
		var setBlock tsgo.Block
		var address tsgo.Expression
		var fieldAdapter api.NameReference
		var admit tsgo.Expression
		if field.Name() == "_" {
			settable = false
			if interfaceField {
				getBody = factory.ParenthesizedExpression(
					factory.Identifier("undefined"),
				)
			} else {
				var adapterErr error
				fieldAdapter, adapterErr = context.Names().InterfaceAdapter(
					field.Type(),
					nil,
				)
				if adapterErr != nil {
					return nil, nil, false, adapterErr
				}
				zero, zeroErr := context.Values().Zero(
					context.WithRole(api.RoleStructZeroField),
					nil,
					field.Type(),
				)
				if zeroErr != nil {
					return nil, nil, false, zeroErr
				}
				scaffold.requests = append(
					scaffold.requests,
					fieldAdapter.Requests()...,
				)
				scaffold.requests = append(
					scaffold.requests,
					zero.Requests()...,
				)
				if len(zero.Before()) == 0 {
					getBody = factory.ParenthesizedExpression(zero.Value())
				} else {
					statements := append(
						[]tsgo.Statement(nil),
						zero.Before()...,
					)
					statements = append(
						statements,
						factory.ReturnStatement(zero.Value()),
					)
					getBody = factory.Block(statements, true)
				}
			}
		} else {
			access := accesses[index]
			if access == nil {
				return nil, nil, false, &api.GeneratedArtifactShapeError{
					Artifact: field.String(),
					Reason:   "reflected struct field access is absent",
				}
			}
			if !interfaceField && propertyFacts {
				fieldAdapter, descriptorErr = context.Names().InterfaceAdapter(
					field.Type(),
					nil,
				)
				if descriptorErr != nil {
					return nil, nil, false, descriptorErr
				}
				scaffold.requests = append(
					scaffold.requests,
					fieldAdapter.Requests()...,
				)
				property, selected, propertyErr := structValuePropertyFact(
					context,
					names,
					field,
					access,
					descriptor,
					fieldAdapter,
					settable,
					scaffold,
				)
				if propertyErr != nil {
					return nil, nil, false, propertyErr
				}
				if selected {
					fields = append(fields, property)
					continue
				}
			}
			target := access.target
			var fieldRequests []api.RootRequest
			getBody, setBlock, fieldRequests, descriptorErr =
				storageStructFieldCallbacks(
					context,
					field,
					target,
					settable,
				)
			if descriptorErr != nil {
				return nil, nil, false, descriptorErr
			}
			fieldAddress, addressErr := reflectedStoreTargetAddress(
				context,
				names,
				field.Type(),
				target,
			)
			if addressErr != nil {
				return nil, nil, false, addressErr
			}
			address = addressCallback(
				factory,
				[]tsgo.ParameterDeclaration{
					untypedParameter(factory, "instance"),
				},
				fieldAddress,
			)
			scaffold.requests = append(
				scaffold.requests,
				fieldRequests...,
			)
			scaffold.requests = append(
				scaffold.requests,
				fieldAddress.Requests()...,
			)
			if interfaceField && settable {
				panicReference, panicErr := context.Names().Runtime(
					api.RuntimePanic,
					api.ImportPhaseValue,
				)
				if panicErr != nil {
					return nil, nil, false, panicErr
				}
				scaffold.panicRef = panicReference
				admitted, admittedRequests, admittedErr := admittedInterfaceValue(
					context,
					field.Type(),
					factory.Identifier("value"),
					"Value.Set",
					scaffold,
				)
				if admittedErr != nil {
					return nil, nil, false, admittedErr
				}
				admit = factory.ArrowFunction(
					nil,
					nil,
					[]tsgo.ParameterDeclaration{untypedParameter(factory, "value")},
					nil,
					factory.EqualsGreaterThanToken(),
					factory.ParenthesizedExpression(admitted),
				)
				scaffold.requests = append(
					scaffold.requests,
					panicReference.Requests()...,
				)
				scaffold.requests = append(
					scaffold.requests,
					admittedRequests...,
				)
			}
			if !interfaceField {
				var adapterErr error
				fieldAdapter, adapterErr = context.Names().InterfaceAdapter(
					field.Type(),
					nil,
				)
				if adapterErr != nil {
					return nil, nil, false, adapterErr
				}
				scaffold.requests = append(
					scaffold.requests,
					fieldAdapter.Requests()...,
				)
			}
		}
		if interfaceField {
			fields = append(fields, structInterfaceFieldFact(
				scaffold,
				descriptor,
				getBody,
				setBlock,
				admit,
				address,
			))
		} else {
			fields = append(fields, structValueFieldFact(
				scaffold,
				descriptor,
				fieldAdapter,
				getBody,
				setBlock,
				address,
			))
		}
	}
	clone, err := structCloneCallback(context, sourceType, scaffold)
	if err != nil {
		return nil, nil, false, err
	}
	return structRegistrationCall(
			factory,
			operations,
			"$registerStruct",
			descriptorName,
			adapter,
			storageResolver,
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{untypedParameter(factory, "fields")},
				nil,
				factory.EqualsGreaterThanToken(),
				factory.ParenthesizedExpression(
					factory.ArrayLiteralExpression(fields, true),
				),
			),
			clone,
		), api.CombineRequests(
			operations.Requests(),
			adapter.Requests(),
			scaffold.requests,
		), true, nil
}

func structUsesProviderRepresentation(
	context api.Context,
	names api.ReflectionNames,
	sourceType types.Type,
) (bool, error) {
	providerRepresented := false
	if model, defined := definedtype.Resolve(sourceType); defined {
		provider, err := model.ProviderCarrier(context)
		if err != nil {
			return false, err
		}
		providerRepresented = provider
	}
	if named, isNamed := types.Unalias(sourceType).(*types.Named); isNamed &&
		named.Obj() != nil {
		owned, err := names.ProviderOwnedDeclaration(named.Obj())
		if err != nil {
			return false, err
		}
		providerRepresented = providerRepresented || owned
	}
	return providerRepresented, nil
}

func structCloneCallback(
	context api.Context,
	sourceType types.Type,
	scaffold *locationScaffold,
) (tsgo.Expression, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil || named.TypeParams().Len() != 0 {
		return nil, nil
	}
	copyReference, err := context.Names().NamedStructOperation(
		named.Origin().Obj(),
		api.NamedStructOperationCopy,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, copyReference.Requests()...)
	memberName, err := api.NamedStructOperationMemberName(
		api.NamedStructOperationCopy,
	)
	if err != nil {
		return nil, err
	}
	factory := scaffold.factory
	return factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{untypedParameter(factory, "value")},
		nil,
		factory.EqualsGreaterThanToken(),
		factory.CallExpression(
			factory.PropertyAccessExpression(
				copyReference.Expression(factory),
				nil,
				factory.Identifier(memberName),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{factory.Identifier("value")},
			tsgo.NodeFlagsNone,
		),
	), nil
}

func structValueFieldFact(
	scaffold *locationScaffold,
	descriptor api.NameReference,
	adapter api.NameReference,
	get tsgo.ConciseBody,
	set tsgo.Block,
	address tsgo.Expression,
) tsgo.Expression {
	factory := scaffold.factory
	member := "readonlyValue"
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
		factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "instance")},
			nil,
			factory.EqualsGreaterThanToken(),
			get,
		),
	}
	if set != nil {
		member = "value"
		arguments = append(arguments, factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "instance"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			set,
		))
	}
	if address != nil {
		arguments = append(arguments, address)
	}
	return structFieldBuilderCall(factory, member, arguments)
}

func structInterfaceFieldFact(
	scaffold *locationScaffold,
	descriptor api.NameReference,
	get tsgo.ConciseBody,
	set tsgo.Block,
	admit tsgo.Expression,
	address tsgo.Expression,
) tsgo.Expression {
	factory := scaffold.factory
	member := "readonlyInterface"
	arguments := []tsgo.Expression{
		arrow(factory, scaffold.descriptorType, descriptor.Expression(factory)),
	}
	if set != nil {
		member = "interfaceValue"
		arguments = append(arguments, admit)
	}
	arguments = append(arguments, factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{untypedParameter(factory, "instance")},
		nil,
		factory.EqualsGreaterThanToken(),
		get,
	))
	if set != nil {
		arguments = append(arguments, factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "instance"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			set,
		))
	}
	if address != nil {
		arguments = append(arguments, address)
	}
	return structFieldBuilderCall(factory, member, arguments)
}

func structFieldBuilderCall(
	factory tsgo.Factory,
	member string,
	arguments []tsgo.Expression,
) tsgo.Expression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier("fields"),
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

func structRegistrationCall(
	factory tsgo.Factory,
	operations api.NameReference,
	member string,
	descriptorName string,
	adapter api.NameReference,
	storage tsgo.Expression,
	fields tsgo.Expression,
	clone tsgo.Expression,
) tsgo.Statement {
	arguments := []tsgo.Expression{
		factory.Identifier(descriptorName),
		factory.ArrowFunction(
			nil,
			nil,
			nil,
			nil,
			factory.EqualsGreaterThanToken(),
			factory.ParenthesizedExpression(adapter.Expression(factory)),
		),
	}
	if storage != nil {
		arguments = append(arguments, storage)
	}
	arguments = append(arguments, fields)
	if clone != nil {
		arguments = append(arguments, clone)
	}
	return factory.ExpressionStatement(factory.CallExpression(
		factory.PropertyAccessExpression(
			operations.Expression(factory),
			nil,
			factory.Identifier(member),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	))
}

func untypedParameter(
	factory tsgo.Factory,
	name string,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		nil,
		nil,
	)
}
