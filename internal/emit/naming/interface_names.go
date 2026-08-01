package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const interfaceTargetNameHexLength = 20

func (n *File) InterfaceAdapter(
	sourceType types.Type,
	targetType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "interface-adapter source type is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(sourceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceAdapter(
		artifactKey,
		sourceType,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceAdapterRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	var targetKey string
	var targetInterface *types.Interface
	if targetType != nil {
		targetInterface, targetKey, err =
			n.canonicalInterfaceContract(targetType)
		if err != nil {
			return api.NameReference{}, err
		}
	}
	demands, err := n.owner.registry.interfaceAdapterContractRequests(
		binding,
		targetKey,
		targetInterface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return reference.WithRequests(
		api.CombineRequests(reference.Requests(), demands)...,
	)
}

func (n *File) InterfaceContractDemand(
	sourceType types.Type,
	targetType types.Type,
) ([]api.RootRequest, error) {
	sourceInterface, sourceKey, err :=
		n.canonicalInterfaceContract(sourceType)
	if err != nil {
		return nil, err
	}
	targetInterface, targetKey, err :=
		n.canonicalInterfaceContract(targetType)
	if err != nil {
		return nil, err
	}
	return n.owner.registry.recordInterfaceContractDemand(
		sourceKey,
		sourceInterface,
		targetKey,
		targetInterface,
	)
}

func (n *File) InterfaceDynamicType(
	sourceType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(sourceType, nil),
			Reason: "interface-dynamic-type source type is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		sourceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceDynamicTypeToken(
		artifactKey,
		sourceType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceDynamicTypeTokenRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetValueSurface,
	)
}

func (n *File) ProviderInterfaceBridge(
	sourceType types.Type,
) (api.NameReference, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return api.NameReference{}, false, nil
	}
	contract, ok := named.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return api.NameReference{}, false, nil
	}
	providerInterface, providerOwned, err := n.owner.registry.ProviderInterface(
		named.Origin().Obj(),
	)
	if err != nil || !providerOwned {
		if err != nil {
			return api.NameReference{}, providerOwned, err
		}
		return api.NameReference{}, false, nil
	}
	if providerInterface.Mode() ==
		gostdlib.ProviderInterfaceModeSealedNative {
		return api.NameReference{}, false, nil
	}
	artifactKey, err := typeidentity.BuildKey(
		named,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	binding, err := n.owner.registry.internProviderInterfaceBridge(
		artifactKey,
		named,
	)
	if err != nil {
		return api.NameReference{}, true, err
	}
	requirement, err := api.NewProviderInterfaceBridgeRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, true, err
	}
	reference, err := n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
	)
	return reference, true, err
}

func (n *File) ProviderInterface(
	sourceType types.Type,
) (gostdlib.ProviderInterface, bool, error) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return gostdlib.ProviderInterface{}, false, nil
	}
	return n.owner.registry.ProviderInterface(named.Origin().Obj())
}

