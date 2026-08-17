package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildZeroMethodAdapter(
	context api.Context,
	name string,
	runtimeValueName string,
	adapterFactoryName string,
	dynamicTypeName string,
	payloadType tsgo.TypeNode,
	sourceType types.Type,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	factory := context.Factory()
	left := factory.Identifier("left")
	right := factory.Identifier("right")
	value := factory.Identifier("value")
	equalBody, equalRequests, err := equalOperationBody(
		context,
		sourceType,
		left,
		right,
	)
	if err != nil {
		return nil, nil, err
	}
	hashBody, hashRequests, err := hashOperationBody(
		context,
		dynamicTypeName,
		sourceType,
		value,
	)
	if err != nil {
		return nil, nil, err
	}
	formatBody, formatRequests, err := formatOperationBody(
		context,
		sourceType,
		value,
	)
	if err != nil {
		return nil, nil, err
	}
	stringType := factory.KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindStringKeyword,
	)
	initializer := factory.CallExpression(
		factory.Identifier(adapterFactoryName),
		nil,
		[]tsgo.TypeNode{payloadType},
		[]tsgo.Expression{
			factory.Identifier(dynamicTypeName),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					operationParameter(factory, "left", payloadType),
					operationParameter(factory, "right", payloadType),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
				factory.EqualsGreaterThanToken(),
				factory.Block(equalBody, true),
			),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					operationParameter(factory, "value", payloadType),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
				factory.EqualsGreaterThanToken(),
				factory.Block(hashBody, true),
			),
			formatStringValue(factory, sourceType),
			factory.ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					operationParameter(factory, "value", payloadType),
					operationParameter(factory, "verb", stringType),
					operationParameter(factory, "_flags", stringType),
					operationParameter(
						factory,
						"precision",
						formatPrecisionType(factory),
					),
				},
				stringType,
				factory.EqualsGreaterThanToken(),
				factory.Block(formatBody, true),
			),
		},
		tsgo.NodeFlagsNone,
	)
	statement := factory.VariableStatement(
		modifiers,
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(name),
				nil,
				interfacecontract.AdapterConstructor(
					factory,
					runtimeValueName,
					payloadType,
				),
				initializer,
			)},
			tsgo.NodeFlagsConst,
		),
	)
	return []tsgo.Statement{statement}, api.CombineRequests(
		equalRequests,
		hashRequests,
		formatRequests,
	), nil
}

func operationParameter(
	factory tsgo.Factory,
	name string,
	typeNode tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		typeNode,
		nil,
	)
}
