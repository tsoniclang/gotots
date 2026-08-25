package certify

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

func verifyProviderProfileInterfaceClosure(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) error {
	providers, err := ordinaryProviderInterfaces(modules, facetModules)
	if err != nil {
		return err
	}
	identities := sourceTypeIdentities(source)
	interfaces := sourceProviderInterfaceTypes(source, providers)
	var differences []string
	for _, module := range facetModules {
		for _, profile := range module.CallableProfiles {
			selected := make(map[string]gostdlib.ProviderInterfaceDocument)
			queue := make([]string, 0, len(profile.Interfaces))
			for _, candidate := range profile.Interfaces {
				selected[candidate.SourceIdentity] = candidate.ProviderInterface
				if candidate.Protocol == nil {
					queue = append(queue, candidate.SourceIdentity)
				}
			}
			visited := make(map[string]struct{})
			for len(queue) != 0 {
				identity := queue[0]
				queue = queue[1:]
				if _, seen := visited[identity]; seen {
					continue
				}
				visited[identity] = struct{}{}
				contract := interfaces[identity]
				if contract == nil {
					continue
				}
				children := make(map[string]struct{})
				for index := range contract.NumMethods() {
					collectBoundProviderInterfaces(
						contract.Method(index).Type(),
						identities,
						providers,
						make(map[types.Type]struct{}),
						children,
					)
				}
				ordered := sortedIdentityKeys(children)
				for _, child := range ordered {
					if child == identity {
						continue
					}
					queue = append(queue, child)
					ordinary := providers[child]
					required, requiredErr := providerInterfaceNeedsCooperative(
						child,
						ordinary,
					)
					if requiredErr != nil {
						return requiredErr
					}
					cooperative, ok := selected[child]
					if required && (!ok || !providerInterfaceIsCooperative(
						ordinary,
						cooperative,
					)) {
						differences = append(differences, fmt.Sprintf(
							"%s interface %s transitively requires cooperative %s",
							profile.SourceIdentity,
							identity,
							child,
						))
					}
				}
			}
		}
	}
	if len(differences) == 0 {
		return nil
	}
	sort.Strings(differences)
	return certifyError(
		"verify provider profile interface closure",
		"provider surface",
		strings.Join(differences, "; "),
	)
}

func sourceProviderInterfaceTypes(
	source goSurface,
	providers map[string]gostdlib.ProviderInterfaceDocument,
) map[string]*types.Interface {
	result := make(map[string]*types.Interface, len(providers))
	for identity := range providers {
		var sourceType types.Type
		if identity == gostdlib.LanguageErrorInterfaceIdentity {
			sourceType = types.Universe.Lookup("error").Type()
		} else if evidence, ok := source.objects[identity]; ok &&
			evidence.object != nil {
			sourceType = evidence.object.Type()
		}
		if sourceType == nil {
			continue
		}
		named, ok := types.Unalias(sourceType).(*types.Named)
		if !ok {
			continue
		}
		contract, ok := named.Underlying().(*types.Interface)
		if ok {
			result[identity] = contract.Complete()
		}
	}
	return result
}

func sortedIdentityKeys(source map[string]struct{}) []string {
	result := make([]string, 0, len(source))
	for identity := range source {
		result = append(result, identity)
	}
	sort.Strings(result)
	return result
}

