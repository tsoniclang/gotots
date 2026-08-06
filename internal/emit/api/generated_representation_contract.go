package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
)

type GoRuntimeType uint8

const (
	GoRuntimeTypeInvalid GoRuntimeType = iota
	GoRuntimeTypeBuiltinError
	GoRuntimeTypeError
	GoRuntimeTypePanicNilError
	GoRuntimeTypePanicNilPointer
)

func (k GoRuntimeType) Valid() bool {
	return k == GoRuntimeTypeBuiltinError ||
		k == GoRuntimeTypeError ||
		k == GoRuntimeTypePanicNilError ||
		k == GoRuntimeTypePanicNilPointer
}

type GoRuntimeContract interface {
	Owns(*types.Package) bool
	Classify(types.Type) GoRuntimeType
}

func (c Context) WithGoRuntimeContract(contract GoRuntimeContract) Context {
	if contract == nil {
		panic("Go runtime contract is nil")
	}
	c.goRuntime = contract
	return c
}

func (c Context) GoRuntimeType(sourceType types.Type) GoRuntimeType {
	if c.goRuntime == nil {
		return GoRuntimeTypeInvalid
	}
	return c.goRuntime.Classify(sourceType)
}

type GeneratedArtifactKind uint8

const (
	GeneratedArtifactInvalid GeneratedArtifactKind = iota
	GeneratedArtifactAnonymousStruct
	GeneratedArtifactMapSpecialization
	GeneratedArtifactInterfaceAdapter
	GeneratedArtifactAnonymousInterface
	GeneratedArtifactInterfaceMethodToken
	GeneratedArtifactInterfaceDynamicTypeToken
	GeneratedArtifactGenericCapability
	GeneratedArtifactCallableABI
	GeneratedArtifactInterfaceMethodCallable
	GeneratedArtifactPointerRepresentation
	GeneratedArtifactProviderInterfaceBridge
	GeneratedArtifactProviderStatefulRepresentation
	GeneratedArtifactDeferredCallableRegistry
	GeneratedArtifactGenericConcretization
	GeneratedArtifactReflectionType
	GeneratedArtifactUnsafeCodec
)

func (k GeneratedArtifactKind) Valid() bool {
	return k == GeneratedArtifactAnonymousStruct ||
		k == GeneratedArtifactMapSpecialization ||
		k == GeneratedArtifactInterfaceAdapter ||
		k == GeneratedArtifactAnonymousInterface ||
		k == GeneratedArtifactInterfaceMethodToken ||
		k == GeneratedArtifactInterfaceDynamicTypeToken ||
		k == GeneratedArtifactGenericCapability ||
		k == GeneratedArtifactCallableABI ||
		k == GeneratedArtifactInterfaceMethodCallable ||
		k == GeneratedArtifactPointerRepresentation ||
		k == GeneratedArtifactProviderInterfaceBridge ||
		k == GeneratedArtifactProviderStatefulRepresentation ||
		k == GeneratedArtifactDeferredCallableRegistry ||
		k == GeneratedArtifactGenericConcretization ||
		k == GeneratedArtifactReflectionType ||
		k == GeneratedArtifactUnsafeCodec
}

type GeneratedArtifactPlacement uint8

const (
	GeneratedArtifactPlacementInvalid GeneratedArtifactPlacement = iota
	GeneratedArtifactPlacementCompilation
	GeneratedArtifactPlacementLexical
	GeneratedArtifactPlacementContract
)

func (p GeneratedArtifactPlacement) Valid() bool {
	return p == GeneratedArtifactPlacementCompilation ||
		p == GeneratedArtifactPlacementLexical ||
		p == GeneratedArtifactPlacementContract
}

type GeneratedArtifact struct {
	kind            GeneratedArtifactKind
	sourceType      types.Type
	artifact        string
	targetName      string
	placement       GeneratedArtifactPlacement
	outputPath      string
	lexicalOwner    ArtifactOwner
	anchor          *types.TypeName
	generic         GenericOperationSelection
	runtime         RuntimeSymbol
	concretization  *GenericConcretization
	reflectionType  *types.TypeName
	providerProfile []gostdlib.ProviderCallableProfileInterface
}

