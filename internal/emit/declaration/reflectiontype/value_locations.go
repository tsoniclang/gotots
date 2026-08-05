package reflectiontype

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// locationScaffold carries the shared references every generated location
// callback of one walked type needs: the exact adapter guard, the runtime
// box type, the panic owner, and the descriptor thunk return type.
type locationScaffold struct {
	factory    tsgo.Factory
	adapter    api.NameReference
	boxType    api.NameReference
	panicRef   api.NameReference
	targetType api.NameReference
	requests   []api.RootRequest
}

// extendedValueProperties derives the location-model callbacks admitted by
// one type's kind beyond the scalar projections: zero evidence and string
// boxing for basic scalars, aliasing element navigation for pointers to
// directly represented structs, and field navigation for structs. A struct
// participating in the value model with a field kind outside the supported
// scalar set fails at emission, never silently at runtime.
func extendedValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		return basicValueProperties(context, scaffold, selected)
	case *types.Pointer:
		if _, structPointee := types.Unalias(selected.Elem()).
			Underlying().(*types.Struct); !structPointee {
			return nil, nil
		}
		return pointerValueProperties(
			context,
			names,
			reflectionType,
			selected.Elem(),
			scaffold,
		)
	case *types.Struct:
		return structValueProperties(
			context,
			names,
			reflectionType,
			selected,
			scaffold,
		)
	default:
		return nil, nil
	}
}

// basicValueProperties adds the zero-evidence callback for every basic
// scalar and the canonical boxing callback for strings.
func basicValueProperties(
	context api.Context,
	scaffold *locationScaffold,
	basic *types.Basic,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	zero, err := scalarZeroExpression(context, factory, basic)
	if err != nil || zero == nil {
		return nil, err
	}
	isZero := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindBooleanKeyword),
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(guardedProjection(
			scaffold,
			"Value.IsZero",
			factory.BinaryExpression(
				nil,
				boxPayload(factory),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				zero,
			),
		)),
	)
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "isZero", isZero),
	}
	if basic.Info()&types.IsString != 0 {
		parameterType, typeErr := context.Names().ProviderPrimitive(
			api.PrimitiveString,
		)
		if typeErr != nil {
			return nil, typeErr
		}
		scaffold.requests = append(
			scaffold.requests,
			parameterType.Requests()...,
		)
		boxString := factory.ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
				nil,
				nil,
				factory.Identifier("value"),
				nil,
				factory.TypeReferenceNode(
					parameterType.EntityName(factory),
					nil,
				),
				nil,
			)},
			factory.TypeReferenceNode(
				scaffold.boxType.EntityName(factory),
				nil,
			),
			factory.EqualsGreaterThanToken(),
			factory.NewExpression(
				scaffold.adapter.Expression(factory),
				nil,
				[]tsgo.Expression{factory.Identifier("value")},
			),
		)
		properties = append(
			properties,
			expressionProperty(factory, "boxString", boxString),
		)
	}
	return properties, nil
}

// scalarZeroExpression is the exact target zero literal of one basic
// scalar under the product scalar representation. Unsupported basic kinds
// yield no zero evidence rather than an approximate literal.
func scalarZeroExpression(
	context api.Context,
	factory tsgo.Factory,
	basic *types.Basic,
) (tsgo.Expression, error) {
	info := basic.Info()
	switch {
	case info&types.IsBoolean != 0:
		return factory.FalseLiteral(), nil
	case info&types.IsString != 0:
		return factory.StringLiteral("", tsgo.TokenFlagsNone), nil
	case info&types.IsFloat != 0:
		return factory.NumericLiteral("0", tsgo.TokenFlagsNone), nil
	case info&types.IsInteger != 0:
		alias, ok := basictype.PrimitiveAlias(context.TypesSizes(), basic)
		if !ok {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: basic.String(),
				Reason:   "reflection value scalar has no primitive alias",
			}
		}
		return api.IntegerLiteral(factory, context.ScalarABI(), alias, "0")
	default:
		return nil, nil
	}
}

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

// supportedValueStruct admits one struct into the value location model
// exactly when every field is a plain basic scalar. Aggregate field kinds
// join with their own construct slices; until then the demand fails closed
// at emission.
func supportedValueStruct(structLike types.Type) (*types.Struct, error) {
	structType, ok := types.Unalias(structLike).Underlying().(*types.Struct)
	if !ok {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: structLike.String(),
			Reason:   "reflection value location target is not a struct",
		}
	}
	for index := range structType.NumFields() {
		fieldType := structType.Field(index).Type()
		basic, basicOK := types.Unalias(fieldType).(*types.Basic)
		if !basicOK || basic.Info()&(types.IsBoolean|types.IsString|
			types.IsInteger|types.IsFloat) == 0 {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: structType.String(),
				Reason: "reflection value struct field type " +
					fieldType.String() +
					" is outside the supported scalar location model",
			}
		}
	}
	return structType, nil
}

