package certify

import (
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type providerCallableProfileBuild struct {
	profile    gostdlib.ProviderCallableProfileDocument
	interfaces []gostdlib.ProviderCallableProfileInterfaceDocument
}

func buildProviderCallableProfile(
	selectedToolchain toolchain,
	source goSurface,
	seed providerCallableProfileSeed,
	targets map[string]tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (providerCallableProfileBuild, error) {
	evidence, ok := source.objects[seed.SourceIdentity]
	if !ok {
		return providerCallableProfileBuild{}, certifyError(
			"build provider callable profile",
			seed.SourceIdentity,
			"selected-GOROOT callable is absent",
		)
	}
	callable, ok := evidence.object.(*types.Func)
	if !ok {
		return providerCallableProfileBuild{}, certifyError(
			"build provider callable profile",
			seed.SourceIdentity,
			"source owner is not a function or method",
		)
	}
	signature, ok := callable.Type().(*types.Signature)
	if !ok || (signature.Recv() != nil) != seed.Receiver {
		return providerCallableProfileBuild{}, certifyError(
			"build provider callable profile",
			seed.SourceIdentity,
			"receiver evidence does not match the source callable",
		)
	}
	if err := validateProfileRootBounds(
		seed.CanonicalParameters,
		signature.Params(),
		"parameter",
		seed.SourceIdentity,
	); err != nil {
		return providerCallableProfileBuild{}, err
	}
	if err := validateProfileRootBounds(
		seed.CanonicalResults,
		signature.Results(),
		"result",
		seed.SourceIdentity,
	); err != nil {
		return providerCallableProfileBuild{}, err
	}
	profileTarget, ok := targets[seed.Export]
	if !ok {
		return providerCallableProfileBuild{}, certifyError(
			"build provider callable profile",
			seed.Export,
			"profile target export is absent",
		)
	}
	owner, err := singleImplementationOwner(
		seed.Export,
		profileTarget.ImplementationOwners(),
	)
	if err != nil {
		return providerCallableProfileBuild{}, err
	}
	effect, err := exportCallableEffect(project, profileTarget, effectMarker)
	if err != nil {
		return providerCallableProfileBuild{}, err
	}
	interfaces := make(
		[]gostdlib.ProviderCallableProfileInterfaceDocument,
		0,
		len(seed.Interfaces),
	)
	keyInterfaces := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(seed.Interfaces),
	)
	for _, selected := range seed.Interfaces {
		interfaceEvidence, ok := source.objects[selected.SourceIdentity]
		if !ok {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
				selected.SourceIdentity,
				"selected-GOROOT interface is absent",
			)
		}
		typeName, ok := interfaceEvidence.object.(*types.TypeName)
		if !ok || typeName.Pkg() == nil {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
				selected.SourceIdentity,
				"profile-interface owner is not a package type",
			)
		}
		sourcePackage := source.packages[typeName.Pkg().Path()]
		target, ok := targets[selected.Export]
		if !ok {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
				selected.Export,
				"profile-interface target export is absent",
			)
		}
		providerInterface, err := buildProviderInterface(
			selectedToolchain,
			sourcePackage,
			typeName,
			target,
			project,
			effectMarker,
		)
		if err != nil {
			return providerCallableProfileBuild{}, err
		}
		if providerInterface == nil ||
			providerInterface.Mode != gostdlib.ProviderInterfaceModeBridge {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
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
		keyInterfaces = append(keyInterfaces, gostdlib.ProviderCallableProfileKeyInterface{
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
	profileKey, err := gostdlib.BuildProviderCallableProfileKey(keyInterfaces)
	if err != nil {
		return providerCallableProfileBuild{}, err
	}
	sort.Slice(interfaces, func(left, right int) bool {
		return interfaces[left].SourceIdentity < interfaces[right].SourceIdentity
	})
	identities := make([]string, len(interfaces))
	for index, selected := range interfaces {
		identities[index] = selected.SourceIdentity
	}
	return providerCallableProfileBuild{
		profile: gostdlib.ProviderCallableProfileDocument{
			SourceIdentity:      seed.SourceIdentity,
			ProfileKey:          profileKey,
			Export:              seed.Export,
			Receiver:            seed.Receiver,
			CanonicalParameters: slices.Clone(seed.CanonicalParameters),
			CanonicalResults:    slices.Clone(seed.CanonicalResults),
			GuardInterfaces:     slices.Clone(seed.GuardInterfaces),
			Interfaces:          identities,
			Effect:              effect,
			ImplementationOwner: owner,
			TargetFingerprint:   profileTarget.Fingerprint(),
		},
		interfaces: interfaces,
	}, nil
}

func sameProviderCallableProfileInterface(
	left gostdlib.ProviderCallableProfileInterfaceDocument,
	right gostdlib.ProviderCallableProfileInterfaceDocument,
) bool {
	return left.SourceIdentity == right.SourceIdentity &&
		left.Export == right.Export &&
		left.TargetFingerprint == right.TargetFingerprint &&
		left.ProviderInterface.Mode == right.ProviderInterface.Mode &&
		slices.Equal(
			left.ProviderInterface.Methods,
			right.ProviderInterface.Methods,
		)
}

func validateProfileRootBounds(
	indexes []int,
	values *types.Tuple,
	name string,
	sourceIdentity string,
) error {
	length := 0
	if values != nil {
		length = values.Len()
	}
	for _, index := range indexes {
		if index < 0 || index >= length {
			return certifyError(
				"build provider callable profile",
				sourceIdentity,
				name+" root is outside the source callable signature",
			)
		}
	}
	return nil
}
