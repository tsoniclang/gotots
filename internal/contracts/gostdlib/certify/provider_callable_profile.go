package certify

import (
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type providerCallableProfileBuild struct {
	profile gostdlib.ProviderCallableProfileDocument
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
	if err := validateCanonicalProfileValues(
		source,
		seed.CanonicalValues,
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
	if err := validateImplementedResultInterfaces(
		source,
		seed,
		signature,
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
	if profileTarget.TypeParameterCount() != len(seed.CanonicalTypeArguments) {
		return providerCallableProfileBuild{}, certifyError(
			"build provider callable profile",
			seed.Export,
			"target type-parameter count does not match canonical type arguments",
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
	callableParameters, keyCallables, err := buildProfileCallableParameters(
		seed,
		signature,
		profileTarget,
		project,
		effectMarker,
	)
	if err != nil {
		return providerCallableProfileBuild{}, err
	}
	interfaces := make(
		[]gostdlib.ProviderCallableProfileInterfaceDocument,
		0,
		len(seed.Interfaces)+len(seed.Protocols),
	)
	keyInterfaces := make(
		[]gostdlib.ProviderCallableProfileKeyInterface,
		0,
		len(seed.Interfaces)+len(seed.Protocols),
	)
	guardInterfaces := slices.Clone(seed.GuardInterfaces)
	appendInterface := func(
		identity string,
		export string,
		protocol *gostdlib.ProviderProtocolInterfaceDocument,
		protocolValueParameter *int,
		providerInterface *gostdlib.ProviderInterfaceDocument,
		target tsgo.ProjectExport,
	) error {
		if providerInterface == nil ||
			providerInterface.Mode != gostdlib.ProviderInterfaceModeBridge {
			return certifyError(
				"build provider callable profile",
				identity,
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
			SourceIdentity: identity,
			Methods:        methods,
		})
		document := gostdlib.ProviderCallableProfileInterfaceDocument{
			SourceIdentity:    identity,
			Export:            export,
			ProviderInterface: *providerInterface,
			TargetFingerprint: target.Fingerprint(),
		}
		if protocol != nil {
			selected := *protocol
			document.Protocol = &selected
			parameter := *protocolValueParameter
			document.ProtocolValueParameter = &parameter
		}
		interfaces = append(interfaces, document)
		return nil
	}
	for _, selected := range seed.Interfaces {
		target, ok := targets[selected.Export]
		if !ok {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
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
			return providerCallableProfileBuild{}, err
		}
		if err := appendInterface(
			selected.SourceIdentity,
			selected.Export,
			nil,
			nil,
			providerInterface,
			target,
		); err != nil {
			return providerCallableProfileBuild{}, err
		}
	}
	for _, selected := range seed.Protocols {
		if selected.ValueParameter == nil ||
			*selected.ValueParameter < 0 ||
			*selected.ValueParameter >= signature.Params().Len() {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
				seed.SourceIdentity,
				"protocol value parameter is outside the source callable signature",
			)
		}
		identity, err := gostdlib.BuildProviderProtocolInterfaceIdentity(
			selected.document,
		)
		if err != nil {
			return providerCallableProfileBuild{}, err
		}
		target, ok := targets[selected.Export]
		if !ok {
			return providerCallableProfileBuild{}, certifyError(
				"build provider callable profile",
				selected.Export,
				"protocol target export is absent",
			)
		}
		providerInterface, err := buildProtocolProviderInterface(
			selected.document,
			identity,
			signature,
			target,
			project,
			effectMarker,
		)
		if err != nil {
			return providerCallableProfileBuild{}, err
		}
		protocol := selected.document
		if err := appendInterface(
			identity,
			selected.Export,
			&protocol,
			selected.ValueParameter,
			providerInterface,
			target,
		); err != nil {
			return providerCallableProfileBuild{}, err
		}
		guardInterfaces = append(guardInterfaces, identity)
	}
	profileKey, err := gostdlib.BuildImplementedResultProfileKey(
		keyInterfaces,
		keyCallables,
		seed.ImplementedResultInterfaces,
	)
	if err != nil {
		return providerCallableProfileBuild{}, err
	}
	sort.Slice(interfaces, func(left, right int) bool {
		return interfaces[left].SourceIdentity < interfaces[right].SourceIdentity
	})
	return providerCallableProfileBuild{
		profile: gostdlib.ProviderCallableProfileDocument{
			SourceIdentity:         seed.SourceIdentity,
			ProfileKey:             profileKey,
			Export:                 seed.Export,
			Required:               seed.Required,
			Receiver:               seed.Receiver,
			CanonicalParameters:    slices.Clone(seed.CanonicalParameters),
			CanonicalResults:       slices.Clone(seed.CanonicalResults),
			CanonicalValues:        slices.Clone(seed.CanonicalValues),
			CanonicalTypeArguments: slices.Clone(seed.CanonicalTypeArguments),
			GuardInterfaces:        guardInterfaces,
			ContractInterfaces:     slices.Clone(seed.ContractInterfaces),
			FromProviderInterfaces: slices.Clone(seed.FromProviderInterfaces),
			ImplementedResultInterfaces: slices.Clone(
				seed.ImplementedResultInterfaces,
			),
			Interfaces:          interfaces,
			CallableParameters:  callableParameters,
			Effect:              effect,
			ImplementationOwner: owner,
			TargetFingerprint:   profileTarget.Fingerprint(),
		},
	}, nil
}

func buildProfileCallableParameters(
	seed providerCallableProfileSeed,
	signature *types.Signature,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (
	[]gostdlib.ProviderCallableParameterDocument,
	[]gostdlib.ProviderCallableProfileKeyCallable,
	error,
) {
	var documents []gostdlib.ProviderCallableParameterDocument
	var keys []gostdlib.ProviderCallableProfileKeyCallable
	targetOffset := 0
	if seed.Receiver {
		targetOffset = 1
	}
	for _, parameter := range seed.CanonicalParameters {
		if _, callable := gostdlibsource.DirectCallableParameterSignature(
			signature.Params().At(parameter).Type(),
		); !callable {
			continue
		}
		effect, err := parameterCallableEffect(
			project,
			target,
			parameter+targetOffset,
			effectMarker,
		)
		if err != nil {
			return nil, nil, err
		}
		if effect != gostdlib.EffectSynchronous &&
			effect != gostdlib.EffectAwaitable {
			return nil, nil, certifyError(
				"build provider callable profile",
				seed.SourceIdentity,
				"transported callable parameter is neither direct nor awaitable",
			)
		}
		documents = append(
			documents,
			gostdlib.ProviderCallableParameterDocument{
				Parameter: parameter,
				Effect:    effect,
			},
		)
		keys = append(keys, gostdlib.ProviderCallableProfileKeyCallable{
			Parameter: parameter,
			Effect:    effect,
		})
	}
	return documents, keys, nil
}

func validateImplementedResultInterfaces(
	source goSurface,
	seed providerCallableProfileSeed,
	signature *types.Signature,
) error {
	if len(seed.ImplementedResultInterfaces) == 0 {
		return nil
	}
	canonicalResults := make(map[int]struct{}, len(seed.CanonicalResults))
	for _, index := range seed.CanonicalResults {
		canonicalResults[index] = struct{}{}
	}
	for _, identity := range seed.ImplementedResultInterfaces {
		evidence, ok := source.objects[identity]
		if !ok {
			return certifyError(
				"build provider callable profile",
				identity,
				"implemented result interface is absent from the selected Go surface",
			)
		}
		typeName, ok := evidence.object.(*types.TypeName)
		if !ok {
			return certifyError(
				"build provider callable profile",
				identity,
				"implemented result owner is not a named type",
			)
		}
		if _, ok := types.Unalias(typeName.Type()).Underlying().(*types.Interface); !ok {
			return certifyError(
				"build provider callable profile",
				identity,
				"implemented result owner is not an interface",
			)
		}
		matched := false
		for index := range signature.Results().Len() {
			if _, canonical := canonicalResults[index]; !canonical {
				continue
			}
			if types.Identical(
				types.Unalias(signature.Results().At(index).Type()),
				types.Unalias(typeName.Type()),
			) {
				matched = true
				break
			}
		}
		if !matched {
			return certifyError(
				"build provider callable profile",
				identity,
				"implemented result interface is not a direct canonical result",
			)
		}
	}
	return nil
}

func buildProtocolProviderInterface(
	protocol gostdlib.ProviderProtocolInterfaceDocument,
	identity string,
	owner *types.Signature,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (*gostdlib.ProviderInterfaceDocument, error) {
	interfaceType, err := gostdlibsource.ResolveProviderProtocolInterface(protocol, owner)
	if err != nil {
		return nil, err
	}
	return buildProviderInterfaceContract(
		interfaceType,
		target,
		project,
		effectMarker,
		func(method *types.Func) (providerInterfaceMethodSource, error) {
			methodDocument, ok := gostdlib.ProviderProtocolMethod(
				protocol,
				method.Name(),
			)
			if !ok {
				return providerInterfaceMethodSource{}, certifyError(
					"build provider protocol",
					identity,
					"method is absent from its protocol",
				)
			}
			methodIdentity, signature, err := gostdlib.ProviderProtocolMethodSource(
				identity,
				methodDocument,
			)
			if err != nil {
				return providerInterfaceMethodSource{}, err
			}
			return providerInterfaceMethodSource{
				identity:  methodIdentity,
				signature: signature,
				location:  "provider-protocol:" + identity,
			}, nil
		},
	)
}

func validateCanonicalProfileValues(
	source goSurface,
	identities []string,
	profileIdentity string,
) error {
	for _, identity := range identities {
		evidence, ok := source.objects[identity]
		if !ok {
			return certifyError(
				"build provider callable profile",
				profileIdentity,
				"canonical value source is absent: "+identity,
			)
		}
		variable, ok := evidence.object.(*types.Var)
		if !ok || variable.IsField() || variable.Pkg() == nil ||
			variable.Parent() != variable.Pkg().Scope() {
			return certifyError(
				"build provider callable profile",
				profileIdentity,
				"canonical value is not a package variable: "+identity,
			)
		}
	}
	return nil
}

func buildProfileProviderInterface(
	selectedToolchain toolchain,
	source goSurface,
	seed providerCallableProfileInterfaceSeed,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (*gostdlib.ProviderInterfaceDocument, error) {
	if interfaceEvidence, ok := source.objects[seed.SourceIdentity]; ok {
		typeName, ok := interfaceEvidence.object.(*types.TypeName)
		if !ok || typeName.Pkg() == nil {
			return nil, certifyError(
				"build provider callable profile",
				seed.SourceIdentity,
				"profile-interface owner is not a package type",
			)
		}
		return buildProviderInterface(
			selectedToolchain,
			source.packages[typeName.Pkg().Path()],
			typeName,
			target,
			project,
			effectMarker,
		)
	}
	providerInterface, ok, err := buildLanguageProviderInterface(
		seed.SourceIdentity,
		target,
		project,
		effectMarker,
	)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, certifyError(
			"build provider callable profile",
			seed.SourceIdentity,
			"selected-GOROOT interface is absent",
		)
	}
	return providerInterface, nil
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