func NewCompilationProviderProfileBridgeArtifact(
	sourceType *types.Named,
	profile []gostdlib.ProviderCallableProfileInterface,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if !validGeneratedArtifactType(
		GeneratedArtifactProviderInterfaceBridge,
		sourceType,
	) || len(profile) == 0 || artifact == "" || targetName == "" ||
		outputPath == "" {
		return nil, &RootRequestError{
			Reason: "compilation provider-profile bridge artifact is invalid",
		}
	}
	for _, selected := range profile {
		if !selected.Valid() {
			return nil, &RootRequestError{
				Reason: "compilation provider-profile bridge contract is invalid",
			}
		}
	}
	return &GeneratedArtifact{
		kind:            GeneratedArtifactProviderInterfaceBridge,
		sourceType:      sourceType,
		artifact:        artifact,
		targetName:      targetName,
		placement:       GeneratedArtifactPlacementCompilation,
		outputPath:      outputPath,
		providerProfile: slices.Clone(profile),
	}, nil
}

func NewCompilationReflectionTypeArtifact(
	sourceType types.Type,
	reflectionType *types.TypeName,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if sourceType == nil || reflectionType == nil || reflectionType.IsAlias() ||
		artifact == "" || targetName == "" || outputPath == "" {
		return nil, &RootRequestError{Reason: "reflection-type artifact is invalid"}
	}
	contract, ok := reflectionType.Type().Underlying().(*types.Interface)
	if !ok || !contract.Complete().IsMethodSet() {
		return nil, &RootRequestError{Reason: "reflection-type contract is invalid"}
	}
	return &GeneratedArtifact{
		kind:           GeneratedArtifactReflectionType,
		sourceType:     sourceType,
		artifact:       artifact,
		targetName:     targetName,
		placement:      GeneratedArtifactPlacementCompilation,
		outputPath:     outputPath,
		reflectionType: reflectionType,
	}, nil
}

