package typefacet

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Emission struct {
	members  []tsgo.ClassElement
	heritage []tsgo.HeritageClause
	requests []api.RootRequest
}

func (e Emission) Members() []tsgo.ClassElement {
	return slices.Clone(e.members)
}

func (e Emission) Heritage() []tsgo.HeritageClause {
	return slices.Clone(e.heritage)
}

func (e Emission) Requests() []api.RootRequest {
	return slices.Clone(e.requests)
}

func Build(
	context api.Context,
	sourceType types.Type,
	storageType tsgo.TypeNode,
	facets []api.TypeRepresentationFacet,
	ambient bool,
) (Emission, error) {
	if sourceType == nil {
		return Emission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "type-representation marker input is incomplete",
		}
	}
	seen := make(map[api.TypeRepresentationFacet]struct{}, len(facets))
	selected := make([]api.TypeRepresentationFacet, 0, len(facets))
	targetTypes := make(map[api.TypeRepresentationFacet]tsgo.TypeNode)
	var requests []api.RootRequest
	for _, facet := range facets {
		if !facet.Valid() {
			return Emission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "type-representation marker facet is invalid",
			}
		}
		if _, duplicate := seen[facet]; duplicate {
			return Emission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "type-representation marker facet is duplicated",
			}
		}
		seen[facet] = struct{}{}
		targetType := storageType
		if targetType == nil {
			return Emission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "selected type-representation marker has no target type",
			}
		}
		selected = append(selected, facet)
		targetTypes[facet] = targetType
	}
	if len(selected) == 0 {
		return Emission{requests: api.CombineRequests(requests)}, nil
	}
	slices.Sort(selected)
	members := make([]tsgo.ClassElement, 0, len(selected))
	implements := make([]tsgo.ExpressionWithTypeArguments, 0, len(selected))
	for _, facet := range selected {
		targetType := targetTypes[facet]
		tokenSymbol, contractSymbol := runtimeSymbols(facet)
		token, err := context.Names().Runtime(
			tokenSymbol,
			api.ImportPhaseType,
		)
		if err != nil {
			return Emission{}, err
		}
		contract, err := context.Names().Runtime(
			contractSymbol,
			api.ImportPhaseType,
		)
		if err != nil {
			return Emission{}, err
		}
		modifiers := []tsgo.ModifierLike{context.Factory().ReadonlyKeyword()}
		if !ambient {
			modifiers = append(
				[]tsgo.ModifierLike{context.Factory().DeclareKeyword()},
				modifiers...,
			)
		}
		members = append(members, context.Factory().PropertyDeclaration(
			modifiers,
			context.Factory().ComputedPropertyName(
				token.Expression(context.Factory()),
			),
			nil,
			targetType,
			nil,
		))
		implements = append(
			implements,
			context.Factory().ExpressionWithTypeArguments(
				contract.Expression(context.Factory()),
				[]tsgo.TypeNode{targetType},
			),
		)
		requests = append(
			requests,
			token.Requests()...,
		)
		requests = append(
			requests,
			contract.Requests()...,
		)
	}
	return Emission{
		members: members,
		heritage: []tsgo.HeritageClause{
			context.Factory().HeritageClause(
				tsgo.HeritageClauseTokenKindImplementsKeyword,
				implements,
			),
		},
		requests: api.CombineRequests(requests),
	}, nil
}

func runtimeSymbols(
	facet api.TypeRepresentationFacet,
) (api.RuntimeSymbol, api.RuntimeSymbol) {
	switch facet {
	case api.TypeRepresentationStorage:
		return api.RuntimeStorageTypeToken, api.RuntimeStoredValue
	case api.TypeRepresentationContainerStorage:
		return api.RuntimeContainerStorageToken,
			api.RuntimeContainerStoredValue
	default:
		panic("validated type-representation facet is unhandled")
	}
}
