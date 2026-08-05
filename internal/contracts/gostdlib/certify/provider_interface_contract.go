package certify

import (
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"sort"
	"strings"
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
		contractSignature, ok := environmentcontract.MethodSignature(method)
		if !ok {
			return nil, certifyError(
				"build provider interface",
				source.identity,
				"method has no receiver-free contract signature",
			)
		}
		document := gostdlib.ProviderInterfaceMethodDocument{
			SourceIdentity:    source.identity,
			SourceSignature:   source.signature,
			ContractSignature: environmentcontract.StableTypeString(contractSignature),
			SourceLocation:    source.location,
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

func buildProviderInterfaceCapability(
	seed providerInterfaceCapabilitySeed,
	profiles []gostdlib.ProviderCallableProfileDocument,
	providerInterfaces []gostdlib.ProviderInterfaceBindingDocument,
	targets map[string]tsgo.ProjectExport,
	project *tsgo.ProjectInspection,
	effectMarker tsgo.ProjectExport,
) (gostdlib.ProviderInterfaceCapabilityDocument, error) {
	var profile *gostdlib.ProviderCallableProfileDocument
	for index := range profiles {
		selected := &profiles[index]
		if selected.SourceIdentity == seed.ProfileSourceIdentity &&
			selected.Export == seed.ProfileExport {
			profile = selected
			break
		}
	}
	if profile == nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.ProfileSourceIdentity,
			"callable profile is absent",
		)
	}
	baseSourceIdentity := ""
	baseExport := ""
	var targetInterface *gostdlib.ProviderCallableProfileInterfaceDocument
	for index := range profile.Interfaces {
		selected := &profile.Interfaces[index]
		if selected.SourceIdentity == seed.BaseSourceIdentity &&
			selected.Export == seed.BaseExport {
			baseSourceIdentity = selected.SourceIdentity
			baseExport = selected.Export
		}
		if selected.Export == seed.TargetExport {
			targetInterface = selected
		}
	}
	if baseSourceIdentity == "" && baseExport == "" {
		for _, selected := range providerInterfaces {
			if selected.SourceIdentity == seed.BaseSourceIdentity &&
				selected.Export == seed.BaseExport {
				baseSourceIdentity = selected.SourceIdentity
				baseExport = selected.Export
				break
			}
		}
	}
	if baseSourceIdentity == "" || baseExport == "" || targetInterface == nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.TargetExport,
			"base or target certified interface is absent",
		)
	}
	baseTarget, baseOK := targets[seed.BaseExport]
	target, targetOK := targets[seed.TargetExport]
	view, viewOK := targets[seed.ViewExport]
	if !baseOK || !targetOK || !viewOK {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.ViewExport,
			"base, target, or view export is absent",
		)
	}
	parameterIdentity, resultIdentity, err :=
		project.CallableOptionalViewTypes(view)
	if err != nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, err
	}
	if !parameterIdentity.Matches(baseTarget) || !resultIdentity.Matches(target) {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.ViewExport,
			"view parameter or result does not exact-join its provider interfaces",
		)
	}
	typeParameters, err := project.CallableTypeParameterCount(view)
	if err != nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, err
	}
	if typeParameters != 0 {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.ViewExport,
			"view is generic",
		)
	}
	effect, err := exportCallableEffect(project, view, effectMarker)
	if err != nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, err
	}
	if effect != gostdlib.EffectSynchronous {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, certifyError(
			"build provider-interface capability",
			seed.ViewExport,
			"view is not synchronous",
		)
	}
	owner, err := singleImplementationOwner(
		seed.ViewExport,
		view.ImplementationOwners(),
	)
	if err != nil {
		return gostdlib.ProviderInterfaceCapabilityDocument{}, err
	}
	return gostdlib.ProviderInterfaceCapabilityDocument{
		Usage:                 seed.Usage,
		BaseSourceIdentity:    baseSourceIdentity,
		BaseExport:            baseExport,
		ProfileSourceIdentity: profile.SourceIdentity,
		ProfileKey:            profile.ProfileKey,
		TargetSourceIdentity:  targetInterface.SourceIdentity,
		TargetExport:          targetInterface.Export,
		ViewExport:            seed.ViewExport,
		ImplementationOwner:   owner,
		ViewFingerprint:       view.Fingerprint(),
	}, nil
}

type providerInterfaceCapabilitySeed struct {
	Usage                 gostdlib.ProviderInterfaceCapabilityUsage `json:"usage"`
	BaseSourceIdentity    string                                    `json:"baseSourceIdentity"`
	BaseExport            string                                    `json:"baseExport"`
	ProfileSourceIdentity string                                    `json:"profileSourceIdentity"`
	ProfileExport         string                                    `json:"profileExport"`
	TargetExport          string                                    `json:"targetExport"`
	Specifier             string                                    `json:"specifier"`
	SourcePath            string                                    `json:"sourcePath"`
	ViewExport            string                                    `json:"viewExport"`
}

