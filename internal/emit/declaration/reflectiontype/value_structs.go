package reflectiontype

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	sourceType types.Type,
	pointee types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	observation, err := pointertype.Observe(context, sourceType, false)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, observation.Requests()...)
	directClass := observation.Representation() ==
		api.PointerRepresentationDirectClass
	named, namedOK := types.Unalias(pointee).(*types.Named)
	_, structOK := types.Unalias(pointee).Underlying().(*types.Struct)
	if !directClass || !namedOK || named.Obj() == nil || !structOK {
		return []tsgo.ObjectLiteralElementLike{
			pointerZeroProperty(scaffold),
		}, nil
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
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleStructCopyField),
		nil,
		pointee,
		pointee,
		api.ValueTransferCopy,
		api.DirectExpression(guardedForeignPayload(
			scaffold,
			elemAdapter,
			"Value.Set",
		)),
	)
	if err != nil {
		return nil, err
	}
	assignReference, err := context.Names().NamedStructOperation(
		named.Origin().Obj(),
		api.NamedStructOperationAssign,
	)
	if err != nil {
		return nil, err
	}
	assignMember, err := api.NamedStructOperationMemberName(
		api.NamedStructOperationAssign,
	)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, copied.Requests()...)
	scaffold.requests = append(
		scaffold.requests,
		assignReference.Requests()...,
	)
	assignments := append([]tsgo.Statement(nil), copied.Before()...)
	assignments = append(assignments, factory.ExpressionStatement(
		factory.CallExpression(
			factory.PropertyAccessExpression(
				assignReference.Expression(factory),
				nil,
				factory.Identifier(assignMember),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			[]tsgo.Expression{
				factory.Identifier("instance"),
				copied.Value(),
			},
			tsgo.NodeFlagsNone,
		),
	))
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
		pointerZeroProperty(scaffold),
	}, nil
}

// pointerZeroProperty is the boxed nil pointer of one pointer type.
func pointerZeroProperty(
	scaffold *locationScaffold,
) tsgo.ObjectLiteralElementLike {
	factory := scaffold.factory
	return expressionProperty(factory, "zero", factory.ArrowFunction(
		nil,
		nil,
		nil,
		factory.TypeReferenceNode(
			scaffold.boxType.EntityName(factory),
			nil,
		),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(factory.NewExpression(
			scaffold.adapter.Expression(factory),
			nil,
			[]tsgo.Expression{factory.Identifier("undefined")},
		)),
	))
}

// structValueProperties adds the numField and field callbacks of one
// struct: every field location reads and writes the boxed instance through
// its exact generated member spelling, with settability taken from export
// evidence.
func structValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	structType *types.Struct,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	providerRepresented := false
	if model, defined := definedtype.Resolve(sourceType); defined {
		provider, providerErr := model.ProviderCarrier(context)
		if providerErr != nil {
			return nil, providerErr
		}
		providerRepresented = provider
	}
	if named, isNamed := types.Unalias(sourceType).(*types.Named); isNamed &&
		named.Obj() != nil {
		owned, ownedErr := names.ProviderOwnedDeclaration(named.Obj())
		if ownedErr != nil {
			return nil, ownedErr
		}
		providerRepresented = providerRepresented || owned
	}
	var cloned tsgo.ObjectLiteralElementLike
	if !providerRepresented {
		if named, isNamed := types.Unalias(sourceType).(*types.Named); isNamed && named.Obj() != nil && named.TypeParams().Len() == 0 {
			var clonedErr error
			cloned, clonedErr = structClonedProperty(
				context,
				sourceType,
				scaffold,
			)
			if clonedErr != nil {
				return nil, clonedErr
			}
		}
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
		if providerRepresented {
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
				[]tsgo.Statement{factory.ReturnStatement(runtimePanic(
					scaffold,
					"reflect: field "+field.Name()+
						" of "+structType.String()+
						" is outside the generated location model",
				))},
			))
			continue
		}
		descriptor, descriptorErr := names.ReflectionValueType(
			field.Type(),
			reflectionType,
		)
		if descriptorErr != nil {
			return nil, descriptorErr
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
			setBlock = factory.Block(
				[]tsgo.Statement{factory.ExpressionStatement(runtimePanic(
					scaffold,
					"reflect: Value.Set using unaddressable value",
				))},
				true,
			)
			if _, isInterface := types.Unalias(field.Type()).Underlying().(*types.Interface); isInterface {
				boxedField = factory.Identifier("undefined")
			} else {
				fieldAdapter, adapterErr := context.Names().InterfaceAdapter(
					field.Type(),
					nil,
				)
				if adapterErr != nil {
					return nil, adapterErr
				}
				zero, zeroErr := context.Values().Zero(
					context.WithRole(api.RoleStructZeroField),
					nil,
					field.Type(),
				)
				if zeroErr != nil {
					return nil, zeroErr
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
					scaffold,
				)
			if descriptorErr != nil {
				return nil, descriptorErr
			}
			scaffold.requests = append(
				scaffold.requests,
				fieldRequests...,
			)
		}
		location := locationLiteral(scaffold, locationCallbacks{
			descriptor: descriptor,
			settable:   settable,
			get:        boxedField,
			getBlock:   boxedFieldBlock,
			set:        setBlock,
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
	properties := []tsgo.ObjectLiteralElementLike{
		numField,
		expressionProperty(factory, "field", field),
	}
	if cloned != nil {
		properties = append(properties, cloned)
	}
	return properties, nil
}

// structClonedProperty binds the canonical class copy operation so
// Interface projections of located struct values return an exact copy,
// never an aliasing view. Structs without a named copy owner fail closed.
func structClonedProperty(
	context api.Context,
	sourceType types.Type,
	scaffold *locationScaffold,
) (tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil || named.TypeParams().Len() != 0 {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: sourceType.String(),
			Reason: "reflection value struct " + sourceType.String() +
				" has no canonical copy owner",
		}
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
	clone := factory.CallExpression(
		factory.PropertyAccessExpression(
			copyReference.Expression(factory),
			nil,
			factory.Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{boxPayload(factory)},
		tsgo.NodeFlagsNone,
	)
	return expressionProperty(factory, "cloned", factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.TypeReferenceNode(
			scaffold.boxType.EntityName(factory),
			nil,
		),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.Interface",
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{clone},
			),
		)),
	)), nil
}
