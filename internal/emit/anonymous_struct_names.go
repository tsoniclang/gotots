package emit

import (
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	anonymousstruct "github.com/tsoniclang/gotots/internal/emit/type/anonymousstruct"
	"github.com/tsoniclang/gotots/internal/output"
)

type anonymousStructPlacement struct {
	kind         api.GeneratedArtifactPlacement
	outputPath   string
	lexicalOwner api.ArtifactOwner
	anchor       *types.TypeName
}

func (n *fileNames) AnonymousStruct(
	structType *types.Struct,
	demand api.AnonymousStructDemand,
) (api.NameReference, error) {
	if structType == nil || !demand.Valid() {
		return api.NameReference{}, &api.NameError{
			Reason: "anonymous-struct demand is invalid",
		}
	}
	keys, err := anonymousstruct.BuildKeys(
		structType,
		n.anonymousNamedObjectIdentity,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	placement, err := n.anonymousStructPlacement(structType)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, err := n.owner.registry.internAnonymousStruct(
		keys,
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
		api.ImportPhaseValue,
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
		api.AnonymousStructDemandEqual:
		return []api.ArtifactFacet{api.ArtifactFacetStaticSurface}
	default:
		return nil
	}
}

func (n *fileNames) anonymousStructPlacement(
	structType *types.Struct,
) (anonymousStructPlacement, error) {
	components := anonymousstruct.LocalComponents(structType)
	if len(components) == 0 {
		return anonymousStructPlacement{
			kind:       api.GeneratedArtifactPlacementCompilation,
			outputPath: output.AnonymousStructSupportPath,
		}, nil
	}
	sourceOwner, sourceOwned := n.artifactOwner.Source()
	if !sourceOwned ||
		n.artifactSource == nil ||
		n.artifactFile == nil {
		return anonymousStructPlacement{}, &api.GeneratedArtifactPlacementError{
			TypeName: components[0].Name(),
			Reason:   "local component has no reconstructible source artifact",
		}
	}
	var anchor *types.TypeName
	var anchorScope *types.Scope
	anchorDepth := -1
	for _, component := range components {
		if component.Pkg() != sourceOwner.Pkg() ||
			component.Pos() < n.artifactSource.Pos() ||
			component.Pos() > n.artifactSource.End() ||
			component.Pos() < n.artifactFile.Pos() ||
			component.Pos() > n.artifactFile.End() {
			return anonymousStructPlacement{}, &api.GeneratedArtifactPlacementError{
				TypeName: component.Name(),
				Reason:   "local component is outside the owning source artifact",
			}
		}
		depth, ok := lexicalScopeDepth(
			component.Parent(),
			n.packageScope,
		)
		if !ok {
			return anonymousStructPlacement{}, &api.GeneratedArtifactPlacementError{
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
			return anonymousStructPlacement{}, &api.GeneratedArtifactPlacementError{
				TypeName: component.Name(),
				Reason:   "local components have no common legal lexical scope",
			}
		}
	}
	return anonymousStructPlacement{
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

func (r *declarationRegistry) internAnonymousStruct(
	keys anonymousstruct.Keys,
	structType *types.Struct,
	placement anonymousStructPlacement,
) (anonymousStructBinding, error) {
	if r == nil ||
		structType == nil ||
		keys.Fingerprint == "" ||
		keys.Artifact == "" ||
		!placement.kind.Valid() {
		return anonymousStructBinding{}, &api.NameError{
			Reason: "anonymous-struct canonicalization input is invalid",
		}
	}
	for _, artifact := range r.anonymousStructBuckets[keys.Fingerprint] {
		candidate, ok := r.anonymousStructs[artifact]
		if !ok || !candidate.owner.Valid() {
			return anonymousStructBinding{}, &api.NameError{
				Reason: "anonymous-struct fingerprint bucket is inconsistent",
			}
		}
		if types.Identical(candidate.owner.SourceType(), structType) {
			if !sameAnonymousStructPlacement(candidate.owner, placement) {
				return anonymousStructBinding{}, &api.NameError{
					Name:   candidate.name,
					Reason: "identical anonymous struct received inconsistent semantic placement",
				}
			}
			return candidate, nil
		}
	}
	if existing, ok := r.anonymousStructs[keys.Artifact]; ok {
		if !types.Identical(existing.owner.SourceType(), structType) {
			return anonymousStructBinding{}, &api.NameError{
				Name:   existing.name,
				Reason: "anonymous-struct artifact-key collision joined non-identical Go types",
			}
		}
		return existing, nil
	}
	name, err := anonymousstruct.TargetName(keys.Artifact)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	if existing := r.anonymousStructNames[name]; existing != "" &&
		existing != keys.Artifact {
		return anonymousStructBinding{}, &api.NameError{
			Name:   name,
			Reason: "anonymous-struct target-name prefix collision",
		}
	}
	owner, err := newAnonymousStructArtifact(
		structType,
		keys.Artifact,
		name,
		placement,
	)
	if err != nil {
		return anonymousStructBinding{}, err
	}
	binding := anonymousStructBinding{
		owner:       owner,
		fingerprint: keys.Fingerprint,
		name:        name,
	}
	r.anonymousStructs[keys.Artifact] = binding
	r.anonymousStructBuckets[keys.Fingerprint] = append(
		r.anonymousStructBuckets[keys.Fingerprint],
		keys.Artifact,
	)
	r.anonymousStructNames[name] = keys.Artifact
	return binding, nil
}

func newAnonymousStructArtifact(
	structType *types.Struct,
	artifact string,
	name string,
	placement anonymousStructPlacement,
) (*api.GeneratedArtifact, error) {
	if placement.kind == api.GeneratedArtifactPlacementLexical {
		return api.NewLexicalGeneratedArtifact(
			structType,
			artifact,
			name,
			placement.lexicalOwner,
			placement.anchor,
		)
	}
	return api.NewCompilationGeneratedArtifact(
		structType,
		artifact,
		name,
		placement.outputPath,
	)
}

func sameAnonymousStructPlacement(
	artifact *api.GeneratedArtifact,
	placement anonymousStructPlacement,
) bool {
	return artifact.Placement() == placement.kind &&
		artifact.OutputPath() == placement.outputPath &&
		artifact.LexicalOwner() == placement.lexicalOwner &&
		artifact.LexicalAnchor() == placement.anchor
}

func (n *fileNames) anonymousNamedObjectIdentity(
	object *types.TypeName,
) (string, error) {
	if object == nil || object.Pkg() == nil {
		return "", &api.NameError{
			Name:   objectName(object),
			Reason: "anonymous-struct named component has no package identity",
		}
	}
	if object.Parent() == object.Pkg().Scope() {
		binding, ok := n.owner.registry.byObject[object]
		if !ok || binding.sourcePath == "" || binding.name == "" {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "anonymous-struct named component has no declaration identity",
			}
		}
		return object.Pkg().Path() + "|" +
			binding.sourcePath + "|" +
			binding.name, nil
	}
	name := n.owner.targetNameByObject[object]
	sourceFile := n.artifactFile
	if name == "" ||
		n.artifactPath == "" ||
		sourceFile == nil ||
		object.Pos() < sourceFile.Pos() ||
		object.Pos() > sourceFile.End() {
		return "", &api.NameError{
			Name:   object.Name(),
			Reason: "anonymous-struct local component has no lexical declaration identity",
		}
	}
	offset := int64(object.Pos() - sourceFile.Pos())
	return object.Pkg().Path() + "|" +
		n.artifactPath + "|" +
		object.Name() + "|" +
		name + "|" +
		strconv.FormatInt(offset, 10), nil
}
