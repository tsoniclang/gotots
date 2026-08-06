package naming

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

func (n *File) GenericKernel(
	owner *types.Func,
) (api.NameReference, error) {
	if !validGenericKernelOwner(owner) {
		return api.NameReference{}, &api.NameError{
			Reason: "generic kernel owner is invalid",
		}
	}
	selected, providerOwned, err :=
		n.owner.registry.ProviderGenericKernel(owner)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		return n.providerGenericKernelReference(owner, selected)
	}
	return n.derivedSourceReference(
		owner,
		api.GenericKernelSuffix,
		api.ArtifactFacetCallableSignature,
	)
}

func (n *File) DeferredGenericKernel(
	owner *types.Func,
) (api.DeferredGenericCallableReference, error) {
	if !validGenericKernelOwner(owner) {
		return api.DeferredGenericCallableReference{}, &api.NameError{
			Reason: "deferred generic kernel owner is invalid",
		}
	}
	selected, providerOwned, err :=
		n.owner.registry.ProviderGenericKernel(owner)
	if err != nil {
		return api.DeferredGenericCallableReference{}, err
	}
	if providerOwned {
		reference, referenceErr :=
			n.providerGenericKernelReference(owner, selected)
		if referenceErr != nil {
			return api.DeferredGenericCallableReference{}, referenceErr
		}
		return api.NewDeferredGenericCallableReference(
			reference,
			api.DeferredGenericRecoveryOmitted,
		)
	}
	reference, err := n.derivedSourceReference(
		owner,
		api.GenericKernelSuffix+api.DeferredEntrySuffix,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		return api.DeferredGenericCallableReference{}, err
	}
	return api.NewDeferredGenericCallableReference(
		reference,
		api.DeferredGenericRecoveryFirst,
	)
}

func (n *File) DeferredGenericCallable(
	owner *types.Func,
) (api.DeferredGenericCallableReference, error) {
	if !validGenericKernelOwner(owner) {
		return api.DeferredGenericCallableReference{}, &api.NameError{
			Reason: "deferred generic callable owner is invalid",
		}
	}
	contract, providerOwned, err := n.providerFacetOwner(owner)
	if err != nil {
		return api.DeferredGenericCallableReference{}, err
	}
	if providerOwned {
		_, recoveryOwned := n.owner.registry.provider.Facet(
			contract.Identity(),
			gostdlib.FacetRecoveryCallable,
			gostdlib.FacetCapabilityRecovery,
		)
		if recoveryOwned {
			reference, _, referenceErr := n.providerFacetReference(
				owner,
				gostdlib.FacetRecoveryCallable,
				gostdlib.FacetCapabilityRecovery,
				api.ImportPhaseValue,
			)
			if referenceErr != nil {
				return api.DeferredGenericCallableReference{}, referenceErr
			}
			return api.NewDeferredGenericCallableReference(
				reference,
				api.DeferredGenericRecoveryLast,
			)
		}
		reference, referenceErr := n.Reference(owner)
		if referenceErr != nil {
			return api.DeferredGenericCallableReference{}, referenceErr
		}
		return api.NewDeferredGenericCallableReference(
			reference,
			api.DeferredGenericRecoveryOmitted,
		)
	}
	reference, err := n.derivedSourceReference(
		owner,
		api.DeferredEntrySuffix,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		return api.DeferredGenericCallableReference{}, err
	}
	return api.NewDeferredGenericCallableReference(
		reference,
		api.DeferredGenericRecoveryFirst,
	)
}

func validGenericKernelOwner(
	owner *types.Func,
) bool {
	return owner != nil && owner.Origin() == owner
}

