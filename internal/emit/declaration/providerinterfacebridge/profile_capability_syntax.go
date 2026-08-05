package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func profileCapabilityDeclarations(
	factory tsgo.Factory,
	capability profileCapabilitySelection,
	baseContract string,
	baseReverse string,
	runtimeBase string,
	modifiers []tsgo.ModifierLike,
) []tsgo.Statement {
	return []tsgo.Statement{
		profileRawCapabilityGuard(
			factory,
			capability,
			baseContract,
			runtimeBase,
		),
		profileCapabilityView(
			factory,
			capability,
			baseContract,
			baseReverse,
			modifiers,
		),
	}
}

func profileRawCapabilityGuard(
	factory tsgo.Factory,
	capability profileCapabilitySelection,
	baseContract string,
	runtimeBase string,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	return factory.FunctionDeclaration(
		nil,
		nil,
		factory.Identifier(capability.rawGuardName),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, profileNamedType(factory, baseContract)),
		},
		factory.TypePredicateNode(
			nil,
			value,
			profileNamedType(factory, capability.targetBridge.Contract().Name()),
		),
		factory.Block([]tsgo.Statement{
			factory.ReturnStatement(factory.BinaryExpression(
				nil,
				factory.PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					profileInstanceOf(factory, value, runtimeBase),
				),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandAmpersandToken,
				),
				factory.CallExpression(
					factory.PropertyAccessExpression(
						value,
						nil,
						factory.Identifier(interfacecontract.ImplementsMember),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{
						factory.Identifier(
							capability.targetCanonical.ContractName(),
						),
					},
					tsgo.NodeFlagsNone,
				),
			)),
		}, true),
	)
}

func profileCapabilityView(
	factory tsgo.Factory,
	capability profileCapabilitySelection,
	baseContract string,
	baseReverse string,
	modifiers []tsgo.ModifierLike,
) tsgo.FunctionDeclaration {
	value := factory.Identifier("value")
	generated := factory.Identifier("generated")
	return factory.FunctionDeclaration(
		modifiers,
		nil,
		factory.Identifier(capability.name),
		nil,
		[]tsgo.ParameterDeclaration{
			parameter(factory, value, profileNullableType(factory, baseContract)),
		},
		profileNullableType(factory, capability.targetBridge.Contract().Name()),
		factory.Block([]tsgo.Statement{
			profileUndefinedReturn(factory, value),
			factory.IfStatement(
				profileInstanceOf(factory, value, baseReverse),
				factory.Block([]tsgo.Statement{
					profileGeneratedCapabilityValue(factory, value, generated),
					factory.ReturnStatement(factory.ConditionalExpression(
						factory.CallExpression(
							factory.Identifier(capability.targetCanonical.GuardName()),
							nil,
							nil,
							[]tsgo.Expression{generated},
							tsgo.NodeFlagsNone,
						),
						factory.QuestionToken(),
						factory.CallExpression(
							factory.PropertyAccessExpression(
								capability.targetBridge.Bridge().Expression(factory),
								nil,
								factory.Identifier(api.ProviderBridgeToMember),
								tsgo.NodeFlagsNone,
							),
							nil,
							nil,
							[]tsgo.Expression{generated},
							tsgo.NodeFlagsNone,
						),
						factory.ColonToken(),
						factory.Identifier("undefined"),
					)),
				}, true),
				nil,
			),
			factory.ReturnStatement(factory.ConditionalExpression(
				factory.CallExpression(
					factory.Identifier(capability.rawGuardName),
					nil,
					nil,
					[]tsgo.Expression{value},
					tsgo.NodeFlagsNone,
				),
				factory.QuestionToken(),
				value,
				factory.ColonToken(),
				factory.Identifier("undefined"),
			)),
		}, true),
	)
}

func profileGeneratedCapabilityValue(
	factory tsgo.Factory,
	value tsgo.Expression,
	generated tsgo.BindingName,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					generated,
					nil,
					nil,
					factory.CallExpression(
						factory.PropertyAccessExpression(
							value,
							nil,
							factory.Identifier(profileGeneratedValueMember),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
