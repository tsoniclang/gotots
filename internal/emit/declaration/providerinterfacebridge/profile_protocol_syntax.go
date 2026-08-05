package providerinterfacebridge

import (
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func profileReverseCapabilityFieldDeclaration(
	factory tsgo.Factory,
	capability capabilitySelection,
) tsgo.PropertyDeclaration {
	return factory.PropertyDeclaration(
		[]tsgo.ModifierLike{
			factory.PrivateKeyword(),
			factory.ReadonlyKeyword(),
		},
		factory.Identifier(capability.fieldName),
		nil,
		profileNullableType(factory, capability.canonical.TypeName()),
		nil,
	)
}

func profileReverseConstructor(
	factory tsgo.Factory,
	payloadType tsgo.TypeNode,
	contractName string,
	capabilities []capabilitySelection,
	conflicts []capabilityConflict,
	panicName string,
) tsgo.ConstructorDeclaration {
	value := factory.Identifier("value")
	body := make([]tsgo.Statement, 0, len(capabilities)*2+len(conflicts)+1)
	for _, capability := range capabilities {
		body = append(body, profileGeneratedCapabilityDeclaration(
			factory,
			capability,
			value,
		))
	}
	for _, conflict := range conflicts {
		body = append(body, capabilityConflictStatement(
			factory,
			conflict,
			panicName,
		))
	}
	body = append(body, factory.ExpressionStatement(factory.CallExpression(
		factory.SuperExpression(),
		nil,
		nil,
		[]tsgo.Expression{
			value,
			capabilityContracts(factory, contractName, capabilities),
		},
		tsgo.NodeFlagsNone,
	)))
	for _, capability := range capabilities {
		body = append(body, capabilityFieldAssignment(factory, capability))
	}
	return factory.ConstructorDeclaration(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{parameter(factory, value, payloadType)},
		nil,
		factory.Block(body, true),
	)
}

func profileGeneratedCapabilityDeclaration(
	factory tsgo.Factory,
	capability capabilitySelection,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		nil,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(capability.fieldName),
					nil,
					nil,
					factory.ConditionalExpression(
						factory.CallExpression(
							factory.Identifier(capability.canonical.GuardName()),
							nil,
							nil,
							[]tsgo.Expression{value},
							tsgo.NodeFlagsNone,
						),
						factory.QuestionToken(),
						value,
						factory.ColonToken(),
						factory.Identifier("undefined"),
					),
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