func (n *File) providerGenericKernelReference(
	owner *types.Func,
	selected gostdlib.Facet,
) (api.NameReference, error) {
	qualifier, request, err := n.providerImport(
		selected.ModuleSpecifier(),
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	selection, err := gostdlib.NewFacetUseSelection(
		selected.Kind(),
		gostdlib.FacetCapabilityKernel,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if err := n.observer.RequireUse(
		owner,
		environmentcontract.UseDemandCallable,
		selection,
	); err != nil {
		return api.NameReference{}, err
	}
	return api.NewProviderQualifiedNameReference(
		qualifier,
		selected.Export(),
		request,
	)
}

func (n *File) GenericConcretizationPlacement(
	owner *types.Func,
	arguments api.TypeArgumentList,
) (
	api.GeneratedArtifactPlacement,
	api.ArtifactOwner,
	*types.TypeName,
	error,
) {
	if owner == nil || owner.Origin() != owner || arguments.Len() == 0 {
		return api.GeneratedArtifactPlacementInvalid,
			api.ArtifactOwner{}, nil, &api.NameError{
				Reason: "generic concretization placement input is invalid",
			}
	}
	seen := make(map[*types.TypeName]struct{})
	var components []*types.TypeName
	for _, argument := range arguments.Values() {
		for _, component := range typeidentity.LocalComponents(argument) {
			if _, duplicate := seen[component]; duplicate {
				continue
			}
			seen[component] = struct{}{}
			components = append(components, component)
		}
	}
	placement, err := n.generatedArtifactPlacementForComponents(components)
	if err != nil {
		return api.GeneratedArtifactPlacementInvalid,
			api.ArtifactOwner{}, nil, err
	}
	return placement.kind,
		placement.lexicalOwner,
		placement.anchor,
		nil
}

func (n *File) GenericConcretization(
	concretization *api.GenericConcretization,
) (api.GenericConcretizationReference, error) {
	if !concretization.Valid() {
		return api.GenericConcretizationReference{}, &api.NameError{
			Reason: "generic concretization reference is invalid",
		}
	}
	binding, err := n.owner.registry.internGenericConcretization(
		concretization,
	)
	if err != nil {
		return api.GenericConcretizationReference{}, err
	}
	request, err := api.NewGenericConcretizationRequest(binding.owner)
	if err != nil {
		return api.GenericConcretizationReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		request,
		api.ArtifactFacetCallableSignature,
	)
	if err != nil {
		return api.GenericConcretizationReference{}, err
	}
	return api.NewGenericConcretizationReference(
		concretization,
		reference.Name(),
		reference.Requests()...,
	)
}

func (n *File) DeferredGenericConcretization(
	concretization *api.GenericConcretization,
) (api.NameReference, error) {
	if !concretization.Valid() {
		return api.NameReference{}, &api.NameError{
			Reason: "deferred generic concretization reference is invalid",
		}
	}
	binding, err := n.owner.registry.internGenericConcretization(
		concretization,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewDeferredGenericConcretizationRequest(
		binding.owner,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name+api.DeferredEntrySuffix,
		request,
		api.ArtifactFacetCallableSignature,
	)
}

func (r *Registry) internGenericConcretization(
	concretization *api.GenericConcretization,
) (genericConcretizationBinding, error) {
	if r == nil || !concretization.Valid() {
		return genericConcretizationBinding{}, &api.NameError{
			Reason: "generic concretization canonicalization input is invalid",
		}
	}
	artifactKey := concretization.Key()
	if existing, ok := r.genericConcretizations[artifactKey]; ok {
		selected, valid := existing.owner.GenericConcretization()
		if !valid || selected != concretization ||
			existing.owner.Placement() != concretization.Placement() ||
			existing.owner.LexicalOwner() != concretization.LexicalOwner() ||
			existing.owner.LexicalAnchor() != concretization.LexicalAnchor() {
			return genericConcretizationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "generic concretization key joined non-identical artifacts",
			}
		}
		return existing, nil
	}
	if len(artifactKey) < 20 {
		return genericConcretizationBinding{}, &api.NameError{
			Reason: "generic concretization artifact key is invalid",
		}
	}
	name := concretization.Owner().Name() +
		"$concrete_" + artifactKey[:20]
	if err := reserveGeneratedName(
		r.genericConcretizationNames,
		name,
		artifactKey,
		"generic concretization",
	); err != nil {
		return genericConcretizationBinding{}, err
	}
	var owner *api.GeneratedArtifact
	var err error
	if concretization.Placement() ==
		api.GeneratedArtifactPlacementLexical {
		owner, err = api.NewLexicalGenericConcretizationArtifact(
			concretization,
			artifactKey,
			name,
		)
	} else {
		outputPath, pathErr := output.GenericConcretizationPath(artifactKey)
		if pathErr != nil {
			return genericConcretizationBinding{}, pathErr
		}
		owner, err = api.NewCompilationGenericConcretizationArtifact(
			concretization,
			artifactKey,
			name,
			outputPath,
		)
	}
	if err != nil {
		return genericConcretizationBinding{}, err
	}
	binding := genericConcretizationBinding{owner: owner, name: name}
	r.genericConcretizations[artifactKey] = binding
	return binding, nil
}
