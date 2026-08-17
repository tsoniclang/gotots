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
	targetType api.NameReference,
	sourceType types.Type,
	descriptorName string,
	structType *types.Struct,
) (tsgo.Statement, []api.RootRequest, bool, error) {
	factory := context.Factory()
	adapter, err := context.Names().InterfaceAdapter(sourceType, nil)
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
				factory.ArrayLiteralExpression(messages, true),
				nil,
			), api.CombineRequests(
				operations.Requests(),
				adapter.Requests(),
			), true, nil
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, false, err
	}
	scaffold := &locationScaffold{
		factory:    factory,
		adapter:    adapter,
		panicRef:   panicReference,
		targetType: targetType,
	}
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
		var boxedField tsgo.Expression
		var boxedFieldBlock tsgo.Block
		var setBlock tsgo.Block
		if field.Name() == "_" {
			settable = false
			setBlock = unaddressableFieldSetter(scaffold)
			if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); isInterface {
				boxedField = factory.Identifier("undefined")
			} else {
				fieldAdapter, adapterErr := context.Names().InterfaceAdapter(
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
				boxedZero := factory.NewExpression(
					fieldAdapter.Expression(factory),
					nil,
					[]tsgo.Expression{zero.Value()},
				)
				if len(zero.Before()) == 0 {
					boxedField = boxedZero
				} else {
					statements := append(
						[]tsgo.Statement(nil),
						zero.Before()...,
					)
					statements = append(
						statements,
						factory.ReturnStatement(boxedZero),
					)
					boxedFieldBlock = factory.Block(statements, true)
				}
			}
		} else {
			var fieldRequests []api.RootRequest
			boxedField, boxedFieldBlock, setBlock, fieldRequests, descriptorErr =
				nonBlankStructFieldCallbacks(
					context,
					sourceType,
					field,
					settable,
					scaffold,
				)
			if descriptorErr != nil {
				return nil, nil, false, descriptorErr
			}
			scaffold.requests = append(
				scaffold.requests,
				fieldRequests...,
			)
		}
		fields = append(fields, structFieldOperations(
			scaffold,
			descriptor,
			settable,
			boxedField,
			boxedFieldBlock,
			setBlock,
		))
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
			factory.ArrayLiteralExpression(fields, true),
			clone,
		), api.CombineRequests(
			operations.Requests(),
			adapter.Requests(),
			panicReference.Requests(),
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

func structFieldOperations(
	scaffold *locationScaffold,
	descriptor api.NameReference,
	settable bool,
	get tsgo.Expression,
	getBlock tsgo.Block,
	set tsgo.Block,
) tsgo.ObjectLiteralExpression {
	factory := scaffold.factory
	var getBody tsgo.ConciseBody
	if getBlock != nil {
		getBody = getBlock
	} else {
		getBody = factory.ParenthesizedExpression(get)
	}
	return factory.ObjectLiteralExpression([]tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "type", arrow(
			factory,
			scaffold.targetType,
			descriptor.Expression(factory),
		)),
		booleanProperty(factory, "settable", settable),
		expressionProperty(factory, "get", factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{untypedParameter(factory, "instance")},
			nil,
			factory.EqualsGreaterThanToken(),
			getBody,
		)),
		expressionProperty(factory, "set", factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{
				untypedParameter(factory, "instance"),
				untypedParameter(factory, "value"),
			},
			nil,
			factory.EqualsGreaterThanToken(),
			set,
		)),
	}, true)
}

func structRegistrationCall(
	factory tsgo.Factory,
	operations api.NameReference,
	member string,
	descriptorName string,
	adapter api.NameReference,
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
		fields,
	}
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
