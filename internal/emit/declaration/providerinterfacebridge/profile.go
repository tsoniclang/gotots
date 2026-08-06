package providerinterfacebridge

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	gostdlibsource "github.com/tsoniclang/gotots/internal/contracts/gostdlib/sourcecontract"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const profileGeneratedSuffix = "$Generated"

func BuildProfile(
	context api.Context,
	children api.ChildEmitter,
	name string,
	source *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	providerCapabilities []CapabilityContract,
	capabilityContracts []ProfileCapabilityContract,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if name == "" || source == nil || source.Obj() == nil || len(profile) == 0 {
		return nil, nil, shapeError(name, "profile bridge identity is invalid")
	}
	contract, ok := source.Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return nil, nil, shapeError(name, "profile bridge source is not an interface")
	}
	certificate, found, err := profileBridgeCertificate(source, profile)
	if err != nil || !found {
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, shapeError(name, "profile bridge certificate is absent")
	}
	methods, matched, err := gostdlibsource.SelectProviderInterfaceMethods(
		certificate,
		contract.Complete(),
	)
	if err != nil {
		return nil, nil, err
	}
	if !matched || len(methods) != contract.NumMethods() {
		return nil, nil, shapeError(name, "profile bridge method certificate exact join failed")
	}
	providerContext, err := context.WithProviderScalarRepresentation()
	if err != nil {
		return nil, nil, err
	}
	providerContext, err = providerContext.WithProviderProfile(profile)
	if err != nil {
		return nil, nil, err
	}
	canonical, err := context.Names().InterfaceContract(source)
	if err != nil {
		return nil, nil, err
	}
	directBridge, directBridgeFound, err :=
		context.Names().ProviderInterfaceBridge(source)
	if err != nil {
		return nil, nil, err
	}
	runtimeBase, err := context.Names().Runtime(
		api.RuntimeProviderInterfaceBridge,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	runtimeValue, err := context.Names().Runtime(
		api.RuntimeInterfaceValue,
		api.ImportPhaseType,
	)
	if err != nil {
		return nil, nil, err
	}
	contractName := name + api.ProviderProfileContractSuffix
	reverseName := name + profileGeneratedSuffix
	protocolCapabilities, protocolRequests, err := selectCapabilities(
		context,
		name,
		source,
		providerCapabilities,
		certificate.ProviderInterface(),
		true,
	)
	if err != nil {
		return nil, nil, err
	}
	protocolConflicts, err := capabilityConflicts(protocolCapabilities)
	if err != nil {
		return nil, nil, err
	}
	var panicReference api.NameReference
	if len(protocolConflicts) != 0 {
		panicReference, err = context.Names().Runtime(
			api.RuntimePanic,
			api.ImportPhaseValue,
		)
		if err != nil {
			return nil, nil, err
		}
	}
	capabilities, capabilityRequests, err := selectProfileCapabilities(
		context,
		name,
		source,
		profile,
		capabilityContracts,
	)
	if err != nil {
		return nil, nil, err
	}
	contractMembers := make([]tsgo.TypeElement, 0, len(methods))
	forwardMembers := make(
		[]tsgo.ClassElement,
		0,
		len(methods)+len(protocolCapabilities)*2+3,
	)
	reverseMembers := make(
		[]tsgo.ClassElement,
		0,
		len(methods)+len(protocolCapabilities)*2+2,
	)
	var requests []api.RootRequest
	for _, capability := range protocolCapabilities {
		forwardMembers = append(
			forwardMembers,
			capabilityFieldDeclaration(context.Factory(), capability),
		)
		reverseMembers = append(
			reverseMembers,
			profileReverseCapabilityFieldDeclaration(
				context.Factory(),
				capability,
			),
		)
	}
	for _, selected := range methods {
		contractMember, memberRequests, memberErr := profileContractMethod(
			providerContext,
			children,
			selected.Method,
			selected.Certificate,
		)
		if memberErr != nil {
			return nil, nil, memberErr
		}
		forward, forwardRequests, forwardErr := profileForwardMethod(
			context,
			children,
			name,
			source,
			profile,
			selected.Method,
			selected.Certificate,
		)
		if forwardErr != nil {
			return nil, nil, forwardErr
		}
		reverse, reverseRequests, reverseErr := profileReverseMethod(
			context,
			providerContext,
			children,
			name,
			source,
			profile,
			selected.Method,
			selected.Certificate,
		)
		if reverseErr != nil {
			return nil, nil, reverseErr
		}
		contractMembers = append(contractMembers, contractMember)
		forwardMembers = append(forwardMembers, forward)
		reverseMembers = append(reverseMembers, reverse)
		requests = append(requests, memberRequests...)
		requests = append(requests, forwardRequests...)
		requests = append(requests, reverseRequests...)
	}
	forwardCapabilityMembers, forwardCapabilityRequests, err :=
		emitCapabilityMethods(
			context,
			children,
			name,
			source,
			protocolCapabilities,
		)
	if err != nil {
		return nil, nil, err
	}
	reverseCapabilityMembers, reverseCapabilityRequests, err :=
		emitProfileReverseCapabilityMethods(
			context,
			providerContext,
			children,
			name,
			source,
			profile,
			protocolCapabilities,
		)
	if err != nil {
		return nil, nil, err
	}
	forwardMembers = append(forwardMembers, forwardCapabilityMembers...)
	reverseMembers = append(reverseMembers, reverseCapabilityMembers...)
	forwardMembers = append(
		[]tsgo.ClassElement{
			constructor(
				context.Factory(),
				profileNamedType(context.Factory(), contractName),
				canonical.ContractName(),
				protocolCapabilities,
				protocolConflicts,
				panicReference.Name(),
			),
			profileFromMethod(
				context.Factory(),
				name,
				reverseName,
				contractName,
				canonical.TypeName(),
			),
			profileToMethod(
				context.Factory(),
				name,
				reverseName,
				directBridge.Name(),
				contractName,
				canonical.TypeName(),
			),
		},
		forwardMembers...,
	)
	reverseMembers = append(
		[]tsgo.ClassElement{
			profileReverseConstructor(
				context.Factory(),
				profileNamedType(context.Factory(), canonical.TypeName()),
				canonical.ContractName(),
				protocolCapabilities,
				protocolConflicts,
				panicReference.Name(),
			),
			profileGeneratedValueMethod(
				context.Factory(),
				canonical.TypeName(),
			),
		},
		reverseMembers...,
	)
	statements := []tsgo.Statement{
		profileContractDeclaration(
			context.Factory(),
			contractName,
			runtimeValue.Name(),
			contractMembers,
			modifiers,
		),
		profileBridgeClass(
			context.Factory(),
			reverseName,
			runtimeBase.Name(),
			canonical.TypeName(),
			contractName,
			reverseMembers,
			nil,
		),
		profileBridgeClass(
			context.Factory(),
			name,
			runtimeBase.Name(),
			contractName,
			canonical.TypeName(),
			forwardMembers,
			modifiers,
		),
	}
	for _, capability := range capabilities {
		statements = append(
			statements,
			profileCapabilityDeclarations(
				context.Factory(),
				capability,
				contractName,
				reverseName,
				runtimeBase.Name(),
				modifiers,
			)...,
		)
	}
	requests = append(requests, canonical.Requests()...)
	if directBridgeFound {
		requests = append(requests, directBridge.Requests()...)
	}
	requests = append(requests, runtimeBase.Requests()...)
	requests = append(requests, runtimeValue.Requests()...)
	requests = append(requests, protocolRequests...)
	requests = append(requests, panicReference.Requests()...)
	requests = append(requests, forwardCapabilityRequests...)
	requests = append(requests, reverseCapabilityRequests...)
	requests = append(requests, capabilityRequests...)
	return statements, api.CombineRequests(requests), nil
}

func profileBridgeCertificate(
	source *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
) (gostdlib.ProviderCallableProfileInterface, bool, error) {
	identity, err := gostdlibsource.ObjectIdentity(source.Obj())
	if err != nil {
		return gostdlib.ProviderCallableProfileInterface{}, false, err
	}
	for _, selected := range profile {
		if selected.SourceIdentity() == identity {
			return selected, true, nil
		}
	}
	return gostdlib.ProviderCallableProfileInterface{}, false, nil
}
