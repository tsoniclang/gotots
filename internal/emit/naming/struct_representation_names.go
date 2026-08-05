package naming

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
	"go/types"
)

func (n *File) DefinedValueRepresentation(
	typeName *types.TypeName,
) (api.DefinedValueRepresentation, error) {
	if typeName == nil {
		return api.DefinedValueRepresentation{}, &api.NameError{
			Reason: "defined-value owner is nil",
		}
	}
	binding, ok := n.owner.byObject[typeName]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[typeName]
	}
	if !ok || binding.kind != targetBindingProvider {
		return api.NewDefinedValueRepresentation(
			api.DefinedValueRepresentationGeneratedWrapper,
			api.NameReference{},
		)
	}
	switch binding.providerDefinedValue {
	case gostdlib.DefinedValueRepresentationIdentity:
		return api.NewProviderIdentityDefinedValueRepresentation()
	case gostdlib.DefinedValueRepresentationOperations:
		reference, providerOwned, err := n.providerFacetReference(
			typeName,
			gostdlib.FacetDefinedValueOperations,
			gostdlib.FacetCapabilityProject,
			api.ImportPhaseValue,
		)
		if err != nil {
			return api.DefinedValueRepresentation{}, err
		}
		if !providerOwned {
			return api.DefinedValueRepresentation{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "provider defined-value operations lost provider ownership",
			}
		}
		return api.NewDefinedValueRepresentation(
			api.DefinedValueRepresentationProviderOperations,
			reference,
		)
	default:
		return api.DefinedValueRepresentation{}, &api.NameError{
			Name:   typeName.Name(),
			Reason: "provider defined-value representation is uncertified",
		}
	}
}

func (n *File) TypeRepresentation(
	typeName *types.TypeName,
	facet api.TypeRepresentationFacet,
) ([]api.RootRequest, error) {
	if typeName == nil || !facet.Valid() {
		return nil, &api.NameError{
			Reason: "type-representation request is invalid",
		}
	}
	if typeName.Pkg() != nil &&
		typeName.Parent() != nil &&
		typeName.Parent() != typeName.Pkg().Scope() {
		placement, err := n.generatedArtifactPlacement(typeName.Type())
		if err != nil {
			return nil, err
		}
		if placement.kind != api.GeneratedArtifactPlacementLexical ||
			placement.anchor != typeName {
			return nil, &api.NameError{
				Name:   typeName.Name(),
				Reason: "local type representation has no exact lexical owner",
			}
		}
		request, err := api.NewLexicalTypeRepresentationRequest(
			placement.lexicalOwner,
			typeName,
			facet,
		)
		if err != nil {
			return nil, err
		}
		return []api.RootRequest{request}, nil
	}
	request, err := api.NewTypeRepresentationRequest(typeName, facet)
	if err != nil {
		return nil, err
	}
	return []api.RootRequest{request}, nil
}

func (n *File) AnonymousStructTypeRepresentation(
	structType *types.Struct,
	facet api.TypeRepresentationFacet,
) ([]api.RootRequest, error) {
	if structType == nil || structType.NumFields() == 0 || !facet.Valid() {
		return nil, &api.NameError{
			Reason: "anonymous-struct type representation is invalid",
		}
	}
	binding, err := n.anonymousStructBinding(structType)
	if err != nil {
		return nil, err
	}
	request, err := api.NewGeneratedTypeRepresentationRequest(
		binding.owner,
		facet,
	)
	if err != nil {
		return nil, err
	}
	requests := []api.RootRequest{request}
	if binding.owner.Placement() == api.GeneratedArtifactPlacementLexical {
		return requests, nil
	}
	if n.artifactOwner.Valid() &&
		n.artifactOwner != api.MustGeneratedArtifactOwner(binding.owner) {
		dependency, dependencyErr :=
			api.NewGeneratedArtifactDependencyRequest(
				binding.owner,
				api.ArtifactFacetInstanceTypeSurface,
			)
		if dependencyErr != nil {
			return nil, dependencyErr
		}
		requests = append(requests, dependency)
	}
	return requests, nil
}