func ordinaryProviderInterfaces(
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) (map[string]gostdlib.ProviderInterfaceDocument, error) {
	result := make(map[string]gostdlib.ProviderInterfaceDocument)
	add := func(identity string, provider gostdlib.ProviderInterfaceDocument) error {
		if _, duplicate := result[identity]; duplicate {
			return certifyError(
				"verify interface parameter profile coverage",
				identity,
				"ordinary provider interface has multiple owners",
			)
		}
		result[identity] = provider
		return nil
	}
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if binding.ProviderInterface != nil {
				if err := add(binding.Identity, *binding.ProviderInterface); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, module := range facetModules {
		for _, binding := range module.ProviderInterfaces {
			if err := add(binding.SourceIdentity, binding.ProviderInterface); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func sourceTypeIdentities(source goSurface) map[types.Object]string {
	result := make(map[types.Object]string, len(source.objects)+1)
	for identity, evidence := range source.objects {
		if evidence.object != nil {
			result[evidence.object] = identity
		}
	}
	result[types.Universe.Lookup("error")] = gostdlib.LanguageErrorInterfaceIdentity
	return result
}

func collectBoundProviderInterfaces(
	source types.Type,
	identities map[types.Object]string,
	providers map[string]gostdlib.ProviderInterfaceDocument,
	visited map[types.Type]struct{},
	result map[string]struct{},
) {
	if source == nil {
		return
	}
	source = types.Unalias(source)
	if _, seen := visited[source]; seen {
		return
	}
	visited[source] = struct{}{}
	switch selected := source.(type) {
	case *types.Named:
		if _, ok := selected.Underlying().(*types.Interface); ok {
			if identity, found := identities[selected.Obj()]; found {
				if _, bound := providers[identity]; bound {
					result[identity] = struct{}{}
					return
				}
			}
		}
		collectBoundProviderInterfaces(
			selected.Underlying(), identities, providers, visited, result,
		)
	case *types.Struct:
		for index := range selected.NumFields() {
			collectBoundProviderInterfaces(
				selected.Field(index).Type(), identities, providers, visited, result,
			)
		}
	case *types.Interface:
		selected = selected.Complete()
		for index := range selected.NumMethods() {
			collectBoundProviderInterfaces(
				selected.Method(index).Type(), identities, providers, visited, result,
			)
		}
	case *types.Pointer:
		collectBoundProviderInterfaces(selected.Elem(), identities, providers, visited, result)
	case *types.Slice:
		collectBoundProviderInterfaces(selected.Elem(), identities, providers, visited, result)
	case *types.Array:
		collectBoundProviderInterfaces(selected.Elem(), identities, providers, visited, result)
	case *types.Map:
		collectBoundProviderInterfaces(selected.Key(), identities, providers, visited, result)
		collectBoundProviderInterfaces(selected.Elem(), identities, providers, visited, result)
	case *types.Chan:
		collectBoundProviderInterfaces(selected.Elem(), identities, providers, visited, result)
	case *types.Signature:
		collectBoundProviderInterfaces(selected.Params(), identities, providers, visited, result)
		collectBoundProviderInterfaces(selected.Results(), identities, providers, visited, result)
	case *types.Tuple:
		for index := range selected.Len() {
			collectBoundProviderInterfaces(
				selected.At(index).Type(), identities, providers, visited, result,
			)
		}
	}
}

func providerInterfaceNeedsCooperative(
	identity string,
	provider gostdlib.ProviderInterfaceDocument,
) (bool, error) {
	if provider.Mode == gostdlib.ProviderInterfaceModeSealedNative {
		return false, nil
	}
	required := false
	for _, method := range provider.Methods {
		if method.Kind != gostdlib.ProviderInterfaceMethodCallable {
			continue
		}
		switch method.Effect {
		case gostdlib.EffectSynchronous:
			required = true
		case gostdlib.EffectAwaitable:
		case gostdlib.EffectAsynchronous:
			return false, certifyError(
				"verify interface parameter profile coverage",
				identity,
				"ordinary provider interface method is Promise-only",
			)
		default:
			return false, certifyError(
				"verify interface parameter profile coverage",
				identity,
				"ordinary provider interface method effect is invalid",
			)
		}
	}
	return required, nil
}

func providerInterfaceIsCooperative(
	ordinary gostdlib.ProviderInterfaceDocument,
	cooperative gostdlib.ProviderInterfaceDocument,
) bool {
	if ordinary.Mode != gostdlib.ProviderInterfaceModeBridge ||
		cooperative.Mode != gostdlib.ProviderInterfaceModeBridge ||
		len(ordinary.Methods) != len(cooperative.Methods) {
		return false
	}
	cooperativeMethods := make(
		map[string]gostdlib.ProviderInterfaceMethodDocument,
		len(cooperative.Methods),
	)
	for _, method := range cooperative.Methods {
		cooperativeMethods[method.SourceIdentity] = method
	}
	for _, method := range ordinary.Methods {
		selected, ok := cooperativeMethods[method.SourceIdentity]
		if !ok || selected.Kind != method.Kind ||
			selected.SourceSignature != method.SourceSignature ||
			selected.ContractSignature != method.ContractSignature {
			return false
		}
		if method.Kind == gostdlib.ProviderInterfaceMethodCallable &&
			selected.Effect != gostdlib.EffectAwaitable {
			return false
		}
	}
	return true
}
