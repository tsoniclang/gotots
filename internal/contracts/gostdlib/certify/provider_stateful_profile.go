package certify

import (
	"go/types"
	"slices"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type providerStatefulProfileBuild struct {
	profile gostdlib.ProviderStatefulProfileDocument
}

func buildProviderStatefulProfile(
	selectedToolchain toolchain,
	source goSurface,
	seed providerStatefulProfileSeed,
	targets map[string]tsgo.ProjectExport,
	bindings map[string]gostdlib.BindingDocument,
	facets []facetSeed,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (providerStatefulProfileBuild, error) {
	evidence, ok := source.objects[seed.SourceIdentity]
	if !ok {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"selected-GOROOT type is absent",
		)
	}
	typeName, ok := evidence.object.(*types.TypeName)
	if !ok || typeName.IsAlias() || !typeName.Exported() {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"source owner is not an exported named concrete type",
		)
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"source owner has no named type",
		)
	}
	if _, isInterface := named.Underlying().(*types.Interface); isInterface {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"source owner is an interface",
		)
	}
	ordinary, ok := bindings[seed.SourceIdentity]
	if !ok || ordinary.Kind != gostdlib.BindingType ||
		ordinary.Access != gostdlib.AccessExport ||
		ordinary.Representation != gostdlib.RepresentationDirect {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"source type has no direct public provider binding",
		)
	}
	retained, err := retainedProviderInterfaces(named)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	seedIdentities := make([]string, len(seed.Interfaces))
	for index, selected := range seed.Interfaces {
		seedIdentities[index] = selected.SourceIdentity
	}
	retainedIdentities := make([]string, 0, len(retained))
	for identity := range retained {
		retainedIdentities = append(retainedIdentities, identity)
	}
	sort.Strings(retainedIdentities)
	if !slices.Equal(seedIdentities, retainedIdentities) {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.SourceIdentity,
			"configured retained interfaces do not exact-join the selected Go representation",
		)
	}
	target, ok := targets[seed.Export]
	if !ok {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.Export,
			"profile target export is absent",
		)
	}
	if target.TypeParameterCount() != len(seed.TypeArguments) {
		return providerStatefulProfileBuild{}, certifyError(
			"build provider stateful profile",
			seed.Export,
			"target type-parameter count does not match retained interfaces",
		)
	}
	implementationOwner, err := singleImplementationOwner(
		seed.Export,
		target.ImplementationOwners(),
	)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	interfaces, keyInterfaces, err := buildStatefulProfileInterfaces(
		selectedToolchain,
		source,
		seed,
		targets,
		project,
		effectMarker,
	)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	profileKey, err := gostdlib.BuildProviderCallableProfileKey(keyInterfaces)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	methods, err := buildStatefulProfileMethods(
		source,
		typeName,
		target,
		bindings,
		project,
		effectMarker,
	)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	operations, err := buildStatefulProfileOperations(seed, facets, target)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	fields, err := buildStatefulProfileFields(
		selectedToolchain,
		source,
		typeName,
		named,
		target,
	)
	if err != nil {
		return providerStatefulProfileBuild{}, err
	}
	if err := verifyStatefulProfileTargetMembers(
		target,
		fields,
		methods,
		operations,
	); err != nil {
		return providerStatefulProfileBuild{}, err
	}
	return providerStatefulProfileBuild{
		profile: gostdlib.ProviderStatefulProfileDocument{
			SourceIdentity:      seed.SourceIdentity,
			ProfileKey:          profileKey,
			Export:              seed.Export,
			Interfaces:          interfaces,
			TypeArguments:       slices.Clone(seed.TypeArguments),
			Operations:          slices.Clone(operations),
			Fields:              fields,
			Methods:             methods,
			ImplementationOwner: implementationOwner,
			TargetFingerprint:   target.Fingerprint(),
		},
	}, nil
}

func buildStatefulProfileInterfaces(
	selectedToolchain toolchain,
	source goSurface,
	seed providerStatefulProfileSeed,
	targets map[string]tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (
	[]gostdlib.ProviderCallableProfileInterfaceDocument,
	[]gostdlib.ProviderCallableProfileKeyInterface,
	error,
) {
	interfaces := make(
		[]gostdlib.ProviderCallableProfileInterfaceDocument,
		0,
		len(seed.Interfaces),
	)
	keys := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(seed.Interfaces),
	)
	for _, selected := range seed.Interfaces {
		target, ok := targets[selected.Export]
		if !ok {
			return nil, nil, certifyError(
				"build provider stateful profile",
				selected.Export,
				"profile-interface target export is absent",
			)
		}
		providerInterface, err := buildProfileProviderInterface(
			selectedToolchain,
			source,
			selected,
			target,
			project,
			effectMarker,
		)
		if err != nil {
			return nil, nil, err
		}
		if providerInterface == nil ||
			providerInterface.Mode != gostdlib.ProviderInterfaceModeBridge {
			return nil, nil, certifyError(
				"build provider stateful profile",
				selected.SourceIdentity,
				"profile interface is not a complete public method set",
			)
		}
		methods := make(
			[]gostdlib.ProviderCallableProfileKeyMethod,
			0,
			len(providerInterface.Methods),
		)
		for _, method := range providerInterface.Methods {
			methods = append(methods, gostdlib.ProviderCallableProfileKeyMethod{
				SourceIdentity: method.SourceIdentity,
				Effect:         method.Effect,
			})
		}
		keys = append(keys, gostdlib.ProviderCallableProfileKeyInterface{
			SourceIdentity: selected.SourceIdentity,
			Methods:        methods,
		})
		interfaces = append(interfaces, gostdlib.ProviderCallableProfileInterfaceDocument{
			SourceIdentity:    selected.SourceIdentity,
			Export:            selected.Export,
			ProviderInterface: *providerInterface,
			TargetFingerprint: target.Fingerprint(),
		})
	}
	return interfaces, keys, nil
}

