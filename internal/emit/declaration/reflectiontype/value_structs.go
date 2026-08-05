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
	structType, structOK := types.Unalias(pointee).
		Underlying().(*types.Struct)
	if !directClass || !structOK || !locationModelStruct(structType) {
		// The pointee is outside the location model: the pointer keeps
		// its exact nil and zero evidence while elem stays a loud typed
		// boundary through operation absence.
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
	fullySupported := !providerRepresented &&
		locationModelStruct(structType)
	var cloned tsgo.ObjectLiteralElementLike
	if fullySupported {
		if _, isNamed := types.Unalias(sourceType).(*types.Named); isNamed {
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
		if providerRepresented || !locationModelField(field.Type()) {
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
	properties := []tsgo.ObjectLiteralElementLike{
		numField,
		expressionProperty(factory, "field", field),
	}
	if cloned != nil {
		properties = append(properties, cloned)
	}
	return properties, nil
}

// locationModelField reports whether one struct field participates in the
// generated location model. Unsupported field kinds keep exact loud typed
// boundaries at their Field cases rather than blocking the whole type.
func locationModelField(fieldType types.Type) bool {
	basic, ok := types.Unalias(fieldType).(*types.Basic)
	return ok && basic.Info()&(types.IsBoolean|types.IsString|
		types.IsInteger|types.IsFloat) != 0
}

// locationModelStruct reports whether every field of one struct is inside
// the location model, which gates whole-value operations such as the
// interface clone.
func locationModelStruct(structType *types.Struct) bool {
	for index := range structType.NumFields() {
		if !locationModelField(structType.Field(index).Type()) {
			return false
		}
	}
	return true
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

// pointerCellValueProperties adds the elem callback of one pointer whose
// pointee is represented through a runtime pointer storage cell (slices,
// basic scalars, and maps): the location reads and replaces the pointee
// through the cell's value member, so mutations stay visible to the
// original variable.
func pointerCellValueProperties(
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
	// Scalar pointer cells hold the raw carrier storage: reads wrap the
	// raw value into the pointee's branded representation for boxing, and
	// writes project the boxed payload back to raw storage. Container
	// cells hold the represented value directly.
	cellRead := cellValue
	cellWrite := guardedForeignPayload(
		scaffold,
		sliceAdapter,
		"Value.Set",
	)
	if _, basicPointee := types.Unalias(pointee).
		Underlying().(*types.Basic); basicPointee {
		wrapped, readRequests, readErr := constructedScalarValue(
			context,
			pointee,
			cellValue,
		)
		if readErr != nil {
			return nil, readErr
		}
		scaffold.requests = append(scaffold.requests, readRequests...)
		cellRead = wrapped
		projected, writeRequests, writeErr := projectedScalarPayload(
			context,
			pointee,
			cellWrite,
		)
		if writeErr != nil {
			return nil, writeErr
		}
		scaffold.requests = append(scaffold.requests, writeRequests...)
		cellWrite = projected
	}
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			sliceAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{cellRead},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.BinaryExpression(
				nil,
				cellValue,
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				cellWrite,
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
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "elem", elem),
		pointerZeroProperty(scaffold),
	}
	if pointeeBasic, basicOK := types.Unalias(pointee).
		Underlying().(*types.Basic); basicOK {
		pointeeZero, zeroErr := scalarZeroExpression(
			context,
			factory,
			pointeeBasic,
		)
		if zeroErr != nil {
			return nil, zeroErr
		}
		if pointeeZero != nil {
			runtimePointer, pointerErr := context.Names().Runtime(
				api.RuntimePointer,
				api.ImportPhaseValue,
			)
			if pointerErr != nil {
				return nil, pointerErr
			}
			scaffold.requests = append(
				scaffold.requests,
				runtimePointer.Requests()...,
			)
			properties = append(properties, expressionProperty(
				factory,
				"newPointer",
				factory.ArrowFunction(
					nil,
					nil,
					nil,
					factory.TypeReferenceNode(
						scaffold.boxType.EntityName(factory),
						nil,
					),
					factory.EqualsGreaterThanToken(),
					factory.ParenthesizedExpression(
						factory.NewExpression(
							scaffold.adapter.Expression(factory),
							nil,
							[]tsgo.Expression{factory.CallExpression(
								factory.PropertyAccessExpression(
									runtimePointer.Expression(factory),
									nil,
									factory.Identifier("cell"),
									tsgo.NodeFlagsNone,
								),
								nil,
								nil,
								[]tsgo.Expression{pointeeZero},
								tsgo.NodeFlagsNone,
							)},
						),
					),
				),
			))
		}
	}
	return properties, nil
}
