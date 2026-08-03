package certify

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

type interfaceParameterRequirement struct {
	parameter int
	identity  string
}

func verifyInterfaceParameterProfileCoverage(
	source goSurface,
	modules []gostdlib.ModuleDocument,
	facetModules []gostdlib.FacetModuleDocument,
) error {
	providers, err := ordinaryProviderInterfaces(modules, facetModules)
	if err != nil {
		return err
	}
	identities := sourceTypeIdentities(source)
	expected := make(map[string]int)
	actual := make(map[string]int)
	for _, module := range modules {
		for _, binding := range module.Bindings {
			if binding.Kind != gostdlib.BindingFunction {
				continue
			}
			evidence, ok := source.objects[binding.Identity]
			if !ok {
				continue
			}
			signature, ok := evidence.object.Type().(*types.Signature)
			if !ok || sourceCallableTypeParameterCount(signature) != 0 {
				continue
			}
			requirements, requirementErr := interfaceParameterRequirements(
				signature,
				identities,
				providers,
			)
			if requirementErr != nil {
				return requirementErr
			}
			if len(requirements) != 0 {
				expected[interfaceParameterRequirementKey(
					binding.Identity,
					requirements,
				)]++
			}
		}
	}
	for _, module := range facetModules {
		for _, profile := range module.CallableProfiles {
			evidence, ok := source.objects[profile.SourceIdentity]
			if !ok {
				continue
			}
			signature, ok := evidence.object.Type().(*types.Signature)
			if !ok || sourceCallableTypeParameterCount(signature) != 0 {
				continue
			}
			requirements, requirementErr := profileInterfaceParameterRequirements(
				profile,
				signature,
				identities,
				providers,
			)
			if requirementErr != nil {
				return requirementErr
			}
			if len(requirements) != 0 {
				actual[interfaceParameterRequirementKey(
					profile.SourceIdentity,
					requirements,
				)]++
			}
		}
	}
	var differences []string
	for key, count := range expected {
		if actual[key] != count {
			differences = append(differences, fmt.Sprintf(
				"%s expected %d profile(s), certified %d",
				key,
				count,
				actual[key],
			))
		}
	}
	for key, count := range actual {
		if expected[key] != count {
			differences = append(differences, fmt.Sprintf(
				"%s certified %d profile(s), expected %d",
				key,
				count,
				expected[key],
			))
		}
	}
	if len(differences) == 0 {
		return nil
	}
	sort.Strings(differences)
	return certifyError(
		"verify interface parameter profile coverage",
		"provider surface",
		strings.Join(differences, "; "),
	)
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

func interfaceParameterRequirements(
	signature *types.Signature,
	identities map[types.Object]string,
	providers map[string]gostdlib.ProviderInterfaceDocument,
) ([]interfaceParameterRequirement, error) {
	var result []interfaceParameterRequirement
	for parameter := range signature.Params().Len() {
		roots := make(map[string]struct{})
		collectBoundProviderInterfaces(
			signature.Params().At(parameter).Type(),
			identities,
			providers,
			make(map[types.Type]struct{}),
			roots,
		)
		for identity := range roots {
			required, err := providerInterfaceNeedsCooperative(
				identity,
				providers[identity],
			)
			if err != nil {
				return nil, err
			}
			if required {
				result = append(result, interfaceParameterRequirement{
					parameter: parameter,
					identity:  identity,
				})
			}
		}
	}
	sortInterfaceParameterRequirements(result)
	return result, nil
}

func profileInterfaceParameterRequirements(
	profile gostdlib.ProviderCallableProfileDocument,
	signature *types.Signature,
	identities map[types.Object]string,
	providers map[string]gostdlib.ProviderInterfaceDocument,
) ([]interfaceParameterRequirement, error) {
	profileInterfaces := make(
		map[string]gostdlib.ProviderInterfaceDocument,
		len(profile.Interfaces),
	)
	for _, selected := range profile.Interfaces {
		profileInterfaces[selected.SourceIdentity] = selected.ProviderInterface
	}
	var result []interfaceParameterRequirement
	for _, parameter := range profile.CanonicalParameters {
		if parameter < 0 || parameter >= signature.Params().Len() {
			continue
		}
		roots := make(map[string]struct{})
		collectBoundProviderInterfaces(
			signature.Params().At(parameter).Type(),
			identities,
			providers,
			make(map[types.Type]struct{}),
			roots,
		)
		for identity := range roots {
			ordinary, ok := providers[identity]
			if !ok {
				continue
			}
			required, err := providerInterfaceNeedsCooperative(identity, ordinary)
			if err != nil {
				return nil, err
			}
			cooperative, ok := profileInterfaces[identity]
			if !required || !ok || !providerInterfaceIsCooperative(ordinary, cooperative) {
				continue
			}
			result = append(result, interfaceParameterRequirement{
				parameter: parameter,
				identity:  identity,
			})
		}
	}
	sortInterfaceParameterRequirements(result)
	return result, nil
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
			selected.SourceSignature != method.SourceSignature {
			return false
		}
		if method.Kind == gostdlib.ProviderInterfaceMethodCallable &&
			selected.Effect != gostdlib.EffectAwaitable {
			return false
		}
	}
	return true
}

func sortInterfaceParameterRequirements(source []interfaceParameterRequirement) {
	sort.Slice(source, func(left, right int) bool {
		if source[left].parameter != source[right].parameter {
			return source[left].parameter < source[right].parameter
		}
		return source[left].identity < source[right].identity
	})
}

func interfaceParameterRequirementKey(
	identity string,
	requirements []interfaceParameterRequirement,
) string {
	var result strings.Builder
	result.WriteString(identity)
	for _, selected := range requirements {
		fmt.Fprintf(
			&result,
			"|parameter=%d|interface=%s",
			selected.parameter,
			selected.identity,
		)
	}
	return result.String()
}
