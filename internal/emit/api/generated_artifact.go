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
)

func (k GeneratedArtifactKind) Valid() bool {
	return k == GeneratedArtifactAnonymousStruct ||
		k == GeneratedArtifactMapSpecialization ||
		k == GeneratedArtifactInterfaceAdapter ||
		k == GeneratedArtifactAnonymousInterface ||
		k == GeneratedArtifactInterfaceMethodToken ||
		k == GeneratedArtifactInterfaceDynamicTypeToken ||
		k == GeneratedArtifactGenericCapability
}

type GeneratedArtifactPlacement uint8

const (
	GeneratedArtifactPlacementInvalid GeneratedArtifactPlacement = iota
	GeneratedArtifactPlacementCompilation
	GeneratedArtifactPlacementLexical
)

func (p GeneratedArtifactPlacement) Valid() bool {
	return p == GeneratedArtifactPlacementCompilation ||
		p == GeneratedArtifactPlacementLexical
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
}

func NewCompilationGeneratedArtifact(
	kind GeneratedArtifactKind,
	sourceType types.Type,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if kind == GeneratedArtifactGenericCapability ||
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
		(o.kind == GeneratedArtifactGenericCapability) != o.generic.Valid() {
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
		return ok && source.Recv() == nil
	case GeneratedArtifactInterfaceDynamicTypeToken:
		return interfaceAdapterType(sourceType)
	case GeneratedArtifactGenericCapability:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && validGenericOperationSignature(source)
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
