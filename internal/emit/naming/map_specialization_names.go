package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const mapTargetNameHexLength = 20

func (n *File) MapSpecialization(
	sourceType types.Type,
	demand api.MapSpecializationDemand,
) (api.NameReference, error) {
	mapType, ok := representedMapType(sourceType)
	if !ok || !demand.Valid() {
		return api.NameReference{}, &api.NameError{
			Reason: "map-specialization demand is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		mapType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(mapType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internMapSpecialization(
		artifactKey,
		mapType,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewMapSpecializationRequest(
		binding.owner,
		demand,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := []api.RootRequest{requirement}
	if binding.owner.Placement() ==
		api.GeneratedArtifactPlacementLexical {
		return api.NewNameReference(binding.name, requests...)
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
		dependency, dependencyError :=
			api.NewGeneratedArtifactDependencyRequest(
				binding.owner,
				mapSpecializationFacet(demand),
			)
		if dependencyError != nil {
			return api.NameReference{}, dependencyError
		}
		requests = append(requests, dependency)
	}
	if binding.owner.OutputPath() == n.targetPath {
		return api.NewNameReference(binding.name, requests...)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		binding.owner.OutputPath(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	phase := api.ImportPhaseValue
	if demand == api.MapSpecializationDemandDefinition {
		phase = api.ImportPhaseType
	}
	importRequest, err := api.NewImportRequest(
		n.factory,
		phase,
		modulePath,
		binding.name,
		binding.name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, importRequest)
	return api.NewNameReference(binding.name, requests...)
}

func representedMapType(sourceType types.Type) (*types.Map, bool) {
	sourceType = types.Unalias(sourceType)
	if named, ok := sourceType.(*types.Named); ok {
		sourceType = named.Underlying()
	}
	mapType, ok := sourceType.(*types.Map)
	return mapType, ok
}

func mapSpecializationFacet(
	demand api.MapSpecializationDemand,
) api.ArtifactFacet {
	if demand == api.MapSpecializationDemandDefinition {
		return api.ArtifactFacetInstanceTypeSurface
	}
	return api.ArtifactFacetStaticSurface
}

func (r *Registry) internMapSpecialization(
	artifactKey string,
	mapType *types.Map,
	placement generatedArtifactPlacement,
) (mapSpecializationBinding, error) {
	if r == nil ||
		mapType == nil ||
		artifactKey == "" ||
		!placement.kind.Valid() {
		return mapSpecializationBinding{}, &api.NameError{
			Reason: "map-specialization canonicalization input is invalid",
		}
	}
	if existing, ok := r.mapSpecializations[artifactKey]; ok {
		existingType, valid := existing.owner.MapType()
		if !valid || !types.Identical(existingType, mapType) {
			return mapSpecializationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "map-specialization artifact-key collision joined non-identical Go types",
			}
		}
		if !sameMapSpecializationPlacement(existing.owner, placement) {
			return mapSpecializationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "identical map specialization received inconsistent semantic placement",
			}
		}
		return existing, nil
	}
	name, err := mapSpecializationTargetName(artifactKey)
	if err != nil {
		return mapSpecializationBinding{}, err
	}
	if existing := r.mapSpecializationNames[name]; existing != "" &&
		existing != artifactKey {
		return mapSpecializationBinding{}, &api.NameError{
			Name:   name,
			Reason: "map-specialization target-name prefix collision",
		}
	}
	owner, err := newMapSpecializationArtifact(
		mapType,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return mapSpecializationBinding{}, err
	}
	binding := mapSpecializationBinding{
		owner: owner,
		name:  name,
	}
	r.mapSpecializations[artifactKey] = binding
	r.mapSpecializationNames[name] = artifactKey
	return binding, nil
}

func mapSpecializationTargetName(artifactKey string) (string, error) {
	if len(artifactKey) < mapTargetNameHexLength {
		return "", &api.NameError{
			Reason: "map-specialization artifact key is invalid",
		}
	}
	return "$goMap_" + artifactKey[:mapTargetNameHexLength], nil
}

func newMapSpecializationArtifact(
	mapType *types.Map,
	artifact string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			api.GeneratedArtifactMapSpecialization,
			mapType,
			artifact,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	outputPath, err := output.MapSpecializationPath(artifact)
	if err != nil {
		return nil, err
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactMapSpecialization,
		mapType,
		artifact,
		name,
		outputPath,
	)
}

func sameMapSpecializationPlacement(
	artifact *api.GeneratedArtifact,
	placement generatedArtifactPlacement,
) bool {
	return artifact.Placement() == placement.kind &&
		artifact.LexicalOwner() == placement.lexicalOwner &&
		artifact.LexicalAnchor() == placement.anchor
}
