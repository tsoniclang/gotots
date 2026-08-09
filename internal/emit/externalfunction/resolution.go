package externalfunction

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/externals"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	"github.com/tsoniclang/gotots/internal/load"
)

type indexedExternalFunction struct {
	function *types.Func
	site     declarationindex.Site
	source   *load.Package
	contract environmentcontract.ObjectContract
}

type BindingError struct {
	Identity string
	Reason   string
}

func (e *BindingError) Error() string {
	if e.Identity == "" {
		return "link external Go function: " + e.Reason
	}
	return "link external Go function " + e.Identity + ": " + e.Reason
}

func Resolve(
	source *load.Program,
	sites map[types.Object]declarationindex.Site,
	provider *externalcertify.Certificate,
	standardLibrary *gostdlibcertify.Certificate,
	providerScalar api.ScalarABI,
) (map[*types.Func]api.ExternalFunctionTarget, []string, error) {
	resolved := make(map[*types.Func]api.ExternalFunctionTarget)
	if provider == nil {
		return resolved, nil, nil
	}
	if err := validateExternalProviderProfile(
		source,
		provider,
		standardLibrary,
		providerScalar,
	); err != nil {
		return nil, nil, err
	}
	bindings := provider.Bindings()
	requested := externalFunctionIdentities(bindings)
	byIdentity, err := indexExternalFunctions(source, sites, requested)
	if err != nil {
		return nil, nil, err
	}
	modules := make([]string, 0)
	for _, binding := range bindings {
		owner, selected := byIdentity[binding.SourceIdentity()]
		if !selected {
			continue
		}
		if err := validateExternalSourceBinding(owner, binding); err != nil {
			return nil, nil, err
		}
		target, module, err := externalFunctionTarget(
			owner,
			binding,
			byIdentity,
		)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := resolved[owner.function]; duplicate {
			return nil, nil, &BindingError{
				Identity: binding.SourceIdentity(),
				Reason:   "certificate binding is duplicated",
			}
		}
		resolved[owner.function] = target
		if module != "" {
			modules = append(modules, module)
		}
	}
	sort.Strings(modules)
	return resolved, slices.Compact(modules), nil
}

func validateExternalProviderProfile(
	source *load.Program,
	provider *externalcertify.Certificate,
	standardLibrary *gostdlibcertify.Certificate,
	providerScalar api.ScalarABI,
) error {
	if source == nil || !provider.Valid() {
		return &BindingError{Reason: "provider certificate is invalid"}
	}
	if standardLibrary == nil || !standardLibrary.Valid() ||
		provider.StandardLibraryDigest() != standardLibrary.ManifestDigest() {
		return &BindingError{
			Reason: "provider and standard-library certificates do not exact-join",
		}
	}
	selectedProfile, ok := provider.BuildProfile()
	if !ok {
		return &BindingError{Reason: "provider build profile is absent"}
	}
	selectedKey, err := environmentcontract.ToolchainKey(selectedProfile)
	if err != nil {
		return err
	}
	sourceKey, err := environmentcontract.ToolchainKey(source.BuildProfile())
	if err != nil {
		return err
	}
	if selectedKey != sourceKey || provider.Backend() != "node" ||
		!providerScalar.Valid() ||
		provider.ProviderIntegerRepresentation() !=
			providerScalar.IntegerRepresentation().String() {
		return &BindingError{
			Reason: "provider target profile does not match compilation",
		}
	}
	return nil
}

