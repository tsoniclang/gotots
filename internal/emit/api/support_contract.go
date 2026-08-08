package api

import (
	"encoding/hex"
	"fmt"
	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"slices"
)

type CallableABIResolver interface {
	ResolveCallableABI(*types.Func) (callableabi.Callable, bool)
}

func (c Context) WithCallableABIResolver(
	resolver CallableABIResolver,
) Context {
	c.callableABI = resolver
	return c
}

func (c Context) ResolveCallableABI(
	function *types.Func,
) (callableabi.Callable, bool) {
	if c.callableABI == nil || function == nil {
		return callableabi.Callable{}, false
	}
	return c.callableABI.ResolveCallableABI(function)
}

const ProviderProfileCapabilitySuffix = "$Capability$"

func ProviderProfileCapabilityName(
	bridgeName string,
	contractKey string,
) (string, error) {
	if bridgeName == "" || contractKey == "" {
		return "", &NameError{
			Reason: "provider-profile capability name input is invalid",
		}
	}
	return bridgeName + ProviderProfileCapabilitySuffix +
		hex.EncodeToString([]byte(contractKey)), nil
}

type ProviderProfileBridgeReference struct {
	bridge   NameReference
	contract NameReference
	profile  []gostdlib.ProviderCallableProfileInterface
}

func NewProviderProfileBridgeReference(
	bridge NameReference,
	contract NameReference,
	profile []gostdlib.ProviderCallableProfileInterface,
) (ProviderProfileBridgeReference, error) {
	if bridge.Name() == "" || contract.Name() == "" || len(profile) == 0 {
		return ProviderProfileBridgeReference{}, &NameError{
			Reason: "provider-profile bridge reference is invalid",
		}
	}
	for _, selected := range profile {
		if !selected.Valid() {
			return ProviderProfileBridgeReference{}, &NameError{
				Reason: "provider-profile bridge certificate is invalid",
			}
		}
	}
	return ProviderProfileBridgeReference{
		bridge:   bridge,
		contract: contract,
		profile:  slices.Clone(profile),
	}, nil
}

func (r ProviderProfileBridgeReference) Bridge() NameReference {
	return r.bridge
}

func (r ProviderProfileBridgeReference) Contract() NameReference {
	return r.contract
}

func (r ProviderProfileBridgeReference) Profile() []gostdlib.ProviderCallableProfileInterface {
	return slices.Clone(r.profile)
}

func (r ProviderProfileBridgeReference) Requests() []RootRequest {
	return CombineRequests(r.bridge.Requests(), r.contract.Requests())
}

func (c Context) WithProviderScalarABI(scalar ScalarABI) Context {
	if !scalar.Valid() {
		panic("provider scalar ABI is invalid")
	}
	c.providerScalar = scalar
	return c
}

func (c Context) WithProviderScalarRepresentation() (Context, error) {
	if !c.providerScalar.Valid() {
		return Context{}, &ContextError{Reason: "provider scalar ABI is absent"}
	}
	c.scalar = c.providerScalar
	c.providerScalarRepresentation = true
	return c, nil
}

func (c Context) WithProviderProfile(
	profile []gostdlib.ProviderCallableProfileInterface,
) (Context, error) {
	if len(profile) == 0 {
		return Context{}, &InvariantError{
			Role:   c.Role(),
			Reason: "provider profile interface contract is empty",
		}
	}
	for _, selected := range profile {
		if !selected.Valid() {
			return Context{}, &InvariantError{
				Role:   c.Role(),
				Reason: "provider profile interface contract is invalid",
			}
		}
	}
	c.providerProfile = slices.Clone(profile)
	return c, nil
}

func (c Context) ProviderScalarABI() (ScalarABI, bool) {
	return c.providerScalar, c.providerScalar.Valid()
}

func (c Context) ProviderScalarRepresentation() bool {
	return c.providerScalarRepresentation
}

func (c Context) ProviderProfile() []gostdlib.ProviderCallableProfileInterface {
	return slices.Clone(c.providerProfile)
}

type DefinedValueRepresentationKind uint8

const (
	DefinedValueRepresentationInvalid DefinedValueRepresentationKind = iota
	DefinedValueRepresentationGeneratedWrapper
	DefinedValueRepresentationProviderCanonical
	DefinedValueRepresentationProviderOperations
	DefinedValueRepresentationGeneratedNumeric
)

