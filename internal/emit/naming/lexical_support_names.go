package naming

import (
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/output"
	"go/ast"
	"go/types"
	"strconv"
)

func WithLexicalTypeRequirements(
	context api.Context,
	source ast.Node,
	owner api.ArtifactOwner,
	requirements []api.DeclarationRequirement,
) (api.Context, error) {
	sourceOwner, sourceOwned := owner.Source()
	_, initializer, initializerOwned := owner.PackageInitializer()
	validSource := source != nil
	switch {
	case sourceOwned:
		validSource = validSource &&
			sourceOwner.Pos() >= source.Pos() &&
			sourceOwner.Pos() <= source.End()
	case initializerOwned:
		validSource = validSource &&
			initializer.Rhs.Pos() >= source.Pos() &&
			initializer.Rhs.End() <= source.End()
	default:
		validSource = false
	}
	if !validSource {
		return api.Context{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "lexical generated artifact has no exact source owner",
		}
	}
	byAnchor := make(
		map[*types.TypeName][]api.DeclarationRequirement,
	)
	for _, requirement := range requirements {
		var anchor *types.TypeName
		if artifact, generated :=
			requirement.LexicalGeneratedArtifact(); generated {
			if artifact.Placement() !=
				api.GeneratedArtifactPlacementLexical ||
				artifact.ReconstructionOwner() != owner {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "source artifact received a foreign lexical generated artifact",
				}
			}
			anchor = artifact.LexicalAnchor()
		} else if _, generated := requirement.GeneratedArtifact(); generated {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "source artifact received a non-lexical generated artifact",
			}
		} else if typeName, _, namedStruct :=
			requirement.NamedStructOperation(); namedStruct {
			if requirement.Owner() != owner {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "source artifact received a foreign lexical named-type requirement",
				}
			}
			anchor = typeName
		} else if typeName, _, _, represented :=
			requirement.TypeRepresentation(); represented {
			if typeName == nil || requirement.Owner() != owner {
				return api.Context{}, &api.InvariantError{
					Role:   context.Role(),
					Reason: "source artifact received a foreign lexical type representation",
				}
			}
			anchor = typeName
		} else {
			continue
		}
		if anchor == nil ||
			anchor.Pos() < source.Pos() ||
			anchor.Pos() > source.End() {
			return api.Context{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "lexical generated artifact anchor is outside its source declaration",
			}
		}
		byAnchor[anchor] = append(byAnchor[anchor], requirement)
	}
	return context.WithLexicalTypeRequirements(owner, byAnchor), nil
}

type constantProjectionImport struct {
	constant   *types.Const
	projection types.BasicKind
}

