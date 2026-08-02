package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func (r *Registry) internProviderStatefulRepresentation(
	artifactKey string,
	sourceType *types.Named,
) (providerStatefulRepresentationBinding, error) {
	if r == nil || sourceType == nil || sourceType.Obj() == nil ||
		artifactKey == "" {
		return providerStatefulRepresentationBinding{}, &api.NameError{
			Reason: "provider stateful-representation canonicalization input is invalid",
		}
	}
	if _, interfaceType := sourceType.Underlying().(*types.Interface); interfaceType {
		return providerStatefulRepresentationBinding{}, &api.NameError{
			Name:   sourceType.Obj().Name(),
			Reason: "provider stateful-representation source is an interface",
		}
	}
	if existing, found := r.providerStatefulRepresentations[artifactKey]; found {
		existingType, valid := existing.owner.ProviderStatefulRepresentationType()
		if !valid || !types.Identical(existingType, sourceType) {
			return providerStatefulRepresentationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "provider stateful-representation key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	name, err := providerStatefulTargetName(artifactKey)
	if err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	if err := reserveGeneratedName(
		r.providerStatefulRepresentationNames,
		name,
		artifactKey,
		"provider stateful-representation",
	); err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	outputPath, err := output.ProviderStatefulRepresentationPath(artifactKey)
	if err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactProviderStatefulRepresentation,
		sourceType,
		artifactKey,
		name,
		outputPath,
	)
	if err != nil {
		return providerStatefulRepresentationBinding{}, err
	}
	binding := providerStatefulRepresentationBinding{owner: owner, name: name}
	r.providerStatefulRepresentations[artifactKey] = binding
	return binding, nil
}

func providerStatefulTargetName(artifactKey string) (string, error) {
	if len(artifactKey) < interfaceTargetNameHexLength {
		return "", &api.NameError{
			Reason: "provider stateful-representation artifact key is invalid",
		}
	}
	return "$goProviderState_" + artifactKey[:interfaceTargetNameHexLength], nil
}
