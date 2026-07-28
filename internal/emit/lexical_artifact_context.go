package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func withLexicalAnonymousStructs(
	context api.Context,
	source ast.Node,
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	sourceOwner, sourceOwned := owner.Source()
	if source == nil ||
		!sourceOwned ||
		sourceOwner.Pos() < source.Pos() ||
		sourceOwner.Pos() > source.End() {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical generated artifact has no exact source owner",
		}
	}
	byAnchor := make(
		map[*types.TypeName][]api.DeclarationRequirement,
	)
	for _, requirement := range requirements {
		if requirement.Kind() !=
			api.DeclarationRequirementAnonymousStruct {
			continue
		}
		artifact, _, ok := requirement.AnonymousStruct()
		if !ok ||
			artifact.Placement() !=
				api.GeneratedArtifactPlacementLexical ||
			artifact.ReconstructionOwner() != owner {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "source artifact received a foreign lexical generated artifact",
			}
		}
		anchor := artifact.LexicalAnchor()
		if anchor == nil ||
			anchor.Pos() < source.Pos() ||
			anchor.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "lexical generated artifact anchor is outside its source declaration",
			}
		}
		byAnchor[anchor] = append(byAnchor[anchor], requirement)
	}
	return context.WithLexicalAnonymousStructs(owner, byAnchor), nil
}
