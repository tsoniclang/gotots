package interfaceadapter

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericinstance "github.com/tsoniclang/gotots/internal/emit/generic/instance"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func contractHeritage(
	context api.Context,
	children api.ChildEmitter,
	contracts []Contract,
) (tsgo.HeritageClause, []api.RootRequest, error) {
	if len(contracts) == 0 {
		return nil, nil, nil
	}
	implements := make(
		[]tsgo.ExpressionWithTypeArguments,
		0,
		len(contracts),
	)
	var requests []api.RootRequest
	for _, contract := range contracts {
		if !contract.valid() {
			return nil, nil, &api.GeneratedArtifactShapeError{
				Reason: "interface-adapter heritage contract is invalid",
			}
		}
		reference, err := context.Names().InterfaceType(contract.sourceType)
		if err != nil {
			return nil, nil, err
		}
		var arguments []tsgo.TypeNode
		var argumentRequests []api.RootRequest
		if named, ok := types.Unalias(contract.sourceType).(*types.Named); ok &&
			named.TypeArgs().Len() != 0 {
			arguments, argumentRequests, err = genericinstance.EmitTypeArguments(
				context,
				children,
				nil,
				named.Origin().Obj(),
				api.TypeArgumentsFromGo(named.TypeArgs()),
			)
			if err != nil {
				return nil, nil, err
			}
		}
		implements = append(
			implements,
			context.Factory().ExpressionWithTypeArguments(
				reference.Expression(context.Factory()),
				arguments,
			),
		)
		requests = append(requests, reference.Requests()...)
		requests = append(requests, argumentRequests...)
	}
	return context.Factory().HeritageClause(
		tsgo.HeritageClauseTokenKindImplementsKeyword,
		implements,
	), api.CombineRequests(requests), nil
}