func validateProviderInterfaceSeeds(
	source []providerInterfaceSeed,
) ([]providerInterfaceSeed, error) {
	result := append([]providerInterfaceSeed(nil), source...)
	sort.Slice(result, func(left, right int) bool {
		return result[left].SourceIdentity < result[right].SourceIdentity
	})
	previous := ""
	for _, seed := range result {
		if seed.SourceIdentity == "" || seed.SourceIdentity <= previous ||
			seed.Specifier == "" || seed.SourcePath == "" || seed.Export == "" {
			return nil, certifyError(
				"configure provider interfaces",
				seed.SourceIdentity,
				"provider-interface identity or target is incomplete or duplicated",
			)
		}
		previous = seed.SourceIdentity
		if !validFacetModule(seed.Specifier, seed.SourcePath) {
			return nil, certifyError(
				"configure provider interfaces",
				seed.SourceIdentity,
				"provider-interface module is invalid",
			)
		}
	}
	return result, nil
}

func validateProviderInterfaceCapabilitySeeds(
	source []providerInterfaceCapabilitySeed,
	callableProfiles []providerCallableProfileSeed,
	providerInterfaces []providerInterfaceSeed,
) ([]providerInterfaceCapabilitySeed, error) {
	result := append([]providerInterfaceCapabilitySeed(nil), source...)
	sort.Slice(result, func(left, right int) bool {
		return providerInterfaceCapabilitySeedKey(result[left]) <
			providerInterfaceCapabilitySeedKey(result[right])
	})
	profiles := make(map[string]providerCallableProfileSeed, len(callableProfiles))
	for _, selected := range callableProfiles {
		profiles[selected.Specifier+"\x00"+selected.SourceIdentity+"\x00"+selected.Export] = selected
	}
	bases := make(map[string]providerInterfaceSeed, len(providerInterfaces))
	for _, selected := range providerInterfaces {
		bases[selected.Specifier+"\x00"+selected.SourceIdentity+"\x00"+selected.Export] = selected
	}
	previous := ""
	viewOwners := make(map[string]struct{}, len(result))
	for _, seed := range result {
		key := providerInterfaceCapabilitySeedKey(seed)
		if key == "" || key == previous || !seed.Usage.Valid() ||
			seed.BaseSourceIdentity == "" ||
			seed.BaseExport == "" || seed.ProfileSourceIdentity == "" ||
			seed.ProfileExport == "" || seed.TargetExport == "" ||
			seed.ViewExport == "" || !validFacetModule(seed.Specifier, seed.SourcePath) {
			return nil, certifyError(
				"configure provider-interface capabilities",
				key,
				"capability identity or module is incomplete or duplicated",
			)
		}
		previous = key
		profile, ok := profiles[seed.Specifier+"\x00"+seed.ProfileSourceIdentity+"\x00"+seed.ProfileExport]
		if !ok || profile.SourcePath != seed.SourcePath ||
			!profileOwnsInterfaceExport(profile, seed.TargetExport) {
			return nil, certifyError(
				"configure provider-interface capabilities",
				key,
				"base or target interface does not exact-join the selected callable profile",
			)
		}
		baseInProfile := profileOwnsInterface(
			profile,
			seed.BaseSourceIdentity,
			seed.BaseExport,
		)
		_, providerBase := bases[seed.Specifier+"\x00"+
			seed.BaseSourceIdentity+"\x00"+seed.BaseExport]
		if (seed.Usage == gostdlib.ProviderInterfaceCapabilityUsageGeneratedBridge &&
			baseInProfile == providerBase) ||
			(seed.Usage == gostdlib.ProviderInterfaceCapabilityUsageProviderInternal &&
				!providerBase) {
			return nil, certifyError(
				"configure provider-interface capabilities",
				key,
				"base interface does not exact-join its certified usage",
			)
		}
		viewKey := seed.Specifier + "\x00" + seed.ViewExport
		if _, duplicate := viewOwners[viewKey]; duplicate {
			return nil, certifyError(
				"configure provider-interface capabilities",
				key,
				"view export is duplicated",
			)
		}
		viewOwners[viewKey] = struct{}{}
	}
	return result, nil
}

func profileOwnsInterface(
	profile providerCallableProfileSeed,
	sourceIdentity string,
	export string,
) bool {
	for _, selected := range profile.Interfaces {
		if selected.SourceIdentity == sourceIdentity &&
			selected.Export == export {
			return true
		}
	}
	return false
}

func profileOwnsInterfaceExport(
	profile providerCallableProfileSeed,
	export string,
) bool {
	for _, selected := range profile.Interfaces {
		if selected.Export == export {
			return true
		}
	}
	for _, selected := range profile.Protocols {
		if selected.Export == export {
			return true
		}
	}
	return false
}

func providerInterfaceCapabilitySeedKey(
	seed providerInterfaceCapabilitySeed,
) string {
	return seed.Specifier + "\x00" + seed.BaseSourceIdentity + "\x00" +
		seed.TargetExport
}

func validFacetModule(specifier string, sourcePath string) bool {
	subpath, ok := providerSubpath(specifier)
	return ok && strings.HasPrefix(subpath, "./internal/facets/") &&
		strings.HasPrefix(sourcePath, "src/internal/facets/") &&
		strings.HasSuffix(sourcePath, ".ts")
}
