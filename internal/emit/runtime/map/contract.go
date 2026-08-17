package mapruntime

import (
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func valueContract(
	factory tsgo.Factory,
	name string,
	members memberNames,
) tsgo.ClassDeclaration {
	keyType := typeName(factory, keyTypeName)
	valueType := typeName(factory, valueTypeName)
	return typescriptclass.Declaration(
		factory,
		[]tsgo.ModifierLike{
			factory.ExportKeyword(),
			factory.AbstractKeyword(),
		},
		factory.Identifier(name),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, keyTypeName),
			typeParameter(factory, valueTypeName),
		},
		nil,
		[]tsgo.ClassElement{
			methodContract(
				factory,
				members.lookup,
				[]tsgo.ParameterDeclaration{
					parameter(factory, keyName, keyType),
				},
				valueType,
			),
			methodContract(
				factory,
				members.lookupOK,
				[]tsgo.ParameterDeclaration{
					parameter(factory, keyName, keyType),
				},
				factory.TupleTypeNode([]tsgo.TypeNode{
					valueType,
					factory.KeywordTypeNode(
						tsgo.KeywordTypeSyntaxKindBooleanKeyword,
					),
				}),
			),
			methodContract(
				factory,
				members.store,
				[]tsgo.ParameterDeclaration{
					parameter(factory, keyName, keyType),
					parameter(factory, valueName, valueType),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			),
			methodContract(
				factory,
				members.delete,
				[]tsgo.ParameterDeclaration{
					parameter(factory, keyName, keyType),
				},
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			),
			methodContract(
				factory,
				members.length,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindNumberKeyword,
				),
			),
			methodContract(
				factory,
				members.isNil,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindBooleanKeyword,
				),
			),
			methodContract(
				factory,
				members.clear,
				nil,
				factory.KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			),
			methodContract(
				factory,
				members.keys,
				nil,
				factory.ArrayTypeNode(keyType),
			),
		},
	)
}

func methodContract(
	factory tsgo.Factory,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
) tsgo.MethodDeclaration {
	return factory.MethodDeclaration(
		[]tsgo.ModifierLike{factory.AbstractKeyword()},
		nil,
		factory.Identifier(name),
		nil,
		nil,
		parameters,
		result,
		nil,
	)
}
