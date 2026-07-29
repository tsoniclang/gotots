package panicruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	RaiseName         = "raise"
	RaiseRuntimeName  = "raiseRuntime"
	CreateRuntimeName = "createRuntime"
	RethrowName       = "rethrow"
	TakeName          = "take"
	RecoveredName     = "recovered"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
	panicName string,
	valueName string,
	runtimeValueName string,
	recoveryName string,
	errorTokenName string,
	runtimeErrorTokenName string,
) (tsgo.Statement, error) {
	switch symbol {
	case api.RuntimePanic:
		return panicCarrier(factory, panicName, valueName, runtimeValueName), nil
	case api.RuntimePanicValue:
		return runtimePanicValue(
			factory,
			runtimeValueName,
			valueName,
			errorTokenName,
			runtimeErrorTokenName,
		), nil
	case api.RuntimeRecovery:
		return recovery(factory, recoveryName, panicName, valueName), nil
	default:
		return nil, &api.RuntimeSymbolError{Symbol: symbol}
	}
}

func Call(
	factory tsgo.Factory,
	className string,
	message tsgo.Expression,
) tsgo.CallExpression {
	return staticCall(factory, className, RaiseRuntimeName, message)
}

func CallValue(
	factory tsgo.Factory,
	className string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return staticCall(factory, className, RaiseName, value)
}

func CreateRuntime(
	factory tsgo.Factory,
	className string,
	message tsgo.Expression,
) tsgo.CallExpression {
	return staticCall(factory, className, CreateRuntimeName, message)
}

func Rethrow(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.ThrowStatement {
	return factory.ThrowStatement(value)
}

func staticCall(
	factory tsgo.Factory,
	className string,
	memberName string,
	value tsgo.Expression,
) tsgo.CallExpression {
	return factory.CallExpression(
		factory.PropertyAccessExpression(
			factory.Identifier(className),
			nil,
			factory.Identifier(memberName),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		[]tsgo.Expression{value},
		tsgo.NodeFlagsNone,
	)
}

func panicCarrier(
	factory tsgo.Factory,
	className string,
	valueName string,
	runtimeValueName string,
) tsgo.ClassDeclaration {
	valueType := factory.TypeReferenceNode(factory.Identifier(valueName), nil)
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		nil,
		nil,
		[]tsgo.ClassElement{
			factory.ConstructorDeclaration(
				[]tsgo.ModifierLike{factory.PrivateKeyword()},
				nil,
				[]tsgo.ParameterDeclaration{
					parameterProperty(
						factory,
						"value",
						valueType,
						factory.PublicKeyword(),
						factory.ReadonlyKeyword(),
					),
				},
				nil,
				factory.Block(nil, true),
			),
			createRuntime(factory, className, runtimeValueName),
			raiseValue(factory, className, valueType),
			raiseRuntime(factory, className, runtimeValueName),
			rethrow(factory),
		},
	)
}

func rethrow(factory tsgo.Factory) tsgo.MethodDeclaration {
	return staticNeverMethod(
		factory,
		RethrowName,
		parameter(
			factory,
			"failure",
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindObjectKeyword,
			),
		),
		factory.Identifier("failure"),
	)
}

func createRuntime(
	factory tsgo.Factory,
	className string,
	runtimeValueName string,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(CreateRuntimeName),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(
				factory,
				"message",
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindStringKeyword,
				),
			),
		},
		factory.TypeReferenceNode(factory.Identifier(className), nil),
		factory.Block(
			[]tsgo.Statement{
				factory.ReturnStatement(
					factory.NewExpression(
						factory.Identifier(className),
						nil,
						[]tsgo.Expression{
							factory.NewExpression(
								factory.Identifier(runtimeValueName),
								nil,
								[]tsgo.Expression{
									factory.Identifier("message"),
								},
							),
						},
					),
				),
			},
			true,
		),
	)
}

func raiseValue(
	factory tsgo.Factory,
	className string,
	valueType tsgo.TypeNode,
) tsgo.MethodDeclaration {
	value := factory.Identifier("value")
	return staticNeverMethod(
		factory,
		RaiseName,
		parameter(factory, "value", valueType),
		factory.NewExpression(
			factory.Identifier(className),
			nil,
			[]tsgo.Expression{value},
		),
	)
}

func raiseRuntime(
	factory tsgo.Factory,
	className string,
	runtimeValueName string,
) tsgo.MethodDeclaration {
	return staticNeverMethod(
		factory,
		RaiseRuntimeName,
		parameter(
			factory,
			"message",
			factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindStringKeyword,
			),
		),
		factory.NewExpression(
			factory.Identifier(className),
			nil,
			[]tsgo.Expression{
				factory.NewExpression(
					factory.Identifier(runtimeValueName),
					nil,
					[]tsgo.Expression{factory.Identifier("message")},
				),
			},
		),
	)
}

func staticNeverMethod(
	factory tsgo.Factory,
	name string,
	parameterDeclaration tsgo.ParameterDeclaration,
	thrown tsgo.Expression,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.StaticKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameterDeclaration},
		factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindNeverKeyword),
		factory.Block(
			[]tsgo.Statement{factory.ThrowStatement(thrown)},
			true,
		),
	)
}