// locationCallbacks is the exact content of one generated location
// literal: the descriptor thunk target, generated settability evidence,
// the boxing read expression, and the write body.
type locationCallbacks struct {
	descriptor api.NameReference
	settable   bool
	get        tsgo.Expression
	set        tsgo.Block
}

func locationLiteral(
	scaffold *locationScaffold,
	callbacks locationCallbacks,
) tsgo.ObjectLiteralExpression {
	factory := scaffold.factory
	return factory.ObjectLiteralExpression(
		[]tsgo.ObjectLiteralElementLike{
			expressionProperty(factory, "type", arrow(
				factory,
				scaffold.targetType,
				callbacks.descriptor.Expression(factory),
			)),
			booleanProperty(factory, "settable", callbacks.settable),
			expressionProperty(factory, "get", factory.ArrowFunction(
				nil,
				nil,
				nil,
				factory.TypeReferenceNode(
					scaffold.boxType.EntityName(factory),
					nil,
				),
				factory.EqualsGreaterThanToken(),
				factory.ParenthesizedExpression(callbacks.get),
			)),
			expressionProperty(factory, "set", factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("value"),
					nil,
					factory.TypeReferenceNode(
						scaffold.boxType.EntityName(factory),
						nil,
					),
					nil,
				)},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
				factory.EqualsGreaterThanToken(),
				callbacks.set,
			)),
		},
		true,
	)
}

// guardedProjection wraps one projection of the walked type's payload in
// the exact generated adapter guard, panicking on a foreign box.
func guardedProjection(
	scaffold *locationScaffold,
	operation string,
	projection tsgo.Expression,
) tsgo.Expression {
	return scaffold.factory.ConditionalExpression(
		adapterGuard(scaffold.factory, scaffold.adapter, "box"),
		scaffold.factory.QuestionToken(),
		projection,
		scaffold.factory.ColonToken(),
		runtimePanic(
			scaffold,
			"reflect: "+operation+" received a foreign interface box",
		),
	)
}

// guardedForeignPayload projects the payload of one written box through
// the target member type's exact adapter guard.
func guardedForeignPayload(
	scaffold *locationScaffold,
	adapter api.NameReference,
	operation string,
) tsgo.Expression {
	factory := scaffold.factory
	return factory.ConditionalExpression(
		adapterGuard(factory, adapter, "value"),
		factory.QuestionToken(),
		memberAccess(factory, "value", "$go$value"),
		factory.ColonToken(),
		runtimePanic(
			scaffold,
			"reflect: "+operation+" received a foreign interface box",
		),
	)
}

func foreignBoxGuardStatement(
	scaffold *locationScaffold,
	operation string,
) tsgo.Statement {
	factory := scaffold.factory
	return factory.IfStatement(
		factory.PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			adapterGuard(factory, scaffold.adapter, "box"),
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(runtimePanic(
				scaffold,
				"reflect: "+operation+
					" received a foreign interface box",
			)),
		}, true),
		nil,
	)
}

func runtimePanic(
	scaffold *locationScaffold,
	message string,
) tsgo.Expression {
	factory := scaffold.factory
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			scaffold.panicRef.Expression(factory),
			nil,
			factory.Identifier("raiseRuntime"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.StringLiteral(
			message,
			tsgo.TokenFlagsNone,
		)},
		tsgo.NodeFlagsNone,
	)
}

func adapterGuard(
	factory tsgo.Factory,
	adapter api.NameReference,
	operand string,
) tsgo.Expression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			adapter.Expression(factory),
			nil,
			factory.Identifier("$is"),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier(operand)},
		tsgo.NodeFlagsNone,
	)
}

func boxPayload(factory tsgo.Factory) tsgo.Expression {
	return memberAccess(factory, "box", "$go$value")
}

func memberAccess(
	factory tsgo.Factory,
	target string,
	member string,
) tsgo.Expression {
	return factory.PropertyAccessExpression(
		factory.Identifier(target),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
}

func boxParameter(scaffold *locationScaffold) tsgo.ParameterDeclaration {
	return scaffold.factory.ParameterDeclaration(
		nil,
		nil,
		scaffold.factory.Identifier("box"),
		nil,
		scaffold.factory.TypeReferenceNode(
			scaffold.boxType.EntityName(scaffold.factory),
			nil,
		),
		nil,
	)
}

func constStatement(
	factory tsgo.Factory,
	name string,
	initializer tsgo.Expression,
) tsgo.Statement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					initializer,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
