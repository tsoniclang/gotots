package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/methodidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

func (r *Registry) internInterfaceAdapter(
	artifactKey string,
	sourceType types.Type,
	placement generatedArtifactPlacement,
) (interfaceAdapterBinding, error) {
	if r == nil ||
		!interfaceAdapterSource(sourceType) ||
		artifactKey == "" ||
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
	name, err := interfaceTargetName("$goInterfaceAdapter_", artifactKey)
	if err != nil {
		return interfaceAdapterBinding{}, err
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
		owner: owner,
		name:  name,
		key:   artifactKey,
	}
	r.interfaceAdapters[artifactKey] = binding
	return binding, nil
}

func (r *Registry) internAnonymousInterface(
	artifactKey string,
	interfaceType *types.Interface,
	placement generatedArtifactPlacement,
) (anonymousInterfaceBinding, error) {
	if r == nil ||
		interfaceType == nil ||
		!interfaceType.Complete().IsMethodSet() ||
		artifactKey == "" ||
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
	name, err := interfaceTargetName("$goInterface_", artifactKey)
	if err != nil {
		return anonymousInterfaceBinding{}, err
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

func (r *Registry) internInterfaceMethodToken(
	artifactKey string,
	method *types.Func,
	signature *types.Signature,
) (interfaceMethodTokenBinding, error) {
	if r == nil ||
		method == nil ||
		signature == nil ||
		signature.Recv() != nil ||
		artifactKey == "" {
		return interfaceMethodTokenBinding{}, &api.NameError{
			Reason: "interface-method-token canonicalization input is invalid",
		}
	}
	if existing, ok := r.interfaceMethodTokens[artifactKey]; ok {
		existingSignature, valid :=
			existing.owner.InterfaceMethodSignature()
		if !valid ||
			!types.Identical(existingSignature, signature) ||
			!methodidentity.Equivalent(existing.method, method) {
			return interfaceMethodTokenBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "interface-method-token key joined non-identical contracts",
			}
		}
		return existing, nil
	}
	name, err := interfaceTargetName("$goInterfaceMethod_", artifactKey)
	if err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	if err := reserveGeneratedName(
		r.interfaceMethodNames,
		name,
		artifactKey,
		"interface-method-token",
	); err != nil {
		return interfaceMethodTokenBinding{}, err
	}
	owner, err := api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactInterfaceMethodToken,
		signature,
		artifactKey,
		name,
		output.InterfaceMethodSupportPath,
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

func (r *Registry) internInterfaceDynamicTypeToken(
	artifactKey string,
	sourceType types.Type,
) (interfaceDynamicTypeTokenBinding, error) {
	if r == nil ||
		!interfaceAdapterSource(sourceType) ||
		artifactKey == "" {
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
	name, err := interfaceTargetName("$goDynamicType_", artifactKey)
	if err != nil {
		return interfaceDynamicTypeTokenBinding{}, err
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

func interfaceTargetName(prefix string, artifactKey string) (string, error) {
	if len(artifactKey) < interfaceTargetNameHexLength {
		return "", &api.NameError{
			Reason: "interface artifact key is invalid",
		}
	}
	return prefix + artifactKey[:interfaceTargetNameHexLength], nil
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
			Reason: kind + " target-name prefix collision",
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
	outputPath, err := output.InterfaceAdapterPath(artifactKey)
	if err != nil {
		return nil, err
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactInterfaceAdapter,
		sourceType,
		artifactKey,
		name,
		outputPath,
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
	outputPath, err := output.AnonymousInterfacePath(artifactKey)
	if err != nil {
		return nil, err
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactAnonymousInterface,
		interfaceType,
		artifactKey,
		name,
		outputPath,
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