func (k DefinedValueRepresentationKind) Valid() bool {
	return k >= DefinedValueRepresentationGeneratedWrapper &&
		k <= DefinedValueRepresentationGeneratedNumeric
}

type DefinedValueRepresentation struct {
	kind       DefinedValueRepresentationKind
	operations NameReference
}

func NewDefinedValueRepresentation(
	kind DefinedValueRepresentationKind,
	operations NameReference,
) (DefinedValueRepresentation, error) {
	hasOperations := operations.Name() != ""
	if !kind.Valid() ||
		(kind == DefinedValueRepresentationProviderOperations) != hasOperations {
		return DefinedValueRepresentation{}, &NameError{
			Name:   operations.Name(),
			Reason: "defined-value representation is invalid",
		}
	}
	return DefinedValueRepresentation{kind: kind, operations: operations}, nil
}

func (r DefinedValueRepresentation) Kind() DefinedValueRepresentationKind {
	return r.kind
}

func (r DefinedValueRepresentation) Operations() (NameReference, bool) {
	return r.operations, r.operations.Name() != ""
}

const TargetGlobalAnchorName = "globalThis"

type TargetIntrinsic uint8

const (
	TargetIntrinsicInvalid TargetIntrinsic = iota
	TargetIntrinsicNumber
	TargetIntrinsicString
	TargetIntrinsicBigInt
	TargetIntrinsicMath
	TargetIntrinsicObject
	TargetIntrinsicPromise
	TargetIntrinsicError
)

func (i TargetIntrinsic) Valid() bool {
	return i >= TargetIntrinsicNumber && i <= TargetIntrinsicError
}

func (i TargetIntrinsic) String() string {
	switch i {
	case TargetIntrinsicNumber:
		return "Number"
	case TargetIntrinsicString:
		return "String"
	case TargetIntrinsicBigInt:
		return "BigInt"
	case TargetIntrinsicMath:
		return "Math"
	case TargetIntrinsicObject:
		return "Object"
	case TargetIntrinsicPromise:
		return "Promise"
	case TargetIntrinsicError:
		return "Error"
	default:
		return fmt.Sprintf("target-intrinsic(%d)", i)
	}
}

func (i TargetIntrinsic) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	if !i.Valid() {
		panic("invalid target intrinsic")
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(TargetGlobalAnchorName),
		nil,
		factory.Identifier(i.String()),
		tsgo.NodeFlagsNone,
	)
}

func (i TargetIntrinsic) UnshadowedExpression(
	factory tsgo.Factory,
) tsgo.Identifier {
	if !i.Valid() {
		panic("invalid target intrinsic")
	}
	return factory.Identifier(i.String())
}

func (i TargetIntrinsic) ReservesTypeName() bool {
	return i == TargetIntrinsicObject || i == TargetIntrinsicPromise
}

func IsReservedTargetTypeName(name string) bool {
	return name == TargetIntrinsicObject.String() ||
		name == TargetIntrinsicPromise.String()
}

func (i TargetIntrinsic) TypeName(factory tsgo.Factory) tsgo.Identifier {
	if !i.ReservesTypeName() {
		panic("invalid target intrinsic")
	}
	return factory.Identifier(i.String())
}

type ProviderInterfaceCapabilityReference struct {
	base        NameReference
	view        NameReference
	target      NameReference
	certificate gostdlib.ProviderInterfaceCapability
}

func NewProviderInterfaceCapabilityReference(
	base NameReference,
	view NameReference,
	target NameReference,
	certificate gostdlib.ProviderInterfaceCapability,
) (ProviderInterfaceCapabilityReference, error) {
	if !certificate.Valid() || base.Name() != certificate.BaseExport() ||
		view.Name() != certificate.ViewExport() ||
		target.Name() != certificate.TargetExport() {
		return ProviderInterfaceCapabilityReference{}, &NameError{
			Name:   certificate.ViewExport(),
			Reason: "provider-interface capability reference is inconsistent",
		}
	}
	return ProviderInterfaceCapabilityReference{
		base:        base,
		view:        view,
		target:      target,
		certificate: certificate,
	}, nil
}

