package certify

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildLanguageProviderInterfaceBinding(
	seed providerInterfaceSeed,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (gostdlib.ProviderInterfaceBindingDocument, error) {
	providerInterface, ok, err := buildLanguageProviderInterface(
		seed.SourceIdentity,
		target,
		project,
		effectMarker,
	)
	if err != nil {
		return gostdlib.ProviderInterfaceBindingDocument{}, err
	}
	if !ok {
		return gostdlib.ProviderInterfaceBindingDocument{}, certifyError(
			"build provider interface",
			seed.SourceIdentity,
			"language interface identity is unsupported",
		)
	}
	return gostdlib.ProviderInterfaceBindingDocument{
		SourceIdentity:    seed.SourceIdentity,
		Export:            seed.Export,
		ProviderInterface: *providerInterface,
		TargetFingerprint: target.Fingerprint(),
	}, nil
}

func buildLanguageProviderInterface(
	identity string,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (*gostdlib.ProviderInterfaceDocument, bool, error) {
	typeName, ok := languageInterface(identity)
	if !ok {
		return nil, false, nil
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return nil, true, certifyError(
			"build provider interface",
			identity,
			"language interface is not named",
		)
	}
	interfaceType, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, true, certifyError(
			"build provider interface",
			identity,
			"language interface has no method set",
		)
	}
	providerInterface, err := buildProviderInterfaceContract(
		interfaceType.Complete(),
		target,
		project,
		effectMarker,
		func(method *types.Func) (providerInterfaceMethodSource, error) {
			if identity != gostdlib.LanguageErrorInterfaceIdentity ||
				method == nil || method != interfaceType.Complete().Method(0).Origin() {
				return providerInterfaceMethodSource{}, certifyError(
					"build provider interface",
					identity,
					"language method is outside the selected contract",
				)
			}
			methodIdentity, signature, err :=
				gostdlibsource.ProviderInterfaceMethod(method)
			if err != nil {
				return providerInterfaceMethodSource{}, err
			}
			return providerInterfaceMethodSource{
				identity:  methodIdentity,
				signature: signature,
				location:  "builtin",
			}, nil
		},
	)
	if err != nil {
		return nil, true, err
	}
	return providerInterface, true, nil
}

func languageInterface(identity string) (*types.TypeName, bool) {
	if identity != gostdlib.LanguageErrorInterfaceIdentity {
		return nil, false
	}
	typeName, ok := types.Universe.Lookup("error").(*types.TypeName)
	return typeName, ok
}
