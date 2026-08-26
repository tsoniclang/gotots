package providerinterfacebridge

import (
	"go/types"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type capabilitySelection struct {
	contract  *types.Interface
	key       string
	fieldName string
	reference api.ProviderInterfaceCapabilityReference
	profile   []gostdlib.ProviderCallableProfileInterface
	canonical api.InterfaceContractReference
	methods   []capabilityMethod
}

type capabilityMethod struct {
	method      *types.Func
	certificate gostdlib.ProviderInterfaceMethod
}

type capabilityConflict struct {
	left  string
	right string
}

func selectCapabilities(
	context api.Context,
	name string,
	source *types.Named,
	contracts []CapabilityContract,
	provider gostdlib.ProviderInterface,
	directProviderUse bool,
) ([]capabilitySelection, []api.RootRequest, error) {
	selected := make([]capabilitySelection, 0, len(contracts))
	var requests []api.RootRequest
	for _, candidate := range contracts {
		if candidate.Contract == nil || candidate.Key == "" {
			return nil, nil, shapeError(name, "capability contract is invalid")
		}
		reference, found, err := context.Names().ProviderInterfaceCapability(
			source,
			candidate.Contract,
			candidate.Key,
		)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			return nil, nil, shapeError(
				name,
				"demanded provider-interface capability has no certificate",
			)
		}
		certificate, ok := reference.Certificate()
		if !ok {
			return nil, nil, shapeError(name, "capability certificate is invalid")
		}
		baseSelected, baseRequests, err := capabilityBaseSelected(
			context,
			source,
			provider,
			certificate.BaseInterface(),
			directProviderUse,
		)
		if err != nil {
			return nil, nil, err
		}
		if !baseSelected {
			continue
		}
		targetCertificate := certificate.TargetInterface()
		providerContract := targetCertificate.ProviderInterface()
		if providerContract.Mode() != gostdlib.ProviderInterfaceModeBridge {
			return nil, nil, shapeError(
				name,
				"capability target is not a bridge interface",
			)
		}
		methods, err := selectCapabilityMethods(
			name,
			candidate.Contract,
			targetCertificate,
		)
		if err != nil {
			return nil, nil, err
		}
		methods, err = capabilityMethodsBeyondBase(name, source, methods)
		if err != nil {
			return nil, nil, err
		}
		canonical, err := context.Names().InterfaceContract(
			candidate.Contract,
		)
		if err != nil {
			return nil, nil, err
		}
		selected = append(selected, capabilitySelection{
			contract:  candidate.Contract,
			key:       candidate.Key,
			fieldName: "$go$capability_" + strconv.Itoa(len(selected)),
			reference: reference,
			profile:   certificate.ProfileInterfaces(),
			canonical: canonical,
			methods:   methods,
		})
		requests = append(
			requests,
			baseRequests...,
		)
		requests = append(
			requests,
			reference.Requests()...,
		)
		requests = append(requests, canonical.Requests()...)
	}
	for _, candidate := range contracts {
		matched := false
		for _, capability := range selected {
			if types.Identical(candidate.Contract, capability.contract) {
				matched = true
				break
			}
		}
		if !matched {
			return nil, nil, shapeError(
				name,
				"demanded provider-interface capability has no ABI-compatible certificate",
			)
		}
	}
	return selected, api.CombineRequests(requests), nil
}

func capabilityBaseSelected(
	context api.Context,
	source *types.Named,
	provider gostdlib.ProviderInterface,
	base gostdlib.ProviderCallableProfileInterface,
	directProviderUse bool,
) (bool, []api.RootRequest, error) {
	if source == nil {
		return false, nil, shapeError(
			"",
			"provider-interface capability base source is invalid",
		)
	}
	contract, ok := source.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return false, nil, shapeError(
			"",
			"provider-interface capability base source is invalid",
		)
	}
	baseProvider := base.ProviderInterface()
	direct := sameProviderInterfaceABI(provider, baseProvider)
	if directProviderUse {
		return direct, nil, nil
	}
	if direct {
		return false, nil, nil
	}
	methods, matched, err := gostdlibsource.SelectProviderInterfaceMethods(
		base,
		contract.Complete(),
	)
	if err != nil {
		return false, nil, err
	}
	if !matched || len(methods) != len(baseProvider.Methods()) {
		return false, nil, nil
	}
	var requests []api.RootRequest
	for _, method := range methods {
		reference, referenceErr :=
			context.Names().InterfaceMethodCallable(method.Method)
		if referenceErr != nil {
			return false, nil, referenceErr
		}
		if method.Certificate.Effect() != gostdlib.EffectSynchronous {
			return false, nil, nil
		}
		requests = append(requests, reference.Requests()...)
	}
	return true, api.CombineRequests(requests), nil
}

