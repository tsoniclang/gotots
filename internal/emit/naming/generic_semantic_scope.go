package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type genericGeneratedModuleScope struct {
	placement api.GeneratedArtifactPlacement
	module    string
}

type genericGeneratedNameScope struct {
	placement    api.GeneratedArtifactPlacement
	lexicalOwner api.ArtifactOwner
	anchor       *types.TypeName
	module       string
	name         string
}

func reserveGenericGeneratedName(
	names map[genericGeneratedNameScope]string,
	scope genericGeneratedNameScope,
	artifactKey string,
	kind string,
) error {
	compilation := scope.placement ==
		api.GeneratedArtifactPlacementCompilation &&
		scope.module != "" && !scope.lexicalOwner.Valid() && scope.anchor == nil
	lexical := scope.placement == api.GeneratedArtifactPlacementLexical &&
		scope.module == "" && scope.lexicalOwner.Valid() && scope.anchor != nil
	if names == nil || scope.name == "" || artifactKey == "" ||
		(!compilation && !lexical) {
		return &api.NameError{
			Name:   scope.name,
			Reason: kind + " semantic name scope is invalid",
		}
	}
	if existing := names[scope]; existing != "" && existing != artifactKey {
		return &api.NameError{
			Name:   scope.name,
			Reason: kind + " semantic name collision",
		}
	}
	names[scope] = artifactKey
	return nil
}

func reserveGenericConcretizationModule(
	modules map[genericGeneratedModuleScope]*types.Func,
	scope genericGeneratedModuleScope,
	owner *types.Func,
) error {
	if modules == nil ||
		scope.placement != api.GeneratedArtifactPlacementCompilation ||
		scope.module == "" || owner == nil || owner.Origin() != owner {
		return &api.NameError{
			Name:   scope.module,
			Reason: "generic concretization semantic module scope is invalid",
		}
	}
	if existing := modules[scope]; existing != nil && existing != owner {
		return &api.NameError{
			Name:   scope.module,
			Reason: "generic concretization semantic module collision",
		}
	}
	modules[scope] = owner
	return nil
}
