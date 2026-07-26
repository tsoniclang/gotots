package mapruntime

import "github.com/tsoniclang/gotots/internal/target/tsgo"

func lookupStatements(
	factory tsgo.Factory,
	withPresence bool,
) []tsgo.Statement {
	const (
		storageName = "storage"
		storedName  = "storedValue"
	)
	return []tsgo.Statement{
		constantDeclaration(
			factory,
			storageName,
			field(factory, valuesName),
		),
		undefinedReturn(
			factory,
			factory.Identifier(storageName),
			lookupResult(factory, withPresence, false),
		),
		constantDeclaration(
			factory,
			storedName,
			methodCall(
				factory,
				factory.Identifier(storageName),
				"get",
				factory.Identifier(keyName),
			),
		),
		undefinedReturn(
			factory,
			factory.Identifier(storedName),
			lookupResult(factory, withPresence, false),
		),
		factory.ReturnStatement(
			lookupResult(factory, withPresence, true),
		),
	}
}

func constantDeclaration(
	factory tsgo.Factory,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					nil,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func undefinedReturn(
	factory tsgo.Factory,
	value tsgo.Expression,
	result tsgo.Expression,
) tsgo.IfStatement {
	return factory.IfStatement(
		factory.BinaryExpression(
			nil,
			value,
			nil,
			factory.BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			factory.Identifier("undefined"),
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(result),
		}, true),
		nil,
	)
}

func lookupResult(
	factory tsgo.Factory,
	withPresence bool,
	present bool,
) tsgo.Expression {
	value := tsgo.Expression(field(factory, zeroName))
	if present {
		value = factory.Identifier("storedValue")
	}
	if !withPresence {
		return value
	}
	presence := tsgo.Expression(factory.FalseLiteral())
	if present {
		presence = factory.TrueLiteral()
	}
	return factory.ArrayLiteralExpression(
		[]tsgo.Expression{value, presence},
		false,
	)
}
