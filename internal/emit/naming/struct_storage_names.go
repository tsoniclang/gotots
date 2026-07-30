package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

func (n *File) PointerRepresentation(
	pointer *types.Pointer,
) (api.PointerRepresentationReference, error) {
	if pointer == nil {
		return api.PointerRepresentationReference{}, &api.NameError{
			Reason: "pointer-representation type is nil",
		}
	}
	key, err := typeidentity.BuildParameterizedKey(
		pointer,
		n.generatedNamedObjectIdentity,
		func(parameter *types.TypeParam) (string, error) {
			if parameter == nil || parameter.Obj() == nil {
				return "", &api.NameError{
					Reason: "pointer representation has an unbound type parameter",
				}
			}
			return n.generatedNamedObjectIdentity(parameter.Obj())
		},
	)
	if err != nil {
		return api.PointerRepresentationReference{}, err
	}
	digest := sha256.Sum256([]byte("pointer-representation|" + key))
	artifactKey := hex.EncodeToString(digest[:])
	binding, err := n.owner.registry.internPointerRepresentation(
		artifactKey,
		pointer,
	)
	if err != nil {
		return api.PointerRepresentationReference{}, err
	}
	definition, err := api.NewPointerRepresentationRequest(
		binding.owner,
		false,
	)
	if err != nil {
		return api.PointerRepresentationReference{}, err
	}
	return api.NewPointerRepresentationReference(
		binding.owner,
		definition,
	)
}

func (r *Registry) internPointerRepresentation(
	artifactKey string,
	pointer *types.Pointer,
) (pointerRepresentationBinding, error) {
	if r == nil || len(artifactKey) != sha256.Size*2 || pointer == nil {
		return pointerRepresentationBinding{}, &api.NameError{
			Reason: "pointer-representation canonicalization input is invalid",
		}
	}
	if existing, ok := r.pointerRepresentations[artifactKey]; ok {
		bound, valid := existing.owner.PointerRepresentation()
		if !valid || !types.Identical(bound, pointer) {
			return pointerRepresentationBinding{}, &api.NameError{
				Reason: "pointer-representation key joined non-identical types",
			}
		}
		return existing, nil
	}
	name := "$goPointer_" + artifactKey[len(artifactKey)-20:]
	owner, err := api.NewContractGeneratedArtifact(
		api.GeneratedArtifactPointerRepresentation,
		pointer,
		artifactKey,
		name,
	)
	if err != nil {
		return pointerRepresentationBinding{}, err
	}
	binding := pointerRepresentationBinding{owner: owner}
	r.pointerRepresentations[artifactKey] = binding
	return binding, nil
}

func (n *File) NamedStructStorage(
	typeName *types.TypeName,
) (api.NameReference, error) {
	request, err := n.namedStructOperationRequest(
		typeName,
		api.NamedStructOperationStorage,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, ok := n.owner.byObject[typeName]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[typeName]
	}
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   objectName(typeName),
			Reason: "struct storage owner has no emitted declaration",
		}
	}
	if binding.scheduled() && n.require != nil {
		if err := n.require(typeName); err != nil {
			return api.NameReference{}, err
		}
	}
	exportedName := binding.name + api.StructStorageTypeSuffix
	localName := exportedName
	requests := []api.RootRequest{request}
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(
			typeName,
			api.ArtifactFacetStaticSurface,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if !binding.scheduled() || binding.sourcePath == n.targetPath {
		return api.NewNameReference(localName, requests...)
	}
	referencePath, _, err := n.sourceReferencePath(typeName, binding)
	if err != nil {
		return api.NameReference{}, err
	}
	modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
	if err != nil {
		return api.NameReference{}, err
	}
	baseLocalName, err := n.importName(typeName, binding.name)
	if err != nil {
		return api.NameReference{}, err
	}
	localName = baseLocalName + api.StructStorageTypeSuffix
	importRequest, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseType,
		modulePath,
		exportedName,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, importRequest)
	return api.NewNameReference(localName, requests...)
}

func (n *File) AnonymousStructStorage(
	structType *types.Struct,
) (api.NameReference, error) {
	if structType == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "anonymous-struct storage type is nil",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		structType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(structType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousStruct(
		artifactKey,
		structType,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewAnonymousStructRequest(
		binding.owner,
		api.AnonymousStructDemandStorage,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name := binding.name + api.StructStorageTypeSuffix
	requests := []api.RootRequest{requirement}
	if binding.owner.Placement() == api.GeneratedArtifactPlacementLexical {
		return api.NewNameReference(name, requests...)
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
		dependency, err := api.NewGeneratedArtifactDependencyRequest(
			binding.owner,
			api.ArtifactFacetStaticSurface,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if binding.owner.OutputPath() == n.targetPath {
		return api.NewNameReference(name, requests...)
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
		name,
		name,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests = append(requests, importRequest)
	return api.NewNameReference(name, requests...)
}
