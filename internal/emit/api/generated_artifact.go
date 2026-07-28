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
)

func (k GeneratedArtifactKind) Valid() bool {
	return k == GeneratedArtifactAnonymousStruct ||
		k == GeneratedArtifactMapSpecialization
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
}

func NewCompilationGeneratedArtifact(
	kind GeneratedArtifactKind,
	sourceType types.Type,
	artifact string,
	targetName string,
	outputPath string,
) (*GeneratedArtifact, error) {
	if !validGeneratedArtifactType(kind, sourceType) ||
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
	if !validGeneratedArtifactType(kind, sourceType) ||
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
		!o.placement.Valid() {
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
	default:
		return false
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