func sameProviderInterfaceABI(
	left gostdlib.ProviderInterface,
	right gostdlib.ProviderInterface,
) bool {
	leftMethods := left.Methods()
	rightMethods := right.Methods()
	if left.Mode() != right.Mode() || len(leftMethods) != len(rightMethods) {
		return false
	}
	for _, leftMethod := range leftMethods {
		rightMethod, ok := right.Method(leftMethod.SourceIdentity())
		if !ok || leftMethod.Kind() != rightMethod.Kind() ||
			leftMethod.Member() != rightMethod.Member() ||
			leftMethod.Effect() != rightMethod.Effect() ||
			leftMethod.SourceSignature() != rightMethod.SourceSignature() ||
			leftMethod.ContractSignature() != rightMethod.ContractSignature() {
			return false
		}
	}
	return true
}

func capabilityMethodsBeyondBase(
	name string,
	base *types.Named,
	methods []capabilityMethod,
) ([]capabilityMethod, error) {
	methodSet := types.NewMethodSet(base)
	selected := make([]capabilityMethod, 0, len(methods))
	for _, method := range methods {
		prior := methodSet.Lookup(method.method.Pkg(), method.method.Name())
		if prior == nil {
			selected = append(selected, method)
			continue
		}
		priorMethod, ok := prior.Obj().(*types.Func)
		priorSignature, priorOK := receiverFreeSignature(priorMethod)
		methodSignature, methodOK := receiverFreeSignature(method.method)
		if !ok || !priorOK || !methodOK ||
			!types.Identical(priorSignature, methodSignature) {
			return nil, shapeError(
				name,
				"capability conflicts with the provider base method set",
			)
		}
	}
	return selected, nil
}

func selectCapabilityMethods(
	name string,
	contract *types.Interface,
	target gostdlib.ProviderCallableProfileInterface,
) ([]capabilityMethod, error) {
	methods, matched, err := gostdlibsource.SelectProviderInterfaceMethods(
		target,
		contract,
	)
	if err != nil {
		return nil, err
	}
	if !matched {
		return nil, shapeError(
			name,
			"capability method certificate exact join failed",
		)
	}
	selected := make([]capabilityMethod, 0, len(methods))
	for _, method := range methods {
		certificate := method.Certificate
		if certificate.Kind() != gostdlib.ProviderInterfaceMethodCallable ||
			certificate.Member() == "" ||
			!certificate.Effect().Valid() {
			return nil, shapeError(
				name,
				"capability method is not a callable provider member",
			)
		}
		selected = append(selected, capabilityMethod{
			method:      method.Method,
			certificate: certificate,
		})
	}
	return selected, nil
}

func capabilityConflicts(
	capabilities []capabilitySelection,
) ([]capabilityConflict, error) {
	var conflicts []capabilityConflict
	for left := range capabilities {
		for right := left + 1; right < len(capabilities); right++ {
			conflict, err := capabilitiesConflict(
				capabilities[left],
				capabilities[right],
			)
			if err != nil {
				return nil, err
			}
			if conflict {
				conflicts = append(conflicts, capabilityConflict{
					left:  capabilities[left].fieldName,
					right: capabilities[right].fieldName,
				})
			}
		}
	}
	sort.Slice(conflicts, func(left, right int) bool {
		if conflicts[left].left != conflicts[right].left {
			return conflicts[left].left < conflicts[right].left
		}
		return conflicts[left].right < conflicts[right].right
	})
	return conflicts, nil
}

func capabilitiesConflict(
	left capabilitySelection,
	right capabilitySelection,
) (bool, error) {
	for _, leftMethod := range left.methods {
		for _, rightMethod := range right.methods {
			if leftMethod.method.Name() != rightMethod.method.Name() {
				continue
			}
			leftSignature, leftOK := receiverFreeSignature(leftMethod.method)
			rightSignature, rightOK := receiverFreeSignature(rightMethod.method)
			if !leftOK || !rightOK {
				return false, shapeError(
					"",
					"capability conflict has an invalid method signature",
				)
			}
			if !types.Identical(leftSignature, rightSignature) {
				return true, nil
			}
		}
	}
	return false, nil
}
