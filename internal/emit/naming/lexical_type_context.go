package naming

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func WithLexicalTypeRequirements(
	context api.Context,
	source ast.Node,
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	sourceOwner, sourceOwned := owner.Source()
	_, initializer, initializerOwned := owner.PackageInitializer()
	validSource := source != nil
	switch {
	case sourceOwned:
		validSource = validSource &&
			sourceOwner.Pos() >= source.Pos() &&
			sourceOwner.Pos() <= source.End()
	case initializerOwned:
		validSource = validSource &&
			initializer.Rhs.Pos() >= source.Pos() &&
			initializer.Rhs.End() <= source.End()
	default:
		validSource = false
	}
	if !validSource {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical generated artifact has no exact source owner",
		}
	}
	byAnchor := make(
		map[*types.TypeName][]api.DeclarationRequirement,
	)
	for _, requirement := range requirements {
		var anchor *types.TypeName
		if artifact, generated :=
			requirement.LexicalGeneratedArtifact(); generated {
			if artifact.Placement() !=
				api.GeneratedArtifactPlacementLexical ||
				artifact.ReconstructionOwner() != owner {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "source artifact received a foreign lexical generated artifact",
				}
			}
			anchor = artifact.LexicalAnchor()
		} else if _, generated := requirement.GeneratedArtifact(); generated {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "source artifact received a non-lexical generated artifact",
			}
		} else if typeName, _, namedStruct :=
			requirement.NamedStructOperation(); namedStruct {
			if requirement.Owner() != owner {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "source artifact received a foreign lexical named-type requirement",
				}
			}
			anchor = typeName
		} else {
			continue
		}
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
	return context.WithLexicalTypeRequirements(owner, byAnchor), nil
}
