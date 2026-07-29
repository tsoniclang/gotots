package naming

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/output"
)

type constantProjectionImport struct {
	constant   *types.Const
	projection types.BasicKind
}

// ConstantProjection returns a constant-size reference to one untyped
// constant projection. Its import-local identity includes both the exact
// constant object and target representation; source spelling is never an
// import key.
func (n *File) ConstantProjection(
	selected *types.Const,
	projection types.BasicKind,
) (api.NameReference, error) {
	if selected == nil {
		return api.NameReference{}, &api.NameError{
			Reason: "constant projection constant is nil",
		}
	}
	request, err := api.NewConstantProjectionRequest(selected, projection)
	if err != nil {
		return api.NameReference{}, err
	}
	binding, ok := n.owner.byObject[selected]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[selected]
	}
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   selected.Name(),
			Reason: "constant has no reserved projection owner",
		}
	}
	exportedName, err := api.ConstantProjectionName(
		binding.name,
		projection,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	requests := []api.RootRequest{request}
	if binding.sourceFile != nil && n.require != nil {
		if err := n.require(selected); err != nil {
			return api.NameReference{}, err
		}
	}
	if binding.sourceFile != nil && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(
			selected,
			api.ArtifactFacetValueSurface,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if binding.sourceFile != nil && binding.sourceFile != n.sourceFile {
		referencePath, crossPackage, err := n.sourceReferencePath(
			selected,
			binding,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		if crossPackage {
			localName, err = n.constantProjectionImportName(
				constantProjectionImport{
					constant:   selected,
					projection: projection,
				},
				exportedName,
			)
			if err != nil {
				return api.NameReference{}, err
			}
		}
		modulePath, err := output.ModuleSpecifier(
			n.targetPath,
			referencePath,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		importRequest, err := api.NewImportRequest(
			n.factory,
			api.ImportPhaseValue,
			modulePath,
			exportedName,
			localName,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, importRequest)
	}
	return api.NewNameReference(localName, requests...)
}

func (n *File) constantProjectionImportName(
	identity constantProjectionImport,
	preferred string,
) (string, error) {
	if _, ok := api.ConstantProjectionType(identity.projection); !ok ||
		identity.constant == nil {
		return "", &api.NameError{
			Reason: "constant projection import identity is invalid",
		}
	}
	if existing := n.projections[identity]; existing != "" {
		return existing, nil
	}
	qualifier, err := n.packageImportQualifier(identity.constant.Pkg())
	if err != nil {
		return "", err
	}
	selected := n.allocateImportName(preferred, qualifier)
	n.projections[identity] = selected
	return selected, nil
}
