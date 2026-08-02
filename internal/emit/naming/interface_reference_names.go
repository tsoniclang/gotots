package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/type/typeidentity"
	"github.com/tsoniclang/gotots/internal/output"
)

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
	if !ok || !binding.scheduled() {
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
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(object, facet)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if binding.sourcePath == n.targetPath {
		return api.NewNameReference(exportedName, requests...)
	}
	referencePath := binding.sourcePath
	crossPackage := n.packageScope != nil &&
		object.Pkg() != nil &&
		object.Pkg().Scope() != n.packageScope
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
	return n.generatedReference(
		artifact,
		name,
		requirement,
		facet,
		api.ImportPhaseValue,
	)
}

func (n *File) generatedReference(
	artifact *api.GeneratedArtifact,
	name string,
	requirement api.RootRequest,
	facet api.ArtifactFacet,
	phase api.ImportPhase,
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
		phase,
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
	if !ok || named.Origin() == nil || named.Origin().Obj() == nil {
		return nil, nil, false
	}
	parameters := named.Origin().TypeParams().Len()
	arguments := named.TypeArgs().Len()
	if parameters != arguments {
		return nil, nil, false
	}
	source, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, nil, false
	}
	source = source.Complete()
	return named.Origin().Obj(), source, source.IsMethodSet()
}

func predeclaredError(typeName *types.TypeName) bool {
	return typeName != nil && typeName == types.Universe.Lookup("error")
}

func anonymousInterface(sourceType types.Type) (*types.Interface, bool) {
	source, ok := types.Unalias(sourceType).(*types.Interface)
	if !ok {
		return nil, false
	}
	source = source.Complete()
	return source, source.IsMethodSet()
}

func (n *File) canonicalInterfaceContract(
	sourceType types.Type,
) (*types.Interface, string, error) {
	var source *types.Interface
	if _, selected, ok := namedInterface(sourceType); ok {
		source = selected
	} else {
		var ok bool
		source, ok = anonymousInterface(sourceType)
		if !ok {
			return nil, "", &api.NameError{
				Name:   types.TypeString(sourceType, nil),
				Reason: "interface contract demand type is invalid",
			}
		}
	}
	key, err := typeidentity.BuildKey(
		source,
		n.generatedNamedObjectIdentity,
	)
	if err != nil {
		return nil, "", err
	}
	source, err = n.owner.registry.internInterfaceContract(key, source)
	if err != nil {
		return nil, "", err
	}
	return source, key, nil
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
