package api

import (
	"fmt"
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
		k == GeneratedArtifactProviderStatefulRepresentation
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
	kind         GeneratedArtifactKind
	sourceType   types.Type
	artifact     string
	targetName   string
	placement    GeneratedArtifactPlacement
	outputPath   string
	lexicalOwner ArtifactOwner
	anchor       *types.TypeName
	generic      GenericOperationSelection
	runtime      RuntimeSymbol
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

func (o *GeneratedArtifact) Kind() GeneratedArtifactKind {
	if o == nil {
		return GeneratedArtifactInvalid
	}
	return o.kind
}

func (o *GeneratedArtifact) SourceType() types.Type {
	if o == nil {
		return nil
	}
	return o.sourceType
}

func (o *GeneratedArtifact) StructType() (*types.Struct, bool) {
	if o == nil || o.kind != GeneratedArtifactAnonymousStruct {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Struct)
	return source, ok
}

func (o *GeneratedArtifact) MapType() (*types.Map, bool) {
	if o == nil || o.kind != GeneratedArtifactMapSpecialization {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Map)
	return source, ok
}

func (o *GeneratedArtifact) InterfaceAdapterType() (types.Type, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceAdapter {
		return nil, false
	}
	return o.sourceType, interfaceAdapterType(o.sourceType)
}

func (o *GeneratedArtifact) InterfaceType() (*types.Interface, bool) {
	if o == nil || o.kind != GeneratedArtifactAnonymousInterface {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).Underlying().(*types.Interface)
	return source, ok && source.IsMethodSet()
}

func (o *GeneratedArtifact) InterfaceMethodSignature() (*types.Signature, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodToken {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Signature)
	return source, ok && source.Recv() == nil
}

func (o *GeneratedArtifact) InterfaceMethodCallableSignature() (
	*types.Signature,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodCallable {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Signature)
	return source, ok && source.Recv() == nil
}

func (o *GeneratedArtifact) InterfaceMethodRuntime() (RuntimeSymbol, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceMethodToken {
		return RuntimeInvalid, false
	}
	return o.runtime, true
}

func (o *GeneratedArtifact) InterfaceDynamicType() (types.Type, bool) {
	if o == nil || o.kind != GeneratedArtifactInterfaceDynamicTypeToken {
		return nil, false
	}
	return o.sourceType, interfaceAdapterType(o.sourceType)
}

func (o *GeneratedArtifact) GenericCapability() (
	*types.Signature,
	GenericOperationSelection,
	bool,
) {
	if o == nil || o.kind != GeneratedArtifactGenericCapability {
		return nil, GenericOperationSelection{}, false
	}
	signature, ok := types.Unalias(o.sourceType).(*types.Signature)
	return signature, o.generic, ok && o.generic.Valid()
}

func (o *GeneratedArtifact) CallableABI() (*types.Signature, bool) {
	if o == nil || o.kind != GeneratedArtifactCallableABI {
		return nil, false
	}
	signature, ok := types.Unalias(o.sourceType).(*types.Signature)
	return signature, ok && signature.Recv() == nil
}

func (o *GeneratedArtifact) PointerRepresentation() (*types.Pointer, bool) {
	if o == nil || o.kind != GeneratedArtifactPointerRepresentation {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Pointer)
	return source, ok
}

func (o *GeneratedArtifact) ProviderInterfaceBridgeType() (*types.Named, bool) {
	if o == nil || o.kind != GeneratedArtifactProviderInterfaceBridge {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Named)
	if !ok {
		return nil, false
	}
	_, interfaceType := source.Underlying().(*types.Interface)
	return source, interfaceType && source.Obj() != nil
}

func (o *GeneratedArtifact) ProviderStatefulRepresentationType() (*types.Named, bool) {
	if o == nil || o.kind != GeneratedArtifactProviderStatefulRepresentation {
		return nil, false
	}
	source, ok := types.Unalias(o.sourceType).(*types.Named)
	if !ok || source.Obj() == nil {
		return nil, false
	}
	_, interfaceType := source.Underlying().(*types.Interface)
	return source, !interfaceType
}

func (o *GeneratedArtifact) ArtifactKey() string {
	if o == nil {
		return ""
	}
	return o.artifact
}

func (o *GeneratedArtifact) TargetName() string {
	if o == nil {
		return ""
	}
	return o.targetName
}

func (o *GeneratedArtifact) Placement() GeneratedArtifactPlacement {
	if o == nil {
		return GeneratedArtifactPlacementInvalid
	}
	return o.placement
}

func (o *GeneratedArtifact) OutputPath() string {
	if o == nil {
		return ""
	}
	return o.outputPath
}

func (o *GeneratedArtifact) LexicalOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	return o.lexicalOwner
}

