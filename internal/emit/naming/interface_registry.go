package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"
	"slices"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

func (r *Registry) internInterfaceAdapter(
	artifactKey string,
	sourceType types.Type,
	name string,
	reflectionName string,
	placement generatedArtifactPlacement,
) (interfaceAdapterBinding, error) {
	if r == nil ||
		!interfaceAdapterSource(sourceType) ||
		artifactKey == "" ||
		name == "" ||
		reflectionName == "" ||
		!placement.kind.Valid() {
		return interfaceAdapterBinding{}, &api.NameError{
			Reason: "interface-adapter canonicalization input is invalid",
		}
	}
	if existing, ok := r.interfaceAdapters[artifactKey]; ok {
		existingType, valid := existing.owner.InterfaceAdapterType()
		if !valid || !types.Identical(existingType, sourceType) {
			return interfaceAdapterBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "interface-adapter key joined non-identical Go types",
			}
		}
		if !sameGeneratedPlacement(existing.owner, placement) {
			return interfaceAdapterBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "interface-adapter placement is inconsistent",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.interfaceAdapterNames,
		name,
		artifactKey,
		"interface-adapter",
	); err != nil {
		return interfaceAdapterBinding{}, err
	}
	owner, err := newInterfaceAdapterArtifact(
		sourceType,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return interfaceAdapterBinding{}, err
	}
	binding := interfaceAdapterBinding{
		owner:          owner,
		name:           name,
		reflectionName: reflectionName,
		key:            artifactKey,
	}
	r.interfaceAdapters[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internAnonymousInterface(
	artifactKey string,
	interfaceType *types.Interface,
	name string,
	placement generatedArtifactPlacement,
) (anonymousInterfaceBinding, error) {
	if r == nil ||
		interfaceType == nil ||
		!interfaceType.Complete().IsMethodSet() ||
		artifactKey == "" ||
		name == "" ||
		!placement.kind.Valid() {
		return anonymousInterfaceBinding{}, &api.NameError{
			Reason: "anonymous-interface canonicalization input is invalid",
		}
	}
	if existing, ok := r.anonymousInterfaces[artifactKey]; ok {
		existingType, valid := existing.owner.InterfaceType()
		if !valid || !types.Identical(existingType, interfaceType) {
			return anonymousInterfaceBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "anonymous-interface key joined non-identical Go types",
			}
		}
		if !sameGeneratedPlacement(existing.owner, placement) {
			return anonymousInterfaceBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "anonymous-interface placement is inconsistent",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.anonymousInterfaceNames,
		name,
		artifactKey,
		"anonymous-interface",
	); err != nil {
		return anonymousInterfaceBinding{}, err
	}
	owner, err := newAnonymousInterfaceArtifact(
		interfaceType,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return anonymousInterfaceBinding{}, err
	}
	binding := anonymousInterfaceBinding{owner: owner, name: name}
	r.anonymousInterfaces[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internProviderInterfaceBridge(
	artifactKey string,
	sourceType *types.Named,
	name string,
) (providerInterfaceBridgeBinding, error) {
	if r == nil || sourceType == nil || sourceType.Obj() == nil ||
		artifactKey == "" || name == "" {
		return providerInterfaceBridgeBinding{}, &api.NameError{
			Reason: "provider-interface bridge canonicalization input is invalid",
		}
	}
	contract, ok := sourceType.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return providerInterfaceBridgeBinding{}, &api.NameError{
			Name:   sourceType.Obj().Name(),
			Reason: "provider-interface bridge source is not an interface",
		}
	}
	if existing, found := r.providerInterfaceBridges[artifactKey]; found {
		existingType, valid := existing.owner.ProviderInterfaceBridgeType()
		if !valid || !types.Identical(existingType, sourceType) {
			return providerInterfaceBridgeBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "provider-interface bridge key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.providerInterfaceBridgeNames,
		name,
		artifactKey,
		"provider-interface bridge",
	); err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactProviderInterfaceBridge,
		sourceType,
		artifactKey,
		name,
		output.ProviderInterfaceBridgeSupportPath,
	)
	if err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	binding := providerInterfaceBridgeBinding{
		owner: owner,
		name:  name,
		key:   artifactKey,
	}
	r.providerInterfaceBridges[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internProviderProfileInterfaceBridge(
	sourceKey string,
	sourceType *types.Named,
	name string,
	profile []gostdlib.ProviderCallableProfileInterface,
) (providerInterfaceBridgeBinding, error) {
	if r == nil || sourceType == nil || sourceType.Obj() == nil ||
		sourceKey == "" || name == "" || len(profile) == 0 {
		return providerInterfaceBridgeBinding{}, &api.NameError{
			Reason: "provider-profile bridge canonicalization input is invalid",
		}
	}
	contract, ok := sourceType.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return providerInterfaceBridgeBinding{}, &api.NameError{
			Name:   sourceType.Obj().Name(),
			Reason: "provider-profile bridge source is not an interface",
		}
	}
	selected, err := providerProfileBridgeClosure(sourceType, profile)
	if err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	semanticParts, err := providerProfileBridgeSemanticParts(selected)
	if err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	var descriptor strings.Builder
	descriptor.WriteString(sourceKey)
	descriptor.WriteByte(0)
	descriptor.WriteString(strings.Join(semanticParts, "$And$"))
	digest := sha256.Sum256([]byte(descriptor.String()))
	artifactKey := hex.EncodeToString(digest[:])
	if existing, found := r.providerInterfaceBridges[artifactKey]; found {
		existingType, boundProfile, valid := existing.owner.ProviderProfileInterfaceBridge()
		if !valid || !types.Identical(existingType, sourceType) ||
			!sameProviderProfileInterfaces(boundProfile, selected) {
			return providerInterfaceBridgeBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "provider-profile bridge key joined inconsistent evidence",
			}
		}
		return existing, nil
	}
	name, err = providerProfileBridgeTargetName(name, selected)
	if err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	if err := reserveGeneratedName(
		r.providerInterfaceBridgeNames,
		name,
		artifactKey,
		"provider-profile bridge",
	); err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	owner, err := api.NewCompilationProviderProfileBridgeArtifact(
		sourceType,
		selected,
		artifactKey,
		name,
		output.ProviderInterfaceBridgeSupportPath,
	)
	if err != nil {
		return providerInterfaceBridgeBinding{}, err
	}
	binding := providerInterfaceBridgeBinding{
		owner: owner,
		name:  name,
		key:   artifactKey,
	}
	r.providerInterfaceBridges[artifactKey] = binding
	return binding, nil
}

func providerProfileBridgeTargetName(
	base string,
	selected []gostdlib.ProviderCallableProfileInterface,
) (string, error) {
	if base == "" || len(selected) == 0 {
		return "", &api.NameError{
			Reason: "provider-profile bridge semantic name is invalid",
		}
	}
	parts, err := providerProfileBridgeNameParts(selected)
	if err != nil {
		return "", err
	}
	return base + "$Using$" + strings.Join(parts, "$And$"), nil
}

func sameProviderProfileInterfaces(
	left []gostdlib.ProviderCallableProfileInterface,
	right []gostdlib.ProviderCallableProfileInterface,
) bool {
	if len(left) != len(right) {
		return false
	}
	leftParts, leftErr := providerProfileBridgeSemanticParts(left)
	rightParts, rightErr := providerProfileBridgeSemanticParts(right)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return slices.Equal(leftParts, rightParts)
}

func (r *Registry) internInterfaceMethodToken(
	artifactKey string,
	method *types.Func,
	signature *types.Signature,
	name string,
	runtime api.RuntimeSymbol,
) (interfaceMethodTokenBinding, error) {
	if r == nil ||
		method == nil ||
		signature == nil ||
		signature.Recv() != nil ||
		artifactKey == "" ||
		name == "" {
		return interfaceMethodTokenBinding{}, &api.NameError{
			Reason: "interface-method-token canonicalization input is invalid",
		}
	}
	if existing, ok := r.interfaceMethodTokens[artifactKey]; ok {
		_, valid :=
			existing.owner.InterfaceMethodSignature()
		existingRuntime, runtimeValid :=
			existing.owner.InterfaceMethodRuntime()
		if !valid ||
			!runtimeValid ||
			existingRuntime != runtime ||
			!environmentcontract.EquivalentMethods(existing.method, method) {
			return interfaceMethodTokenBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "interface-method-token key joined non-identical contracts",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.interfaceMethodNames,
		name,
		artifactKey,
		"interface-method-token",
	); err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	owner, err := api.NewCompilationInterfaceMethodTokenArtifact(
		signature,
		artifactKey,
		name,
		output.InterfaceMethodSupportPath,
		runtime,
	)
	if err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	binding := interfaceMethodTokenBinding{
		owner:  owner,
		method: method,
		name:   name,
	}
	r.interfaceMethodTokens[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internInterfaceMethodCallable(
	artifactKey string,
	method *types.Func,
	signature *types.Signature,
	name string,
) (interfaceMethodCallableBinding, error) {
	if r == nil ||
		method == nil ||
		signature == nil ||
		signature.Recv() != nil ||
		artifactKey == "" ||
		name == "" {
		return interfaceMethodCallableBinding{}, &api.NameError{
			Reason: "interface-method callable canonicalization input is invalid",
		}
	}
	if existing, ok := r.interfaceMethodCallables[artifactKey]; ok {
		_, valid := existing.owner.InterfaceMethodCallableSignature()
		if !valid ||
			!environmentcontract.EquivalentMethods(existing.method, method) {
			return interfaceMethodCallableBinding{}, &api.NameError{
				Name: existing.name,
				Reason: "interface-method callable key joined " +
					"non-identical contracts",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.interfaceMethodCallableNames,
		name,
		artifactKey,
		"interface-method callable",
	); err != nil {
		return interfaceMethodCallableBinding{}, err
	}
	owner, err := api.NewContractGeneratedArtifact(
		api.GeneratedArtifactInterfaceMethodCallable,
		signature,
		artifactKey,
		name,
	)
	if err != nil {
		return interfaceMethodCallableBinding{}, err
	}
	binding := interfaceMethodCallableBinding{
		owner:  owner,
		method: method,
		name:   name,
	}
	r.interfaceMethodCallables[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internInterfaceDynamicTypeToken(
	artifactKey string,
	sourceType types.Type,
	name string,
) (interfaceDynamicTypeTokenBinding, error) {
	if r == nil ||
		!interfaceAdapterSource(sourceType) ||
		artifactKey == "" ||
		name == "" {
		return interfaceDynamicTypeTokenBinding{}, &api.NameError{
			Reason: "interface-dynamic-type canonicalization input is invalid",
		}
	}
	if existing, ok := r.interfaceDynamicTypes[artifactKey]; ok {
		existingType, valid := existing.owner.InterfaceDynamicType()
		if !valid || !types.Identical(existingType, sourceType) {
			return interfaceDynamicTypeTokenBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "interface-dynamic-type key joined non-identical Go types",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedName(
		r.interfaceDynamicNames,
		name,
		artifactKey,
		"interface-dynamic-type",
	); err != nil {
		return interfaceDynamicTypeTokenBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactInterfaceDynamicTypeToken,
		sourceType,
		artifactKey,
		name,
		output.InterfaceTypeSupportPath,
	)
	if err != nil {
		return interfaceDynamicTypeTokenBinding{}, err
	}
	binding := interfaceDynamicTypeTokenBinding{owner: owner, name: name}
	r.interfaceDynamicTypes[artifactKey] = binding
	return binding, nil
}

func reserveGeneratedName(
	names map[string]string,
	name string,
	artifactKey string,
	kind string,
) error {
	if existing := names[name]; existing != "" && existing != artifactKey {
		return &api.NameError{
			Name:   name,
			Reason: kind + " target-name collision",
		}
	}
	names[name] = artifactKey
	return nil
}

func newInterfaceAdapterArtifact(
	sourceType types.Type,
	artifactKey string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			api.GeneratedArtifactInterfaceAdapter,
			sourceType,
			artifactKey,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactInterfaceAdapter,
		sourceType,
		artifactKey,
		name,
		output.InterfaceAdapterSupportPath,
	)
}

func newAnonymousInterfaceArtifact(
	interfaceType *types.Interface,
	artifactKey string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			api.GeneratedArtifactAnonymousInterface,
			interfaceType,
			artifactKey,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactAnonymousInterface,
		interfaceType,
		artifactKey,
		name,
		output.AnonymousInterfaceSupportPath,
	)
}

func sameGeneratedPlacement(
	artifact *api.GeneratedArtifact,
	placement generatedArtifactPlacement,
) bool {
	return artifact.Placement() == placement.kind &&
		artifact.LexicalOwner() == placement.lexicalOwner &&
		artifact.LexicalAnchor() == placement.anchor
}
