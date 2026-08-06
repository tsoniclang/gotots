package storagefacet

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	storageParameter = "S"
	valueParameter   = "T"
)

func Build(
	factory tsgo.Factory,
	symbol api.RuntimeSymbol,
) (tsgo.Statement, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return nil, err
	}
	switch symbol {
	case api.RuntimeStorageTypeToken,
		api.RuntimeContainerStorageToken,
		api.RuntimePointerTypeToken:
		return tokenDeclaration(factory, contract.ExportedName()), nil
	case api.RuntimeStoredValue:
		token, tokenErr := api.RuntimeContract(api.RuntimeStorageTypeToken)
		if tokenErr != nil {
			return nil, tokenErr
		}
		return valueContract(
			factory,
			contract.ExportedName(),
			token.ExportedName(),
		), nil
	case api.RuntimeContainerStoredValue:
		token, tokenErr := api.RuntimeContract(
			api.RuntimeContainerStorageToken,
		)
		if tokenErr != nil {
			return nil, tokenErr
		}
		return valueContract(
			factory,
			contract.ExportedName(),
			token.ExportedName(),
		), nil
	case api.RuntimePointerRepresentedValue:
		token, tokenErr := api.RuntimeContract(api.RuntimePointerTypeToken)
		if tokenErr != nil {
			return nil, tokenErr
		}
		return valueContract(
			factory,
			contract.ExportedName(),
			token.ExportedName(),
		), nil
	case api.RuntimeStorageType:
		value, valueErr := api.RuntimeContract(api.RuntimeStoredValue)
		if valueErr != nil {
			return nil, valueErr
		}
		return projectionAlias(
			factory,
			contract.ExportedName(),
			value.ExportedName(),
			valueType(factory),
		), nil
	case api.RuntimeContainerStorageType:
		value, valueErr := api.RuntimeContract(
			api.RuntimeContainerStoredValue,
		)
		if valueErr != nil {
			return nil, valueErr
		}
		return projectionAlias(
			factory,
			contract.ExportedName(),
			value.ExportedName(),
			valueType(factory),
		), nil
	case api.RuntimePointerType:
		value, valueErr := api.RuntimeContract(
			api.RuntimePointerRepresentedValue,
		)
		if valueErr != nil {
			return nil, valueErr
		}
		pointer, pointerErr := api.RuntimeContract(api.RuntimePointer)
		if pointerErr != nil {
			return nil, pointerErr
		}
		logical := valueType(factory)
		return nonDistributiveProjectionAlias(
			factory,
			contract.ExportedName(),
			value.ExportedName(),
			factory.TypeReferenceNode(
				factory.Identifier(pointer.ExportedName()),
				[]tsgo.TypeNode{logical, logical},
			),
		), nil
	default:
		return nil, &api.InvariantError{
			Reason: "storage-facet runtime symbol is unsupported",
		}
	}
}

func tokenDeclaration(
	factory tsgo.Factory,
	name string,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		[]tsgo.ModifierLike{
			factory.ExportKeyword(),
			factory.DeclareKeyword(),
		},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				factory.VariableDeclaration(
					factory.Identifier(name),
					nil,
					factory.TypeOperatorNode(
						tsgo.TypeOperatorNodeOperatorKindUniqueKeyword,
						factory.KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindSymbolKeyword,
						),
					),
					nil,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}

func valueContract(
	factory tsgo.Factory,
	name string,
	token string,
) tsgo.InterfaceDeclaration {
	storageType := factory.TypeReferenceNode(
		factory.Identifier(storageParameter),
		nil,
	)
	return factory.InterfaceDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, storageParameter),
		},
		nil,
		[]tsgo.TypeElement{
			factory.PropertySignatureDeclaration(
				[]tsgo.ModifierLike{factory.ReadonlyKeyword()},
				factory.ComputedPropertyName(factory.Identifier(token)),
				nil,
				storageType,
				factory.OmittedExpression(),
			),
		},
	)
}

func projectionAlias(
	factory tsgo.Factory,
	name string,
	contract string,
	fallback tsgo.TypeNode,
) tsgo.TypeAliasDeclaration {
	valueType := valueType(factory)
	return projectionAliasFromTypes(
		factory,
		name,
		valueType,
		factory.TypeReferenceNode(
			factory.Identifier(contract),
			[]tsgo.TypeNode{
				factory.InferTypeNode(
					typeParameter(factory, storageParameter),
				),
			},
		),
		fallback,
	)
}

func nonDistributiveProjectionAlias(
	factory tsgo.Factory,
	name string,
	contract string,
	fallback tsgo.TypeNode,
) tsgo.TypeAliasDeclaration {
	valueType := valueType(factory)
	return projectionAliasFromTypes(
		factory,
		name,
		factory.TupleTypeNode([]tsgo.TypeNode{valueType}),
		factory.TupleTypeNode([]tsgo.TypeNode{
			factory.TypeReferenceNode(
				factory.Identifier(contract),
				[]tsgo.TypeNode{
					factory.InferTypeNode(
						typeParameter(factory, storageParameter),
					),
				},
			),
		}),
		fallback,
	)
}

func projectionAliasFromTypes(
	factory tsgo.Factory,
	name string,
	checkType tsgo.TypeNode,
	extendsType tsgo.TypeNode,
	fallback tsgo.TypeNode,
) tsgo.TypeAliasDeclaration {
	storageType := factory.TypeReferenceNode(
		factory.Identifier(storageParameter),
		nil,
	)
	return factory.TypeAliasDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, valueParameter),
		},
		factory.ConditionalTypeNode(
			checkType,
			extendsType,
			storageType,
			fallback,
		),
	)
}

func valueType(factory tsgo.Factory) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(
		factory.Identifier(valueParameter),
		nil,
	)
}

func typeParameter(
	factory tsgo.Factory,
	name string,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier(name),
		nil,
		nil,
		nil,
	)
}