func (o *GeneratedArtifact) LexicalAnchor() *types.TypeName {
	if o == nil {
		return nil
	}
	return o.anchor
}

func (o *GeneratedArtifact) ReconstructionOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	if o.placement == GeneratedArtifactPlacementLexical {
		return o.lexicalOwner
	}
	return MustGeneratedArtifactOwner(o)
}

func (o *GeneratedArtifact) Valid() bool {
	if o == nil ||
		!validGeneratedArtifactType(o.kind, o.sourceType) ||
		o.artifact == "" ||
		o.targetName == "" ||
		!o.placement.Valid() ||
		(o.kind == GeneratedArtifactGenericCapability) != o.generic.Valid() ||
		(o.kind == GeneratedArtifactInterfaceMethodToken &&
			!validInterfaceMethodRuntime(o.runtime)) ||
		(o.kind != GeneratedArtifactInterfaceMethodToken &&
			o.runtime != RuntimeInvalid) {
		return false
	}
	switch o.placement {
	case GeneratedArtifactPlacementCompilation:
		return o.outputPath != "" &&
			!o.lexicalOwner.Valid() &&
			o.anchor == nil
	case GeneratedArtifactPlacementLexical:
		sourcePackage := o.lexicalOwner.Package()
		_, sourceOwned := o.lexicalOwner.Source()
		_, _, initializerOwned := o.lexicalOwner.PackageInitializer()
		return o.outputPath == "" &&
			(sourceOwned || initializerOwned) &&
			o.anchor != nil &&
			sourcePackage != nil &&
			o.anchor.Pkg() == sourcePackage &&
			o.anchor.Parent() != nil &&
			o.anchor.Parent() != o.anchor.Pkg().Scope()
	case GeneratedArtifactPlacementContract:
		return (o.kind == GeneratedArtifactCallableABI ||
			o.kind == GeneratedArtifactInterfaceMethodCallable ||
			o.kind == GeneratedArtifactPointerRepresentation) &&
			o.outputPath == "" &&
			!o.lexicalOwner.Valid() &&
			o.anchor == nil
	default:
		return false
	}
}

func validGeneratedArtifactType(
	kind GeneratedArtifactKind,
	sourceType types.Type,
) bool {
	if sourceType == nil || !kind.Valid() {
		return false
	}
	switch kind {
	case GeneratedArtifactAnonymousStruct:
		_, ok := types.Unalias(sourceType).(*types.Struct)
		return ok
	case GeneratedArtifactMapSpecialization:
		source, ok := types.Unalias(sourceType).(*types.Map)
		return ok && types.Comparable(source.Key())
	case GeneratedArtifactInterfaceAdapter:
		return interfaceAdapterType(sourceType)
	case GeneratedArtifactAnonymousInterface:
		source, ok := types.Unalias(sourceType).Underlying().(*types.Interface)
		return ok && source.IsMethodSet()
	case GeneratedArtifactInterfaceMethodToken:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok &&
			source.Recv() == nil &&
			!ContainsGenericTypeParameter(source)
	case GeneratedArtifactInterfaceMethodCallable:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source.Recv() == nil
	case GeneratedArtifactInterfaceDynamicTypeToken:
		return interfaceAdapterType(sourceType)
	case GeneratedArtifactGenericCapability:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && validGenericOperationSignature(source)
	case GeneratedArtifactCallableABI:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source.Recv() == nil
	case GeneratedArtifactPointerRepresentation:
		_, ok := types.Unalias(sourceType).(*types.Pointer)
		return ok
	case GeneratedArtifactProviderInterfaceBridge:
		source, ok := types.Unalias(sourceType).(*types.Named)
		if !ok || source.Obj() == nil {
			return false
		}
		contract, ok := source.Underlying().(*types.Interface)
		return ok && contract.Complete().IsMethodSet()
	case GeneratedArtifactProviderStatefulRepresentation:
		source, ok := types.Unalias(sourceType).(*types.Named)
		if !ok || source.Obj() == nil {
			return false
		}
		_, interfaceType := source.Underlying().(*types.Interface)
		return !interfaceType
	default:
		return false
	}
}

func interfaceAdapterType(sourceType types.Type) bool {
	if sourceType == nil {
		return false
	}
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Interface, *types.Tuple, *types.TypeParam, *types.Union:
		return false
	default:
		return true
	}
}

func validInterfaceMethodRuntime(symbol RuntimeSymbol) bool {
	return symbol == RuntimeInvalid ||
		symbol == RuntimeErrorMethodToken ||
		symbol == RuntimeRuntimeErrorToken
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
