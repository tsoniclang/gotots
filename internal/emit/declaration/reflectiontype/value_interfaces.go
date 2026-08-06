package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// optionalInterfaceBoxType is the runtime value carried by a reflection
// location. Undefined is a valid payload only for a statically typed nil
// interface; the location's descriptor keeps that Value distinct from the
// invalid zero Value.
func optionalInterfaceBoxType(
	factory tsgo.Factory,
	boxType api.NameReference,
) tsgo.TypeNode {
	return factory.UnionTypeNode([]tsgo.TypeNode{
		factory.TypeReferenceNode(boxType.EntityName(factory), nil),
		factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
		),
	})
}

// interfaceValueProperties supplies the exact zero of an interface type.
// Its nil payload remains a valid reflect.Value because Zero carries the
// requested static descriptor separately in the provider runtime.
func interfaceValueProperties(
	scaffold *locationScaffold,
) []tsgo.ObjectLiteralElementLike {
	factory := scaffold.factory
	return []tsgo.ObjectLiteralElementLike{expressionProperty(
		factory,
		"zero",
		factory.ArrowFunction(
			nil,
			nil,
			nil,
			optionalInterfaceBoxType(factory, scaffold.boxType),
			factory.EqualsGreaterThanToken(),
			factory.Identifier("undefined"),
		),
	)}
}

// interfaceFieldCallbacks preserve one interface slot as its existing
// canonical dynamic box. Writes admit nil or a box satisfying the field's
// exact generated method contract; no concrete adapter can represent an
// interface type itself.
func interfaceFieldCallbacks(
	context api.Context,
	field *types.Var,
	fieldAccess tsgo.Expression,
	scaffold *locationScaffold,
) (
	tsgo.Expression,
	tsgo.Block,
	[]api.RootRequest,
	error,
) {
	factory := scaffold.factory
	assigned, requests, err := admittedInterfaceFieldValue(
		context,
		field,
		scaffold,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	set := factory.Block([]tsgo.Statement{factory.ExpressionStatement(
		factory.BinaryExpression(
			nil,
			fieldAccess,
			nil,
			factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
			assigned,
		),
	)}, true)
	return fieldAccess, set, requests, nil
}

func admittedInterfaceFieldValue(
	context api.Context,
	field *types.Var,
	scaffold *locationScaffold,
) (tsgo.Expression, []api.RootRequest, error) {
	factory := scaffold.factory
	value := factory.Identifier("value")
	nilValue := factory.BinaryExpression(
		nil,
		value,
		nil,
		factory.BinaryOperatorToken(
			tsgo.BinaryOperatorEqualsEqualsEqualsToken,
		),
		factory.Identifier("undefined"),
	)
	implements, err := interfacevalue.ContractTest(
		context,
		field.Type(),
		value,
	)
	if err != nil {
		return nil, nil, err
	}
	accepted := factory.BinaryExpression(
		nil,
		nilValue,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorBarBarToken),
		implements.Value(),
	)
	assigned := factory.ConditionalExpression(
		accepted,
		factory.QuestionToken(),
		value,
		factory.ColonToken(),
		runtimePanic(
			scaffold,
			"reflect: Value.Set received a value outside the interface contract",
		),
	)
	return assigned, implements.Requests(), nil
}