func (n *File) InterfaceContract(
	sourceType types.Type,
) (api.InterfaceContractReference, error) {
	if typeName, interfaceType, ok := namedInterface(sourceType); ok {
		named := types.Unalias(sourceType).(*types.Named)
		if named.TypeArgs().Len() != 0 ||
			n.providerInterfaceContract(typeName) {
			return n.generatedInterfaceContract(interfaceType)
		}
		return n.namedInterfaceContract(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.InterfaceContractReference{}, &api.NameError{
			Reason: "interface-contract source type is invalid",
		}
	}
	return n.generatedInterfaceContract(interfaceType)
}

func (n *File) generatedInterfaceContract(
	interfaceType *types.Interface,
) (api.InterfaceContractReference, error) {
	artifactKey, err := typeidentity.BuildKey(
		interfaceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(interfaceType)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousInterface(
		artifactKey,
		interfaceType,
		placement,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	requirement, err := api.NewAnonymousInterfaceRequest(binding.owner)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	requests := []api.RootRequest{requirement}
	if binding.owner.Placement() ==
		api.GeneratedArtifactPlacementCompilation {
		if n.artifactOwner.Valid() &&
			n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
			for _, facet := range []api.ArtifactFacet{
				api.ArtifactFacetInstanceTypeSurface,
				api.ArtifactFacetValueSurface,
			} {
				dependency, dependencyErr :=
					api.NewGeneratedArtifactDependencyRequest(
						binding.owner,
						facet,
					)
				if dependencyErr != nil {
					return api.InterfaceContractReference{}, dependencyErr
				}
				requests = append(requests, dependency)
			}
		}
		if binding.owner.OutputPath() != n.targetPath {
			imports, importErr := n.interfaceContractImports(
				binding.owner.OutputPath(),
				binding.name,
				"",
			)
			if importErr != nil {
				return api.InterfaceContractReference{}, importErr
			}
			requests = append(requests, imports...)
		}
	}
	return api.NewInterfaceContractReference(
		binding.name,
		interfaceContractName(binding.name),
		interfaceGuardName(binding.name),
		requests...,
	)
}

func (n *File) InterfaceType(
	sourceType types.Type,
) (api.NameReference, error) {
	if typeName, interfaceType, ok := namedInterface(sourceType); ok {
		if n.providerInterfaceContract(typeName) {
			providerInterface, _, err :=
				n.owner.registry.ProviderInterface(typeName)
			if err != nil {
				return api.NameReference{}, err
			}
			if providerInterface.Mode() ==
				gostdlib.ProviderInterfaceModeSealedNative {
				return n.TypeReference(typeName)
			}
			return n.generatedInterfaceType(interfaceType)
		}
		return n.TypeReference(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name: types.TypeString(
				sourceType,
				func(sourcePackage *types.Package) string {
					return sourcePackage.Path()
				},
			),
			Reason: "interface type is invalid",
		}
	}
	return n.generatedInterfaceType(interfaceType)
}

func (n *File) generatedInterfaceType(
	interfaceType *types.Interface,
) (api.NameReference, error) {
	artifactKey, err := typeidentity.BuildKey(
		interfaceType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(interfaceType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousInterface(
		artifactKey,
		interfaceType,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewAnonymousInterfaceRequest(binding.owner)
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
		dependency, dependencyErr :=
			api.NewGeneratedArtifactDependencyRequest(
				binding.owner,
				api.ArtifactFacetInstanceTypeSurface,
			)
		if dependencyErr != nil {
			return api.NameReference{}, dependencyErr
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
	importRequest, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseType,
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

func (n *File) providerInterfaceContract(typeName *types.TypeName) bool {
	if n == nil || n.owner == nil || n.owner.registry == nil || typeName == nil {
		return false
	}
	binding, ok := n.owner.registry.byObject[typeName]
	return ok && (binding.kind == targetBindingProvider ||
		binding.kind == targetBindingMissingProvider)
}

func (n *File) namedInterfaceContract(
	typeName *types.TypeName,
) (api.InterfaceContractReference, error) {
	typeReference, err := n.TypeReference(typeName)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	contract, err := n.derivedSourceReference(
		typeName,
		"$contract",
		api.ArtifactFacetValueSurface,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	guard, err := n.derivedSourceReference(
		typeName,
		"$is",
		api.ArtifactFacetValueSurface,
	)
	if err != nil {
		return api.InterfaceContractReference{}, err
	}
	return api.NewInterfaceContractReference(
		typeReference.Name(),
		contract.Name(),
		guard.Name(),
		api.CombineRequests(
			typeReference.Requests(),
			contract.Requests(),
			guard.Requests(),
		)...,
	)
}

func (n *File) derivedSourceReference(
	object types.Object,
	suffix string,
	facet api.ArtifactFacet,
) (api.NameReference, error) {
	if object == nil || suffix == "" {
		return api.NameReference{}, &api.NameError{
			Reason: "derived source reference is invalid",
		}
	}
	binding, ok := n.owner.byObject[object]
	if !ok {
		binding, ok = n.owner.registry.byObject[object]
	}
	if !ok || !binding.scheduled() {
		return api.NameReference{}, &api.NameError{
			Name:   object.Name(),
			Reason: "derived source reference has no declaration",
		}
	}
	if n.require != nil {
		if err := n.require(object); err != nil {
			return api.NameReference{}, err
		}
	}
	exportedName := binding.name + suffix
	var requests []api.RootRequest
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(object, facet)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if binding.sourcePath == n.targetPath {
		return api.NewNameReference(exportedName, requests...)
	}
	referencePath, crossPackage, err := n.sourceReferencePath(object, binding)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	if crossPackage {
		key := referencePath + "\x00" + exportedName
		localName = n.derivedImports[key]
		if localName == "" {
			qualifier, qualifierErr := n.packageImportQualifier(object.Pkg())
			if qualifierErr != nil {
				return api.NameReference{}, qualifierErr
			}
			localName = n.allocateImportName(exportedName, qualifier)
			n.derivedImports[key] = localName
		}
	}
	modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseValue,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, request)
	return api.NewNameReference(localName, requests...)
}

func (n *File) generatedValueReference(
	artifact *api.GeneratedArtifact,
	name string,
	requirement api.RootRequest,
	facet api.ArtifactFacet,
) (api.NameReference, error) {
	requests := []api.RootRequest{requirement}
	if artifact.Placement() == api.GeneratedArtifactPlacementLexical {
		return api.NewNameReference(name, requests...)
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(artifact) {
		dependency, err := api.NewGeneratedArtifactDependencyRequest(
			artifact,
			facet,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if artifact.OutputPath() == n.targetPath {
		return api.NewNameReference(name, requests...)
	}
	modulePath, err := output.ModuleSpecifier(
		n.targetPath,
		artifact.OutputPath(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseValue,
		modulePath,
		name,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, request)
	return api.NewNameReference(name, requests...)
}

func (n *File) interfaceContractImports(
	outputPath string,
	baseName string,
	qualifier string,
) ([]api.RootRequest, error) {
	modulePath, err := output.ModuleSpecifier(n.targetPath, outputPath)
	if err != nil {
		return nil, err
	}
	exports := []struct {
		name  string
		phase api.ImportPhase
	}{
		{baseName, api.ImportPhaseType},
		{interfaceContractName(baseName), api.ImportPhaseValue},
		{interfaceGuardName(baseName), api.ImportPhaseValue},
	}
	requests := make([]api.RootRequest, 0, len(exports))
	for _, exported := range exports {
		localName := exported.name
		if qualifier != "" {
			localName = n.allocateImportName(exported.name, qualifier)
		}
		request, requestErr := api.NewImportRequest(
			n.factory,
			exported.phase,
			modulePath,
			exported.name,
			localName,
		)
		if requestErr != nil {
			return nil, requestErr
		}
		requests = append(requests, request)
	}
	return requests, nil
}

func interfaceContractName(base string) string {
	return base + "$contract"
}

func interfaceGuardName(base string) string {
	return base + "$is"
}

func namedInterface(
	sourceType types.Type,
) (*types.TypeName, *types.Interface, bool) {
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Origin() == nil || named.Origin().Obj() == nil {
		return nil, nil, false
	}
	parameters := named.Origin().TypeParams().Len()
	arguments := named.TypeArgs().Len()
	if parameters != arguments {
		return nil, nil, false
	}
	source, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, nil, false
	}
	source = source.Complete()
	return named.Origin().Obj(), source, source.IsMethodSet()
}

func anonymousInterface(sourceType types.Type) (*types.Interface, bool) {
	source, ok := types.Unalias(sourceType).(*types.Interface)
	if !ok {
		return nil, false
	}
	source = source.Complete()
	return source, source.IsMethodSet()
}

func (n *File) canonicalInterfaceContract(
	sourceType types.Type,
) (*types.Interface, string, error) {
	var source *types.Interface
	if _, selected, ok := namedInterface(sourceType); ok {
		source = selected
	} else {
		var ok bool
		source, ok = anonymousInterface(sourceType)
		if !ok {
			return nil, "", &api.NameError{
				Name:   types.TypeString(sourceType, nil),
				Reason: "interface contract demand type is invalid",
			}
		}
	}
	key, err := typeidentity.BuildKey(
		source,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return nil, "", err
	}
	source, err = n.owner.registry.internInterfaceContract(key, source)
	if err != nil {
		return nil, "", err
	}
	return source, key, nil
}

func interfaceAdapterSource(sourceType types.Type) bool {
	if sourceType == nil {
		return false
	}
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Interface, *types.Tuple, *types.TypeParam, *types.Union:
		return false
	default:
		return true
	}
}
