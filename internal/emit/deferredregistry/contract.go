package deferredregistry

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ContractType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (api.TypeEmission, error) {
	contract, err := observedCallableTypes(
		context,
		children,
		source,
		signature,
	)
	if err != nil {
		return api.TypeEmission{}, err
	}
	undefined := context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
	)
	return api.DirectType(
		context.Factory().TypeLiteralNode([]tsgo.TypeElement{
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(api.DeferredRegistryRegisterName),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						context.Factory(),
						context.Factory().Identifier("source"),
						contract.source,
					),
					parameter(
						context.Factory(),
						context.Factory().Identifier("deferred"),
						contract.deferred,
					),
				},
				contract.source,
			),
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(api.DeferredRegistryResolveName),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{parameter(
					context.Factory(),
					context.Factory().Identifier("source"),
					context.Factory().UnionTypeNode([]tsgo.TypeNode{
						contract.source,
						undefined,
					}),
				)},
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					contract.deferred,
					undefined,
				}),
			),
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(
					api.DeferredRegistryRegisterMethodName,
				),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						context.Factory(),
						context.Factory().Identifier("method"),
						context.Factory().KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindObjectKeyword,
						),
					),
					parameter(
						context.Factory(),
						context.Factory().Identifier("dynamicType"),
						context.Factory().KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindObjectKeyword,
						),
					),
					parameter(
						context.Factory(),
						context.Factory().Identifier("deferred"),
						contract.methodDeferred,
					),
				},
				context.Factory().KeywordTypeNode(
					tsgo.KeywordTypeSyntaxKindVoidKeyword,
				),
			),
			context.Factory().MethodSignatureDeclaration(
				nil,
				context.Factory().Identifier(
					api.DeferredRegistryResolveMethodName,
				),
				nil,
				nil,
				[]tsgo.ParameterDeclaration{
					parameter(
						context.Factory(),
						context.Factory().Identifier("method"),
						context.Factory().KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindObjectKeyword,
						),
					),
					parameter(
						context.Factory(),
						context.Factory().Identifier("receiver"),
						contract.interfaceValue,
					),
				},
				context.Factory().UnionTypeNode([]tsgo.TypeNode{
					contract.methodDeferred,
					undefined,
				}),
			),
		}),
		contract.requests...,
	), nil
}

func observedCallableTypes(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	signature *types.Signature,
) (callableContract, error) {
	abi, err := callable.ABIReference(context, signature)
	if err != nil {
		return callableContract{}, err
	}
	facet, err := context.CallableABIFacet(abi)
	if err != nil {
		return callableContract{}, err
	}
	observation, err := context.ObserveCooperativeCallable(facet)
	if err != nil {
		return callableContract{}, err
	}
	contract, err := callableTypes(
		context,
		children,
		signature,
		observation.Cooperative(),
	)
	contract.requests = api.CombineRequests(
		contract.requests,
		abi.Requests(),
		observation.Requests(),
	)
	return contract, err
}
