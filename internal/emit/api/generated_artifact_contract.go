package api

import (
	"go/types"
)

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
	kind           GeneratedArtifactKind
	sourceType     types.Type
	artifact       string
	targetName     string
	placement      GeneratedArtifactPlacement
	outputPath     string
	lexicalOwner   ArtifactOwner
	anchor         *types.TypeName
	generic        GenericOperationSelection
	runtime        RuntimeSymbol
	concretization *GenericConcretization
	reflectionType *types.TypeName
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
