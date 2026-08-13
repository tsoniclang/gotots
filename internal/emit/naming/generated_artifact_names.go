package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/generic/semanticname"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

func (n *File) DeferredCallableRegistry(
	signature *types.Signature,
) (api.NameReference, error) {
	signature, err := deferredCallableSignature(signature)
	if err != nil {
		return api.NameReference{}, err
	}
	if reason := deferredRegistrySignatureReason(signature); reason != "" {
		return api.NameReference{}, &api.NameError{
			Name:   types.TypeString(signature, packageQualifier),
			Reason: "deferred-callable registry signature is not closed: " + reason,
		}
	}
	signatureKey, err := typeidentity.BuildKey(
		signature,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	digest := sha256.Sum256([]byte("deferred-callable|" + signatureKey))
	artifactKey := hex.EncodeToString(digest[:])
	name, err := n.semanticGeneratedTypeName("$goDeferred$", signature)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internDeferredCallableRegistry(
		artifactKey,
		signature,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	definition, err := api.NewDeferredCallableRegistryRequest(binding.owner)
	if err != nil {
		return api.NameReference{}, err
	}
	return n.generatedValueReference(
		binding.owner,
		binding.name,
		definition,
		api.ArtifactFacetValueSurface,
	)
}

func deferredCallableSignature(
	signature *types.Signature,
) (*types.Signature, error) {
	if signature == nil || api.ContainsGenericTypeParameter(signature) {
		return nil, &api.NameError{
			Name:   types.TypeString(signature, packageQualifier),
			Reason: "deferred-callable registry signature is not closed",
		}
	}
	if signature.Recv() == nil {
		return signature, nil
	}
	return types.NewSignatureType(
		nil,
		nil,
		nil,
		signature.Params(),
		signature.Results(),
		signature.Variadic(),
	), nil
}

func deferredRegistrySignatureReason(signature *types.Signature) string {
	if signature == nil {
		return "nil"
	}
	if signature.Recv() != nil {
		return "receiver is present"
	}
	if api.ContainsGenericTypeParameter(signature) {
		return "generic type parameter is present"
	}
	return ""
}

func packageQualifier(source *types.Package) string {
	if source == nil {
		return ""
	}
	return source.Path()
}

func (r *Registry) internDeferredCallableRegistry(
	artifactKey string,
	signature *types.Signature,
	name string,
) (deferredCallableRegistryBinding, error) {
	if r == nil ||
		len(artifactKey) != sha256.Size*2 ||
		signature == nil ||
		signature.Recv() != nil ||
		api.ContainsGenericTypeParameter(signature) ||
		name == "" {
		return deferredCallableRegistryBinding{}, &api.NameError{
			Reason: "deferred-callable registry identity is invalid",
		}
	}
	if existing, ok := r.deferredCallableRegistries[artifactKey]; ok {
		existingSignature, valid := existing.owner.DeferredCallableRegistry()
		if !valid || !types.Identical(existingSignature, signature) {
			return deferredCallableRegistryBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "deferred-callable registry key joined non-identical signatures",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.deferredCallableRegistryNames,
		name,
		artifactKey,
		"deferred-callable registry",
	); err != nil {
		return deferredCallableRegistryBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactDeferredCallableRegistry,
		signature,
		artifactKey,
		name,
		output.DeferredCallableRegistrySupportPath,
	)
	if err != nil {
		return deferredCallableRegistryBinding{}, err
	}
	binding := deferredCallableRegistryBinding{owner: owner, name: name}
	r.deferredCallableRegistries[artifactKey] = binding
	return binding, nil
}

type generatedInterfaceContractImports struct {
	typeName     string
	contractName string
	guardName    string
	requests     []api.RootRequest
}

func (n *File) generatedArtifactLocalName(
	artifact *api.GeneratedArtifact,
	exported string,
) string {
	identity := generatedArtifactImport{artifact: artifact, exported: exported}
	if selected := n.artifactImports[identity]; selected != "" {
		return selected
	}
	selected := generatedArtifactPreferredLocalName(artifact, exported)
	if n.lexicalNameExists(selected) {
		if selected != exported && !n.lexicalNameExists(exported) {
			selected = exported
		} else {
			selected = n.allocateImportName(
				selected,
				generatedArtifactImportQualifier(artifact),
			)
		}
	} else {
		n.importNames[selected] = struct{}{}
	}
	n.artifactImports[identity] = selected
	return selected
}

func generatedArtifactPreferredLocalName(
	artifact *api.GeneratedArtifact,
	exported string,
) string {
	if artifact == nil || exported == "" {
		return exported
	}
	base := artifact.TargetName()
	if !strings.HasPrefix(exported, base) {
		return exported
	}
	suffix := strings.TrimPrefix(exported, base)
	preferred := ""
	switch artifact.Kind() {
	case api.GeneratedArtifactMapSpecialization:
		preferred = "GoMap"
	case api.GeneratedArtifactInterfaceAdapter:
		preferred = "GoInterfaceAdapter"
	case api.GeneratedArtifactAnonymousInterface:
		preferred = "GoInterface"
	case api.GeneratedArtifactProviderInterfaceBridge:
		preferred = "GoProviderInterfaceBridge"
		if strings.HasPrefix(base, "$goProviderProfileBridge$") {
			preferred = "GoProviderProfileBridge"
		}
	case api.GeneratedArtifactProviderStatefulRepresentation:
		preferred = "GoProviderState"
	case api.GeneratedArtifactDeferredCallableRegistry:
		preferred = "DeferredCallableRegistry"
	default:
		return exported
	}
	return preferred + suffix
}

func (n *File) interfaceContractImports(
	artifact *api.GeneratedArtifact,
	baseName string,
) (generatedInterfaceContractImports, error) {
	modulePath, err := output.ModuleSpecifier(n.targetPath, artifact.OutputPath())
	if err != nil {
		return generatedInterfaceContractImports{}, err
	}
	exports := []struct {
		name  string
		phase api.ImportPhase
	}{
		{baseName, api.ImportPhaseType},
		{interfaceContractName(baseName), api.ImportPhaseValue},
		{interfaceGuardName(baseName), api.ImportPhaseValue},
	}
	result := generatedInterfaceContractImports{
		requests: make([]api.RootRequest, 0, len(exports)),
	}
	localNames := []*string{
		&result.typeName,
		&result.contractName,
		&result.guardName,
	}
	for index, exported := range exports {
		localName := n.generatedArtifactLocalName(artifact, exported.name)
		request, requestErr := api.NewImportRequest(
			n.factory,
			exported.phase,
			modulePath,
			exported.name,
			localName,
		)
		if requestErr != nil {
			return generatedInterfaceContractImports{}, requestErr
		}
		*localNames[index] = localName
		result.requests = append(result.requests, request)
	}
	return result, nil
}

func (n *File) semanticGeneratedTypeName(
	prefix string,
	sourceType types.Type,
) (string, error) {
	semanticType, err := semanticname.TypeWithIdentityTokens(
		sourceType,
		n.generatedNamedObjectToken,
		n.generatedPackageToken,
	)
	if err != nil {
		return "", err
	}
	return prefix + semanticType, nil
}

func (n *File) generatedNamedObjectToken(
	object *types.TypeName,
) (string, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil || object == nil {
		return "", &api.NameError{
			Name:   objectName(object),
			Reason: "generated-artifact named token owner is invalid",
		}
	}
	if object.Pkg() == nil {
		return semanticname.Identifier(object.Name()), nil
	}
	if object.Parent() == object.Pkg().Scope() {
		qualifier := n.owner.registry.ImportQualifier(object.Pkg())
		if qualifier == "" {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generated-artifact named token has no package qualifier",
			}
		}
		return qualifier + "$" +
			semanticname.Identifier(object.Name()), nil
	}
	targetName, ok := n.owner.targetNameByObject[object]
	if !ok || targetName == "" {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "generated-artifact local token has no target name",
		}
	}
	return targetName, nil
}

func (n *File) generatedPackageToken(
	sourcePackage *types.Package,
) (string, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil {
		return "", &api.NameError{
			Reason: "generated-artifact package token owner is invalid",
		}
	}
	return n.owner.registry.semanticPackageToken(sourcePackage)
}

func semanticGeneratedTypeName(
	prefix string,
	sourceType types.Type,
	localIdentity semanticname.LocalNamedIdentity,
) (string, error) {
	var (
		semanticType string
		err          error
	)
	if len(typeidentity.LocalComponents(sourceType)) == 0 {
		semanticType, err = semanticname.Type(sourceType)
	} else {
		semanticType, err = semanticname.TypeWithLocalIdentity(
			sourceType,
			localIdentity,
		)
	}
	if err != nil {
		return "", err
	}
	return prefix + semanticType, nil
}

func (n *File) semanticGeneratedMethodName(
	prefix string,
	method *types.Func,
	signature *types.Signature,
) (string, error) {
	if method == nil || signature == nil {
		return "", &api.NameError{
			Reason: "semantic generated method name is invalid",
		}
	}
	contract, err := n.semanticGeneratedTypeName("", signature)
	if err != nil {
		return "", err
	}
	identity := semanticname.Identifier(
		types.Id(method.Origin().Pkg(), method.Origin().Name()),
	)
	return prefix + identity + "$" + contract, nil
}