func buildStatefulProfileMethods(
	source goSurface,
	typeName *types.TypeName,
	target tsgo.ProjectExport,
	bindings map[string]gostdlib.BindingDocument,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) ([]gostdlib.ProviderStatefulProfileMethodDocument, error) {
	methods := source.packages[typeName.Pkg().Path()].methodsByType[typeName.Name()]
	result := make([]gostdlib.ProviderStatefulProfileMethodDocument, 0, len(methods))
	for _, evidence := range methods {
		binding, supplied := bindings[evidence.contract.Identity()]
		if !supplied {
			continue
		}
		if binding.Kind != gostdlib.BindingFunction ||
			(binding.Access != gostdlib.AccessStaticMethod &&
				binding.Access != gostdlib.AccessInstanceMethod) {
			return nil, certifyError(
				"build provider stateful profile",
				evidence.contract.Identity(),
				"ordinary provider method binding is invalid",
			)
		}
		instance, instanceOK := target.TypeMember(evidence.object.Name())
		static, staticOK := target.ValueMember(evidence.object.Name())
		if !instanceOK || !instance.Visible() || !staticOK || !static.Visible() {
			return nil, certifyError(
				"build provider stateful profile",
				evidence.contract.Identity(),
				"profile must expose matching instance and static methods",
			)
		}
		instanceEffect, err := memberCallableEffect(project, instance, effectMarker)
		if err != nil {
			return nil, err
		}
		staticEffect, err := memberCallableEffect(project, static, effectMarker)
		if err != nil {
			return nil, err
		}
		if instanceEffect != staticEffect {
			return nil, certifyError(
				"build provider stateful profile",
				evidence.contract.Identity(),
				"instance and static method effects disagree",
			)
		}
		instanceOwner, err := singleImplementationOwner(
			evidence.object.Name(),
			instance.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		staticOwner, err := singleImplementationOwner(
			evidence.object.Name(),
			static.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		if instanceOwner != staticOwner {
			return nil, certifyError(
				"build provider stateful profile",
				evidence.contract.Identity(),
				"instance and static implementation owners disagree",
			)
		}
		result = append(result, gostdlib.ProviderStatefulProfileMethodDocument{
			SourceIdentity:            evidence.contract.Identity(),
			Member:                    evidence.object.Name(),
			Effect:                    staticEffect,
			SourceSignature:           evidence.contract.Signature(),
			SourceLocation:            evidence.location,
			ImplementationOwner:       staticOwner,
			InstanceTargetFingerprint: instance.Fingerprint(),
			StaticTargetFingerprint:   static.Fingerprint(),
		})
	}
	if len(result) == 0 {
		return nil, certifyError(
			"build provider stateful profile",
			typeName.Name(),
			"source type has no supplied provider methods",
		)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].SourceIdentity < result[right].SourceIdentity
	})
	return result, nil
}

func retainedProviderInterfaces(
	source *types.Named,
) (map[string]*types.Named, error) {
	result := make(map[string]*types.Named)
	visited := make(map[types.Type]struct{})
	var collect func(types.Type) error
	collect = func(current types.Type) error {
		if current == nil {
			return nil
		}
		current = types.Unalias(current)
		if _, seen := visited[current]; seen {
			return nil
		}
		visited[current] = struct{}{}
		switch selected := current.(type) {
		case *types.Named:
			if contract, ok := selected.Underlying().(*types.Interface); ok {
				if !contract.Complete().IsMethodSet() || selected.Obj() == nil {
					return certifyError(
						"derive provider stateful profile",
						types.TypeString(selected, nil),
						"retained interface is not a named method set",
					)
				}
				identity, err := retainedInterfaceIdentity(selected.Obj())
				if err != nil {
					return err
				}
				result[identity] = selected
				return nil
			}
			return collect(selected.Underlying())
		case *types.Struct:
			for index := range selected.NumFields() {
				if err := collect(selected.Field(index).Type()); err != nil {
					return err
				}
			}
		case *types.Pointer:
			return collect(selected.Elem())
		case *types.Slice:
			return collect(selected.Elem())
		case *types.Array:
			return collect(selected.Elem())
		case *types.Map:
			if err := collect(selected.Key()); err != nil {
				return err
			}
			return collect(selected.Elem())
		case *types.Chan:
			return collect(selected.Elem())
		case *types.Tuple:
			for index := range selected.Len() {
				if err := collect(selected.At(index).Type()); err != nil {
					return err
				}
			}
		case *types.Signature:
			if err := collect(selected.Params()); err != nil {
				return err
			}
			return collect(selected.Results())
		case *types.Interface:
			return certifyError(
				"derive provider stateful profile",
				types.TypeString(selected, nil),
				"retained anonymous interface has no provider identity",
			)
		case *types.TypeParam, *types.Union:
			return certifyError(
				"derive provider stateful profile",
				types.TypeString(selected, nil),
				"retained open type has no closed provider profile",
			)
		}
		return nil
	}
	if err := collect(source.Underlying()); err != nil {
		return nil, err
	}
	return result, nil
}

func retainedInterfaceIdentity(typeName *types.TypeName) (string, error) {
	if typeName == types.Universe.Lookup("error") {
		return gostdlib.LanguageErrorInterfaceIdentity, nil
	}
	contract, err := environmentcontract.Describe(typeName)
	if err != nil {
		return "", err
	}
	return contract.Identity(), nil
}
