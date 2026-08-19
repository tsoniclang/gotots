package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// locationScaffold carries the shared references every generated location
// callback of one walked type needs: the exact adapter guard, the runtime
// box type, the panic owner, and the descriptor thunk return type.
type locationScaffold struct {
	factory        tsgo.Factory
	adapter        api.NameReference
	boxType        api.NameReference
	panicRef       api.NameReference
	descriptorType api.NameReference
	requests       []api.RootRequest
	// payload is the raw represented payload of the walked type's box:
	// the boxed value routed through the defined-type projection when the
	// registered type carries a branded representation.
	payload tsgo.Expression
}

// scaffoldPayload is the raw payload expression of the walked type.
func scaffoldPayload(scaffold *locationScaffold) tsgo.Expression {
	if scaffold.payload != nil {
		return scaffold.payload
	}
	return boxPayload(scaffold.factory)
}

// extendedValueProperties derives the location-model callbacks admitted by
// one non-struct, non-pointer kind beyond its scalar projections.
func extendedValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	sourceType types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	switch selected := types.Unalias(sourceType).Underlying().(type) {
	case *types.Basic:
		return basicValueProperties(context, scaffold, sourceType, selected)
	case *types.Slice:
		return sliceValueProperties(
			context,
			names,
			reflectionType,
			sourceType,
			selected,
			scaffold,
		)
	case *types.Map:
		return mapValueProperties(
			context,
			names,
			reflectionType,
			sourceType,
			selected,
			scaffold,
		)
	case *types.Interface:
		return interfaceValueProperties(scaffold), nil
	default:
		return nil, nil
	}
}

// locationCallbacks is the exact content of one generated location
// literal: the descriptor thunk target, generated settability evidence,
// the boxing read expression, and the write body.
type locationCallbacks struct {
	descriptor api.NameReference
	settable   bool
	get        tsgo.Expression
	getBlock   tsgo.Block
	set        tsgo.Block
}

func locationLiteral(
	scaffold *locationScaffold,
	callbacks locationCallbacks,
) tsgo.ObjectLiteralExpression {
	factory := scaffold.factory
	var get tsgo.ConciseBody
	if callbacks.getBlock != nil {
		get = callbacks.getBlock
	} else {
		get = factory.ParenthesizedExpression(callbacks.get)
	}
	return factory.ObjectLiteralExpression(
		[]tsgo.ObjectLiteralElementLike{
			expressionProperty(factory, "type", arrow(
				factory,
				scaffold.descriptorType,
				callbacks.descriptor.Expression(factory),
			)),
			booleanProperty(factory, "settable", callbacks.settable),
			expressionProperty(factory, "get", factory.ArrowFunction(
				nil,
				nil,
				nil,
				optionalInterfaceBoxType(factory, scaffold.boxType),
				factory.EqualsGreaterThanToken(),
				get,
			)),
			expressionProperty(factory, "set", factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{factory.ParameterDeclaration(
					nil,
					nil,
					factory.Identifier("value"),
					nil,
					optionalInterfaceBoxType(factory, scaffold.boxType),
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
	return guardedForeignOperand(scaffold, adapter, "value", operation)
}

// guardedForeignOperand projects the payload of one named box operand
// through the exact adapter guard of its expected type.
func guardedForeignOperand(
	scaffold *locationScaffold,
	adapter api.NameReference,
	operand string,
	operation string,
) tsgo.Expression {
	factory := scaffold.factory
	return factory.ConditionalExpression(
		adapterGuard(factory, adapter, operand),
		factory.QuestionToken(),
		memberAccess(factory, operand, "$go$value"),
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
