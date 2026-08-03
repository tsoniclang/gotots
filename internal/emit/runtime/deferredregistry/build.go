package deferredregistry

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	sourceType          = "Source"
	deferredType        = "Deferred"
	methodDeferredType  = "MethodDeferred"
	entriesMember       = "$entries"
	methodEntriesMember = "$methodEntries"
)

func Build(
	factory tsgo.Factory,
	className string,
	interfaceValueName string,
) tsgo.ClassDeclaration {
	return factory.ClassDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier(className),
		[]tsgo.TypeParameterDeclaration{
			typeParameter(factory, sourceType, objectType(factory)),
			typeParameter(factory, deferredType, nil),
			typeParameter(factory, methodDeferredType, nil),
		},
		nil,
		[]tsgo.ClassElement{
			entries(factory),
			register(factory),
			resolve(factory),
			methodEntries(factory, methodEntriesMember, methodDeferredType),
			registerMethod(
				factory,
				methodEntriesMember,
				api.DeferredRegistryRegisterMethodName,
				methodDeferredType,
			),
			resolveMethod(
				factory,
				methodEntriesMember,
				api.DeferredRegistryResolveMethodName,
				interfaceValueName,
				methodDeferredType,
			),
		},
	)
}

func typeParameter(
	factory tsgo.Factory,
	name string,
	constraint tsgo.TypeNode,
) tsgo.TypeParameterDeclaration {
	return factory.TypeParameterDeclaration(
		nil,
		factory.Identifier(name),
		constraint,
		nil,
		nil,
	)
}

func typeReference(factory tsgo.Factory, name string) tsgo.TypeReferenceNode {
	return factory.TypeReferenceNode(factory.Identifier(name), nil)
}

func parameter(
	factory tsgo.Factory,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return factory.ParameterDeclaration(
		nil,
		nil,
		factory.Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func objectType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindObjectKeyword)
}

func undefinedType(factory tsgo.Factory) tsgo.TypeNode {
	return factory.KeywordTypeNode(tsgo.KeywordTypeSyntaxKindUndefinedKeyword)
}

func mapType(factory tsgo.Factory, value tsgo.TypeNode) tsgo.TypeNode {
	return factory.TypeReferenceNode(
		factory.Identifier("Map"),
		[]tsgo.TypeNode{objectType(factory), value},
	)
}

func memberAccess(
	factory tsgo.Factory,
	owner string,
	member string,
) tsgo.Expression {
	return factory.PropertyAccessExpression(
		factory.PropertyAccessExpression(
			factory.ThisExpression(),
			nil,
			factory.Identifier(owner),
			tsgo.NodeFlagsNone,
		),
		nil,
		factory.Identifier(member),
		tsgo.NodeFlagsNone,
	)
}
