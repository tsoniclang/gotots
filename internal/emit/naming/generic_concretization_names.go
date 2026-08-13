package naming

import (
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
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

func generatedArtifactNameScope(
	name string,
	placement generatedArtifactPlacement,
	module string,
) genericGeneratedNameScope {
	scope := genericGeneratedNameScope{
		placement:    placement.kind,
		lexicalOwner: placement.lexicalOwner,
		anchor:       placement.anchor,
		name:         name,
	}
	if placement.kind == api.GeneratedArtifactPlacementCompilation {
		scope.module = module
	}
	return scope
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
		return n.providerGenericKernelReference(
			owner,
			selected,
			gostdlib.FacetCapabilityKernel,
		)
	}
	return n.derivedSourceReference(
		owner,
		api.GenericKernelSuffix,
		api.ArtifactFacetCallableSignature,
	)
}

func (n *File) SynchronousGenericKernel(
	owner *types.Func,
) (api.NameReference, error) {
	if !validGenericKernelOwner(owner) {
		return api.NameReference{}, &api.NameError{
			Reason: "synchronous generic kernel owner is invalid",
		}
	}
	selected, providerOwned, err :=
		n.owner.registry.ProviderSynchronousGenericKernel(owner)
	if err != nil {
		return api.NameReference{}, err
	}
	if !providerOwned {
		return api.NameReference{}, &api.NameError{
			Name:   owner.Name(),
			Reason: "synchronous generic kernel is not certified",
		}
	}
	return n.providerGenericKernelReference(
		owner,
		selected,
		gostdlib.FacetCapabilitySynchronousKernel,
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
			n.providerGenericKernelReference(
				owner,
				selected,
				gostdlib.FacetCapabilityKernel,
			)
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
	capability gostdlib.FacetCapability,
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
		capability,
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
	suffix, err := n.genericConcretizationSuffix(concretization)
	if err != nil {
		return api.GenericConcretizationReference{}, err
	}
	binding, err := n.owner.registry.internGenericConcretization(
		concretization,
		suffix,
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
	canonical, ok := binding.owner.GenericConcretization()
	if !ok {
		return api.GenericConcretizationReference{}, &api.NameError{
			Name:   binding.name,
			Reason: "generic concretization canonical binding is invalid",
		}
	}
	return api.NewGenericConcretizationReference(
		canonical,
		reference.Name(),
		binding.suffix,
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
	suffix, err := n.genericConcretizationSuffix(concretization)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internGenericConcretization(
		concretization,
		suffix,
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
	suffix string,
) (genericConcretizationBinding, error) {
	if r == nil || !concretization.Valid() || suffix == "" {
		return genericConcretizationBinding{}, &api.NameError{
			Reason: "generic concretization canonicalization input is invalid",
		}
	}
	artifactKey := concretization.Key()
	if existing, ok := r.genericConcretizations[artifactKey]; ok {
		selected, valid := existing.owner.GenericConcretization()
		if !valid || !selected.Identical(concretization) ||
			existing.suffix != suffix {
			return genericConcretizationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "generic concretization key joined non-identical artifacts",
			}
		}
		return existing, nil
	}
	name, err := semanticname.ConcretizationName(
		concretization.Owner(),
		suffix,
	)
	if err != nil {
		return genericConcretizationBinding{}, err
	}
	module := ""
	if concretization.Placement() ==
		api.GeneratedArtifactPlacementCompilation {
		var moduleErr error
		module, moduleErr = semanticname.ConcretizationModule(
			concretization.Owner(),
		)
		if moduleErr != nil {
			return genericConcretizationBinding{}, moduleErr
		}
	}
	if err := reserveGenericGeneratedName(
		r.genericConcretizationNames,
		genericGeneratedNameScope{
			placement:    concretization.Placement(),
			lexicalOwner: concretization.LexicalOwner(),
			anchor:       concretization.LexicalAnchor(),
			module:       module,
			name:         name,
		},
		artifactKey,
		"generic concretization",
	); err != nil {
		return genericConcretizationBinding{}, err
	}
	if concretization.Placement() ==
		api.GeneratedArtifactPlacementCompilation {
		if err := reserveGenericConcretizationModule(
			r.genericConcretizationModules,
			genericGeneratedModuleScope{
				placement: concretization.Placement(),
				module:    module,
			},
			concretization.Owner(),
		); err != nil {
			return genericConcretizationBinding{}, err
		}
	}
	var owner *api.GeneratedArtifact
	if concretization.Placement() ==
		api.GeneratedArtifactPlacementLexical {
		owner, err = api.NewLexicalGenericConcretizationArtifact(
			concretization,
			artifactKey,
			name,
		)
	} else {
		outputPath, pathErr := output.GenericConcretizationPath(module)
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
	binding := genericConcretizationBinding{
		owner:  owner,
		name:   name,
		suffix: suffix,
	}
	r.genericConcretizations[artifactKey] = binding
	return binding, nil
}

func (n *File) genericConcretizationSuffix(
	concretization *api.GenericConcretization,
) (string, error) {
	if n == nil || !concretization.Valid() {
		return "", &api.NameError{
			Reason: "generic concretization suffix owner is invalid",
		}
	}
	return semanticname.ConcretizationSuffixWithIdentityTokens(
		concretization.Arguments(),
		concretization.Effect().Synchronous(),
		n.generatedNamedObjectToken,
		n.generatedPackageToken,
	)
}