func (n *File) NamedStructOperation(
	typeName *types.TypeName,
	operation api.NamedStructOperation,
) (api.NameReference, error) {
	request, err := n.namedStructOperationRequest(typeName, operation)
	if err != nil {
		return api.NameReference{}, err
	}
	capability, err := providerNamedStructCapability(operation)
	if err != nil {
		return api.NameReference{}, err
	}
	providerReference, providerOwned, err := n.providerFacetReference(
		typeName,
		gostdlib.FacetNamedStructOperations,
		capability,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		statefulReference, profiled, statefulErr := n.providerStatefulOperation(
			typeName,
			capability,
			api.ImportPhaseValue,
			api.ArtifactFacetStaticSurface,
		)
		if statefulErr != nil {
			return api.NameReference{}, statefulErr
		}
		if profiled {
			return statefulReference.WithRequests(
				api.CombineRequests(
					statefulReference.Requests(),
					[]api.RootRequest{request},
				)...,
			)
		}
		return providerReference.WithRequests(
			api.CombineRequests(
				providerReference.Requests(),
				[]api.RootRequest{request},
			)...,
		)
	}
	reference, err := n.reference(
		typeName,
		api.ImportPhaseValue,
		api.ArtifactFacetStaticSurface,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requests := append(reference.Requests(), request)
	return reference.WithRequests(requests...)
}

func (n *File) NamedStructConstructor(
	typeName *types.TypeName,
) (api.NameReference, error) {
	providerReference, providerOwned, err := n.providerFacetReference(
		typeName,
		gostdlib.FacetNamedStructOperations,
		gostdlib.FacetCapabilityMake,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		statefulReference, profiled, statefulErr := n.providerStatefulOperation(
			typeName,
			gostdlib.FacetCapabilityMake,
			api.ImportPhaseValue,
			api.ArtifactFacetStaticSurface,
		)
		if statefulErr != nil {
			return api.NameReference{}, statefulErr
		}
		if profiled {
			return statefulReference, nil
		}
		return providerReference, nil
	}
	return n.Reference(typeName)
}

func (n *File) namedStructOperationRequest(
	typeName *types.TypeName,
	operation api.NamedStructOperation,
) (api.RootRequest, error) {
	if typeName != nil &&
		typeName.Pkg() != nil &&
		typeName.Parent() != nil &&
		typeName.Parent() != typeName.Pkg().Scope() {
		placement, placementErr := n.generatedArtifactPlacement(
			typeName.Type(),
		)
		if placementErr != nil {
			return api.RootRequest{}, placementErr
		}
		if placement.kind != api.GeneratedArtifactPlacementLexical ||
			placement.anchor != typeName {
			return api.RootRequest{}, &api.NameError{
				Name:   typeName.Name(),
				Reason: "local named-struct operation has no exact lexical owner",
			}
		}
		return api.NewLexicalNamedStructOperationRequest(
			placement.lexicalOwner,
			typeName,
			operation,
		)
	}
	return api.NewNamedStructOperationRequest(
		typeName,
		operation,
	)
}

func (n *File) PointerRepresentation(
	pointer *types.Pointer,
) (api.PointerRepresentationReference, error) {
	return n.pointerRepresentation(
		pointer,
		n.generatedNamedObjectIdentity,
	)
}

func (n *File) SourcePointerRepresentation(
	owner types.Object,
	pointer *types.Pointer,
) (api.PointerRepresentationReference, error) {
	owner = api.GenericDeclarationOrigin(owner)
	if owner == nil || owner.Pkg() == nil {
		return api.PointerRepresentationReference{}, &api.NameError{
			Reason: "source pointer representation has no declaration owner",
		}
	}
	return n.pointerRepresentation(
		pointer,
		n.sourceGeneratedNamedObjectIdentity(owner),
	)
}

func (n *File) pointerRepresentation(
	pointer *types.Pointer,
	namedIdentity typeidentity.NamedObjectIdentity,
) (api.PointerRepresentationReference, error) {
	if pointer == nil {
		return api.PointerRepresentationReference{}, &api.NameError{
			Reason: "pointer-representation type is nil",
		}
	}
	pointer = pointerRepresentationFamily(pointer)
	key, err := typeidentity.BuildParameterizedKey(
		pointer,
		namedIdentity,
		func(parameter *types.TypeParam) (string, error) {
			if parameter == nil || parameter.Obj() == nil {
				return "", &api.NameError{
					Reason: "pointer representation has an unbound type parameter",
				}
			}
			return namedIdentity(parameter.Obj())
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

func pointerRepresentationFamily(pointer *types.Pointer) *types.Pointer {
	if pointer == nil {
		return nil
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	if !ok ||
		named.Origin() == nil ||
		named.Origin().TypeParams().Len() == 0 {
		return pointer
	}
	return types.NewPointer(named.Origin())
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
	providerReference, providerOwned, err := n.providerFacetStorageReference(
		typeName,
		gostdlib.FacetNamedStructOperations,
		gostdlib.FacetCapabilityStorage,
		api.ImportPhaseType,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if providerOwned {
		statefulReference, profiled, statefulErr := n.providerStatefulOperation(
			typeName,
			gostdlib.FacetCapabilityStorage,
			api.ImportPhaseType,
			api.ArtifactFacetInstanceTypeSurface,
		)
		if statefulErr != nil {
			return api.NameReference{}, statefulErr
		}
		if profiled {
			return statefulReference.WithRequests(
				api.CombineRequests(
					statefulReference.Requests(),
					[]api.RootRequest{request},
				)...,
			)
		}
		return providerReference.WithRequests(
			api.CombineRequests(
				providerReference.Requests(),
				[]api.RootRequest{request},
			)...,
		)
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
	if structType.NumFields() == 0 {
		return n.Runtime(api.RuntimeEmptyStruct, api.ImportPhaseType)
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
