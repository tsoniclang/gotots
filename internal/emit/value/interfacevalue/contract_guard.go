package interfacevalue

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ContractTest(
	context api.Context,
	targetType types.Type,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	contract, err := context.Names().InterfaceContract(targetType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	guard := tsgo.Expression(context.Factory().Identifier(contract.GuardName()))
	requests := contract.Requests()
	provider, providerOwned, err := context.Names().ProviderInterface(targetType)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if providerOwned && provider.Mode() == gostdlib.ProviderInterfaceModeSealedNative {
		named, ok := types.Unalias(targetType).(*types.Named)
		if !ok || named.Obj() == nil {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "sealed provider interface has no named source type",
			}
		}
		providerType, referenceErr := context.Names().TypeReference(named.Obj())
		if referenceErr != nil {
			return api.ExpressionEmission{}, referenceErr
		}
		interfaceValue, runtimeErr := context.Names().Runtime(
			api.RuntimeInterfaceValue,
			api.ImportPhaseType,
		)
		if runtimeErr != nil {
			return api.ExpressionEmission{}, runtimeErr
		}
		candidate := context.Factory().Identifier("$candidate")
		guard = context.Factory().ParenthesizedExpression(
			context.Factory().ArrowFunction(
				nil,
				nil,
				[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
					nil,
					nil,
					candidate,
					nil,
					context.Factory().UnionTypeNode([]tsgo.TypeNode{
						context.Factory().TypeReferenceNode(
							interfaceValue.EntityName(context.Factory()),
							nil,
						),
						context.Factory().KeywordTypeNode(
							tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
						),
					}),
					nil,
				)},
				context.Factory().TypePredicateNode(
					nil,
					candidate,
					context.Factory().TypeReferenceNode(
						providerType.EntityName(context.Factory()),
						nil,
					),
				),
				context.Factory().EqualsGreaterThanToken(),
				context.Factory().CallExpression(
					context.Factory().Identifier(contract.GuardName()),
					nil,
					nil,
					[]tsgo.Expression{candidate},
					tsgo.NodeFlagsNone,
				),
			),
		)
		requests = api.CombineRequests(
			requests,
			providerType.Requests(),
			interfaceValue.Requests(),
		)
	}
	return api.DirectExpression(
		context.Factory().CallExpression(
			guard,
			nil,
			nil,
			[]tsgo.Expression{value},
			tsgo.NodeFlagsNone,
		),
		requests...,
	), nil
}