func (r ProviderInterfaceCapabilityReference) Base() NameReference {
	return r.base
}

func (r ProviderInterfaceCapabilityReference) View() NameReference {
	return r.view
}

func (r ProviderInterfaceCapabilityReference) Target() NameReference {
	return r.target
}

func (r ProviderInterfaceCapabilityReference) Certificate() (
	gostdlib.ProviderInterfaceCapability,
	bool,
) {
	return r.certificate, r.certificate.Valid()
}

func (r ProviderInterfaceCapabilityReference) Requests() []RootRequest {
	return CombineRequests(
		r.base.Requests(),
		r.view.Requests(),
		r.target.Requests(),
	)
}

func NewProviderInterfaceCapabilityRequest(
	artifact *GeneratedArtifact,
	contract *types.Interface,
	contractKey string,
) (RootRequest, error) {
	requirement, err := NewProviderInterfaceCapabilityRequirement(
		artifact,
		contract,
		contractKey,
	)
	return generatedDefinitionRequest(requirement, err)
}

type RecoveryCallableResolver interface {
	ObserveRecoveryCallable(
		Context,
		CallableFacet,
	) (RecoveryCallableObservation, error)
}

type RecoveryCallableObservation struct {
	recovery bool
	requests []RootRequest
}

func NewRecoveryCallableObservation(
	recovery bool,
	requests ...RootRequest,
) (RecoveryCallableObservation, error) {
	if err := validateReferenceRequests(requests); err != nil {
		return RecoveryCallableObservation{}, &RootRequestError{
			Reason: "recovery callable observation has an invalid request",
		}
	}
	return RecoveryCallableObservation{
		recovery: recovery,
		requests: slices.Clone(requests),
	}, nil
}

func (o RecoveryCallableObservation) Recovery() bool {
	return o.recovery
}

func (o RecoveryCallableObservation) Requests() []RootRequest {
	return slices.Clone(o.requests)
}

func (c Context) WithRecoveryCallableResolver(
	resolver RecoveryCallableResolver,
) Context {
	if resolver == nil {
		panic("recovery callable resolver is nil")
	}
	c.recoveryResolver = resolver
	return c
}

func (c Context) ObserveRecoveryCallable(
	facet CallableFacet,
) (RecoveryCallableObservation, error) {
	if c.recoveryResolver == nil {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable resolver is unavailable",
		}
	}
	if !facet.Valid() {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable facet is invalid",
		}
	}
	if !c.artifactOwner.Valid() {
		return RecoveryCallableObservation{}, &ContextError{
			Reason: "recovery callable consumer has no artifact owner",
		}
	}
	return c.recoveryResolver.ObserveRecoveryCallable(c, facet)
}

func GenericKernelRequired(
	owner *types.Func,
	requirements []DeclarationRequirement,
) (bool, error) {
	if owner == nil || owner.Origin() != owner ||
		len(GenericDeclarationParameters(owner)) == 0 {
		return false, &InvariantError{
			Reason: "generic kernel owner is invalid",
		}
	}
	required := false
	for _, requirement := range requirements {
		switch requirement.Kind() {
		case DeclarationRequirementGenericOperation:
			requirementOwner, _, ok := requirement.GenericOperation()
			if !ok || requirementOwner != owner {
				return false, &InvariantError{
					Reason: "generic operation has foreign kernel ownership",
				}
			}
			required = true
		case DeclarationRequirementGenericRepresentation:
			requirementOwner, _, _, ok :=
				requirement.GenericRepresentation()
			if !ok || requirementOwner != owner {
				return false, &InvariantError{
					Reason: "generic representation has foreign kernel ownership",
				}
			}
		}
	}
	return required, nil
}

func NewSideEffectImportRequest(
	factory tsgo.Factory,
	modulePath string,
) (RootRequest, error) {
	if modulePath == "" {
		return RootRequest{}, &RootRequestError{Reason: "module path is empty"}
	}
	return RootRequest{payload: &rootRequestPayload{
		owner: RootRequestOwner{
			kind:          RootRequestImport,
			importBinding: ImportBindingSideEffect,
			modulePath:    modulePath,
		},
		importPhase:     ImportPhaseValue,
		moduleSpecifier: factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
	}}, nil
}
