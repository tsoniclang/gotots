package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

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
