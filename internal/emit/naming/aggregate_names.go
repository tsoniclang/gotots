package naming

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
	"go/types"
)

func (n *File) MapSpecialization(
	sourceType types.Type,
	demand api.MapSpecializationDemand,
) (api.NameReference, error) {
	mapType, ok := representedMapType(sourceType)
	if !ok || !demand.Valid() {
		return api.NameReference{}, &api.NameError{
			Reason: "map-specialization demand is invalid",
		}
	}
	artifactKey, err := typeidentity.BuildKey(
		mapType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	name, err := n.semanticGeneratedTypeName("$goMap$", mapType)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.generatedArtifactPlacement(mapType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internMapSpecialization(
		artifactKey,
		mapType,
		name,
		placement,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewMapSpecializationRequest(
		binding.owner,
		demand,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	phase := api.ImportPhaseValue
	if demand == api.MapSpecializationDemandDefinition {
		phase = api.ImportPhaseType
	}
	return n.generatedReference(
		binding.owner,
		binding.name,
		requirement,
		mapSpecializationFacet(demand),
		phase,
	)
}

func representedMapType(sourceType types.Type) (*types.Map, bool) {
	sourceType = types.Unalias(sourceType)
	if named, ok := sourceType.(*types.Named); ok {
		sourceType = named.Underlying()
	}
	mapType, ok := sourceType.(*types.Map)
	return mapType, ok
}

func mapSpecializationFacet(
	demand api.MapSpecializationDemand,
) api.ArtifactFacet {
	if demand == api.MapSpecializationDemandDefinition {
		return api.ArtifactFacetInstanceTypeSurface
	}
	return api.ArtifactFacetStaticSurface
}

func (r *Registry) internMapSpecialization(
	artifactKey string,
	mapType *types.Map,
	name string,
	placement generatedArtifactPlacement,
) (mapSpecializationBinding, error) {
	if r == nil ||
		mapType == nil ||
		artifactKey == "" ||
		name == "" ||
		!placement.kind.Valid() {
		return mapSpecializationBinding{}, &api.NameError{
			Reason: "map-specialization canonicalization input is invalid",
		}
	}
	if existing, ok := r.mapSpecializations[artifactKey]; ok {
		existingType, valid := existing.owner.MapType()
		if !valid || !types.Identical(existingType, mapType) {
			return mapSpecializationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "map-specialization artifact-key collision joined non-identical Go types",
			}
		}
		if !sameMapSpecializationPlacement(existing.owner, placement) {
			return mapSpecializationBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "identical map specialization received inconsistent semantic placement",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedScopedName(
		r.mapSpecializationNames,
		name,
		artifactKey,
		"map-specialization",
		placement,
		output.MapSpecializationSupportPath,
	); err != nil {
		return mapSpecializationBinding{}, err
	}
	owner, err := newMapSpecializationArtifact(
		mapType,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return mapSpecializationBinding{}, err
	}
	binding := mapSpecializationBinding{
		owner: owner,
		name:  name,
	}
	r.mapSpecializations[artifactKey] = binding
	return binding, nil
}

func newMapSpecializationArtifact(
	mapType *types.Map,
	artifact string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			api.GeneratedArtifactMapSpecialization,
			mapType,
			artifact,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactMapSpecialization,
		mapType,
		artifact,
		name,
		output.MapSpecializationSupportPath,
	)
}

func sameMapSpecializationPlacement(
	artifact *api.GeneratedArtifact,
	placement generatedArtifactPlacement,
) bool {
	return artifact.Placement() == placement.kind &&
		artifact.LexicalOwner() == placement.lexicalOwner &&
		artifact.LexicalAnchor() == placement.anchor
}

type generatedArtifactPlacement struct {
	kind         api.GeneratedArtifactPlacement
	lexicalOwner api.ArtifactOwner
	anchor       *types.TypeName
}

func (n *File) AnonymousStruct(
	structType *types.Struct,
	demand api.AnonymousStructDemand,
	phase api.ImportPhase,
) (api.NameReference, error) {
	if structType == nil ||
		!demand.Valid() ||
		(phase != api.ImportPhaseType &&
			phase != api.ImportPhaseValue) ||
		(phase == api.ImportPhaseType &&
			demand != api.AnonymousStructDemandDefinition) {
		return api.NameReference{}, &api.NameError{
			Reason: "anonymous-struct demand is invalid",
		}
	}
	if structType.NumFields() == 0 {
		return n.Runtime(api.RuntimeEmptyStruct, phase)
	}
	binding, err := n.anonymousStructBinding(structType)
	if err != nil {
		return api.NameReference{}, err
	}
	requirement, err := api.NewAnonymousStructRequest(
		binding.owner,
		demand,
	)
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
		for _, facet := range anonymousStructDependencyFacets(demand) {
			dependency, dependencyError :=
				api.NewGeneratedArtifactDependencyRequest(
					binding.owner,
					facet,
				)
			if dependencyError != nil {
				return api.NameReference{}, dependencyError
			}
			requests = append(requests, dependency)
		}
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
		phase,
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

func (n *File) anonymousStructBinding(
	structType *types.Struct,
) (anonymousStructBinding, error) {
	artifactKey, err := typeidentity.BuildKey(
		structType,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	name, err := n.semanticGeneratedTypeName("$goStruct$", structType)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	placement, err := n.generatedArtifactPlacement(structType)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	return n.owner.registry.internAnonymousStruct(
		artifactKey,
		structType,
		name,
		placement,
	)
}

func anonymousStructDependencyFacets(
	demand api.AnonymousStructDemand,
) []api.ArtifactFacet {
	switch demand {
	case api.AnonymousStructDemandDefinition:
		return []api.ArtifactFacet{
			api.ArtifactFacetConstructorSurface,
			api.ArtifactFacetInstanceTypeSurface,
		}
	case api.AnonymousStructDemandZero,
		api.AnonymousStructDemandCopy,
		api.AnonymousStructDemandEqual,
		api.AnonymousStructDemandHash,
		api.AnonymousStructDemandConvert,
		api.AnonymousStructDemandStorage:
		return []api.ArtifactFacet{api.ArtifactFacetStaticSurface}
	default:
		return nil
	}
}

func (n *File) generatedArtifactPlacement(
	sourceType types.Type,
) (generatedArtifactPlacement, error) {
	return n.generatedArtifactPlacementForComponents(
		typeidentity.LocalComponents(sourceType),
	)
}

func (n *File) generatedArtifactPlacementForComponents(
	components []*types.TypeName,
) (generatedArtifactPlacement, error) {
	if len(components) == 0 {
		return generatedArtifactPlacement{
			kind: api.GeneratedArtifactPlacementCompilation,
		}, nil
	}
	ownerPackage := n.artifactOwner.Package()
	_, sourceOwned := n.artifactOwner.Source()
	_, _, initializerOwned := n.artifactOwner.PackageInitializer()
	if (!sourceOwned && !initializerOwned) ||
		ownerPackage == nil ||
		n.artifactSource == nil ||
		n.artifactFile == nil {
		return generatedArtifactPlacement{}, &api.GeneratedArtifactPlacementError{
			TypeName: components[0].Name(),
			Reason:   "local component has no reconstructible source artifact",
		}
	}
	var anchor *types.TypeName
	var anchorScope *types.Scope
	anchorDepth := -1
	for _, component := range components {
		if component.Pkg() != ownerPackage ||
			component.Pos() < n.artifactSource.Pos() ||
			component.Pos() > n.artifactSource.End() ||
			component.Pos() < n.artifactFile.Pos() ||
			component.Pos() > n.artifactFile.End() {
			return generatedArtifactPlacement{}, &api.GeneratedArtifactPlacementError{
				TypeName: component.Name(),
				Reason:   "local component is outside the owning source artifact",
			}
		}
		depth, ok := lexicalScopeDepth(
			component.Parent(),
			n.packageScope,
		)
		if !ok {
			return generatedArtifactPlacement{}, &api.GeneratedArtifactPlacementError{
				TypeName: component.Name(),
				Reason:   "local component scope is not package-rooted",
			}
		}
		if depth > anchorDepth ||
			(depth == anchorDepth && component.Pos() > anchor.Pos()) {
			anchor = component
			anchorScope = component.Parent()
			anchorDepth = depth
		}
	}
	for _, component := range components {
		if !scopeContainsScope(component.Parent(), anchorScope) {
			return generatedArtifactPlacement{}, &api.GeneratedArtifactPlacementError{
				TypeName: component.Name(),
				Reason:   "local components have no common legal lexical scope",
			}
		}
	}
	return generatedArtifactPlacement{
		kind:         api.GeneratedArtifactPlacementLexical,
		lexicalOwner: n.artifactOwner,
		anchor:       anchor,
	}, nil
}

func lexicalScopeDepth(
	scope *types.Scope,
	packageScope *types.Scope,
) (int, bool) {
	depth := 0
	for current := scope; current != nil; current = current.Parent() {
		if current == packageScope {
			return depth, true
		}
		depth++
	}
	return 0, false
}

func scopeContainsScope(outer *types.Scope, inner *types.Scope) bool {
	for current := inner; current != nil; current = current.Parent() {
		if current == outer {
			return true
		}
	}
	return false
}

func (r *Registry) internAnonymousStruct(
	artifactKey string,
	structType *types.Struct,
	name string,
	placement generatedArtifactPlacement,
) (anonymousStructBinding, error) {
	if r == nil ||
		structType == nil ||
		artifactKey == "" ||
		name == "" ||
		!placement.kind.Valid() {
		return anonymousStructBinding{}, &api.NameError{
			Reason: "anonymous-struct canonicalization input is invalid",
		}
	}
	if existing, ok := r.anonymousStructs[artifactKey]; ok {
		if !types.Identical(existing.owner.SourceType(), structType) {
			return anonymousStructBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "anonymous-struct artifact-key collision joined non-identical Go types",
			}
		}
		if !sameAnonymousStructPlacement(existing.owner, placement) {
			return anonymousStructBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "identical anonymous struct received inconsistent semantic placement",
			}
		}
		return existing, nil
	}
	if err := reserveGeneratedScopedName(
		r.anonymousStructNames,
		name,
		artifactKey,
		"anonymous-struct",
		placement,
		output.AnonymousStructSupportPath,
	); err != nil {
		return anonymousStructBinding{}, err
	}
	owner, err := newAnonymousStructArtifact(
		structType,
		artifactKey,
		name,
		placement,
	)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	binding := anonymousStructBinding{
		owner: owner,
		name:  name,
	}
	r.anonymousStructs[artifactKey] = binding
	return binding, nil
}

func newAnonymousStructArtifact(
	structType *types.Struct,
	artifact string,
	name string,
	placement generatedArtifactPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			api.GeneratedArtifactAnonymousStruct,
			structType,
			artifact,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	return api.NewCompilationGeneratedArtifact(
		api.GeneratedArtifactAnonymousStruct,
		structType,
		artifact,
		name,
		output.AnonymousStructSupportPath,
	)
}

func sameAnonymousStructPlacement(
	artifact *api.GeneratedArtifact,
	placement generatedArtifactPlacement,
) bool {
	return artifact.Placement() == placement.kind &&
		(artifact.Placement() != api.GeneratedArtifactPlacementCompilation ||
			artifact.OutputPath() == output.AnonymousStructSupportPath) &&
		artifact.LexicalOwner() == placement.lexicalOwner &&
		artifact.LexicalAnchor() == placement.anchor
}

func (n *File) generatedNamedObjectIdentity(
	object *types.TypeName,
) (string, error) {
	if object == nil {
		return "", &api.NameError{
			Name:   objectName(object),
			Reason: "generated-artifact named component has no package identity",
		}
	}
	if object.Pkg() == nil {
		return typeidentity.NamedObjectKey(object)
	}
	if object.Parent() == object.Pkg().Scope() {
		if _, ok := n.owner.registry.byObject[object]; !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "generated-artifact named component has no declaration identity",
			}
		}
		return typeidentity.NamedObjectKey(object)
	}
	_, indexed := n.owner.targetNameByObject[object]
	if !indexed ||
		n.packageScope == nil {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "generated-artifact local component has no lexical declaration identity",
		}
	}
	return typeidentity.LexicalNamedObjectKey(
		object,
		n.artifactOwner,
		n.packageScope,
	)
}
