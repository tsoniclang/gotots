package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	anonymousstruct "github.com/tsoniclang/gotots/internal/emit/type/anonymousstruct"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

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
	components := typeidentity.LocalComponents(sourceType)
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
	placement generatedArtifactPlacement,
) (anonymousStructBinding, error) {
	if r == nil ||
		structType == nil ||
		artifactKey == "" ||
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
	name, err := anonymousstruct.TargetName(artifactKey)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	if existing := r.anonymousStructNames[name]; existing != "" &&
		existing != artifactKey {
		return anonymousStructBinding{}, &api.NameError{
			Name:   name,
			Reason: "anonymous-struct target-name prefix collision",
		}
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
	r.anonymousStructNames[name] = artifactKey
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