func indexExternalFunctions(
	source *load.Program,
	sites map[types.Object]declarationindex.Site,
	requested map[string]struct{},
) (map[string]indexedExternalFunction, error) {
	result := make(map[string]indexedExternalFunction)
	for object, site := range sites {
		function, ok := object.(*types.Func)
		if !ok || function != function.Origin() {
			continue
		}
		contract, err := environmentcontract.Describe(function)
		if err != nil {
			return nil, err
		}
		if _, selected := requested[contract.Identity()]; !selected {
			continue
		}
		if err := addIndexedExternalFunction(result, indexedExternalFunction{
			function: function,
			site:     site,
			source:   site.Source,
			contract: contract,
		}); err != nil {
			return nil, err
		}
	}
	for _, sourcePackage := range source.EnvironmentPackages() {
		if sourcePackage.Kind() != load.PackageExternalContract {
			continue
		}
		scope := sourcePackage.Types().Scope()
		for _, name := range scope.Names() {
			function, ok := scope.Lookup(name).(*types.Func)
			if !ok || function != function.Origin() {
				continue
			}
			contract, err := environmentcontract.Describe(function)
			if err != nil {
				return nil, err
			}
			if _, selected := requested[contract.Identity()]; !selected {
				continue
			}
			if err := addIndexedExternalFunction(result, indexedExternalFunction{
				function: function,
				source:   sourcePackage,
				contract: contract,
			}); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func addIndexedExternalFunction(
	indexed map[string]indexedExternalFunction,
	function indexedExternalFunction,
) error {
	identity := function.contract.Identity()
	if identity == "" || function.function == nil || function.source == nil {
		return &BindingError{
			Identity: identity,
			Reason:   "selected source identity is incomplete",
		}
	}
	if _, duplicate := indexed[identity]; duplicate {
		return &BindingError{
			Identity: identity,
			Reason:   "selected source identity is duplicated",
		}
	}
	indexed[identity] = function
	return nil
}

func externalFunctionIdentities(bindings []externals.Binding) map[string]struct{} {
	result := make(map[string]struct{}, len(bindings)*2)
	for _, binding := range bindings {
		result[binding.SourceIdentity()] = struct{}{}
		if identity, _, _, ok := binding.SourceTarget(); ok {
			result[identity] = struct{}{}
		}
	}
	return result
}

func validateExternalSourceBinding(
	owner indexedExternalFunction,
	binding externals.Binding,
) error {
	declaration, syntaxOwned := owner.site.Declaration.(*ast.FuncDecl)
	bodylessSource := syntaxOwned && declaration.Body == nil
	contractOwned := !syntaxOwned &&
		owner.source.Kind() == load.PackageExternalContract
	if (!bodylessSource && !contractOwned) ||
		owner.contract.Signature() != binding.SourceSignature() ||
		owner.source.ModulePath() != binding.SourceModulePath() ||
		owner.source.ModuleVersion() != binding.SourceModuleVersion() {
		return &BindingError{
			Identity: binding.SourceIdentity(),
			Reason:   "certificate does not match the selected bodyless declaration",
		}
	}
	return nil
}

func externalFunctionTarget(
	owner indexedExternalFunction,
	binding externals.Binding,
	byIdentity map[string]indexedExternalFunction,
) (api.ExternalFunctionTarget, string, error) {
	switch binding.TargetKind() {
	case externals.TargetModule:
		module, export, _, _, ok := binding.ModuleTarget()
		if !ok {
			break
		}
		target, err := api.NewExternalModuleFunctionTarget(module, export)
		return target, module, err
	case externals.TargetSource:
		identity, signature, _, ok := binding.SourceTarget()
		implementation, found := byIdentity[identity]
		declaration, bodyOwner := implementation.site.Declaration.(*ast.FuncDecl)
		_, sourceOwned := owner.site.Declaration.(*ast.FuncDecl)
		if !ok || !found || !sourceOwned || !bodyOwner || declaration.Body == nil ||
			implementation.contract.Signature() != signature ||
			implementation.function.Pkg() != owner.function.Pkg() ||
			!types.Identical(implementation.function.Type(), owner.function.Type()) {
			return api.ExternalFunctionTarget{}, "", &BindingError{
				Identity: binding.SourceIdentity(),
				Reason:   "portable source implementation is absent or incompatible",
			}
		}
		target, err := api.NewExternalSourceFunctionTarget(implementation.function)
		return target, "", err
	}
	return api.ExternalFunctionTarget{}, "", &BindingError{
		Identity: binding.SourceIdentity(),
		Reason:   "certificate target is invalid",
	}
}
