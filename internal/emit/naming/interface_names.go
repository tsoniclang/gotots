package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

const interfaceTargetNameHexLength = 20

func (n *File) InterfaceAdapter(
	sourceType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
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
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		requirement,
		api.ArtifactFacetConstructorSurface,
	)
}

func (n *File) InterfaceDynamicType(
	sourceType types.Type,
) (api.NameReference, error) {
	if !interfaceAdapterSource(sourceType) {
		return api.NameReference{}, &api.NameError{
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

func (n *File) InterfaceContract(
	sourceType types.Type,
) (api.InterfaceContractReference, error) {
	if typeName, _, ok := namedInterface(sourceType); ok {
		return n.namedInterfaceContract(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.InterfaceContractReference{}, &api.NameError{
			Reason: "interface-contract source type is invalid",
		}
	}
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
	if typeName, _, ok := namedInterface(sourceType); ok {
		return n.TypeReference(typeName)
	}
	interfaceType, ok := anonymousInterface(sourceType)
	if !ok {
		return api.NameReference{}, &api.NameError{
			Reason: "interface type is invalid",
		}
	}
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

func (n *File) InterfaceMethodName(method *types.Func) (string, error) {
	if method == nil {
		return "", &api.NameError{Reason: "interface method is nil"}
	}
	if _, ok := methodidentity.Signature(method); !ok {
		return "", &api.NameError{
			Name:   method.Name(),
			Reason: "interface method signature is invalid",
		}
	}
	if method.Exported() {
		return portableIdentifier(method.Name()), nil
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return "", err
	}
	return "$go$private_" + artifactKey[:interfaceTargetNameHexLength], nil
}

func (n *File) InterfaceMethodToken(
	method *types.Func,
) (api.NameReference, error) {
	if symbol, ok := runtimeInterfaceMethodToken(method); ok {
		return n.Runtime(symbol, api.ImportPhaseValue)
	}
	signature, ok := methodidentity.Signature(method)
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   objectName(method),
			Reason: "interface method signature is invalid",
		}
	}
	artifactKey, err := methodidentity.BuildKey(
		method,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internInterfaceMethodToken(
		artifactKey,
		method,
		signature,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewInterfaceMethodTokenRequest(binding.owner)
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

func runtimeInterfaceMethodToken(
	method *types.Func,
) (api.RuntimeSymbol, bool) {
	switch interfacecontract.Method(method) {
	case interfacecontract.MethodError:
		return api.RuntimeErrorMethodToken, true
	case interfacecontract.MethodRuntimeError:
		return api.RuntimeRuntimeErrorToken, true
	default:
		return api.RuntimeInvalid, false
	}
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
	if !ok || binding.sourceFile == nil {
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
	if n.artifactOwner.Valid() {
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
	if !ok || named.Obj() == nil || named.TypeArgs().Len() != 0 {
		return nil, nil, false
	}
	source, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, nil, false
	}
	source = source.Complete()
	return named.Obj(), source, source.IsMethodSet()
}

func anonymousInterface(sourceType types.Type) (*types.Interface, bool) {
	source, ok := types.Unalias(sourceType).(*types.Interface)
	if !ok {
		return nil, false
	}
	source = source.Complete()
	return source, source.IsMethodSet()
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