func (n *File) ConstantValue(
	selected *types.Const,
) (api.NameReference, bool, error) {
	if selected == nil || !constantbinding.RequiresDeferredBinding(selected) {
		return api.NameReference{}, false, &api.NameError{
			Reason: "deferred constant identity is invalid",
		}
	}
	binding, ok := n.owner.byObject[selected]
	if !ok && n.owner.registry != nil {
		binding, ok = n.owner.registry.byObject[selected]
	}
	if !ok {
		return api.NameReference{}, false, &api.NameError{
			Name:   selected.Name(),
			Reason: "constant has no emitted declaration",
		}
	}
	if !binding.sourceOwned() {
		reference, err := n.Reference(selected)
		return reference, false, err
	}
	if err := n.requireUse(
		selected,
		environmentcontract.UseDemandValue,
	); err != nil {
		return api.NameReference{}, false, err
	}
	deferredName, err := constantbinding.DeferredBindingName(binding.name)
	if err != nil {
		return api.NameReference{}, false, err
	}
	localName := deferredName
	var requests []api.RootRequest
	if n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(
			selected,
			api.ArtifactFacetCallableSignature,
		)
		if err != nil {
			return api.NameReference{}, false, err
		}
		requests = append(requests, dependency)
	}
	if binding.sourcePath != n.targetPath {
		referencePath, crossPackage, err := n.sourceReferencePath(
			selected,
			binding,
		)
		if err != nil {
			return api.NameReference{}, false, err
		}
		if crossPackage {
			localName, err = n.importName(selected, deferredName)
			if err != nil {
				return api.NameReference{}, false, err
			}
		}
		modulePath, err := output.ModuleSpecifier(n.targetPath, referencePath)
		if err != nil {
			return api.NameReference{}, false, err
		}
		request, err := api.NewImportRequest(
			n.factory,
			api.ImportPhaseValue,
			modulePath,
			deferredName,
			localName,
		)
		if err != nil {
			return api.NameReference{}, false, err
		}
		requests = append(requests, request)
	}
	reference, err := api.NewNameReference(localName, requests...)
	return reference, true, err
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
	projectionScheduled := binding.scheduled() ||
		binding.kind == targetBindingMissingProvider
	if projectionScheduled {
		if err := n.requireUse(
			selected,
			environmentcontract.UseDemandValue,
		); err != nil {
			return api.NameReference{}, err
		}
	}
	if binding.sourceOwned() && n.artifactOwner.Valid() {
		dependency, err := api.NewArtifactDependencyRequest(
			selected,
			api.ArtifactFacetValueSurface,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		requests = append(requests, dependency)
	}
	if projectionScheduled && binding.sourcePath != n.targetPath {
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

func (n *File) Primitive(alias api.PrimitiveAlias) (api.NameReference, error) {
	if existing := n.primitives[alias]; existing != "" {
		modulePath, err := output.RuntimeModuleSpecifier(
			output.ScalarSupportPath,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		request, err := api.NewPrimitiveAliasRequest(
			n.factory,
			modulePath,
			alias,
			existing,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName, err := api.PrimitiveAliasName(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	localName := exportedName
	if n.lexicalNameExists(localName) {
		base := exportedName + "__from_gotots_support"
		localName = base
		for suffix := uint64(1); n.lexicalNameExists(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.primitives[alias] = localName
	modulePath, err := output.RuntimeModuleSpecifier(
		output.ScalarSupportPath,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	request, err := api.NewPrimitiveAliasRequest(
		n.factory,
		modulePath,
		alias,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}

func (n *File) ProviderPrimitive(
	alias api.PrimitiveAlias,
) (api.NameReference, error) {
	if n == nil || n.owner == nil || n.owner.registry == nil ||
		n.owner.registry.provider == nil ||
		!n.owner.registry.provider.Valid() {
		return api.NameReference{}, &api.NameError{
			Reason: "provider scalar contract is absent",
		}
	}
	module := n.owner.registry.provider.ProviderScalarModule()
	exportedName, err := api.PrimitiveAliasName(alias)
	if err != nil {
		return api.NameReference{}, err
	}
	qualifier, request, err := n.providerImport(module, api.ImportPhaseType)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewQualifiedNameReference(qualifier, exportedName, request)
}

func (n *File) Runtime(
	symbol api.RuntimeSymbol,
	phase api.ImportPhase,
) (api.NameReference, error) {
	contract, err := api.RuntimeContract(symbol)
	if err != nil {
		return api.NameReference{}, err
	}
	modulePath, err := output.RuntimeModuleSpecifier(
		contract.OutputPath(),
	)
	if err != nil {
		return api.NameReference{}, err
	}
	if existing := n.runtime[symbol]; existing != "" {
		request, err := api.NewRuntimeImportRequest(
			n.factory,
			phase,
			modulePath,
			symbol,
			existing,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(existing, request)
	}
	exportedName := contract.ExportedName()
	localName := exportedName
	if n.lexicalNameExists(localName) {
		base := exportedName + "__from_gotots_runtime"
		localName = base
		for suffix := uint64(1); n.lexicalNameExists(localName); suffix++ {
			localName = base + "_" + strconv.FormatUint(suffix, 10)
		}
	}
	n.importNames[localName] = struct{}{}
	n.runtime[symbol] = localName
	request, err := api.NewRuntimeImportRequest(
		n.factory,
		phase,
		modulePath,
		symbol,
		localName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(localName, request)
}