func NewCompilationGeneratedArtifact(
	kind GeneratedArtifactKind,
	sourceType types.Type,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if kind == GeneratedArtifactGenericCapability ||
		kind == GeneratedArtifactCallableABI ||
		kind == GeneratedArtifactInterfaceMethodCallable ||
		kind == GeneratedArtifactInterfaceMethodToken ||
		!validGeneratedArtifactType(kind, sourceType) ||
		artifact == "" ||
		targetName == "" ||
		outputPath == "" {
		return nil, &RootRequestError{
			Reason: "compilation generated artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:       kind,
		sourceType: sourceType,
		artifact:   artifact,
		targetName: targetName,
		placement:  GeneratedArtifactPlacementCompilation,
		outputPath: outputPath,
	}, nil
}

func NewCompilationInterfaceMethodTokenArtifact(
	signature *types.Signature,
	artifact string,
	targetName string,
	outputPath string,
	runtime RuntimeSymbol,
) (*GeneratedArtifact, error) {
	if !validGeneratedArtifactType(
		GeneratedArtifactInterfaceMethodToken,
		signature,
	) ||
		ContainsGenericTypeParameter(signature) ||
		artifact == "" ||
		targetName == "" ||
		outputPath == "" ||
		!validInterfaceMethodRuntime(runtime) {
		return nil, &RootRequestError{
			Reason: "compilation interface-method-token artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:       GeneratedArtifactInterfaceMethodToken,
		sourceType: signature,
		artifact:   artifact,
		targetName: targetName,
		placement:  GeneratedArtifactPlacementCompilation,
		outputPath: outputPath,
		runtime:    runtime,
	}, nil
}

func NewContractGeneratedArtifact(
	kind GeneratedArtifactKind,
	sourceType types.Type,
	artifact string,
	targetName string,
) (*GeneratedArtifact, error) {
	if (kind != GeneratedArtifactCallableABI &&
		kind != GeneratedArtifactInterfaceMethodCallable &&
		kind != GeneratedArtifactPointerRepresentation) ||
		!validGeneratedArtifactType(kind, sourceType) ||
		artifact == "" ||
		targetName == "" {
		return nil, &RootRequestError{
			Reason: "contract generated artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:       kind,
		sourceType: sourceType,
		artifact:   artifact,
		targetName: targetName,
		placement:  GeneratedArtifactPlacementContract,
	}, nil
}

func NewLexicalGeneratedArtifact(
	kind GeneratedArtifactKind,
	sourceType types.Type,
	artifact string,
	targetName string,
	lexicalOwner ArtifactOwner,
	anchor *types.TypeName,
) (*GeneratedArtifact, error) {
	sourcePackage := lexicalOwner.Package()
	_, sourceOwned := lexicalOwner.Source()
	_, _, initializerOwned := lexicalOwner.PackageInitializer()
	if kind == GeneratedArtifactGenericCapability ||
		kind == GeneratedArtifactCallableABI ||
		kind == GeneratedArtifactInterfaceMethodCallable ||
		!validGeneratedArtifactType(kind, sourceType) ||
		artifact == "" ||
		targetName == "" ||
		(!sourceOwned && !initializerOwned) ||
		anchor == nil ||
		sourcePackage == nil ||
		anchor.Pkg() != sourcePackage ||
		anchor.Parent() == nil ||
		anchor.Parent() == anchor.Pkg().Scope() {
		return nil, &RootRequestError{
			Reason: "lexical generated artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:         kind,
		sourceType:   sourceType,
		artifact:     artifact,
		targetName:   targetName,
		placement:    GeneratedArtifactPlacementLexical,
		lexicalOwner: lexicalOwner,
		anchor:       anchor,
	}, nil
}

func NewCompilationGenericCapabilityArtifact(
	selection GenericOperationSelection,
	signature *types.Signature,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if !selection.Valid() ||
		!validGenericOperationSignature(signature) ||
		artifact == "" ||
		targetName == "" ||
		outputPath == "" {
		return nil, &RootRequestError{
			Reason: "compilation generic-capability artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:       GeneratedArtifactGenericCapability,
		sourceType: signature,
		artifact:   artifact,
		targetName: targetName,
		placement:  GeneratedArtifactPlacementCompilation,
		outputPath: outputPath,
		generic:    selection,
	}, nil
}

func NewCompilationGenericConcretizationArtifact(
	concretization *GenericConcretization,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if !concretization.Valid() ||
		concretization.Placement() != GeneratedArtifactPlacementCompilation ||
		artifact == "" || targetName == "" || outputPath == "" {
		return nil, &RootRequestError{
			Reason: "compilation generic-concretization artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:           GeneratedArtifactGenericConcretization,
		sourceType:     concretization.Signature(),
		artifact:       artifact,
		targetName:     targetName,
		placement:      GeneratedArtifactPlacementCompilation,
		outputPath:     outputPath,
		concretization: concretization,
	}, nil
}

func NewLexicalGenericConcretizationArtifact(
	concretization *GenericConcretization,
	artifact string,
	targetName string,
) (*GeneratedArtifact, error) {
	if !concretization.Valid() ||
		concretization.Placement() != GeneratedArtifactPlacementLexical ||
		artifact == "" || targetName == "" {
		return nil, &RootRequestError{
			Reason: "lexical generic-concretization artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:           GeneratedArtifactGenericConcretization,
		sourceType:     concretization.Signature(),
		artifact:       artifact,
		targetName:     targetName,
		placement:      GeneratedArtifactPlacementLexical,
		lexicalOwner:   concretization.LexicalOwner(),
		anchor:         concretization.LexicalAnchor(),
		concretization: concretization,
	}, nil
}

func NewLexicalGenericCapabilityArtifact(
	selection GenericOperationSelection,
	signature *types.Signature,
	artifact string,
	targetName string,
	lexicalOwner ArtifactOwner,
	anchor *types.TypeName,
) (*GeneratedArtifact, error) {
	sourcePackage := lexicalOwner.Package()
	_, sourceOwned := lexicalOwner.Source()
	_, _, initializerOwned := lexicalOwner.PackageInitializer()
	if !selection.Valid() ||
		!validGenericOperationSignature(signature) ||
		artifact == "" ||
		targetName == "" ||
		(!sourceOwned && !initializerOwned) ||
		anchor == nil ||
		sourcePackage == nil ||
		anchor.Pkg() != sourcePackage ||
		anchor.Parent() == nil ||
		anchor.Parent() == anchor.Pkg().Scope() {
		return nil, &RootRequestError{
			Reason: "lexical generic-capability artifact is invalid",
		}
	}
	return &GeneratedArtifact{
		kind:         GeneratedArtifactGenericCapability,
		sourceType:   signature,
		artifact:     artifact,
		targetName:   targetName,
		placement:    GeneratedArtifactPlacementLexical,
		lexicalOwner: lexicalOwner,
		anchor:       anchor,
		generic:      selection,
	}, nil
}

type PointerRepresentation uint8

const (
	PointerRepresentationInvalid          PointerRepresentation = 0
	PointerRepresentationDirectClass      PointerRepresentation = 1
	PointerRepresentationCarrierLogical   PointerRepresentation = 2
	PointerRepresentationCarrierCanonical PointerRepresentation = 3
)

func (r PointerRepresentation) Valid() bool {
	return r == PointerRepresentationDirectClass ||
		r == PointerRepresentationCarrierLogical ||
		r == PointerRepresentationCarrierCanonical
}

func (r PointerRepresentation) String() string {
	switch r {
	case PointerRepresentationDirectClass:
		return "direct-class"
	case PointerRepresentationCarrierLogical:
		return "carrier-logical"
	case PointerRepresentationCarrierCanonical:
		return "carrier-canonical"
	default:
		return "invalid"
	}
}

type PointerRepresentationReference struct {
	artifact *GeneratedArtifact
	requests []RootRequest
}

func NewPointerRepresentationReference(
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (PointerRepresentationReference, error) {
	if _, ok := artifact.PointerRepresentation(); !ok {
		return PointerRepresentationReference{}, &RootRequestError{
			Reason: "pointer-representation reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return PointerRepresentationReference{}, &RootRequestError{
			Reason: "pointer-representation reference request is invalid",
		}
	}
	return PointerRepresentationReference{
		artifact: artifact,
		requests: slices.Clone(requests),
	}, nil
}

func (r PointerRepresentationReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r PointerRepresentationReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type PointerRepresentationObservation struct {
	representation PointerRepresentation
	requests       []RootRequest
}

func NewPointerRepresentationObservation(
	representation PointerRepresentation,
	requests ...RootRequest,
) (PointerRepresentationObservation, error) {
	if !representation.Valid() {
		return PointerRepresentationObservation{}, &RootRequestError{
			Reason: "pointer-representation observation is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return PointerRepresentationObservation{}, &RootRequestError{
			Reason: "pointer-representation observation request is invalid",
		}
	}
	return PointerRepresentationObservation{
		representation: representation,
		requests:       slices.Clone(requests),
	}, nil
}

func (o PointerRepresentationObservation) Representation() PointerRepresentation {
	return o.representation
}

func (o PointerRepresentationObservation) Requests() []RootRequest {
	return slices.Clone(o.requests)
}

type GeneratedArtifactPlacementError struct {
	TypeName string
	Reason   string
}

func (e *GeneratedArtifactPlacementError) Error() string {
	if e.TypeName == "" {
		return "place generated type: " + e.Reason
	}
	return fmt.Sprintf(
		"place generated type containing %q: %s",
		e.TypeName,
		e.Reason,
	)
}

type GeneratedArtifactShapeError struct {
	Artifact string
	Reason   string
}

func (e *GeneratedArtifactShapeError) Error() string {
	if e.Artifact == "" {
		return "emit generated type: " + e.Reason
	}
	return fmt.Sprintf(
		"emit generated type %q: %s",
		e.Artifact,
		e.Reason,
	)
}

type GeneratedArtifactError struct {
	Artifact *GeneratedArtifact
	Cause    error
}

func (e *GeneratedArtifactError) Error() string {
	if e.Artifact == nil {
		return fmt.Sprintf("emit generated artifact: %v", e.Cause)
	}
	anchor := ""
	if selected := e.Artifact.LexicalAnchor(); selected != nil {
		anchor = ", lexical anchor " + selected.Name()
	}
	operation := ""
	if _, selected, ok := e.Artifact.GenericCapability(); ok {
		operation = ", operation " + selected.Operation().String()
	}
	sourceType := types.TypeString(
		e.Artifact.SourceType(),
		func(sourcePackage *types.Package) string {
			if sourcePackage == nil {
				return ""
			}
			return sourcePackage.Path()
		},
	)
	return fmt.Sprintf(
		"emit generated artifact %q (kind %d, key %s, type %s, placement %d%s%s): %v",
		e.Artifact.TargetName(),
		e.Artifact.Kind(),
		e.Artifact.ArtifactKey(),
		sourceType,
		e.Artifact.Placement(),
		anchor,
		operation,
		e.Cause,
	)
}

func (e *GeneratedArtifactError) Unwrap() error {
	return e.Cause
}

func WrapGeneratedArtifactError(
	artifact *GeneratedArtifact,
	err error,
) error {
	if err == nil {
		return nil
	}
	return &GeneratedArtifactError{Artifact: artifact, Cause: err}
}
