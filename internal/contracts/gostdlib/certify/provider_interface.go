package certify

import (
	"go/types"
	"sort"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func buildProviderInterface(
	selectedToolchain toolchain,
	sourcePackage *goPackageSurface,
	typeName *types.TypeName,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (*gostdlib.ProviderInterfaceDocument, error) {
	if sourcePackage == nil || sourcePackage.selected == nil || typeName == nil {
		return nil, certifyError(
			"build provider interface",
			"",
			"source evidence is incomplete",
		)
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return nil, nil
	}
	interfaceType, ok := named.Underlying().(*types.Interface)
	if !ok {
		return nil, nil
	}
	interfaceType = interfaceType.Complete()
	if !interfaceType.IsMethodSet() || interfaceType.NumMethods() == 0 {
		return nil, certifyError(
			"build provider interface",
			typeName.Name(),
			"selected Go interface is not a non-empty method set",
		)
	}
	return buildProviderInterfaceContract(
		interfaceType,
		target,
		project,
		effectMarker,
		func(method *types.Func) (providerInterfaceMethodSource, error) {
			contract, err := environmentcontract.Describe(method)
			if err != nil {
				return providerInterfaceMethodSource{}, err
			}
			location, selected, err := selectedGoSourceLocation(
				selectedToolchain.root,
				sourcePackage.selected.Fset,
				method.Pos(),
			)
			if err != nil {
				return providerInterfaceMethodSource{}, certifyError(
					"build provider interface",
					contract.Identity(),
					err.Error(),
				)
			}
			if !selected {
				return providerInterfaceMethodSource{}, certifyError(
					"build provider interface",
					contract.Identity(),
					"method is outside the selected GOROOT",
				)
			}
			return providerInterfaceMethodSource{
				identity:  contract.Identity(),
				signature: contract.Signature(),
				location:  location,
			}, nil
		},
	)
}

type providerInterfaceMethodSource struct {
	identity  string
	signature string
	location  string
}

type providerInterfaceMethodSourceOwner func(
	*types.Func,
) (providerInterfaceMethodSource, error)

func buildProviderInterfaceContract(
	interfaceType *types.Interface,
	target tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
	sourceOwner providerInterfaceMethodSourceOwner,
) (*gostdlib.ProviderInterfaceDocument, error) {
	if interfaceType == nil || sourceOwner == nil {
		return nil, certifyError(
			"build provider interface",
			"",
			"interface method evidence owner is incomplete",
		)
	}
	methods := make(
		[]gostdlib.ProviderInterfaceMethodDocument,
		0,
		interfaceType.NumMethods(),
	)
	mode := gostdlib.ProviderInterfaceModeBridge
	for index := range interfaceType.NumMethods() {
		method := interfaceType.Method(index).Origin()
		source, err := sourceOwner(method)
		if err != nil {
			return nil, err
		}
		document := gostdlib.ProviderInterfaceMethodDocument{
			SourceIdentity:  source.identity,
			SourceSignature: source.signature,
			SourceLocation:  source.location,
		}
		if !method.Exported() {
			mode = gostdlib.ProviderInterfaceModeSealedNative
			document.Kind = gostdlib.ProviderInterfaceMethodRuntimeOnly
			methods = append(methods, document)
			continue
		}
		member, ok := target.TypeMember(method.Name())
		if !ok || !member.Visible() {
			return nil, certifyError(
				"build provider interface",
				source.identity,
				"exported Go method has no visible provider member",
			)
		}
		owner, err := singleImplementationOwner(
			target.Name()+"."+method.Name(),
			member.ImplementationOwners(),
		)
		if err != nil {
			return nil, err
		}
		effect, err := memberCallableEffect(project, member, effectMarker)
		if err != nil {
			return nil, err
		}
		document.Kind = gostdlib.ProviderInterfaceMethodCallable
		document.Member = method.Name()
		document.Effect = effect
		document.ImplementationOwner = owner
		document.TargetFingerprint = member.Fingerprint()
		methods = append(methods, document)
	}
	sort.Slice(methods, func(left, right int) bool {
		return methods[left].SourceIdentity < methods[right].SourceIdentity
	})
	return &gostdlib.ProviderInterfaceDocument{
		Mode:    mode,
		Methods: methods,
	}, nil
}
